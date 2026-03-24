package response

import (
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

func parseDate(s string) *openapi_types.Date {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &openapi_types.Date{Time: t}
}
