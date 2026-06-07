package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestUpdateTeamRequestToParams_NameOnly(t *testing.T) {
	t.Parallel()

	name := "Team A"

	params, err := UpdateTeamRequestToParams(&openapi.UpdateTeamRequest{Name: &name})

	require.NoError(t, err)
	require.NotNil(t, params.Name)
	assert.Equal(t, "Team A", *params.Name)
	assert.Nil(t, params.CustomFields)
}

func TestUpdateTeamRequestToParams_CustomFieldsOnly(t *testing.T) {
	t.Parallel()

	customFields := map[string]any{"11111111-1111-1111-1111-111111111111": []any{"value"}}

	params, err := UpdateTeamRequestToParams(&openapi.UpdateTeamRequest{CustomFields: &customFields})

	require.NoError(t, err)
	assert.Nil(t, params.Name)
	require.NotNil(t, params.CustomFields)
	assert.Equal(t, customFields, *params.CustomFields)
}

func TestUpdateTeamRequestToParams_EmptyPatch(t *testing.T) {
	t.Parallel()

	params, err := UpdateTeamRequestToParams(&openapi.UpdateTeamRequest{})

	var validationErr *apperr.ValidationError

	require.Error(t, err)
	assert.ErrorAs(t, err, &validationErr)
	assert.Nil(t, params.Name)
	assert.Nil(t, params.CustomFields)
}
