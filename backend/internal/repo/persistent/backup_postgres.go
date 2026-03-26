package persistent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

var csvUUIDColumns = map[string]map[string]bool{
	"users":       {"id": true, "team_id": true},
	"teams":       {"id": true, "captain_id": true, "invite_token": true, "bracket_id": true},
	"challenges":  {"id": true},
	"submissions": {"id": true, "user_id": true, "team_id": true, "challenge_id": true, "banned_team_id": true, "banned_user_id": true},
	"solves":      {"id": true, "user_id": true, "team_id": true, "challenge_id": true, "banned_team_id": true, "banned_user_id": true},
	"awards":      {"id": true, "team_id": true, "created_by": true, "banned_team_id": true},
}

const backupImportBatchSize = 100

var backupEraseTables = []string{
	"submissions", "solves", "awards", "ratings", "hint_unlocks", "files", "hints", "challenge_tags", "challenge_requirements", "solutions", "challenges", "tags", "users", "teams", "notifications", "pages", "comments", "field_values", "fields", "brackets",
}

// quoteIdentifier returns a PostgreSQL-quoted identifier (double-quote escaped).
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// backupEraseTablesQuoted maps each allowed table name to its PostgreSQL-quoted form.
// Used so TRUNCATE is built only from fixed quoted identifiers, not string concatenation of input.
var backupEraseTablesQuoted = func() map[string]string {
	m := make(map[string]string, len(backupEraseTables))
	for _, t := range backupEraseTables {
		m[t] = quoteIdentifier(t)
	}

	return m
}()

var (
	backupSubmissionImportCols = []string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "ip", "created_at", "submission_type", "banned_team_id", "banned_user_id"}

	backupChallengeImportCols = []string{
		"id", "title", "description", "category", "flag_hash", "points",
		"initial_value", "min_value", "decay", "solve_count", "state", "connection_info", "max_attempts", "position",
		"is_regex", "is_case_insensitive", "flag_regex", "flag_format_regex",
	}
	backupHintImportCols       = []string{"id", "challenge_id", "content", "cost", "order_index", "title"}
	backupTagImportCols        = []string{"id", "name", "color"}
	backupChallengeTagCols     = []string{"challenge_id", "tag_id"}
	backupTeamImportCols       = []string{"id", "name", "captain_id", "invite_token", "invite_token_expires_at", "bracket_id", "is_solo", "is_banned", "banned_at", "banned_reason", "is_hidden", "created_at"}
	backupUserImportCols       = []string{"id", "username", "email", "password_hash", "role", "team_id", "is_verified", "verified_at", "is_banned", "banned_at", "banned_reason", "created_at"}
	backupAwardImportCols      = []string{"id", "team_id", "value", "description", "created_by", "created_at", "banned_team_id"}
	backupSolveImportCols      = []string{"id", "user_id", "team_id", "challenge_id", "solved_at", "points_at_solve", "banned_team_id", "banned_user_id"}
	backupHintUnlockImportCols = []string{"id", "hint_id", "team_id", "unlocked_at", "banned_team_id"}
	backupFileImportCols       = []string{"id", "type", "challenge_id", "location", "filename", "size", "sha256", "created_at"}
	backupBracketImportCols    = []string{"id", "name", "description", "is_default", "created_at"}
	backupChallengeReqCols     = []string{"challenge_id", "required_challenge_id"}
	backupSolutionImportCols   = []string{"id", "challenge_id", "content"}
	backupCommentImportCols    = []string{"id", "user_id", "challenge_id", "content", "created_at", "updated_at"}
	backupRatingImportCols     = []string{"id", "challenge_id", "user_id", "team_id", "value", "review", "created_at", "updated_at"}
	backupFieldImportCols      = []string{"id", "name", "field_type", "entity_type", "required", "options", "order_index", "created_at"}
	backupFieldValueImportCols = []string{"id", "field_id", "entity_id", "value", "created_at"}
)

