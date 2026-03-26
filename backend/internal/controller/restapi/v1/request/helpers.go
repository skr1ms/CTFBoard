package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

func ParseUUIDSlice(rawIDs *[]string, fieldName string) ([]uuid.UUID, error) {
	if rawIDs == nil {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(*rawIDs))
	for _, s := range *rawIDs {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, httperr.NewValidationErrorf("invalid %s", fieldName)
		}

		ids = append(ids, id)
	}

	return ids, nil
}

func ValidateConstraints(v validator.Validator, c any) error {
	err := v.Validate(c)
	if err != nil {
		return httperr.NewValidationErrorf("%v", err)
	}

	return nil
}
