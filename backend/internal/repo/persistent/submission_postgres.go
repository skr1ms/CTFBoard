package persistent

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type SubmissionRepo struct {
	BaseRepo
}

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

// CreateBatch inserts multiple submissions using a 3-tier fallback strategy:
// 1) pgx CopyFrom (fastest, requires a *pgx.Conn);
// 2) pgx SendBatch (pipeline, works with pool connections);
// 3) sequential Create calls as a last resort.
func (r *SubmissionRepo) CreateBatch(ctx context.Context, subs []*domain.Submission) error {
	if len(subs) == 0 {
		return nil
	}

	now := time.Now()

	rows := make([][]any, 0, len(subs))
	for _, sub := range subs {
		EnsureID(&sub.ID)

		if sub.CreatedAt.IsZero() {
			sub.CreatedAt = now
		}

		subType := sub.Type
		if subType == "" {
			subType = domain.SubmissionTypeIncorrect
		}

		rows = append(rows, []any{sub.ID, sub.UserID, sub.TeamID, sub.ChallengeID, sub.SubmittedFlag, sub.IsCorrect, subType, sub.IP, sub.CreatedAt})
	}

	conn := r.DB(ctx)
	if copier, ok := conn.(interface {
		CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	}); ok {
		_, err := copier.CopyFrom(ctx,
			pgx.Identifier{"submissions"},
			[]string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "submission_type", "ip", "created_at"},
			pgx.CopyFromRows(rows),
		)
		if err != nil {
			return fmt.Errorf("SubmissionRepo - CreateBatch - CopyFrom: %w", err)
		}

		return nil
	}

	type batchSender interface {
		SendBatch(ctx context.Context, batch *pgx.Batch) pgx.BatchResults
	}

	if sender, ok := conn.(batchSender); ok {
		batch := &pgx.Batch{}
		cols := []string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "submission_type", "ip", "created_at"}

		for _, sub := range subs {
			subType := sub.Type
			if subType == "" {
				subType = domain.SubmissionTypeIncorrect
			}

			q := SB.Insert("submissions").Columns(cols...).
				Values(sub.ID, sub.UserID, sub.TeamID, sub.ChallengeID, sub.SubmittedFlag, sub.IsCorrect, subType, sub.IP, sub.CreatedAt)

			sqlStr, args, err := q.ToSql()
			if err != nil {
				return fmt.Errorf("SubmissionRepo - CreateBatch - build SQL: %w", err)
			}

			batch.Queue(sqlStr, args...)
		}

		br := sender.SendBatch(ctx, batch)
		defer br.Close()

		for i := 0; i < batch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				return fmt.Errorf("SubmissionRepo - CreateBatch - batch exec: %w", err)
			}
		}

		return nil
	}

	for _, sub := range subs {
		err := r.Create(ctx, sub)
		if err != nil {
			return fmt.Errorf("SubmissionRepo - CreateBatch: %w", err)
		}
	}

	return nil
}

