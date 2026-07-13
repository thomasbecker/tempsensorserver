package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type DHTCache struct {
	Temp     *CachedReading
	Humidity *CachedReading
}

type CachedReading struct {
	Sensor Sensor
	ReadAt time.Time
}

func (c *CachedReading) Age() time.Duration {
	if c == nil {
		return 0
	}
	return time.Since(c.ReadAt)
}

func (c *CachedReading) Stale(maxAge time.Duration) bool {
	return c != nil && c.Age() > maxAge
}

type Sensor struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

var tempRegexp = regexp.MustCompile(`(?m)t=(-?\d+)\s*$`)

// ParseSensorMap parses "addr1:id1,addr2:id2,..." into
// a map from device directory name to sensor ID.
func ParseSensorMap(raw string) map[string]string {
	m := make(map[string]string)
	if raw == "" {
		return m
	}
	for _, entry := range strings.Split(raw, ",") {
		parts := strings.SplitN(entry, ":", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m
}

func ReadDS18B20(basePath string, sensorMap map[string]string) []Sensor {
	pattern := filepath.Join(basePath, "28-*")
	dirs, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("error globbing %s: %v", pattern, err)
		return nil
	}

	sort.Strings(dirs)

	var sensors []Sensor
	for i, dir := range dirs {
		path := filepath.Join(dir, "w1_slave")
		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("error reading %s: %v", path, err)
			continue
		}

		content := string(data)
		if !strings.Contains(content, "YES") {
			log.Printf("CRC check failed for %s", path)
			continue
		}

		match := tempRegexp.FindStringSubmatch(content)
		if match == nil {
			log.Printf("no temperature found in %s", path)
			continue
		}

		millideg, err := strconv.ParseInt(match[1], 10, 64)
		if err != nil {
			log.Printf("invalid temperature in %s: %v", path, err)
			continue
		}

		addr := filepath.Base(dir)
		id := strconv.Itoa(i)
		if mapped, ok := sensorMap[addr]; ok {
			id = mapped
		}

		value := fmt.Sprintf("%.3f", float64(millideg)/1000.0)
		sensors = append(sensors, Sensor{
			ID:    id,
			Value: value,
		})
	}

	return sensors
}

const DefaultDHTMaxAge = 30 * time.Minute

const (
	dhtReadAttempts = 3
	// The DHT22 needs >=2s between samples. Retrying sooner just re-reads a
	// sensor that is still settling, and fails again.
	dhtRetryDelay = 2200 * time.Millisecond
)

// Seams so the retry path can be tested without a real GPIO sensor.
var (
	dhtReadFile = os.ReadFile
	dhtSleep    = time.Sleep
)

// dhtTransient reports whether a DHT read failure is worth retrying.
//
// The kernel dht11 driver bit-bangs the GPIO line and loses edges under IRQ
// latency, so it fails with EIO/ETIMEDOUT ("Only 0 signal edges detected") on
// roughly 1 read in 6 even when the sensor is healthy. Those are transient. A
// missing device (ENOENT) is not — retrying it would stall every poll by the
// whole backoff budget for a sensor that simply is not there.
func dhtTransient(err error) bool {
	return errors.Is(err, syscall.EIO) || errors.Is(err, syscall.ETIMEDOUT)
}

// readDHTPair reads temperature and humidity together, retrying transient
// kernel failures. Retrying recovers a genuinely fresh reading; the cache
// below is only a fallback for when even the retries fail, and it serves a
// stale value, so it is strictly better to succeed here.
func readDHTPair(tempFile, humFile string) (string, error, string, error) {
	var temp, hum string
	var errT, errH error

	for attempt := 1; attempt <= dhtReadAttempts; attempt++ {
		temp, errT = readIIOValue(tempFile)
		hum, errH = readIIOValue(humFile)

		if errT == nil && errH == nil {
			break
		}
		if attempt == dhtReadAttempts || !(dhtTransient(errT) || dhtTransient(errH)) {
			break
		}
		dhtSleep(dhtRetryDelay)
	}

	return temp, errT, hum, errH
}

