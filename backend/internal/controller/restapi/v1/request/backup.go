package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

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
