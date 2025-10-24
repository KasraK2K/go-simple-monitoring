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

## Example Request

```bash
curl -X POST "<DOMAIN>/monitoring" -H "Content-Type: application/json" -H "Authorization: Bearer <TOKEN>"
```
