package main

import (
    "encoding/json"
    "fmt"
    "time"

    "go-log/internal/api/logics"
    "go-log/internal/config"
    "go-log/internal/utils"
)

func main() {
    config.InitEnvConfig()
    utils.InitTimeConfig()
    logics.InitMonitoringConfig()

    from := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
    to := time.Now().UTC().Format(time.RFC3339)

    data, err := logics.MonitoringDataGeneratorWithTableFilter("", from, to)
    if err != nil {
        panic(err)
    }

    fmt.Printf("entries: %d\n", len(data))
    if len(data) > 0 {
        b, _ := json.MarshalIndent(data[0], "", "  ")
        fmt.Println(string(b))
    }
}
