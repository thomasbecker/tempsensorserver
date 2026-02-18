package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPushSensor_Success(t *testing.T) {
	var gotMethod, gotPath, gotAuth, gotContentType string
	var gotPayload haPayload

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")
	p.Push([]Sensor{
		{ID: "hot_water_middle", Value: "48.750"},
	})

	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/api/states/sensor.warmwasser_mitte" {
		t.Errorf("path = %q, want /api/states/sensor.warmwasser_mitte", gotPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("auth = %q, want Bearer test-token", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("content-type = %q, want application/json", gotContentType)
	}
	if gotPayload.State != "48.8" {
		t.Errorf("state = %q, want 48.8", gotPayload.State)
	}
	if gotPayload.Attributes["friendly_name"] != "Warmwasser Mitte" {
		t.Errorf("friendly_name = %q, want Warmwasser Mitte",
			gotPayload.Attributes["friendly_name"])
	}
	if gotPayload.Attributes["unit_of_measurement"] != "°C" {
		t.Errorf("unit = %q, want °C",
			gotPayload.Attributes["unit_of_measurement"])
	}
	if gotPayload.Attributes["device_class"] != "temperature" {
		t.Errorf("device_class = %q, want temperature",
			gotPayload.Attributes["device_class"])
	}
	if gotPayload.Attributes["state_class"] != "measurement" {
		t.Errorf("state_class = %q, want measurement",
			gotPayload.Attributes["state_class"])
	}
}

func TestPush_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")
	p.Push([]Sensor{
		{ID: "hot_water_middle", Value: "48.750"},
	})

	if p.failures != 1 {
		t.Errorf("failures = %d, want 1", p.failures)
	}

	p.Push([]Sensor{
		{ID: "hot_water_middle", Value: "48.750"},
	})

	if p.failures != 2 {
		t.Errorf("failures = %d, want 2", p.failures)
	}
}

func TestPush_Unreachable(t *testing.T) {
	p := NewHAPusher("http://192.0.2.1:1", "test-token")
	p.client.Timeout = 100 * time.Millisecond

	p.Push([]Sensor{
		{ID: "hot_water_middle", Value: "48.750"},
	})

	if p.failures != 1 {
		t.Errorf("failures = %d, want 1", p.failures)
	}
}

func TestPush_SkipWhenBusy(t *testing.T) {
	blocked := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blocked
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.Push([]Sensor{{ID: "hot_water_middle", Value: "48.750"}})
	}()

	time.Sleep(50 * time.Millisecond)

	requestCount := 0
	p.Push([]Sensor{{ID: "hot_water_middle", Value: "48.750"}})

	close(blocked)
	wg.Wait()

	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts2.Close()
	p.url = ts2.URL
	p.Push([]Sensor{{ID: "hot_water_middle", Value: "48.750"}})

	if requestCount != 1 {
		t.Errorf("requestCount = %d, want 1 (second push should have been skipped)", requestCount)
	}
}

func TestPush_UnknownSensors(t *testing.T) {
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")
	p.Push([]Sensor{
		{ID: "unknown_sensor_1", Value: "42.0"},
		{ID: "unknown_sensor_2", Value: "43.0"},
	})

	if requestCount != 0 {
		t.Errorf("requestCount = %d, want 0 for unknown sensors", requestCount)
	}
}

func TestPush_EmptySensors(t *testing.T) {
	requestCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")
	p.Push([]Sensor{})

	if requestCount != 0 {
		t.Errorf("requestCount = %d, want 0 for empty sensors", requestCount)
	}
}

func TestPush_AllSixSensors(t *testing.T) {
	paths := make(map[string]bool)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths[r.URL.Path] = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")
	p.Push([]Sensor{
		{ID: "hot_water_middle", Value: "48.750"},
		{ID: "heating_supply", Value: "42.500"},
		{ID: "hot_water_bottom", Value: "45.000"},
		{ID: "heating_return", Value: "38.125"},
		{ID: "utility_room_temperature", Value: "21.3"},
		{ID: "utility_room_humidity", Value: "49.3"},
	})

	expected := []string{
		"/api/states/sensor.warmwasser_mitte",
		"/api/states/sensor.heizung_vorlauf",
		"/api/states/sensor.warmwasser_unten",
		"/api/states/sensor.heizung_rucklauf",
		"/api/states/sensor.technikraum_temperatur",
		"/api/states/sensor.technikraum_luftfeuchtigkeit",
	}

	for _, path := range expected {
		if !paths[path] {
			t.Errorf("missing push to %s", path)
		}
	}

	if p.failures != 0 {
		t.Errorf("failures = %d, want 0", p.failures)
	}
}

func TestPush_HumidityAttributes(t *testing.T) {
	var gotPayload haPayload
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "luftfeuchtigkeit") {
			json.NewDecoder(r.Body).Decode(&gotPayload)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")
	p.Push([]Sensor{
		{ID: "utility_room_humidity", Value: "49.300"},
	})

	if gotPayload.State != "49.3" {
		t.Errorf("state = %q, want 49.3", gotPayload.State)
	}
	if gotPayload.Attributes["unit_of_measurement"] != "%" {
		t.Errorf("unit = %q, want %%", gotPayload.Attributes["unit_of_measurement"])
	}
	if gotPayload.Attributes["device_class"] != "humidity" {
		t.Errorf("device_class = %q, want humidity",
			gotPayload.Attributes["device_class"])
	}
}

func TestPush_FailureRecovery(t *testing.T) {
	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	p := NewHAPusher(ts.URL, "test-token")

	p.Push([]Sensor{{ID: "hot_water_middle", Value: "48.750"}})
	if p.failures != 1 {
		t.Errorf("after first push: failures = %d, want 1", p.failures)
	}

	p.Push([]Sensor{{ID: "hot_water_middle", Value: "48.750"}})
	if p.failures != 0 {
		t.Errorf("after recovery: failures = %d, want 0", p.failures)
	}
}
