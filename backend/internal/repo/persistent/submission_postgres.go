package persistent

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

type SubmissionRepo struct {
	BaseRepo
}

const positiveAdvisoryLockMask = 0x7fffffffffffffff

var _ repo.SubmissionRepository = (*SubmissionRepo)(nil)

func NewSubmissionRepo(pool *pgxpool.Pool) *SubmissionRepo {
	return &SubmissionRepo{BaseRepo: BaseRepo{pool: pool}}
}

type subRow struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	TeamID         *uuid.UUID
	ChallengeID    uuid.UUID
	SubmittedFlag  string
	IsCorrect      bool
	SubmissionType string
	IP             string
	CreatedAt      pgtype.Timestamptz
	BannedUserID   *uuid.UUID
	BannedTeamID   *uuid.UUID
}

func newSubRow(
	id, userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID,
	submittedFlag string, isCorrect bool, submissionType, ip string,
	createdAt pgtype.Timestamptz, bannedUserID, bannedTeamID *uuid.UUID,
) subRow {
	return subRow{
		ID: id, UserID: userID, TeamID: teamID, ChallengeID: challengeID,
		SubmittedFlag: submittedFlag, IsCorrect: isCorrect, SubmissionType: submissionType,
		IP: ip, CreatedAt: createdAt, BannedUserID: bannedUserID, BannedTeamID: bannedTeamID,
	}
}

func toBaseSubmission(r subRow) domain.Submission {
	return domain.Submission{
		ID:            r.ID,
		UserID:        r.UserID,
		TeamID:        r.TeamID,
		ChallengeID:   r.ChallengeID,
		SubmittedFlag: r.SubmittedFlag,
		IsCorrect:     r.IsCorrect,
		Type:          r.SubmissionType,
		IP:            r.IP,
		CreatedAt:     pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(r.CreatedAt)),
		BannedUserID:  r.BannedUserID,
		BannedTeamID:  r.BannedTeamID,
	}
}
