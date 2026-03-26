package request

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
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

	t.Run("empty_configs_returns_empty_slice", func(t *testing.T) {
		req := &openapi.BatchSetConfigRequest{Configs: []openapi.BatchSetConfigItem{}}

		out, err := BatchSetConfigRequestToParams(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(out) != 0 {
			t.Fatalf("expected 0 items, got %d", len(out))
		}
	})

	t.Run("valid_items_converted", func(t *testing.T) {
		vt := openapi.BatchSetConfigItemValueTypeInt
		desc := "an int"
		req := &openapi.BatchSetConfigRequest{
			Configs: []openapi.BatchSetConfigItem{
				{Key: "a", Value: "42", ValueType: &vt, Description: &desc},
				{Key: "b", Value: "hello"},
			},
		}

		out, err := BatchSetConfigRequestToParams(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(out) != 2 {
			t.Fatalf("expected 2 items, got %d", len(out))
		}

		if out[0].Key != "a" || out[0].Value != "42" || out[0].ValueType != domain.CompetitionParamTypeInt || out[0].Description != "an int" {
			t.Errorf("item 0: got %+v", out[0])
		}

		if out[1].Key != "b" || out[1].Value != "hello" || out[1].ValueType != domain.CompetitionParamTypeString || out[1].Description != "" {
			t.Errorf("item 1: got %+v", out[1])
		}
	})

	t.Run("empty_key_returns_error", func(t *testing.T) {
		v, err := validator.New()
		require.NoError(t, err)

		req := &openapi.BatchSetConfigRequest{
			Configs: []openapi.BatchSetConfigItem{{Key: "", Value: "x"}},
		}

		err = ValidateBatchSetConfigRequest(req, v)
		if err == nil {
			t.Fatal("expected error for empty key")
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
