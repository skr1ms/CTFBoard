package persistent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/jackc/pgx/v5/pgxpool"
)

var backupEraseTables = []string{
	"submissions", "solves", "awards", "hint_unlocks", "files", "hints", "challenges", "users", "teams", "notifications", "pages",
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
	backupSubmissionImportCols = []string{"id", "user_id", "team_id", "challenge_id", "submitted_flag", "is_correct", "ip", "created_at"}

	backupChallengeImportCols = []string{
		"id", "title", "description", "category", "flag_hash", "points",
		"initial_value", "min_value", "decay", "solve_count", "is_hidden", "is_regex", "is_case_insensitive", "flag_regex",
	}
	backupHintImportCols       = []string{"id", "challenge_id", "content", "cost", "order_index"}
	backupTeamImportCols       = []string{"id", "name", "captain_id", "invite_token", "is_solo", "is_banned", "banned_at", "banned_reason", "is_hidden", "created_at"}
	backupUserImportCols       = []string{"id", "username", "email", "password_hash", "role", "team_id"}
	backupAwardImportCols      = []string{"id", "team_id", "value", "description", "created_by", "created_at"}
	backupSolveImportCols      = []string{"id", "user_id", "team_id", "challenge_id", "solved_at", "points_at_solve"}
	backupHintUnlockImportCols = []string{"id", "hint_id", "team_id", "unlocked_at"}
	backupFileImportCols       = []string{"id", "type", "challenge_id", "location", "filename", "size", "sha256", "created_at"}
)

