package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func execImportBatches[T any](
	ctx context.Context,
	r *BackupRepo,
	op string,
	items []T,
	table string,
	cols []string,
	suffix string,
	addValues func(squirrel.InsertBuilder, T) squirrel.InsertBuilder,
) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(items))
		q := sqlBuilder().Insert(table).Columns(cols...)

		for _, item := range items[i:end] {
			q = addValues(q, item)
		}

		if err := r.exec(ctx, q.Suffix(suffix)); err != nil {
			return fmt.Errorf("BackupRepo - %s: %w", op, err)
		}
	}

	return nil
}

// ImportTags upserts tags in batches using ON CONFLICT (id) DO UPDATE, so re-importing
// a backup overwrites existing tags rather than failing on duplicate IDs.
func (r *BackupRepo) ImportTags(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportTags", data.Tags, "tags", backupTagImportCols, "ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, color = EXCLUDED.color",
		func(q squirrel.InsertBuilder, t domain.Tag) squirrel.InsertBuilder {
			return q.Values(t.ID, t.Name, t.Color)
		})
}

func (r *BackupRepo) ImportChallengeTags(ctx context.Context, data *domain.BackupData) error {
	var pairs []struct{ challengeID, tagID uuid.UUID }

	for _, ch := range data.Challenges {
		for _, tagID := range ch.TagIDs {
			pairs = append(pairs, struct{ challengeID, tagID uuid.UUID }{ch.ID, tagID})
		}
	}

	if len(pairs) == 0 {
		return nil
	}

	for i := 0; i < len(pairs); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(pairs))

		batch := pairs[i:end]
		q := sqlBuilder().Insert("challenge_tags").Columns(backupChallengeTagCols...)

		for _, p := range batch {
			q = q.Values(p.challengeID, p.tagID)
		}

		err := r.exec(ctx, q.Suffix("ON CONFLICT (challenge_id, tag_id) DO NOTHING"))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportChallengeTags: %w", err)
		}
	}

	return nil
}

// ImportCompetition updates the single competition row (WHERE id=1) with exported data.
// Unlike other Import methods it does not insert - the row is pre-seeded by migrations.
func (r *BackupRepo) ImportCompetition(ctx context.Context, comp *domain.Competition) error {
	if comp == nil {
		return nil
	}

	query := sqlBuilder().Update("competition").
		Set("name", comp.Name).
		Set("start_time", comp.StartTime).
		Set("end_time", comp.EndTime).
		Set("freeze_time", comp.FreezeTime).
		Set("is_paused", comp.IsPaused).
		Set("paused_at", comp.PausedAt).
		Set("is_public", comp.IsPublic).
		Set("flag_regex", comp.FlagRegex).
		Set("mode", string(comp.Mode)).
		Set("allow_team_switch", comp.AllowTeamSwitch).
		Set("min_team_size", comp.MinTeamSize).
		Set("max_team_size", comp.MaxTeamSize).
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	err := r.exec(ctx, query)
	if err != nil {
		return fmt.Errorf("BackupRepo - ImportCompetition: %w", err)
	}

	return nil
}

// ImportChallenges upserts challenges and their hints in two separate batched
// loops. All hints from every challenge are collected into a flat slice first,
// then inserted independently so that hint rows are not duplicated per batch
// boundary. Both inserts use ON CONFLICT UPDATE to overwrite existing rows.
func (r *BackupRepo) ImportChallenges(ctx context.Context, data *domain.BackupData) error {
	var allHints []domain.Hint

	for _, ch := range data.Challenges {
		allHints = append(allHints, ch.Hints...)
	}

	for i := 0; i < len(data.Challenges); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Challenges))

		batch := data.Challenges[i:end]
		q := sqlBuilder().Insert("challenges").Columns(backupChallengeImportCols...)

		for _, ch := range batch {
			state := domain.ChallengeStateOrDefault(ch.State)

			var flagRegex, flagFormatRegex any

			if ch.FlagRegex != "" {
				flagRegex = ch.FlagRegex
			}

			if ch.FlagFormatRegex != nil && *ch.FlagFormatRegex != "" {
				flagFormatRegex = *ch.FlagFormatRegex
			}

			q = q.Values(ch.ID, ch.Title, ch.Description, ch.Category, ch.FlagHash, ch.Points,
				ch.InitialValue, ch.MinValue, ch.Decay, ch.SolveCount, state, ch.ConnectionInfo, ch.MaxAttempts, int64(ch.MaxAttemptsWindow), ch.Position,
				ch.IsRegex, ch.IsCaseInsensitive, flagRegex, flagFormatRegex)
		}

		err := r.exec(ctx, q.Suffix(backupChallengeUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportChallenges: %w", err)
		}
	}

	return execImportBatches(ctx, r, "ImportChallenges hints", allHints, "hints", backupHintImportCols, backupHintUpsertSuffix,
		func(q squirrel.InsertBuilder, h domain.Hint) squirrel.InsertBuilder {
			return q.Values(h.ID, h.ChallengeID, h.Content, h.Cost, h.OrderIndex, h.Title)
		})
}

