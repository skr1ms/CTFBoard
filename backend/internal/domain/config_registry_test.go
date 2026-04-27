package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetConfigDefault_KnownKey(t *testing.T) {
	t.Parallel()

	val, ok := GetConfigDefault("ctf_name")
	assert.True(t, ok)
	assert.Equal(t, "CTF Platform", val)
	val, ok = GetConfigDefault("scoring_type")
	assert.True(t, ok)
	assert.Equal(t, "dynamic", val)
}

func TestGetConfigDefault_UnknownKey(t *testing.T) {
	t.Parallel()

	val, ok := GetConfigDefault("nonexistent_key_xyz")
	assert.False(t, ok)
	assert.Empty(t, val)
}

var validValueTypes = map[CompetitionParamValueType]struct{}{
	CompetitionParamTypeString: {},
	CompetitionParamTypeInt:    {},
	CompetitionParamTypeBool:   {},
	CompetitionParamTypeJSON:   {},
}

func TestConfigRegistry_AllKeysHaveNonEmptyCategoryAndValidValueType(t *testing.T) {
	t.Parallel()
	RangeConfigRegistry(func(key string, def ConfigDef) bool {
		assert.NotEmpty(t, def.Category, "key %q must have non-empty category", key)
		_, ok := validValueTypes[def.ValueType]
		assert.True(t, ok, "key %q has invalid value_type %q", key, def.ValueType)

		return true
	})
}