func (r *SubmissionRepo) GetByChallenge(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge: %w", err)
	}

	rows, err := r.Q(ctx).GetSubmissionsByChallenge(ctx, sqlc.GetSubmissionsByChallengeParams{
		ChallengeID: challengeID,
		FreezeTime:  pgutil.TimeToTimestamptz(freezeTime),
		Limit:       limit32,
		Offset:      offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByChallenge - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			Username: row.Username,
			TeamName: row.TeamName,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) GetByUser(ctx context.Context, userID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser: %w", err)
	}

	rows, err := r.Q(ctx).GetSubmissionsByUser(ctx, sqlc.GetSubmissionsByUserParams{
		UserID:     userID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
		Limit:      limit32,
		Offset:     offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByUser - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) GetByTeam(ctx context.Context, teamID uuid.UUID, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam: %w", err)
	}

	rows, err := r.Q(ctx).GetSubmissionsByTeam(ctx, sqlc.GetSubmissionsByTeamParams{
		TeamID:     &teamID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
		Limit:      limit32,
		Offset:     offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetByTeam - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			Username:          row.Username,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) GetAll(ctx context.Context, freezeTime *time.Time, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll: %w", err)
	}

	rows, err := r.Q(ctx).GetAllSubmissions(ctx, sqlc.GetAllSubmissionsParams{
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
		Limit:      limit32,
		Offset:     offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetAll - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			Username:          row.Username,
			TeamName:          row.TeamName,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) CountByChallenge(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByChallenge(ctx, sqlc.CountSubmissionsByChallengeParams{
		ChallengeID: challengeID,
		FreezeTime:  pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByChallenge: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountByUser(ctx context.Context, userID uuid.UUID, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByUser(ctx, sqlc.CountSubmissionsByUserParams{
		UserID:     userID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByUser: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountByTeam(ctx context.Context, teamID uuid.UUID, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountSubmissionsByTeam(ctx, sqlc.CountSubmissionsByTeamParams{
		TeamID:     &teamID,
		FreezeTime: pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountByTeam: %w", err)
	}

	return n, nil
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
	key := int64(keyU & 0x7fffffffffffffff)

	db := r.DB(ctx)
	if _, err := db.Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", key); err != nil {
		return fmt.Errorf("SubmissionRepo - AcquireAdvisoryLockForSubmit: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) CountAll(ctx context.Context, freezeTime *time.Time) (int64, error) {
	n, err := r.Q(ctx).CountAllSubmissions(ctx, pgutil.TimeToTimestamptz(freezeTime))
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountAll: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) CountFailedByIP(ctx context.Context, ip string, since time.Time) (int64, error) {
	n, err := r.Q(ctx).CountFailedSubmissionsByIP(ctx, sqlc.CountFailedSubmissionsByIPParams{
		IP:        ip,
		CreatedAt: pgutil.TimeToTimestamptz(&since),
	})
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailedByIP: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) GetStats(ctx context.Context, challengeID uuid.UUID, freezeTime *time.Time) (*domain.SubmissionStats, error) {
	row, err := r.Q(ctx).GetSubmissionStats(ctx, sqlc.GetSubmissionStatsParams{
		ChallengeID: challengeID,
		FreezeTime:  pgutil.TimeToTimestamptz(freezeTime),
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetStats: %w", err)
	}

	return &domain.SubmissionStats{
		Total:     int(row.Total),
		Correct:   int(row.Correct),
		Incorrect: int(row.Incorrect),
	}, nil
}

func (r *SubmissionRepo) GetFailsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser: %w", err)
	}

	rows, err := r.Q(ctx).GetFailsByUserID(ctx, sqlc.GetFailsByUserIDParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByUser - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) CountFailsByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	n, err := r.Q(ctx).CountFailsByUserID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailsByUser: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) GetFailsByTeam(ctx context.Context, teamID uuid.UUID, limit, offset int) ([]*domain.SubmissionWithDetails, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam: %w", err)
	}

	rows, err := r.Q(ctx).GetFailsByTeamID(ctx, sqlc.GetFailsByTeamIDParams{
		TeamID: &teamID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("SubmissionRepo - GetFailsByTeam - scan: %w", err)
	}

	result := make([]*domain.SubmissionWithDetails, len(rows))
	for i, row := range rows {
		result[i] = &domain.SubmissionWithDetails{
			Submission: toBaseSubmission(newSubRow(
				row.ID, row.UserID, row.TeamID, row.ChallengeID,
				row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
				row.IP, row.CreatedAt, row.BannedUserID, nil,
			)),
			Username:          row.Username,
			ChallengeTitle:    row.ChallengeTitle,
			ChallengeCategory: row.ChallengeCategory,
		}
	}

	return result, nil
}

func (r *SubmissionRepo) CountFailsByTeam(ctx context.Context, teamID uuid.UUID) (int64, error) {
	n, err := r.Q(ctx).CountFailsByTeamID(ctx, &teamID)
	if err != nil {
		return 0, fmt.Errorf("SubmissionRepo - CountFailsByTeam: %w", err)
	}

	return n, nil
}

func (r *SubmissionRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.SubmissionWithDetails, error) {
	row, err := r.Q(ctx).GetSubmissionByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, apperr.ErrSubmissionNotFound
		}

		return nil, fmt.Errorf("SubmissionRepo - GetByID: %w", err)
	}

	return &domain.SubmissionWithDetails{
		Submission: toBaseSubmission(newSubRow(
			row.ID, row.UserID, row.TeamID, row.ChallengeID,
			row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
			row.IP, row.CreatedAt, row.BannedUserID, nil,
		)),
		Username:          row.Username,
		TeamName:          row.TeamName,
		ChallengeTitle:    row.ChallengeTitle,
		ChallengeCategory: row.ChallengeCategory,
	}, nil
}

func (r *SubmissionRepo) Update(ctx context.Context, ID uuid.UUID, isCorrect bool) error {
	err := r.Q(ctx).UpdateSubmission(ctx, sqlc.UpdateSubmissionParams{
		ID:        ID,
		IsCorrect: isCorrect,
	})
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Update: %w", err)
	}

	return nil
}

// Delete removes a submission by ID. Idempotent: returns nil if the submission does not exist.
func (r *SubmissionRepo) Discard(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DiscardSubmission(ctx, ID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Discard: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	err := r.Q(ctx).DeleteSubmission(ctx, ID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - Delete: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) DeleteByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).DeleteSubmissionsByTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - DeleteByTeamID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) SoftBanByTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).SoftBanSubmissionsByTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - SoftBanByTeamID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) RestoreByBannedTeamID(ctx context.Context, teamID uuid.UUID) error {
	err := r.Q(ctx).RestoreSubmissionsByBannedTeamID(ctx, &teamID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - RestoreByBannedTeamID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) SoftBanByUserID(ctx context.Context, userID uuid.UUID) error {
	err := r.Q(ctx).SoftBanSubmissionsByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - SoftBanByUserID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) RestoreByBannedUserID(ctx context.Context, userID uuid.UUID) error {
	err := r.Q(ctx).RestoreSubmissionsByBannedUserID(ctx, &userID)
	if err != nil {
		return fmt.Errorf("SubmissionRepo - RestoreByBannedUserID: %w", err)
	}

	return nil
}

func (r *SubmissionRepo) GetByIDForUpdate(ctx context.Context, ID uuid.UUID) (*domain.Submission, error) {
	row, err := r.Q(ctx).GetSubmissionByIDForUpdate(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, apperr.ErrSubmissionNotFound
		}

		return nil, fmt.Errorf("SubmissionRepo - GetByIDForUpdate: %w", err)
	}

	s := toBaseSubmission(newSubRow(
		row.ID, row.UserID, row.TeamID, row.ChallengeID,
		row.SubmittedFlag, row.IsCorrect, row.SubmissionType,
		row.IP, row.CreatedAt, row.BannedUserID, row.BannedTeamID,
	))

	return &s, nil
}
