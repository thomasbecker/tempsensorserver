package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	defaultPort         = "8080"
	defaultPollInterval = 10 * time.Second
	defaultW1Path       = "/sys/devices/w1_bus_master1"
)

type sensorResponse struct {
	Sensors []Sensor `json:"sensors"`
}

type server struct {
	cache     atomic.Value
	w1Path    string
	iioPath   string
	sensorMap map[string]string
}

func (s *server) poll() []Sensor {
	sensors := ReadAll(s.w1Path, s.iioPath, s.sensorMap)
	s.cache.Store(sensors)
	log.Printf("polled %d sensors", len(sensors))
	return sensors
}

func (s *server) handleSensors(w http.ResponseWriter, r *http.Request) {
	cached, _ := s.cache.Load().([]Sensor)
	if cached == nil {
		cached = []Sensor{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sensorResponse{Sensors: cached})
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	cached, _ := s.cache.Load().([]Sensor)
	status := "ok"
	if cached == nil || len(cached) == 0 {
		status = "no_data"
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"%s","sensors":%d}`, status, len(cached))
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func findIIODevice() string {
	candidates := []string{
		"/sys/bus/iio/devices/iio:device0",
		"/sys/bus/iio/devices/iio:device1",
	}
	if v := os.Getenv("IIO_DEVICE"); v != "" {
		candidates = []string{v}
	}
	for _, path := range candidates {
		if _, err := os.Stat(path + "/in_temp_input"); err == nil {
			log.Printf("found IIO device at %s", path)
			return path
		}
	}
	log.Println("no IIO device found, DHT22 disabled")
	return ""
}

func main() {
	port := envOrDefault("PORT", defaultPort)
	w1Path := envOrDefault("W1_PATH", defaultW1Path)
	sensorMap := ParseSensorMap(os.Getenv("SENSOR_MAP"))

	pollStr := envOrDefault("POLL_INTERVAL", "10")
	pollSec, err := strconv.Atoi(pollStr)
	if err != nil {
		pollSec = 10
	}
	pollInterval := time.Duration(pollSec) * time.Second

	iioPath := findIIODevice()

	if len(sensorMap) > 0 {
		log.Printf("sensor map: %v", sensorMap)
	}

	srv := &server{
		w1Path:    w1Path,
		iioPath:   iioPath,
		sensorMap: sensorMap,
	}

	var pusher *haPusher
	if haURL := os.Getenv("HA_URL"); haURL != "" {
		if haToken := os.Getenv("HA_TOKEN"); haToken != "" {
			pusher = NewHAPusher(haURL, haToken)
			log.Printf("HA push enabled: %s", haURL)
		}
	}

	sensors := srv.poll()
	if pusher != nil {
		go pusher.Push(sensors)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				sensors := srv.poll()
				if pusher != nil {
					go pusher.Push(sensors)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/sensors", srv.handleSensors)
	mux.HandleFunc("/health", srv.handleHealth)

	httpSrv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("starting on :%s (poll every %s, w1=%s, iio=%s)",
			port, pollInterval, w1Path, iioPath)
		if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutdownCtx)
}