const (
	backupChallengeUpsertSuffix = `ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title, description = EXCLUDED.description, category = EXCLUDED.category,
		flag_hash = EXCLUDED.flag_hash, points = EXCLUDED.points, initial_value = EXCLUDED.initial_value,
		min_value = EXCLUDED.min_value, decay = EXCLUDED.decay, solve_count = EXCLUDED.solve_count,
		is_hidden = EXCLUDED.is_hidden, is_regex = EXCLUDED.is_regex, is_case_insensitive = EXCLUDED.is_case_insensitive,
		flag_regex = EXCLUDED.flag_regex`
	backupHintUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, cost = EXCLUDED.cost, order_index = EXCLUDED.order_index`
	backupTeamUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, captain_id = EXCLUDED.captain_id, invite_token = EXCLUDED.invite_token, is_solo = EXCLUDED.is_solo, is_banned = EXCLUDED.is_banned, banned_at = EXCLUDED.banned_at, banned_reason = EXCLUDED.banned_reason, is_hidden = EXCLUDED.is_hidden`
	backupUserUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, email = EXCLUDED.email, role = EXCLUDED.role, team_id = EXCLUDED.team_id`
	backupUserRestoredPasswordHash = "__RESTORED__"
	backupAwardUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description`
	backupSolveConflictSuffix      = `ON CONFLICT (id) DO NOTHING`
	backupHintUnlockConflictSuffix = `ON CONFLICT (team_id, hint_id) DO NOTHING`
	backupFileUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET type = EXCLUDED.type, location = EXCLUDED.location, filename = EXCLUDED.filename, size = EXCLUDED.size, sha256 = EXCLUDED.sha256`
)

func (r *BackupRepo) exec(ctx context.Context, b squirrel.Sqlizer) error {
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return fmt.Errorf("BackupRepo - exec - ToSql: %w", err)
	}
	_, err = ExtractDB(ctx, r.pool).Exec(ctx, sqlStr, args...)
	if err != nil {
		return fmt.Errorf("BackupRepo - exec - Exec: %w", err)
	}
	return nil
}

type BackupRepo struct {
	pool *pgxpool.Pool
}

var _ repo.BackupRepository = (*BackupRepo)(nil)

func NewBackupRepo(pool *pgxpool.Pool) *BackupRepo {
	return &BackupRepo{pool: pool}
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
	// TRUNCATE ... CASCADE handles FKs and is faster than DELETE; RESTART IDENTITY resets sequences.
	// SQL is built only from precomputed quoted identifiers, no input concatenation.
	sql := "TRUNCATE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE"
	_, err := ExtractDB(ctx, r.pool).Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("BackupRepo - EraseTables: %w", err)
	}
	return nil
}

func (r *BackupRepo) ImportCompetition(ctx context.Context, comp *entity.Competition) error {
	if comp == nil {
		return nil
	}

	query := squirrel.Update("competition").
		Set("name", comp.Name).
		Set("start_time", comp.StartTime).
		Set("end_time", comp.EndTime).
		Set("freeze_time", comp.FreezeTime).
		Set("is_paused", comp.IsPaused).
		Set("mode", string(comp.Mode)).
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	if err := r.exec(ctx, query); err != nil {
		return fmt.Errorf("BackupRepo - ImportCompetition: %w", err)
	}
	return nil
}

func (r *BackupRepo) ImportChallenges(ctx context.Context, data *entity.BackupData) error {
	for _, ch := range data.Challenges {
		query := squirrel.Insert("challenges").
			Columns(backupChallengeImportCols...).
			Values(ch.ID, ch.Title, ch.Description, ch.Category, ch.FlagHash, ch.Points,
				ch.InitialValue, ch.MinValue, ch.Decay, ch.SolveCount, ch.IsHidden, ch.IsRegex, ch.IsCaseInsensitive, ch.FlagRegex).
			Suffix(backupChallengeUpsertSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportChallenges - challenge %s: %w", ch.ID, err)
		}

		for _, hint := range ch.Hints {
			hintQuery := squirrel.Insert("hints").
				Columns(backupHintImportCols...).
				Values(hint.ID, hint.ChallengeID, hint.Content, hint.Cost, hint.OrderIndex).
				Suffix(backupHintUpsertSuffix).
				PlaceholderFormat(squirrel.Dollar)

			if err := r.exec(ctx, hintQuery); err != nil {
				return fmt.Errorf("BackupRepo - ImportChallenges - hint %s: %w", hint.ID, err)
			}
		}
	}
	return nil
}

func (r *BackupRepo) ImportTeams(ctx context.Context, data *entity.BackupData, opts entity.ImportOptions) error {
	for _, t := range data.Teams {
		base := squirrel.Insert("teams").
			Columns(backupTeamImportCols...).
			Values(t.ID, t.Name, t.CaptainID, t.InviteToken, t.IsSolo, t.IsBanned, t.BannedAt, t.BannedReason, t.IsHidden, t.CreatedAt).
			PlaceholderFormat(squirrel.Dollar)

		var query squirrel.InsertBuilder
		if opts.ConflictMode == entity.ConflictModeSkip {
			query = base.Suffix("ON CONFLICT (id) DO NOTHING")
		} else {
			query = base.Suffix(backupTeamUpsertSuffix)
		}

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportTeams - team %s: %w", t.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportUsers(ctx context.Context, data *entity.BackupData, opts entity.ImportOptions) error {
	for _, u := range data.Users {
		base := squirrel.Insert("users").
			Columns(backupUserImportCols...).
			Values(u.ID, u.Username, u.Email, backupUserRestoredPasswordHash, u.Role, u.TeamID).
			PlaceholderFormat(squirrel.Dollar)

		var query squirrel.InsertBuilder
		if opts.ConflictMode == entity.ConflictModeSkip {
			query = base.Suffix("ON CONFLICT (id) DO NOTHING")
		} else {
			query = base.Suffix(backupUserUpsertSuffix)
		}

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportUsers - user %s: %w", u.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportAwards(ctx context.Context, data *entity.BackupData) error {
	for _, a := range data.Awards {
		query := squirrel.Insert("awards").
			Columns(backupAwardImportCols...).
			Values(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt).
			Suffix(backupAwardUpsertSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportAwards - award %s: %w", a.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportSolves(ctx context.Context, data *entity.BackupData) error {
	for _, s := range data.Solves {
		query := squirrel.Insert("solves").
			Columns(backupSolveImportCols...).
			Values(s.ID, s.UserID, s.TeamID, s.ChallengeID, s.SolvedAt, s.PointsAtSolve).
			Suffix(backupSolveConflictSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportSolves - solve %s: %w", s.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportHintUnlocks(ctx context.Context, data *entity.BackupData) error {
	for _, u := range data.HintUnlocks {
		query := squirrel.Insert("hint_unlocks").
			Columns(backupHintUnlockImportCols...).
			Values(u.ID, u.HintID, u.TeamID, u.UnlockedAt).
			Suffix(backupHintUnlockConflictSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportHintUnlocks - unlock %s: %w", u.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportFileMetadata(ctx context.Context, data *entity.BackupData) error {
	for _, f := range data.Files {
		query := squirrel.Insert("files").
			Columns(backupFileImportCols...).
			Values(f.ID, f.Type, f.ChallengeID, f.Location, f.Filename, f.Size, f.SHA256, f.CreatedAt).
			Suffix(backupFileUpsertSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := r.exec(ctx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportFileMetadata - file %s: %w", f.ID, err)
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

func (r *BackupRepo) ImportCSV(ctx context.Context, tableName string, header []string, rows [][]string) (int, []string, error) {
	if !csvImportAllowedTables[tableName] {
		return 0, nil, fmt.Errorf("BackupRepo - ImportCSV: unsupported table %q", tableName)
	}
	if err := validateCSVColumns(tableName, header); err != nil {
		return 0, nil, fmt.Errorf("BackupRepo - ImportCSV: %w", err)
	}

	var imported int
	var csvErrors []string

	for i, row := range rows {
		if len(row) != len(header) {
			csvErrors = append(csvErrors, fmt.Sprintf("row %d: expected %d columns, got %d", i+1, len(header), len(row)))
			continue
		}

		q := squirrel.Insert(tableName).Columns(header...).PlaceholderFormat(squirrel.Dollar)
		vals := make([]any, len(row))
		for j, v := range row {
			vals[j] = v
		}
		q = q.Values(vals...).Suffix("ON CONFLICT DO NOTHING")

		if err := r.exec(ctx, q); err != nil {
			csvErrors = append(csvErrors, fmt.Sprintf("row %d: %v", i+1, err))
			continue
		}
		imported++
	}

	return imported, csvErrors, nil
}