// ImportTeams upserts teams in batches. The ON CONFLICT behaviour is driven by
// opts.ConflictMode: ConflictModeSkip emits DO NOTHING; all other modes use the
// full UPDATE suffix defined in backupTeamUpsertSuffix.
func (r *BackupRepo) ImportTeams(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error {
	suffix := backupTeamUpsertSuffix

	if opts.ConflictMode == domain.ConflictModeSkip {
		suffix = "ON CONFLICT (id) DO NOTHING"
	}

	return execImportBatches(ctx, r, "ImportTeams", data.Teams, "teams", backupTeamImportCols, suffix,
		func(q squirrel.InsertBuilder, t domain.TeamExport) squirrel.InsertBuilder {
			return q.Values(t.ID, t.Name, t.CaptainID, t.InviteToken, t.InviteTokenExpiresAt, t.BracketID, t.IsSolo, t.IsBanned, t.BannedAt, t.BannedReason, t.IsHidden, t.CreatedAt)
		})
}

// ImportUsers upserts users in batches. Passwords are replaced with
// backupUserRestoredPasswordHash (a sentinel invalid hash) so that restored
// accounts cannot log in with their old credentials until a reset is performed.
// Conflict behaviour mirrors ImportTeams.
func (r *BackupRepo) ImportUsers(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error {
	suffix := backupUserUpsertSuffix

	if opts.ConflictMode == domain.ConflictModeSkip {
		suffix = "ON CONFLICT (id) DO NOTHING"
	}

	return execImportBatches(ctx, r, "ImportUsers", data.Users, "users", backupUserImportCols, suffix,
		func(q squirrel.InsertBuilder, u domain.UserExport) squirrel.InsertBuilder {
			return q.Values(u.ID, u.Username, u.Email, backupUserRestoredPasswordHash, u.Role, nil, u.IsVerified, u.VerifiedAt, u.IsBanned, u.BannedAt, u.BannedReason, u.CreatedAt)
		})
}

// UpdateUserTeamIDs bulk-sets team_id on restored user rows using an
// UPDATE … FROM (VALUES …) pattern. Placeholders are constructed manually
// because squirrel does not support multi-column VALUES lists natively; the
// generated SQL is safe - no user input is concatenated, only $N positional
// placeholders are used.
func (r *BackupRepo) UpdateUserTeamIDs(ctx context.Context, data *domain.BackupData) error {
	var withTeam []*domain.UserExport

	for i := range data.Users {
		if data.Users[i].TeamID != nil {
			withTeam = append(withTeam, &data.Users[i])
		}
	}

	if len(withTeam) == 0 {
		return nil
	}

	db := r.DB(ctx)

	for i := 0; i < len(withTeam); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(withTeam))

		batch := withTeam[i:end]

		var (
			placeholders []string
			args         []any
		)

		for j, u := range batch {
			placeholders = append(placeholders, fmt.Sprintf("($%d::uuid, $%d::uuid)", backupUserTeamIDValueColumns*j+1, backupUserTeamIDValueColumns*j+backupUserTeamIDValueColumns))
			args = append(args, u.ID, *u.TeamID)
		}

		q := sqlBuilder().Update("users u").
			Set("team_id", squirrel.Expr("v.team_id")).
			From("(VALUES " + strings.Join(placeholders, ", ") + ") AS v(id, team_id)").
			Where(squirrel.Expr("u.id = v.id")).
			PlaceholderFormat(squirrel.Dollar)

		sqlStr, _, err := q.ToSql()
		if err != nil {
			return fmt.Errorf("BackupRepo - UpdateUserTeamIDs - build SQL: %w", err)
		}

		if _, err := db.Exec(ctx, sqlStr, args...); err != nil {
			return fmt.Errorf("BackupRepo - UpdateUserTeamIDs: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportAwards(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportAwards", data.Awards, "awards", backupAwardImportCols, backupAwardUpsertSuffix,
		func(q squirrel.InsertBuilder, a domain.Award) squirrel.InsertBuilder {
			return q.Values(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt, a.BannedTeamID)
		})
}

func (r *BackupRepo) ImportSolves(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportSolves", data.Solves, "solves", backupSolveImportCols, backupSolveConflictSuffix,
		func(q squirrel.InsertBuilder, s domain.Solve) squirrel.InsertBuilder {
			return q.Values(s.ID, s.UserID, s.TeamID, s.ChallengeID, s.SolvedAt, s.PointsAtSolve, s.BannedTeamID, s.BannedUserID)
		})
}

func (r *BackupRepo) ImportHintUnlocks(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportHintUnlocks", data.HintUnlocks, "hint_unlocks", backupHintUnlockImportCols, backupHintUnlockConflictSuffix,
		func(q squirrel.InsertBuilder, u domain.HintUnlock) squirrel.InsertBuilder {
			return q.Values(u.ID, u.HintID, u.TeamID, u.UnlockedAt, u.BannedTeamID)
		})
}

