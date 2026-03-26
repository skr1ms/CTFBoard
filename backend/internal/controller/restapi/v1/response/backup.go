package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromCSVImportResult(result *usecase.CSVImportResult) openapi.CSVImportResult {
	res := openapi.CSVImportResult{
		Success:       new(result.Success),
		ImportedCount: new(result.ImportedCount),
		SkippedCount:  new(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}

	return res
}

func FromImportResult(result *domain.ImportResult) openapi.ImportResult {
	res := openapi.ImportResult{
		Success:      new(result.Success),
		SkippedCount: new(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}

	return res
}

func FromHealthcheck(status, database string) openapi.HealthcheckResponse {
	return openapi.HealthcheckResponse{
		Status:   new(status),
		Database: new(database),
	}
}
