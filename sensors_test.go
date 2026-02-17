package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDS18B20(t *testing.T) {
	sensors := ReadDS18B20("testdata/w1_bus_master1")

	if len(sensors) != 4 {
		t.Fatalf("expected 4 sensors, got %d", len(sensors))
	}

	expected := []struct {
		id, value string
	}{
		{"0", "48.750"},
		{"1", "22.875"},
		{"2", "46.250"},
		{"3", "21.437"},
	}

	for i, want := range expected {
		got := sensors[i]
		if got.ID != want.id {
			t.Errorf("sensor %d: id = %q, want %q", i, got.ID, want.id)
		}
		if got.Value != want.value {
			t.Errorf("sensor %d: value = %q, want %q", i, got.Value, want.value)
		}
	}
}

func TestReadDS18B20_CRCFailure(t *testing.T) {
	dir := t.TempDir()
	sensorDir := filepath.Join(dir, "28-0000000bad01")
	os.MkdirAll(sensorDir, 0755)
	os.WriteFile(
		filepath.Join(sensorDir, "w1_slave"),
		[]byte("33 00 4b 46 ff ff 02 10 f4 : crc=f4 NO\n33 00 4b 46 ff ff 02 10 f4 t=99999\n"),
		0644,
	)

	sensors := ReadDS18B20(dir)
	if len(sensors) != 0 {
		t.Errorf("expected 0 sensors on CRC failure, got %d", len(sensors))
	}
}

func TestReadDS18B20_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	sensors := ReadDS18B20(dir)
	if len(sensors) != 0 {
		t.Errorf("expected 0 sensors, got %d", len(sensors))
	}
}

func TestReadDS18B20_NegativeTemp(t *testing.T) {
	dir := t.TempDir()
	sensorDir := filepath.Join(dir, "28-0000000neg01")
	os.MkdirAll(sensorDir, 0755)
	os.WriteFile(
		filepath.Join(sensorDir, "w1_slave"),
		[]byte("33 00 4b 46 ff ff 02 10 f4 : crc=f4 YES\n33 00 4b 46 ff ff 02 10 f4 t=-1250\n"),
		0644,
	)

	sensors := ReadDS18B20(dir)
	if len(sensors) != 1 {
		t.Fatalf("expected 1 sensor, got %d", len(sensors))
	}
	if sensors[0].Value != "-1.250" {
		t.Errorf("value = %q, want %q", sensors[0].Value, "-1.250")
	}
}

func TestReadDHT22(t *testing.T) {
	sensors := ReadDHT22("testdata/iio_device")

	if len(sensors) != 2 {
		t.Fatalf("expected 2 sensors, got %d", len(sensors))
	}

	if sensors[0].ID != "100" || sensors[0].Value != "21.3" {
		t.Errorf("temp sensor = %+v, want id=100 value=21.3", sensors[0])
	}
	if sensors[1].ID != "101" || sensors[1].Value != "49.3" {
		t.Errorf("humidity sensor = %+v, want id=101 value=49.3", sensors[1])
	}
}

func TestReadDHT22_NoDevice(t *testing.T) {
	sensors := ReadDHT22("")
	if sensors != nil {
		t.Errorf("expected nil for empty path, got %v", sensors)
	}
}

func TestReadDHT22_MissingFiles(t *testing.T) {
	dir := t.TempDir()
	sensors := ReadDHT22(dir)
	if len(sensors) != 0 {
		t.Errorf("expected 0 sensors, got %d", len(sensors))
	}
}

func TestReadAll(t *testing.T) {
	sensors := ReadAll("testdata/w1_bus_master1", "testdata/iio_device")
	if len(sensors) != 6 {
		t.Fatalf("expected 6 sensors, got %d", len(sensors))
	}
}

func TestReadAll_NoDHT(t *testing.T) {
	sensors := ReadAll("testdata/w1_bus_master1", "")
	if len(sensors) != 4 {
		t.Fatalf("expected 4 sensors (DS18B20 only), got %d", len(sensors))
	}
}
