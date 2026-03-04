package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromCSVImportResult(result *usecase.CSVImportResult) openapi.CSVImportResult {
	res := openapi.CSVImportResult{
		Success:       ptr(result.Success),
		ImportedCount: ptr(result.ImportedCount),
		SkippedCount:  ptr(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}
	return res
}

func FromImportResult(result *entity.ImportResult) openapi.ImportResult {
	res := openapi.ImportResult{
		Success:      ptr(result.Success),
		SkippedCount: ptr(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}
	return res
}

func FromHealthcheck(status, database string) openapi.HealthcheckResponse {
	return openapi.HealthcheckResponse{
		Status:   ptr(status),
		Database: ptr(database),
	}
}
