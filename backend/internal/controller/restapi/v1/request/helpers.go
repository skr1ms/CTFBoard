package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

func ParseUUIDSlice(rawIDs *[]string, fieldName string) ([]uuid.UUID, error) {
	if rawIDs == nil {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(*rawIDs))
	for _, s := range *rawIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, apperr.NewValidationErrorf("invalid %s", fieldName)
		}

		ids = append(ids, id)
	}

	return ids, nil
}
