package team

import (
	"encoding/json"

	"github.com/google/uuid"
)

// parseUUIDSliceFromDetails extracts a []uuid.UUID from an audit log Details map.
// It delegates string-slice coercion to toStringSlice so the value is handled
// correctly whether it was stored as []string, []any, or a JSON-encoded array.
func parseUUIDSliceFromDetails(details map[string]any, key string) []uuid.UUID {
	raw, ok := details[key]
	if !ok || raw == nil {
		return nil
	}

	strSlice := toStringSlice(raw)

	var ids []uuid.UUID

	for _, s := range strSlice {
		if id, err := uuid.Parse(s); err == nil {
			ids = append(ids, id)
		}
	}

	return ids
}

// toStringSlice coerces an arbitrary value to []string. It handles three cases:
// []string (direct return), []any (per-element assertion), and anything else
// (round-trip through JSON marshalling). Returns nil on any failure.
func toStringSlice(raw any) []string {
	if raw == nil {
		return nil
	}

	if s, ok := raw.([]string); ok {
		return s
	}

	if s, ok := raw.([]any); ok {
		out := make([]string, 0, len(s))
		for _, v := range s {
			if str, ok := v.(string); ok {
				out = append(out, str)
			}
		}

		return out
	}

	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var out []string
	if err := json.Unmarshal(encoded, &out); err != nil {
		return nil
	}

	return out
}

// parseFieldValuesFromDetails reconstructs the custom field-value snapshot stored in an
// audit log. The value is a []any of {"field_id": string, "value": string} maps; missing
// "value" keys are treated as empty strings rather than errors.
func parseFieldValuesFromDetails(details map[string]any, key string) map[string]string {
	raw, ok := details[key]
	if !ok || raw == nil {
		return nil
	}

	slice, ok := raw.([]any)
	if !ok {
		return nil
	}

	out := make(map[string]string)

	for _, item := range slice {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		fid, okF := m["field_id"].(string)
		val, okV := m["value"].(string)

		if okF && fid != "" {
			if okV {
				out[fid] = val
			} else {
				out[fid] = ""
			}
		}
	}

	return out
}
