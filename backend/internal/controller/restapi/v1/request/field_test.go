package request

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestCreateFieldRequestToParamsMapsMetadata(t *testing.T) {
	t.Parallel()

	description := "Profile description"
	required := true
	public := true
	editable := true
	options := []string{"A", "B"}
	orderIndex := 2
	req := &openapi.CreateFieldRequest{
		Name:        "division",
		Description: &description,
		FieldType:   openapi.CreateFieldRequestFieldTypeSelect,
		EntityType:  openapi.CreateFieldRequestEntityTypeUser,
		Required:    &required,
		Public:      &public,
		Editable:    &editable,
		Options:     &options,
		OrderIndex:  &orderIndex,
	}

	got, err := CreateFieldRequestToParams(req)

	require.NoError(t, err)
	assert.Equal(t, "division", got.Name)
	assert.Equal(t, description, got.Description)
	assert.Equal(t, domain.FieldTypeSelect, got.FieldType)
	assert.Equal(t, domain.EntityTypeUser, got.EntityType)
	assert.True(t, got.Required)
	assert.True(t, got.Public)
	assert.True(t, got.Editable)
	assert.Equal(t, options, got.Options)
	assert.Equal(t, orderIndex, got.OrderIndex)
}

func TestUpdateFieldRequestToParamsMapsMetadata(t *testing.T) {
	t.Parallel()

	description := "Updated description"
	public := true
	editable := true
	req := &openapi.UpdateFieldRequest{
		Name:        "bio",
		Description: &description,
		FieldType:   openapi.UpdateFieldRequestFieldTypeText,
		Public:      &public,
		Editable:    &editable,
	}

	got, err := UpdateFieldRequestToParams(req)

	require.NoError(t, err)
	assert.Equal(t, "bio", got.Name)
	assert.Equal(t, description, got.Description)
	assert.Equal(t, domain.FieldTypeText, got.FieldType)
	assert.True(t, got.Public)
	assert.True(t, got.Editable)
}
