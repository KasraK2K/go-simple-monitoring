# Log

## Run Project:

```bash
go run ./cmd
```

---

## Build for linux:

```bash
GOOS=linux GOARCH=amd64 \
CGO_ENABLED=0 \
go build \
  -ldflags="-s -w" \
  -trimpath \
  -o monitoring \
  ./cmd
```

---

## Log File Storage

The system automatically logs monitoring data to daily files in `YYYY-MM-DD.log` format based on the `configs.json` configuration.

### Storage Requirements

| Interval | Daily Size | Weekly Size | Monthly Size | Yearly Size |
| -------- | ---------- | ----------- | ------------ | ----------- |
| 2s       | ~59 MB     | ~413 MB     | ~1.8 GB      | ~21.5 GB    |
| 5s       | ~23.6 MB   | ~165 MB     | ~708 MB      | ~8.6 GB     |
| 10s      | ~11.8 MB   | ~83 MB      | ~354 MB      | ~4.3 GB     |

**Note**: Use the `CleanOldLogs()` function to automatically remove logs older than specified days to manage disk space.

---

## Example Request

```bash
curl -X POST "<DOMAIN>/monitoring" -H "Content-Type: application/json" -H "Authorization: Bearer <TOKEN>"
```
