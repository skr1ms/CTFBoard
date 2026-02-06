package team

import "github.com/skr1ms/CTFBoard/internal/entity"

type ConfirmReason string

const (
	ConfirmReasonNone          ConfirmReason = ""
	ConfirmReasonSoloTeamReset ConfirmReason = "solo_team_reset"
	ConfirmReasonProgressLoss  ConfirmReason = "progress_loss"
)

type AffectedData struct {
	SolveCount int `json:"solve_count"`
	Points     int `json:"points"`
}

type OperationResult struct {
	Team               *entity.Team
	RequiresConfirm    bool
	ConfirmationReason ConfirmReason
	AffectedData       *AffectedData
}
