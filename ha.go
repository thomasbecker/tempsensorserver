package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type sensorMeta struct {
	EntityID     string
	FriendlyName string
	Unit         string
	DeviceClass  string
}

var sensorMetaMap = map[string]sensorMeta{
	"hot_water_middle": {
		EntityID:     "sensor.warmwasser_mitte",
		FriendlyName: "Warmwasser Mitte",
		Unit:         "°C",
		DeviceClass:  "temperature",
	},
	"heating_supply": {
		EntityID:     "sensor.heizung_vorlauf",
		FriendlyName: "Heizung Vorlauf",
		Unit:         "°C",
		DeviceClass:  "temperature",
	},
	"hot_water_bottom": {
		EntityID:     "sensor.warmwasser_unten",
		FriendlyName: "Warmwasser Unten",
		Unit:         "°C",
		DeviceClass:  "temperature",
	},
	"heating_return": {
		EntityID:     "sensor.heizung_rucklauf",
		FriendlyName: "Heizung Rücklauf",
		Unit:         "°C",
		DeviceClass:  "temperature",
	},
	"utility_room_temperature": {
		EntityID:     "sensor.technikraum_temperatur",
		FriendlyName: "Technikraum Temperatur",
		Unit:         "°C",
		DeviceClass:  "temperature",
	},
	"utility_room_humidity": {
		EntityID:     "sensor.technikraum_luftfeuchtigkeit",
		FriendlyName: "Technikraum Luftfeuchtigkeit",
		Unit:         "%",
		DeviceClass:  "humidity",
	},
}

type haPayload struct {
	State      string            `json:"state"`
	Attributes map[string]string `json:"attributes"`
}

type haPusher struct {
	url      string
	token    string
	client   *http.Client
	mu       sync.Mutex
	failures int
}

func NewHAPusher(url, token string) *haPusher {
	return &haPusher{
		url:   url,
		token: token,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (p *haPusher) Push(sensors []Sensor) {
	if !p.mu.TryLock() {
		log.Println("ha: push still in progress, skipping")
		return
	}
	defer p.mu.Unlock()

	pushed := 0
	for _, s := range sensors {
		meta, ok := sensorMetaMap[s.ID]
		if !ok {
			continue
		}
		if err := p.pushSensor(s, meta); err != nil {
			p.failures++
			if p.failures == 1 || p.failures%10 == 0 {
				log.Printf("ha: push %s failed (%d consecutive): %v",
					meta.EntityID, p.failures, err)
			}
			continue
		}
		pushed++
	}

	if pushed > 0 && p.failures > 0 {
		log.Printf("ha: recovered after %d failures", p.failures)
	}
	if pushed > 0 {
		p.failures = 0
	}
	log.Printf("ha: pushed %d sensors", pushed)
}

func (p *haPusher) pushSensor(s Sensor, meta sensorMeta) error {
	val, err := strconv.ParseFloat(s.Value, 64)
	if err != nil {
		return fmt.Errorf("parse value %q: %w", s.Value, err)
	}

	payload := haPayload{
		State: fmt.Sprintf("%.1f", val),
		Attributes: map[string]string{
			"friendly_name":       meta.FriendlyName,
			"unit_of_measurement": meta.Unit,
			"device_class":        meta.DeviceClass,
			"state_class":         "measurement",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	url := p.url + "/api/states/" + meta.EntityID
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status %d for %s", resp.StatusCode, meta.EntityID)
	}

	return nil
}