const (
	backupChallengeUpsertSuffix = `ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title, description = EXCLUDED.description, category = EXCLUDED.category,
		flag_hash = EXCLUDED.flag_hash, points = EXCLUDED.points, initial_value = EXCLUDED.initial_value,
		min_value = EXCLUDED.min_value, decay = EXCLUDED.decay, solve_count = EXCLUDED.solve_count,
		state = EXCLUDED.state, connection_info = EXCLUDED.connection_info, max_attempts = EXCLUDED.max_attempts, position = EXCLUDED.position,
		is_regex = EXCLUDED.is_regex, is_case_insensitive = EXCLUDED.is_case_insensitive,
		flag_regex = EXCLUDED.flag_regex, flag_format_regex = EXCLUDED.flag_format_regex`
	backupHintUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, cost = EXCLUDED.cost, order_index = EXCLUDED.order_index, title = EXCLUDED.title`
	backupTeamUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, captain_id = EXCLUDED.captain_id, invite_token = EXCLUDED.invite_token, invite_token_expires_at = EXCLUDED.invite_token_expires_at, bracket_id = EXCLUDED.bracket_id, is_solo = EXCLUDED.is_solo, is_banned = EXCLUDED.is_banned, banned_at = EXCLUDED.banned_at, banned_reason = EXCLUDED.banned_reason, is_hidden = EXCLUDED.is_hidden`
	backupUserUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, email = EXCLUDED.email, role = EXCLUDED.role, team_id = EXCLUDED.team_id, is_verified = EXCLUDED.is_verified, verified_at = EXCLUDED.verified_at, is_banned = EXCLUDED.is_banned, banned_at = EXCLUDED.banned_at, banned_reason = EXCLUDED.banned_reason`
	backupUserRestoredPasswordHash = "__RESTORED__"
	backupAwardUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description, banned_team_id = EXCLUDED.banned_team_id`
	backupSolveConflictSuffix      = `ON CONFLICT (id) DO NOTHING`
	backupHintUnlockConflictSuffix = `ON CONFLICT (team_id, hint_id) DO NOTHING`
	backupFileUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET type = EXCLUDED.type, location = EXCLUDED.location, filename = EXCLUDED.filename, size = EXCLUDED.size, sha256 = EXCLUDED.sha256`
	backupBracketUpsertSuffix      = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, is_default = EXCLUDED.is_default`
	backupChallengeReqConflict     = `ON CONFLICT (challenge_id, required_challenge_id) DO NOTHING`
	backupSolutionUpsertSuffix     = `ON CONFLICT (challenge_id) DO UPDATE SET content = EXCLUDED.content`
	backupRatingUpsertSuffix       = `ON CONFLICT (team_id, challenge_id) DO UPDATE SET user_id = EXCLUDED.user_id, value = EXCLUDED.value, review = EXCLUDED.review, updated_at = EXCLUDED.updated_at`
	backupCommentUpsertSuffix      = `ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, updated_at = EXCLUDED.updated_at`
	backupFieldUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, field_type = EXCLUDED.field_type, entity_type = EXCLUDED.entity_type, required = EXCLUDED.required, options = EXCLUDED.options, order_index = EXCLUDED.order_index`
	backupFieldValueUpsertSuffix   = `ON CONFLICT (field_id, entity_id) DO UPDATE SET value = EXCLUDED.value`
)

func (r *BackupRepo) exec(ctx context.Context, b squirrel.Sqlizer) error {
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("BackupRepo - exec - ToSql: %w", err)
	}

	_, err = r.DB(ctx).Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("BackupRepo - exec - Exec: %w", err)
	}

	return nil
}

type BackupRepo struct {
	BaseRepo
}

var _ repo.BackupRepository = (*BackupRepo)(nil)

func NewBackupRepo(pool *pgxpool.Pool) *BackupRepo {
	return &BackupRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *BackupRepo) EraseAllTables(ctx context.Context) error {
	return r.EraseTables(ctx, backupEraseTables)
}

func (r *BackupRepo) EraseTables(ctx context.Context, tables []string) error {
	if len(tables) == 0 {
		return nil
	}

	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		q, ok := backupEraseTablesQuoted[table]
		if !ok {
			return fmt.Errorf("BackupRepo - EraseTables: table %q is not allowed", table)
		}

		quoted = append(quoted, q)
	}
	// TRUNCATE ... CASCADE handles FKs and is faster than DELETE; RESTART IDentity resets sequences.
	// SQL is built only from precomputed quoted identifiers, no input concatenation.
	sql := "TRUNCATE " + strings.Join(quoted, ", ") + " RESTART IDentity CASCADE"

	_, err := r.DB(ctx).Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("BackupRepo - EraseTables: %w", err)
	}

	return nil
}

func (r *BackupRepo) ImportTags(ctx context.Context, data *domain.BackupData) error {
	if len(data.Tags) == 0 {
		return nil
	}

	for i := 0; i < len(data.Tags); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Tags))

		batch := data.Tags[i:end]
		q := SB.Insert("tags").Columns(backupTagImportCols...)

		for _, t := range batch {
			q = q.Values(t.ID, t.Name, t.Color)
		}

		err := r.exec(ctx, q.Suffix("ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, color = EXCLUDED.color"))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportTags: %w", err)
		}
	}

	return nil
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
		q := SB.Insert("challenge_tags").Columns(backupChallengeTagCols...)

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

func (r *BackupRepo) ImportCompetition(ctx context.Context, comp *domain.Competition) error {
	if comp == nil {
		return nil
	}

	query := SB.Update("competition").
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

func (r *BackupRepo) ImportChallenges(ctx context.Context, data *domain.BackupData) error {
	var allHints []domain.Hint

	for _, ch := range data.Challenges {
		allHints = append(allHints, ch.Hints...)
	}

	for i := 0; i < len(data.Challenges); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Challenges))

		batch := data.Challenges[i:end]
		q := SB.Insert("challenges").Columns(backupChallengeImportCols...)

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
				ch.InitialValue, ch.MinValue, ch.Decay, ch.SolveCount, state, ch.ConnectionInfo, ch.MaxAttempts, ch.Position,
				ch.IsRegex, ch.IsCaseInsensitive, flagRegex, flagFormatRegex)
		}

		err := r.exec(ctx, q.Suffix(backupChallengeUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportChallenges: %w", err)
		}
	}

	for i := 0; i < len(allHints); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(allHints))

		batch := allHints[i:end]
		q := SB.Insert("hints").Columns(backupHintImportCols...)

		for _, h := range batch {
			q = q.Values(h.ID, h.ChallengeID, h.Content, h.Cost, h.OrderIndex, h.Title)
		}

		err := r.exec(ctx, q.Suffix(backupHintUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportChallenges hints: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportTeams(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error {
	suffix := backupTeamUpsertSuffix

	if opts.ConflictMode == domain.ConflictModeSkip {
		suffix = "ON CONFLICT (id) DO NOTHING"
	}

	for i := 0; i < len(data.Teams); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Teams))

		batch := data.Teams[i:end]
		q := SB.Insert("teams").Columns(backupTeamImportCols...)

		for _, t := range batch {
			q = q.Values(t.ID, t.Name, t.CaptainID, t.InviteToken, t.InviteTokenExpiresAt, t.BracketID, t.IsSolo, t.IsBanned, t.BannedAt, t.BannedReason, t.IsHidden, t.CreatedAt)
		}

		err := r.exec(ctx, q.Suffix(suffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportTeams: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportUsers(ctx context.Context, data *domain.BackupData, opts domain.ImportOptions) error {
	suffix := backupUserUpsertSuffix

	if opts.ConflictMode == domain.ConflictModeSkip {
		suffix = "ON CONFLICT (id) DO NOTHING"
	}

	for i := 0; i < len(data.Users); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Users))

		batch := data.Users[i:end]
		q := SB.Insert("users").Columns(backupUserImportCols...)

		for _, u := range batch {
			q = q.Values(u.ID, u.Username, u.Email, backupUserRestoredPasswordHash, u.Role, nil, u.IsVerified, u.VerifiedAt, u.IsBanned, u.BannedAt, u.BannedReason, u.CreatedAt)
		}

		err := r.exec(ctx, q.Suffix(suffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportUsers: %w", err)
		}
	}

	return nil
}

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
			placeholders = append(placeholders, fmt.Sprintf("($%d::uuid, $%d::uuid)", 2*j+1, 2*j+2))
			args = append(args, u.ID, *u.TeamID)
		}

		q := SB.Update("users u").
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
	for i := 0; i < len(data.Awards); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Awards))

		batch := data.Awards[i:end]
		q := SB.Insert("awards").Columns(backupAwardImportCols...)

		for _, a := range batch {
			q = q.Values(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt, a.BannedTeamID)
		}

		err := r.exec(ctx, q.Suffix(backupAwardUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportAwards: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportSolves(ctx context.Context, data *domain.BackupData) error {
	for i := 0; i < len(data.Solves); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Solves))

		batch := data.Solves[i:end]
		q := SB.Insert("solves").Columns(backupSolveImportCols...)

		for _, s := range batch {
			q = q.Values(s.ID, s.UserID, s.TeamID, s.ChallengeID, s.SolvedAt, s.PointsAtSolve, s.BannedTeamID, s.BannedUserID)
		}

		err := r.exec(ctx, q.Suffix(backupSolveConflictSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportSolves: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportHintUnlocks(ctx context.Context, data *domain.BackupData) error {
	for i := 0; i < len(data.HintUnlocks); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.HintUnlocks))

		batch := data.HintUnlocks[i:end]
		q := SB.Insert("hint_unlocks").Columns(backupHintUnlockImportCols...)

		for _, u := range batch {
			q = q.Values(u.ID, u.HintID, u.TeamID, u.UnlockedAt, u.BannedTeamID)
		}

		err := r.exec(ctx, q.Suffix(backupHintUnlockConflictSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportHintUnlocks: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportFileMetadata(ctx context.Context, data *domain.BackupData) error {
	for i := 0; i < len(data.Files); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Files))

		batch := data.Files[i:end]
		q := SB.Insert("files").Columns(backupFileImportCols...)

		for _, f := range batch {
			q = q.Values(f.ID, f.Type, f.ChallengeID, f.Location, f.Filename, f.Size, f.SHA256, f.CreatedAt)
		}

		err := r.exec(ctx, q.Suffix(backupFileUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportFileMetadata: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportBrackets(ctx context.Context, data *domain.BackupData) error {
	if len(data.Brackets) == 0 {
		return nil
	}

	for i := 0; i < len(data.Brackets); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Brackets))

		batch := data.Brackets[i:end]
		q := SB.Insert("brackets").Columns(backupBracketImportCols...)

		for _, b := range batch {
			var desc *string

			if b.Description != "" {
				desc = &b.Description
			}

			q = q.Values(b.ID, b.Name, desc, b.IsDefault, b.CreatedAt)
		}

		err := r.exec(ctx, q.Suffix(backupBracketUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportBrackets: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportChallengeRequirements(ctx context.Context, data *domain.BackupData) error {
	if len(data.ChallengeRequirements) == 0 {
		return nil
	}

	for i := 0; i < len(data.ChallengeRequirements); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.ChallengeRequirements))

		batch := data.ChallengeRequirements[i:end]
		q := SB.Insert("challenge_requirements").Columns(backupChallengeReqCols...)

		for _, p := range batch {
			q = q.Values(p.ChallengeID, p.RequiredChallengeID)
		}

		err := r.exec(ctx, q.Suffix(backupChallengeReqConflict))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportChallengeRequirements: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportSolutions(ctx context.Context, data *domain.BackupData) error {
	if len(data.Solutions) == 0 {
		return nil
	}

	for i := 0; i < len(data.Solutions); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Solutions))

		batch := data.Solutions[i:end]
		q := SB.Insert("solutions").Columns(backupSolutionImportCols...)

		for _, s := range batch {
			q = q.Values(s.ID, s.ChallengeID, s.Content)
		}

		err := r.exec(ctx, q.Suffix(backupSolutionUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportSolutions: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportRatings(ctx context.Context, data *domain.BackupData) error {
	if len(data.Ratings) == 0 {
		return nil
	}

	for i := 0; i < len(data.Ratings); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Ratings))

		batch := data.Ratings[i:end]
		q := SB.Insert("ratings").Columns(backupRatingImportCols...)

		for _, rating := range batch {
			q = q.Values(rating.ID, rating.ChallengeID, rating.UserID, rating.TeamID, rating.Value, rating.Review, rating.CreatedAt, rating.UpdatedAt)
		}

		err := r.exec(ctx, q.Suffix(backupRatingUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportRatings: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportComments(ctx context.Context, data *domain.BackupData) error {
	if len(data.Comments) == 0 {
		return nil
	}

	for i := 0; i < len(data.Comments); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Comments))

		batch := data.Comments[i:end]
		q := SB.Insert("comments").Columns(backupCommentImportCols...)

		for _, c := range batch {
			q = q.Values(c.ID, c.UserID, c.ChallengeID, c.Content, c.CreatedAt, c.UpdatedAt)
		}

		err := r.exec(ctx, q.Suffix(backupCommentUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportComments: %w", err)
		}
	}

	return nil
}

func (r *BackupRepo) ImportFields(ctx context.Context, data *domain.BackupData) error {
	if len(data.Fields) == 0 {
		return nil
	}

	for i := 0; i < len(data.Fields); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.Fields))

		batch := data.Fields[i:end]
		q := SB.Insert("fields").Columns(backupFieldImportCols...)

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
	if len(data.FieldValues) == 0 {
		return nil
	}

	for i := 0; i < len(data.FieldValues); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(data.FieldValues))

		batch := data.FieldValues[i:end]
		q := SB.Insert("field_values").Columns(backupFieldValueImportCols...)

		for _, v := range batch {
			q = q.Values(v.ID, v.FieldID, v.EntityID, v.Value, v.CreatedAt)
		}

		err := r.exec(ctx, q.Suffix(backupFieldValueUpsertSuffix))
		if err != nil {
			return fmt.Errorf("BackupRepo - ImportFieldValues: %w", err)
		}
	}

	return nil
}

var csvImportAllowedTables = map[string]bool{
	"users":       true,
	"teams":       true,
	"challenges":  true,
	"submissions": true,
	"solves":      true,
	"awards":      true,
}

var csvAllowedColumns = map[string]map[string]bool{
	"challenges":  toSet(backupChallengeImportCols),
	"teams":       toSet(backupTeamImportCols),
	"users":       toSet(backupUserImportCols),
	"awards":      toSet(backupAwardImportCols),
	"solves":      toSet(backupSolveImportCols),
	"submissions": toSet(backupSubmissionImportCols),
}

func toSet(cols []string) map[string]bool {
	m := make(map[string]bool, len(cols))
	for _, c := range cols {
		m[c] = true
	}

	return m
}

func validateCSVColumns(table string, header []string) error {
	allowed, ok := csvAllowedColumns[table]
	if !ok {
		return fmt.Errorf("no allowed columns defined for table %q", table)
	}

	for _, col := range header {
		if !allowed[col] {
			return fmt.Errorf("column %q is not allowed for table %q", col, table)
		}
	}

	return nil
}

func validateCSVRowValues(table string, header, row []string) string {
	uuidCols := csvUUIDColumns[table]

	for j, col := range header {
		if j >= len(row) {
			break
		}

		v := strings.TrimSpace(row[j])
		if uuidCols[col] && v != "" {
			if _, err := uuid.Parse(v); err != nil {
				return fmt.Sprintf("column %q: invalid UUID %q", col, v)
			}
		}

		if col == "email" && v != "" && !validator.EmailRegex.MatchString(v) {
			return fmt.Sprintf("column email: invalid format %q", v)
		}
	}

	return ""
}

func (r *BackupRepo) ImportCSV(ctx context.Context, tableName string, header []string, rows [][]string) (int, []string, error) {
	if !csvImportAllowedTables[tableName] {
		return 0, nil, fmt.Errorf("BackupRepo - ImportCSV: unsupported table %q", tableName)
	}

	err := validateCSVColumns(tableName, header)
	if err != nil {
		return 0, nil, fmt.Errorf("BackupRepo - ImportCSV: %w", err)
	}

	var csvErrors []string

	type validRow struct {
		index int
		vals  []any
	}

	var validRows []validRow

	for i, row := range rows {
		if len(row) != len(header) {
			csvErrors = append(csvErrors, fmt.Sprintf("row %d: expected %d columns, got %d", i+1, len(header), len(row)))

			continue
		}

		if msg := validateCSVRowValues(tableName, header, row); msg != "" {
			csvErrors = append(csvErrors, fmt.Sprintf("row %d: %s", i+1, msg))

			continue
		}

		vals := make([]any, len(row))
		for j, v := range row {
			vals[j] = v
		}

		validRows = append(validRows, validRow{index: i + 1, vals: vals})
	}

	var imported int

	for i := 0; i < len(validRows); i += backupImportBatchSize {
		end := min(i+backupImportBatchSize, len(validRows))

		batch := validRows[i:end]
		q := SB.Insert(tableName).Columns(header...)

		for _, rw := range batch {
			q = q.Values(rw.vals...)
		}

		q = q.Suffix("ON CONFLICT (id) DO NOTHING")

		err := r.exec(ctx, q)
		if err != nil {
			for _, rw := range batch {
				single := SB.Insert(tableName).Columns(header...).Values(rw.vals...).Suffix("ON CONFLICT (id) DO NOTHING")

				err := r.exec(ctx, single)
				if err != nil {
					csvErrors = append(csvErrors, fmt.Sprintf("row %d: %v", rw.index, err))
				} else {
					imported++
				}
			}

			continue
		}

		imported += len(batch)
	}

	return imported, csvErrors, nil
}
