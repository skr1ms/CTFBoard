package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func AppealDecisionFromParams(params openapi.GetAdminAppealsParams) *domain.AppealDecision {
	if params.Decision == nil {
		return nil
	}

	decision := domain.AppealDecision(string(*params.Decision))

	return &decision
}

func ReviewAppealRequestToParams(req *openapi.ReviewAppealRequest) (domain.AppealDecision, *string) {
	decision := domain.AppealDecision(string(req.Decision))

	var adminResponse *string

	if req.AdminResponse != nil && *req.AdminResponse != "" {
		adminResponse = req.AdminResponse
	}

	return decision, adminResponse
}
