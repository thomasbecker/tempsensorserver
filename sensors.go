package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Sensor struct {
	ID    string `json:"id"`
	Value string `json:"value"`
}

var tempRegexp = regexp.MustCompile(`(?m)t=(-?\d+)\s*$`)

func ReadDS18B20(basePath string) []Sensor {
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

		value := fmt.Sprintf("%.3f", float64(millideg)/1000.0)
		sensors = append(sensors, Sensor{
			ID:    strconv.Itoa(i),
			Value: value,
		})
	}

	return sensors
}

func ReadDHT22(iioPath string) []Sensor {
	if iioPath == "" {
		return nil
	}

	tempFile := filepath.Join(iioPath, "in_temp_input")
	humFile := filepath.Join(iioPath, "in_humidityrelative_input")

	var sensors []Sensor

	if val, err := readIIOValue(tempFile); err == nil {
		sensors = append(sensors, Sensor{
			ID:    "100",
			Value: val,
		})
	} else {
		log.Printf("error reading DHT22 temp: %v", err)
	}

	if val, err := readIIOValue(humFile); err == nil {
		sensors = append(sensors, Sensor{
			ID:    "101",
			Value: val,
		})
	} else {
		log.Printf("error reading DHT22 humidity: %v", err)
	}

	return sensors
}

func readIIOValue(path string) (string, error) {
	data, err := os.ReadFile(path)
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

func ReadAll(w1Path, iioPath string) []Sensor {
	sensors := ReadDS18B20(w1Path)
	sensors = append(sensors, ReadDHT22(iioPath)...)
	return sensors
}