func ReadDHT22Cached(iioPath string, prev *DHTCache, maxAge time.Duration) (*DHTCache, []Sensor) {
	if iioPath == "" {
		return prev, nil
	}

	cache := &DHTCache{}
	if prev != nil {
		cache.Temp = prev.Temp
		cache.Humidity = prev.Humidity
	}

	tempFile := filepath.Join(iioPath, "in_temp_input")
	humFile := filepath.Join(iioPath, "in_humidityrelative_input")
	now := time.Now()

	tempVal, tempErr, humVal, humErr := readDHTPair(tempFile, humFile)

	if tempErr == nil {
		cache.Temp = &CachedReading{
			Sensor: Sensor{ID: "utility_room_temperature", Value: tempVal},
			ReadAt: now,
		}
		if prev != nil && prev.Temp != nil && prev.Temp.ReadAt != now {
			if prev.Temp.Age() > 30*time.Second {
				log.Printf("DHT22 temp: recovered after %s", prev.Temp.Age().Truncate(time.Second))
			}
		}
	} else {
		log.Printf("error reading DHT22 temp after %d attempts: %v", dhtReadAttempts, tempErr)
		if cache.Temp != nil {
			log.Printf("DHT22 temp: serving cached value (age: %s)", cache.Temp.Age().Truncate(time.Second))
		}
	}

	if humErr == nil {
		cache.Humidity = &CachedReading{
			Sensor: Sensor{ID: "utility_room_humidity", Value: humVal},
			ReadAt: now,
		}
		if prev != nil && prev.Humidity != nil && prev.Humidity.ReadAt != now {
			if prev.Humidity.Age() > 30*time.Second {
				log.Printf("DHT22 humidity: recovered after %s", prev.Humidity.Age().Truncate(time.Second))
			}
		}
	} else {
		log.Printf("error reading DHT22 humidity after %d attempts: %v", dhtReadAttempts, humErr)
		if cache.Humidity != nil {
			log.Printf("DHT22 humidity: serving cached value (age: %s)", cache.Humidity.Age().Truncate(time.Second))
		}
	}

	var sensors []Sensor
	if cache.Temp != nil && !cache.Temp.Stale(maxAge) {
		sensors = append(sensors, cache.Temp.Sensor)
	} else if cache.Temp != nil && cache.Temp.Stale(maxAge) {
		log.Printf("DHT22 temp: cached value expired (age: %s > %s)", cache.Temp.Age().Truncate(time.Second), maxAge)
		cache.Temp = nil
	}
	if cache.Humidity != nil && !cache.Humidity.Stale(maxAge) {
		sensors = append(sensors, cache.Humidity.Sensor)
	} else if cache.Humidity != nil && cache.Humidity.Stale(maxAge) {
		log.Printf("DHT22 humidity: cached value expired (age: %s > %s)", cache.Humidity.Age().Truncate(time.Second), maxAge)
		cache.Humidity = nil
	}

	return cache, sensors
}

func readIIOValue(path string) (string, error) {
	data, err := dhtReadFile(path)
	if err != nil {
		return "", err
	}

	raw := strings.TrimSpace(string(data))
	milliVal, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid value %q in %s: %w", raw, path, err)
	}

	return fmt.Sprintf("%.1f", float64(milliVal)/1000.0), nil
}

func ReadAll(w1Path, iioPath string, sensorMap map[string]string, dhtCache *DHTCache, dhtMaxAge time.Duration) (*DHTCache, []Sensor) {
	sensors := ReadDS18B20(w1Path, sensorMap)
	newCache, dhtSensors := ReadDHT22Cached(iioPath, dhtCache, dhtMaxAge)
	sensors = append(sensors, dhtSensors...)
	return newCache, sensors
}
