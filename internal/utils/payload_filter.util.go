package utils

import (
    "encoding/json"
)

// FilterMonitoringPayload trims a remote /api/v1/monitoring JSON payload
// to only include fields used by the dashboard UI. It accepts payloads in the
// following shapes and returns a semantically equivalent payload:
//   - [ {SystemMonitoring}, ... ]
//   - { data: [ {SystemMonitoring}, ... ] }
// If filtering fails for any reason, the function returns the original payload and no error.
func FilterMonitoringPayload(payload []byte) ([]byte, error) {
    // Try array of objects first
    var arr []map[string]any
    if err := json.Unmarshal(payload, &arr); err == nil && len(arr) > 0 {
        trimmed := make([]map[string]any, 0, len(arr))
        for _, item := range arr {
            trimmed = append(trimmed, trimMonitoringItem(item))
        }
        out, err := json.Marshal(trimmed)
        if err != nil {
            return payload, nil
        }
        return out, nil
    }

    // Try wrapper { data: [...] } – return as array to keep downstream format consistent
    var wrapper map[string]any
    if err := json.Unmarshal(payload, &wrapper); err == nil {
        if data, ok := wrapper["data"]; ok {
            if rawArr, ok := data.([]any); ok && len(rawArr) > 0 {
                trimmed := make([]map[string]any, 0, len(rawArr))
                for _, v := range rawArr {
                    if m, ok := v.(map[string]any); ok {
                        trimmed = append(trimmed, trimMonitoringItem(m))
                    }
                }
                // Return just the array to match expected payload shape
                out, err := json.Marshal(trimmed)
                if err != nil {
                    return payload, nil
                }
                return out, nil
            }
        }
    }

    // Fallback – keep original
    return payload, nil
}

func trimMonitoringItem(item map[string]any) map[string]any {
    result := make(map[string]any)

    // timestamp/time
    if ts, ok := item["timestamp"]; ok {
        result["timestamp"] = ts
    } else if t, ok := item["time"]; ok {
        result["timestamp"] = t
    }

    // cpu.usage_percent or fallbacks
    if cpu, ok := asObject(item["cpu"]); ok {
        if val, ok := cpu["usage_percent"]; ok {
            result["cpu"] = map[string]any{"usage_percent": val}
        }
    }
    if _, ok := result["cpu"]; !ok {
        if v, ok := item["cpu_usage"]; ok {
            result["cpu"] = map[string]any{"usage_percent": v}
        } else if v, ok := item["cpu_usage_percent"]; ok {
            result["cpu"] = map[string]any{"usage_percent": v}
        }
    }

    // ram.used_pct or fallback
    if ram, ok := asObject(item["ram"]); ok {
        if val, ok := ram["used_pct"]; ok {
            result["ram"] = map[string]any{"used_pct": val}
        }
    }
    if _, ok := result["ram"]; !ok {
        if v, ok := item["ram_used_percent"]; ok {
            result["ram"] = map[string]any{"used_pct": v}
        }
    }

    // disk_space array – keep only essential fields
    if arr, ok := asArray(item["disk_space"]); ok && len(arr) > 0 {
        disks := make([]map[string]any, 0, len(arr))
        for _, d := range arr {
            if disk, ok := asObject(d); ok {
                trimmed := map[string]any{}
                if v, ok := disk["path"]; ok { trimmed["path"] = v }
                if v, ok := disk["device"]; ok { trimmed["device"] = v }
                if v, ok := disk["filesystem"]; ok { trimmed["filesystem"] = v }
                if v, ok := disk["total_bytes"]; ok { trimmed["total_bytes"] = v }
                if v, ok := disk["used_bytes"]; ok { trimmed["used_bytes"] = v }
                if v, ok := disk["available_bytes"]; ok { trimmed["available_bytes"] = v }
                if v, ok := disk["used_pct"]; ok { trimmed["used_pct"] = v }
                disks = append(disks, trimmed)
            }
        }
        if len(disks) > 0 {
            result["disk_space"] = disks
        }
    }

    // network_io.bytes_recv / bytes_sent or flat fallbacks
    if nio, ok := asObject(item["network_io"]); ok {
        trimmed := map[string]any{}
        if v, ok := nio["bytes_recv"]; ok { trimmed["bytes_recv"] = v }
        if v, ok := nio["bytes_sent"]; ok { trimmed["bytes_sent"] = v }
        if len(trimmed) > 0 {
            result["network_io"] = trimmed
        }
    } else {
        trimmed := map[string]any{}
        if v, ok := item["network_bytes_recv"]; ok { trimmed["bytes_recv"] = v }
        if v, ok := item["network_bytes_sent"]; ok { trimmed["bytes_sent"] = v }
        if len(trimmed) > 0 {
            result["network_io"] = trimmed
        }
    }

    // process.load_avg_*
    if proc, ok := asObject(item["process"]); ok {
        trimmed := map[string]any{}
        if v, ok := proc["load_avg_1"]; ok { trimmed["load_avg_1"] = v }
        if v, ok := proc["load_avg_5"]; ok { trimmed["load_avg_5"] = v }
        if v, ok := proc["load_avg_15"]; ok { trimmed["load_avg_15"] = v }
        if len(trimmed) > 0 {
            result["process"] = trimmed
        }
    } else {
        trimmed := map[string]any{}
        if v, ok := item["process_load_avg_1"]; ok { trimmed["load_avg_1"] = v }
        if v, ok := item["process_load_avg_5"]; ok { trimmed["load_avg_5"] = v }
        if v, ok := item["process_load_avg_15"]; ok { trimmed["load_avg_15"] = v }
        if len(trimmed) > 0 {
            result["process"] = trimmed
        }
    }

    // heartbeat – keep UI fields
    if hbArr, ok := asArray(item["heartbeat"]); ok && len(hbArr) > 0 {
        out := make([]map[string]any, 0, len(hbArr))
        for _, e := range hbArr {
            if m, ok := asObject(e); ok {
                trimmed := map[string]any{}
                if v, ok := m["name"]; ok { trimmed["name"] = v }
                if v, ok := m["url"]; ok { trimmed["url"] = v }
                if v, ok := m["status"]; ok { trimmed["status"] = v }
                if v, ok := m["response_ms"]; ok { trimmed["response_ms"] = v }
                if v, ok := m["response_time"]; ok { trimmed["response_time"] = v }
                if v, ok := m["last_checked"]; ok { trimmed["last_checked"] = v }
                out = append(out, trimmed)
            }
        }
        if len(out) > 0 {
            result["heartbeat"] = out
        }
    }

    // server_metrics – keep as-is if present (items here are already compact)
    if smArr, ok := asArray(item["server_metrics"]); ok && len(smArr) > 0 {
        result["server_metrics"] = smArr
    }

    return result
}

func asObject(v any) (map[string]any, bool) {
    if v == nil { return nil, false }
    m, ok := v.(map[string]any)
    return m, ok
}

func asArray(v any) ([]any, bool) {
    if v == nil { return nil, false }
    a, ok := v.([]any)
    return a, ok
}
