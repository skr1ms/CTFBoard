package response

import "time"

func DebugInfo(now time.Time) map[string]any {
	return map[string]any{
		"debug":     true,
		"timestamp": now.UTC().Format(time.RFC3339),
	}
}
