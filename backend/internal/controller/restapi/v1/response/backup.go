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
	if result == nil {
		return openapi.ImportResult{}
	}

	res := openapi.ImportResult{
		Success:      new(result.Success),
		SkippedCount: new(result.SkippedCount),
	}
	if len(result.Errors) > 0 {
		res.Errors = &result.Errors
	}

	if len(result.Warnings) > 0 {
		res.Warnings = &result.Warnings
	}

	return res
}

func FromImportJob(job *domain.ImportJob) openapi.ImportJobResponse {
	status := openapi.ImportJobResponseStatus(job.Status)
	phase := openapi.ImportJobResponsePhase(job.Phase)

	res := openapi.ImportJobResponse{
		ID:              &job.ID,
		ArchiveFilename: &job.ArchiveFilename,
		ArchiveSize:     &job.ArchiveSize,
		Status:          &status,
		Phase:           &phase,
		CreatedAt:       &job.CreatedAt,
		StartedAt:       job.StartedAt,
		FinishedAt:      job.FinishedAt,
		UpdatedAt:       &job.UpdatedAt,
		Error:           job.Error,
		RequestedBy:     job.RequestedBy,
	}

	if job.Result != nil {
		result := FromImportResult(job.Result)
		res.Result = &result
	}

	if job.ClientIP != "" {
		res.ClientIP = &job.ClientIP
	}

	return res
}

func FromHealthcheck(status, database string) openapi.HealthcheckResponse {
	return openapi.HealthcheckResponse{
		Status:   new(status),
		Database: new(database),
	}
}
