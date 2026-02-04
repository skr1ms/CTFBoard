package request

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func SetConfigRequestToValueType(v *openapi.RequestSetConfigRequestValueType) entity.ConfigValueType {
	if v == nil {
		return entity.ConfigTypeString
	}
	switch *v {
	case openapi.Int:
		return entity.ConfigTypeInt
	case openapi.Bool:
		return entity.ConfigTypeBool
	case openapi.JSON:
		return entity.ConfigTypeJSON
	case openapi.String:
		return entity.ConfigTypeString
	default:
		return entity.ConfigTypeString
	}
}
