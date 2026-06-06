package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

// CreateAuditLog inserts a team audit-log entry, generating a new UUID and
// setting CreatedAt to the current time. The Details field, when non-nil, is
// marshalled to JSON; an empty JSON object is stored otherwise.
func (r *TeamRepo) CreateAuditLog(ctx context.Context, log *domain.TeamAuditLog) error {
	log.ID = uuid.New()
	log.CreatedAt = time.Now()
	detailsJSON := []byte("{}")

	if log.Details != nil {
		var jsonErr error

		detailsJSON, jsonErr = json.Marshal(log.Details)
		if jsonErr != nil {
			return fmt.Errorf("TeamRepo - CreateAuditLog - MarshalDetails: %w", jsonErr)
		}
	}

	err := r.Q(ctx).CreateTeamAuditLog(ctx, sqlc.CreateTeamAuditLogParams{
		ID:        log.ID,
		TeamID:    log.TeamID,
		UserID:    log.UserID,
		Action:    string(log.Action),
		Details:   detailsJSON,
		CreatedAt: pgutil.TimeToTimestamptz(&log.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("TeamRepo - CreateAuditLog: %w", err)
	}

	return nil
}

// GetLatestAuditLogByTeamIDAndAction returns the most recent audit log entry for the
// given team and action, or (nil, nil) when no such entry exists (ErrNoRows is not an error here).
// The JSON details field is unmarshalled into map[string]any.
func (r *TeamRepo) GetLatestAuditLogByTeamIDAndAction(ctx context.Context, teamID uuid.UUID, action string) (*domain.TeamAuditLog, error) {
	row, err := r.Q(ctx).GetLatestTeamAuditLogByTeamIDAndAction(ctx, sqlc.GetLatestTeamAuditLogByTeamIDAndActionParams{
		TeamID: teamID,
		Action: action,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("TeamRepo - GetLatestAuditLogByTeamIDAndAction: %w", err)
	}

	var details map[string]any

	if len(row.Details) > 0 {
		err := json.Unmarshal(row.Details, &details)
		if err != nil {
			return nil, fmt.Errorf("TeamRepo - GetLatestAuditLogByTeamIDAndAction - Unmarshal details: %w", err)
		}
	}

	createdAt := pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))

	return &domain.TeamAuditLog{
		ID:        row.ID,
		TeamID:    row.TeamID,
		UserID:    row.UserID,
		Action:    domain.TeamAuditAction(row.Action),
		Details:   details,
		CreatedAt: createdAt,
	}, nil
}
