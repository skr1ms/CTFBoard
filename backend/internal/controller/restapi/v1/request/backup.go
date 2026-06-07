package request

import (
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const zipHeaderMinLen = 2

func ValidateZIPArchive(data []byte) error {
	if len(data) < zipHeaderMinLen || data[0] != 'P' || data[1] != 'K' {
		return apperr.NewValidationErrorf("file must be a ZIP archive")
	}

	return nil
}

func ExportOptionsFromParams(params openapi.GetAdminExportParams) domain.ExportOptions {
	return domain.ExportOptions{
		IncludeUsers:       params.IncludeUsers != nil && *params.IncludeUsers,
		IncludeTeams:       params.IncludeTeams == nil || *params.IncludeTeams,
		IncludeSolves:      params.IncludeSolves != nil && *params.IncludeSolves,
		IncludeHintUnlocks: params.IncludeHintUnlocks != nil && *params.IncludeHintUnlocks,
		IncludeAwards:      params.IncludeAwards == nil || *params.IncludeAwards,
	}
}

func ExportZIPOptionsFromParams(params openapi.GetAdminExportZipParams) domain.ExportOptions {
	includeFiles := params.IncludeFiles == nil || *params.IncludeFiles

	return domain.ExportOptions{
		IncludeUsers:       false,
		IncludeTeams:       true,
		IncludeSolves:      false,
		IncludeHintUnlocks: false,
		IncludeAwards:      true,
		IncludeFiles:       includeFiles,
	}
}

func ExportCSVTableFromParams(params openapi.GetAdminExportCsvParams) (string, error) {
	if !params.Table.Valid() {
		return "", apperr.NewValidationErrorf("table must be one of: users, teams, challenges, submissions, solves, awards")
	}

	return string(params.Table), nil
}

func ImportCSVTableFromMultipart(body *openapi.PostAdminImportCsvMultipartBody) (string, error) {
	if body.Table == "" {
		return "", apperr.NewValidationErrorf("table parameter is required")
	}

	if !body.Table.Valid() {
		return "", apperr.NewValidationErrorf("table must be one of: users, teams, challenges, submissions, solves, awards")
	}

	return string(body.Table), nil
}

func AdminResetRequestToParams(req *openapi.AdminResetRequest) domain.AdminResetOptions {
	opts := domain.AdminResetOptions{}

	if req.Pages != nil {
		opts.Pages = *req.Pages
	}

	if req.Notifications != nil {
		opts.Notifications = *req.Notifications
	}

	if req.Challenges != nil {
		opts.Challenges = *req.Challenges
	}

	if req.Accounts != nil {
		opts.Accounts = *req.Accounts
	}

	if req.Submissions != nil {
		opts.Submissions = *req.Submissions
	}

	return opts
}

func ImportOptionsFromMultipart(body *openapi.PostAdminImportMultipartBody, adminID uuid.UUID, adminIP string) (domain.ImportOptions, error) {
	cm := domain.ConflictModeOverwrite

	if body.ConflictMode != nil {
		if !body.ConflictMode.Valid() {
			return domain.ImportOptions{}, apperr.NewValidationErrorf("conflict_mode must be one of: overwrite, skip")
		}

		cm = domain.ConflictMode(*body.ConflictMode)
	}

	return domain.ImportOptions{
		EraseExisting:      body.EraseExisting != nil && *body.EraseExisting,
		ValidateFiles:      body.ValidateFiles != nil && *body.ValidateFiles,
		ConflictMode:       cm,
		PreserveAdminRoles: body.PreserveAdminRoles != nil && *body.PreserveAdminRoles,
		AdminUserID:        &adminID,
		AdminIP:            adminIP,
	}, nil
}
