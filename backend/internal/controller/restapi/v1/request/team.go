package request

import "github.com/skr1ms/CTFBoard/internal/openapi"

func CreateTeamRequestToParams(req *openapi.RequestCreateTeamRequest) (name string, confirmReset bool) {
	confirmReset = false
	if req.ConfirmReset != nil {
		confirmReset = *req.ConfirmReset
	}
	return req.Name, confirmReset
}

func JoinTeamRequestToParams(req *openapi.RequestJoinTeamRequest) (inviteToken string, confirmReset bool) {
	confirmReset = false
	if req.ConfirmReset != nil {
		confirmReset = *req.ConfirmReset
	}
	return req.InviteToken, confirmReset
}

func TransferCaptainRequestToNewCaptainID(req *openapi.RequestTransferCaptainRequest) string {
	return req.NewCaptainID
}

func BanTeamRequestToReason(req *openapi.RequestBanTeamRequest) string {
	return req.Reason
}

func SetHiddenRequestToHidden(req *openapi.RequestSetHiddenRequest) bool {
	if req.Hidden != nil {
		return *req.Hidden
	}
	return false
}
