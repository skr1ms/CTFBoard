package request

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestBatchSetConfigRequestToParams(t *testing.T) {
	t.Run("nil_request_returns_nil", func(t *testing.T) {
		out, err := BatchSetConfigRequestToParams(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if out != nil {
			t.Fatalf("expected nil, got %v", out)
		}
	})

	t.Run("invalid_value_type_returns_error", func(t *testing.T) {
		vt := openapi.BatchSetConfigItemValueType("invalid")
		req := &openapi.BatchSetConfigRequest{
			Configs: []openapi.BatchSetConfigItem{{Key: "k", Value: "v", ValueType: &vt}},
		}

		_, err := BatchSetConfigRequestToParams(req)
		if err == nil {
			t.Fatal("expected error for invalid value_type")
		}
	})
}
