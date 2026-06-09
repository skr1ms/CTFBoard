package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
)

type ChallengeRepo struct {
	BaseRepo
}

var _ repo.ChallengeRepository = (*ChallengeRepo)(nil)

func NewChallengeRepo(pool *pgxpool.Pool) *ChallengeRepo {
	return &ChallengeRepo{BaseRepo: BaseRepo{pool: pool}}
}

type challengeRow struct {
	ID                uuid.UUID
	Title             string
	Description       string
	Category          *string
	Points            int32
	InitialValue      int32
	MinValue          int32
	Decay             int32
	SolveCount        int32
	FlagHash          string
	Attribution       string
	ConnectionInfo    string
	MaxAttempts       int32
	MaxAttemptsWindow int64
	Position          int32
	NextChallengeID   *uuid.UUID
	State             string
	IsRegex           bool
	IsCaseInsensitive bool
	FlagRegex         *string
	FlagFormatRegex   *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// challengeFields is a reusable value type that holds the raw sqlc-scanned
// fields shared by every challenge query. Each sqlc row type gets its own
// toChallengeRow* adapter below; call toDomainChallenge(adapter(row)) to build
// a domain.Challenge without repeating the 20-field struct literal.
type challengeFields struct {
	ID                uuid.UUID
	Title             string
	Description       string
	Category          string
	Points            int32
	InitialValue      int32
	MinValue          int32
	Decay             int32
	SolveCount        int32
	FlagHash          string
	Attribution       string
	ConnectionInfo    string
	MaxAttempts       int32
	MaxAttemptsWindow int64
	Position          int32
	NextChallengeID   *uuid.UUID
	State             string
	IsRegex           bool
	IsCaseInsensitive bool
	FlagRegex         *string
	FlagFormatRegex   *string
	CreatedAt         pgtype.Timestamptz
	UpdatedAt         pgtype.Timestamptz
}

func (f challengeFields) toChallengeRow() challengeRow {
	return challengeRow{
		ID:                f.ID,
		Title:             f.Title,
		Description:       f.Description,
		Category:          lo.EmptyableToPtr(f.Category),
		Points:            f.Points,
		InitialValue:      f.InitialValue,
		MinValue:          f.MinValue,
		Decay:             f.Decay,
		SolveCount:        f.SolveCount,
		FlagHash:          f.FlagHash,
		Attribution:       f.Attribution,
		ConnectionInfo:    f.ConnectionInfo,
		MaxAttempts:       f.MaxAttempts,
		MaxAttemptsWindow: f.MaxAttemptsWindow,
		Position:          f.Position,
		NextChallengeID:   f.NextChallengeID,
		State:             f.State,
		IsRegex:           f.IsRegex,
		IsCaseInsensitive: f.IsCaseInsensitive,
		FlagRegex:         f.FlagRegex,
		FlagFormatRegex:   f.FlagFormatRegex,
		CreatedAt:         pgutil.TimestamptzToTimeZero(f.CreatedAt),
		UpdatedAt:         pgutil.TimestamptzToTimeZero(f.UpdatedAt),
	}
}

func fieldsFromGetByID(r sqlc.GetChallengeByIDRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetByIDForUpdate(r sqlc.GetChallengeByIDForUpdateRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetByIDs(r sqlc.GetChallengesByIDsRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetForTeamByTag(r sqlc.GetChallengesForTeamByTagRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetByTag(r sqlc.GetChallengesByTagRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetForTeam(r sqlc.GetChallengesForTeamRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetAll(r sqlc.GetChallengesRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetAllForBackup(r sqlc.GetChallengesAllRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetMissingByTeamID(r sqlc.GetMissingChallengesByTeamIDRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func fieldsFromGetMissingByUserID(r sqlc.GetMissingChallengesByUserIDRow) challengeFields {
	return challengeFields{ID: r.ID, Title: r.Title, Description: r.Description, Category: r.Category, Points: r.Points, InitialValue: r.InitialValue, MinValue: r.MinValue, Decay: r.Decay, SolveCount: r.SolveCount, FlagHash: r.FlagHash, Attribution: r.Attribution, ConnectionInfo: r.ConnectionInfo, MaxAttempts: r.MaxAttempts, MaxAttemptsWindow: r.MaxAttemptsWindow, Position: r.Position, NextChallengeID: r.NextChallengeID, State: r.State, IsRegex: r.IsRegex, IsCaseInsensitive: r.IsCaseInsensitive, FlagRegex: r.FlagRegex, FlagFormatRegex: r.FlagFormatRegex, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func toDomainChallenge(r challengeRow) *domain.Challenge {
	return &domain.Challenge{
		ID:                r.ID,
		Title:             r.Title,
		Description:       r.Description,
		Category:          lo.FromPtr(r.Category),
		Points:            int(r.Points),
		InitialValue:      int(r.InitialValue),
		MinValue:          int(r.MinValue),
		Decay:             int(r.Decay),
		SolveCount:        int(r.SolveCount),
		FlagHash:          r.FlagHash,
		Attribution:       r.Attribution,
		ConnectionInfo:    r.ConnectionInfo,
		MaxAttempts:       int(r.MaxAttempts),
		MaxAttemptsWindow: time.Duration(r.MaxAttemptsWindow),
		Position:          int(r.Position),
		NextChallengeID:   r.NextChallengeID,
		State:             r.State,
		IsRegex:           r.IsRegex,
		IsCaseInsensitive: r.IsCaseInsensitive,
		FlagRegex:         r.FlagRegex,
		FlagFormatRegex:   r.FlagFormatRegex,
		CreatedAt:         r.CreatedAt,
		UpdatedAt:         r.UpdatedAt,
	}
}

func (r *ChallengeRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Challenge, error) {
	row, err := GetOrNotFound(func() (sqlc.GetChallengeByIDRow, error) { return r.Q(ctx).GetChallengeByID(ctx, ID) },
		apperr.ErrChallengeNotFound, "ChallengeRepo - GetByID")
	if err != nil {
		return nil, err
	}

	return toDomainChallenge(fieldsFromGetByID(row).toChallengeRow()), nil
}

func (r *ChallengeRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*domain.Challenge, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]*domain.Challenge{}, nil
	}

	rows, err := r.Q(ctx).GetChallengesByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - GetByIDs: %w", err)
	}

	out := make(map[uuid.UUID]*domain.Challenge, len(rows))
	for _, row := range rows {
		out[row.ID] = toDomainChallenge(fieldsFromGetByIDs(row).toChallengeRow())
	}

	return out, nil
}

func (r *ChallengeRepo) listForTeamByTag(ctx context.Context, teamID, tagID uuid.UUID) ([]*domain.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesForTeamByTag(ctx, sqlc.GetChallengesForTeamByTagParams{TagID: tagID, TeamID: teamID})
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listForTeamByTag: %w", err)
	}

	out := make([]*domain.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeWithSolved{
			Challenge: toDomainChallenge(fieldsFromGetForTeamByTag(row).toChallengeRow()),
			Solved:    row.Solved == 1,
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listByTag(ctx context.Context, tagID uuid.UUID) ([]*domain.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesByTag(ctx, tagID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listByTag: %w", err)
	}

	out := make([]*domain.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeWithSolved{
			Challenge: toDomainChallenge(fieldsFromGetByTag(row).toChallengeRow()),
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listForTeam(ctx context.Context, teamID uuid.UUID) ([]*domain.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesForTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listForTeam: %w", err)
	}

	out := make([]*domain.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeWithSolved{
			Challenge: toDomainChallenge(fieldsFromGetForTeam(row).toChallengeRow()),
			Solved:    row.Solved == 1,
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listAllChallenges(ctx context.Context) ([]*domain.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallenges(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listAllChallenges: %w", err)
	}

	out := make([]*domain.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeWithSolved{
			Challenge: toDomainChallenge(fieldsFromGetAll(row).toChallengeRow()),
		})
	}

	return out, nil
}

func (r *ChallengeRepo) listAllChallengesForBackup(ctx context.Context) ([]*domain.ChallengeWithSolved, error) {
	rows, err := r.Q(ctx).GetChallengesAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("ChallengeRepo - listAllChallengesForBackup: %w", err)
	}

	out := make([]*domain.ChallengeWithSolved, 0, len(rows))
	for _, row := range rows {
		out = append(out, &domain.ChallengeWithSolved{
			Challenge: toDomainChallenge(fieldsFromGetAllForBackup(row).toChallengeRow()),
		})
	}

	return out, nil
}

// GetAll dispatches to one of four query paths depending on which of teamID and
// tagID are provided: team+tag, tag-only, team-only, or all challenges. When a
// teamID is given the query joins solves to populate the Solved flag per entry.
func (r *ChallengeRepo) GetAll(ctx context.Context, teamID, tagID *uuid.UUID) ([]*domain.ChallengeWithSolved, error) {
	if tagID != nil && teamID != nil {
		return r.listForTeamByTag(ctx, *teamID, *tagID)
	}

	if tagID != nil {
		return r.listByTag(ctx, *tagID)
	}

	if teamID != nil {
		return r.listForTeam(ctx, *teamID)
	}

	return r.listAllChallenges(ctx)
}

func (r *ChallengeRepo) GetAllForBackup(ctx context.Context) ([]*domain.ChallengeWithSolved, error) {
	return r.listAllChallengesForBackup(ctx)
}

func (r *ChallengeRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	_, err := r.Q(ctx).DeleteChallenge(ctx, ID)
	if err != nil && !pgutil.IsNoRows(err) {
		return fmt.Errorf("ChallengeRepo - Delete: %w", err)
	}

	return nil
}

func (r *ChallengeRepo) IncrementSolveCount(ctx context.Context, ID uuid.UUID) (int, error) {
	n, err := r.Q(ctx).IncrementChallengeSolveCount(ctx, ID)
	if err != nil {
		return 0, fmt.Errorf("ChallengeRepo - IncrementSolveCount: %w", err)
	}

	return int(n), nil
}

func (r *ChallengeRepo) UpdatePoints(ctx context.Context, ID uuid.UUID, points int) error {
	pts, err := intToInt32Safe(points)
	if err != nil {
		return fmt.Errorf("ChallengeRepo - UpdatePoints: %w", err)
	}

	_, err = GetOrNotFound(func() (uuid.UUID, error) {
		return r.Q(ctx).UpdateChallengePoints(ctx, sqlc.UpdateChallengePointsParams{ID: ID, Points: pts})
	}, apperr.ErrChallengeNotFound, "ChallengeRepo - UpdatePoints")

	return err
}

func (r *ChallengeRepo) GetFlags(ctx context.Context, ID uuid.UUID) (*domain.ChallengeFlags, error) {
	row, err := GetOrNotFound(func() (sqlc.GetChallengeFlagsRow, error) { return r.Q(ctx).GetChallengeFlags(ctx, ID) },
		apperr.ErrChallengeNotFound, "ChallengeRepo - GetFlags")
	if err != nil {
		return nil, err
	}

	return &domain.ChallengeFlags{
		FlagHash:          row.FlagHash,
		IsRegex:           row.IsRegex,
		IsCaseInsensitive: row.IsCaseInsensitive,
		FlagRegex:         row.FlagRegex,
		FlagFormatRegex:   row.FlagFormatRegex,
	}, nil
}
