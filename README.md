# tempsensorserver

Lightweight Go service for Raspberry Pi that serves
DS18B20 (1-Wire) and DHT22 temperature/humidity sensor
data as JSON on port 8080.

Replaces the previous Java/Jetty implementation. Key
improvement: background polling with caching — HTTP
requests never block on sensor I/O.

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

Environment variables (all optional):

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `POLL_INTERVAL` | `10` | Sensor poll interval (seconds) |
| `W1_PATH` | `/sys/devices/w1_bus_master1` | 1-Wire sysfs path |
| `IIO_DEVICE` | auto-detect | IIO device path for DHT22 |

## Endpoints

#### `GET /sensors`

```json
{
  "sensors": [
    {"id": "0", "value": "48.750"},
    {"id": "1", "value": "22.875"},
    {"id": "2", "value": "46.250"},
    {"id": "3", "value": "21.437"},
    {"id": "100", "value": "21.3"},
    {"id": "101", "value": "49.3"}
  ]
}
```

IDs 0-3: DS18B20 (sorted by device address).
IDs 100/101: DHT22 temperature/humidity.

#### `GET /health`

```json
{"status":"ok","sensors":6}
```

## Monitoring

Logs go to stdout/stderr (visible via `journalctl -u tempsensorserver`).

The systemd unit auto-restarts on failure with a 5s delay
and has a 32MB memory limit.