func (r *BackupRepo) ImportFileMetadata(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportFileMetadata", data.Files, "files", backupFileImportCols, backupFileUpsertSuffix,
		func(q squirrel.InsertBuilder, f domain.File) squirrel.InsertBuilder {
			return q.Values(f.ID, f.Type, f.ChallengeID, f.Location, f.Filename, f.Size, f.SHA256, f.CreatedAt)
		})
}

func (r *BackupRepo) ImportBrackets(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportBrackets", data.Brackets, "brackets", backupBracketImportCols, backupBracketUpsertSuffix,
		func(q squirrel.InsertBuilder, b domain.Bracket) squirrel.InsertBuilder {
			var desc *string

			if b.Description != "" {
				desc = &b.Description
			}

			return q.Values(b.ID, b.Name, desc, b.IsDefault, b.CreatedAt)
		})
}

func (r *BackupRepo) ImportChallengeRequirements(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportChallengeRequirements", data.ChallengeRequirements, "challenge_requirements", backupChallengeReqCols, backupChallengeReqConflict,
		func(q squirrel.InsertBuilder, p domain.ChallengeRequirementPair) squirrel.InsertBuilder {
			return q.Values(p.ChallengeID, p.RequiredChallengeID)
		})
}

func (r *BackupRepo) ImportSolutions(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportSolutions", data.Solutions, "solutions", backupSolutionImportCols, backupSolutionUpsertSuffix,
		func(q squirrel.InsertBuilder, s domain.SolutionBackup) squirrel.InsertBuilder {
			return q.Values(s.ID, s.ChallengeID, s.Content)
		})
}

func (r *BackupRepo) ImportRatings(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportRatings", data.Ratings, "ratings", backupRatingImportCols, backupRatingUpsertSuffix,
		func(q squirrel.InsertBuilder, rating domain.Rating) squirrel.InsertBuilder {
			return q.Values(rating.ID, rating.ChallengeID, rating.UserID, rating.TeamID, rating.Value, rating.Review, rating.CreatedAt, rating.UpdatedAt)
		})
}

func (r *BackupRepo) ImportComments(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportComments", data.Comments, "comments", backupCommentImportCols, backupCommentUpsertSuffix,
		func(q squirrel.InsertBuilder, c domain.Comment) squirrel.InsertBuilder {
			return q.Values(c.ID, c.UserID, c.ChallengeID, c.Content, c.CreatedAt, c.UpdatedAt)
		})
}

func (r *BackupRepo) ImportFields(ctx context.Context, data *domain.BackupData) error {
	if len(data.Fields) == 0 {
		return nil
	}

	for i := 0; i < len(data.Fields); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Fields))

		batch := data.Fields[i:end]
		q := sqlBuilder().Insert("fields").Columns(backupFieldImportCols...)

		for _, f := range batch {
			opts, err := json.Marshal(f.Options)
			if err != nil {
				return fmt.Errorf("BackupRepo - ImportFields - marshal options: %w", err)
			}

			var optsVal any = opts
			if len(opts) == 0 {
				optsVal = nil
			}

			q = q.Values(f.ID, f.Name, string(f.FieldType), string(f.EntityType), f.Required, optsVal, f.OrderIndex, f.CreatedAt)
		}

		err := r.exec(ctx, q.Suffix(backupFieldUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportFields: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportFieldValues(ctx context.Context, data *domain.BackupData) error {
	return execImportBatches(ctx, r, "ImportFieldValues", data.FieldValues, "field_values", backupFieldValueImportCols, backupFieldValueUpsertSuffix,
		func(q squirrel.InsertBuilder, v domain.FieldValue) squirrel.InsertBuilder {
			return q.Values(v.ID, v.FieldID, v.EntityID, v.Value, v.CreatedAt)
		})
}
