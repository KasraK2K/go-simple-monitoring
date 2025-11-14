package utils

import "strings"

// HasStorage checks if the desired backend exists in the configured list.
func HasStorage(backends []string, want string) bool {
    want = strings.ToLower(strings.TrimSpace(want))
    for _, b := range backends {
        if strings.ToLower(strings.TrimSpace(b)) == want {
            return true
        }
    }
    return false
}

