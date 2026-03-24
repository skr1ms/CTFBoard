package response

import (
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromCSVImportResult(result *usecase.CSVImportResult) openapi.CSVImportResult {
	res := openapi.CSVImportResult{
		Success:       httputil.Ptr(result.Success),
		ImportedCount: httputil.Ptr(result.ImportedCount),
		SkippedCount:  httputil.Ptr(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}
	return res
}

func FromImportResult(result *domain.ImportResult) openapi.ImportResult {
	res := openapi.ImportResult{
		Success:      httputil.Ptr(result.Success),
		SkippedCount: httputil.Ptr(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}
	return res
}

func FromHealthcheck(status, database string) openapi.HealthcheckResponse {
	return openapi.HealthcheckResponse{
		Status:   httputil.Ptr(status),
		Database: httputil.Ptr(database),
	}
}
