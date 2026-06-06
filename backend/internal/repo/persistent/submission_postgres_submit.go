package persistent

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

func (r *SubmissionRepo) Create(ctx context.Context, sub *domain.Submission) error {
	EnsureID(&sub.ID)

	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = time.Now()
	}

	subType := sub.Type
	if subType == "" {
		subType = domain.SubmissionTypeIncorrect
	}

	err := r.Q(ctx).CreateSubmission(ctx, sqlc.CreateSubmissionParams{
		ID:             sub.ID,
		UserID:         sub.UserID,
		TeamID:         sub.TeamID,
		ChallengeID:    sub.ChallengeID,
		SubmittedFlag:  sub.SubmittedFlag,
		IsCorrect:      sub.IsCorrect,
		SubmissionType: subType,
		IP:             sub.IP,
		CreatedAt:      pgutil.TimeToTimestamptz(&sub.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Create: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) CountSubmissionsByTeamAndChallenge(ctx context.Context, teamID, challengeID uuid.UUID) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByTeamAndChallenge(ctx, sqlc.CountSubmissionsByTeamAndChallengeParams{
		TeamID:      &teamID,
		ChallengeID: challengeID,
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountSubmissionsByTeamAndChallenge: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountSubmissionsByTeamAndChallengeInWindow(ctx context.Context, teamID, challengeID uuid.UUID, windowStart time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByTeamAndChallengeInWindow(ctx, sqlc.CountSubmissionsByTeamAndChallengeInWindowParams{
		TeamID:      &teamID,
		ChallengeID: challengeID,
		CreatedAt:   pgutil.TimeToTimestamptz(&windowStart),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountSubmissionsByTeamAndChallengeInWindow: %w", err)
	}

	return n, nil
}

// AcquireAdvisoryLockForSubmit acquires a PostgreSQL transaction-level advisory lock
// to serialize concurrent flag submissions for the same (team, challenge) pair.
// The lock key is derived by XOR-ing the first 8 bytes of teamID and challengeID,
// then masking to a positive int64.
func (r *SubmissionRepo) AcquireAdvisoryLockForSubmit(ctx context.Context, teamID, challengeID uuid.UUID) error {
	keyU := binary.BigEndian.Uint64(teamID[0:8]) ^ binary.BigEndian.Uint64(challengeID[0:8])
	key := int64(keyU & positiveAdvisoryLockMask)

	if err := r.AcquireAdvisoryLock(ctx, key); err != nil {
		return fmt.Errorf("SubmissionRepo - AcquireAdvisoryLockForSubmit: %w", err)
	}

	return nil
}
