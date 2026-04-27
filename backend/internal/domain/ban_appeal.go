package domain

import (
	"time"

	"github.com/google/uuid"
)

// AppealDecision is a string-backed enum for the outcome of a ban appeal review.
type AppealDecision string

const (
	AppealDecisionPending  AppealDecision = "pending"
	AppealDecisionResolved AppealDecision = "resolved"
	AppealDecisionRejected AppealDecision = "rejected"
)

// BanAppeal represents a banned user's request to have their ban reviewed.
type BanAppeal struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Message       string
	CreatedAt     time.Time
	ReviewedAt    *time.Time
	AdminResponse *string
	Decision      AppealDecision
}
