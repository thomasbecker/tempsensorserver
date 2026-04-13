# tempsensorserver

Lightweight Go service for Raspberry Pi that reads
DS18B20 (1-Wire) and DHT22 temperature/humidity sensors,
serves them as JSON on port 8080, and pushes state
updates to Home Assistant.

Replaces the previous Java/Jetty implementation. Key
features: background polling with caching, DHT22 fault
tolerance (serves cached readings for up to 30 minutes
on read errors), and direct HA REST API integration.

## Build

```bash
make test    # run tests
make build   # cross-compile for Pi (linux/arm)
```

Requires Go 1.22+. Zero external dependencies.

## Deploy

```bash
make deploy  # scp binary + systemd unit, restart service
```

The deploy target copies the binary to
`/usr/local/bin/tempsensorserver` on the Pi and installs
the systemd unit.

## Pi Setup (one-time)

#### 1-Wire (DS18B20)

Already configured if the old service was running. The
kernel module `w1-gpio` and `w1-therm` must be loaded,
and `/boot/config.txt` must have:

```
dtoverlay=w1-gpio
```

#### DHT22 (optional)

Enable the kernel IIO driver:

```
# Add to /boot/config.txt (or /boot/firmware/config.txt)
dtoverlay=dht11,gpiopin=<PIN>
```

Replace `<PIN>` with the GPIO pin number the DHT22 data
line is connected to (check `/home/pirate/dht_out.py` on
the Pi for the current pin). Reboot after adding the
overlay.

The service auto-detects the IIO device at startup. If
no IIO device is found, it serves DS18B20 data only.

## Configuration

Environment variables (all optional unless noted):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `POLL_INTERVAL` | `10` | Sensor poll interval (seconds) |
| `W1_PATH` | `/sys/devices/w1_bus_master1` | 1-Wire sysfs path |
| `IIO_DEVICE` | auto-detect | IIO device path for DHT22 |
| `SENSOR_MAP` | _(none)_ | Map device addresses to named IDs (see below) |
| `HA_URL` | _(none)_ | Home Assistant base URL for state push |
| `HA_TOKEN` | _(none)_ | Long-lived access token for HA REST API |

#### SENSOR_MAP

Maps 1-Wire device addresses to human-readable sensor IDs:

```
SENSOR_MAP=28-02131ad2cdaa:hot_water_middle,28-02131a6a33aa:heating_supply,28-02131b0312aa:hot_water_bottom,28-02131aca29aa:heating_return
```

Without a map, sensors are numbered 0, 1, 2, ... by address order.

#### Home Assistant Push

When both `HA_URL` and `HA_TOKEN` are set, the service pushes
sensor states to HA's REST API after every poll cycle. Each
sensor is mapped to an HA entity (e.g. `sensor.warmwasser_mitte`)
with appropriate `device_class`, `unit_of_measurement`, and
`state_class` attributes. See `ha.go` for the full mapping.

## Endpoints

#### `GET /sensors`

```json
{
  "sensors": [
    {"id": "heating_supply", "value": "22.875"},
    {"id": "heating_return", "value": "21.125"},
    {"id": "hot_water_middle", "value": "46.312"},
    {"id": "hot_water_bottom", "value": "44.375"},
    {"id": "utility_room_temperature", "value": "20.6"},
    {"id": "utility_room_humidity", "value": "45.3"}
  ]
}
```

With `SENSOR_MAP`, IDs are human-readable names. Without it,
DS18B20 sensors are numbered 0, 1, 2, ... by address order.
DHT22 sensors are always `utility_room_temperature` and
`utility_room_humidity`.

#### `GET /health`

```json
{
  "status": "ok",
  "sensors": 6,
  "dht22_available": true,
  "dht22_age_seconds": 8
}
```

When DHT22 readings are served from cache after a read error:

```json
{
  "status": "ok",
  "sensors": 6,
  "dht22_stale": true,
  "dht22_age_seconds": 150,
  "dht22_available": true
}
```

| Field | Description |
|---|---|
| `status` | `ok` or `no_data` (no sensors at all) |
| `sensors` | Number of sensors in the last poll |
| `dht22_available` | Whether DHT22 data is being served (fresh or cached) |
| `dht22_stale` | `true` if DHT22 data is from cache (age > 30s) |
| `dht22_age_seconds` | Age of the DHT22 reading in seconds |

## DHT22 Fault Tolerance

The DHT22 sensor is read via the kernel IIO driver and is
prone to intermittent I/O errors and timeouts. To avoid
sensor count drops (6 → 4) that trigger monitoring alerts:

- On a successful read, the value is cached with a timestamp.
- On a read error, the last cached value is served if it is
  less than 30 minutes old. The `/health` endpoint reports
  `dht22_stale: true` with the age.
- After 30 minutes without a successful read, the cached
  value expires and the sensor is dropped from the response.

Log messages for observability:

```
DHT22 temp: serving cached value (age: 2m30s)
DHT22 temp: recovered after 3m0s
DHT22 temp: cached value expired (age: 30m0s > 30m0s)
```

## Monitoring

Logs go to stdout/stderr (visible via `journalctl -u tempsensorserver`).

The systemd unit auto-restarts on failure with a 5s delay
and has a 32MB memory limit.

Grafana alerts (provisioned in the homeassistant repo):

| Alert | What it checks |
|---|---|
| `pi-jetty-down` | >80% HTTP failure rate over 5min, sustained 10min |
| `pi-sensors-slow` | Average response time >10s over 5min, sustained 5min |

The generic `service-endpoint-down` alert excludes
`1wire-sensors` to avoid double notifications.
