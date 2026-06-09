package persistent

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

const (
	backupImportBatchSize        = 100
	backupUserTeamIDValueColumns = 2
)

var backupEraseTables = []string{
	"submissions", "solves", "awards", "ratings", "hint_unlocks", "files", "hints", "challenge_topics", "challenge_tags", "challenge_requirements", "solutions", "challenges", "topics", "tags", "users", "teams", "comments", "field_values", "fields", "brackets",
}

// quoteIdentifier returns a PostgreSQL-quoted identifier (double-quote escaped).
func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// backupEraseTablesQuoted maps each allowed table name to its PostgreSQL-quoted form
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
		"initial_value", "min_value", "decay", "solve_count", "state", "attribution", "connection_info", "max_attempts", "max_attempts_window", "position", "next_challenge_id",
		"is_regex", "is_case_insensitive", "flag_regex", "flag_format_regex",
	}
	backupHintImportCols       = []string{"id", "challenge_id", "content", "cost", "order_index", "title"}
	backupTagImportCols        = []string{"id", "name", "color"}
	backupChallengeTagCols     = []string{"challenge_id", "tag_id"}
	backupTopicImportCols      = []string{"id", "name", "created_at"}
	backupChallengeTopicCols   = []string{"challenge_id", "topic_id"}
	backupTeamImportCols       = []string{"id", "name", "captain_id", "invite_token", "invite_token_expires_at", "bracket_id", "is_solo", "is_banned", "banned_at", "banned_reason", "is_hidden", "created_at"}
	backupUserImportCols       = []string{"id", "username", "email", "password_hash", "role", "team_id", "is_verified", "verified_at", "is_banned", "banned_at", "banned_reason", "created_at"}
	backupAwardImportCols      = []string{"id", "team_id", "value", "description", "created_by", "created_at", "banned_team_id"}
	backupSolveImportCols      = []string{"id", "user_id", "team_id", "challenge_id", "solved_at", "points_at_solve", "banned_team_id", "banned_user_id"}
	backupHintUnlockImportCols = []string{"id", "hint_id", "team_id", "unlocked_at", "banned_team_id"}
	backupFileImportCols       = []string{"id", "type", "challenge_id", "page_id", "location", "filename", "size", "sha256", "created_at"}
	backupBracketImportCols    = []string{"id", "name", "description", "is_default", "created_at"}
	backupChallengeReqCols     = []string{"challenge_id", "required_challenge_id"}
	backupSolutionImportCols   = []string{"id", "challenge_id", "content", "state"}
	backupCommentImportCols    = []string{"id", "user_id", "challenge_id", "content", "created_at", "updated_at"}
	backupRatingImportCols     = []string{"id", "challenge_id", "user_id", "team_id", "banned_team_id", "value", "review", "created_at", "updated_at"}
	backupFieldImportCols      = []string{"id", "name", "description", "field_type", "entity_type", "required", "is_public", "editable", "options", "order_index", "created_at"}
	backupFieldValueImportCols = []string{"id", "field_id", "entity_id", "value", "created_at"}
)

const (
	backupChallengeUpsertSuffix = `ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title, description = EXCLUDED.description, category = EXCLUDED.category,
		flag_hash = EXCLUDED.flag_hash, points = EXCLUDED.points, initial_value = EXCLUDED.initial_value,
		min_value = EXCLUDED.min_value, decay = EXCLUDED.decay, solve_count = EXCLUDED.solve_count,
		state = EXCLUDED.state, attribution = EXCLUDED.attribution, connection_info = EXCLUDED.connection_info, max_attempts = EXCLUDED.max_attempts, max_attempts_window = EXCLUDED.max_attempts_window, position = EXCLUDED.position, next_challenge_id = EXCLUDED.next_challenge_id,
		is_regex = EXCLUDED.is_regex, is_case_insensitive = EXCLUDED.is_case_insensitive,
		flag_regex = EXCLUDED.flag_regex, flag_format_regex = EXCLUDED.flag_format_regex`
	backupHintUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, cost = EXCLUDED.cost, order_index = EXCLUDED.order_index, title = EXCLUDED.title`
	backupTopicUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, created_at = EXCLUDED.created_at`
	backupTeamUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, captain_id = EXCLUDED.captain_id, invite_token = EXCLUDED.invite_token, invite_token_expires_at = EXCLUDED.invite_token_expires_at, bracket_id = EXCLUDED.bracket_id, is_solo = EXCLUDED.is_solo, is_banned = EXCLUDED.is_banned, banned_at = EXCLUDED.banned_at, banned_reason = EXCLUDED.banned_reason, is_hidden = EXCLUDED.is_hidden`
	backupUserUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, email = EXCLUDED.email, role = EXCLUDED.role, team_id = EXCLUDED.team_id, is_verified = EXCLUDED.is_verified, verified_at = EXCLUDED.verified_at, is_banned = EXCLUDED.is_banned, banned_at = EXCLUDED.banned_at, banned_reason = EXCLUDED.banned_reason`
	backupUserRestoredPasswordHash = "__RESTORED__"
	backupAwardUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description, banned_team_id = EXCLUDED.banned_team_id`
	backupSolveConflictSuffix      = `ON CONFLICT (id) DO NOTHING`
	backupHintUnlockConflictSuffix = `ON CONFLICT (team_id, hint_id) DO NOTHING`
	backupFileUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET type = EXCLUDED.type, challenge_id = EXCLUDED.challenge_id, page_id = EXCLUDED.page_id, location = EXCLUDED.location, filename = EXCLUDED.filename, size = EXCLUDED.size, sha256 = EXCLUDED.sha256`
	backupBracketUpsertSuffix      = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, is_default = EXCLUDED.is_default`
	backupChallengeReqConflict     = `ON CONFLICT (challenge_id, required_challenge_id) DO NOTHING`
	backupSolutionUpsertSuffix     = `ON CONFLICT (challenge_id) DO UPDATE SET content = EXCLUDED.content, state = EXCLUDED.state`
	backupRatingUpsertSuffix       = `ON CONFLICT (team_id, challenge_id) DO UPDATE SET user_id = EXCLUDED.user_id, banned_team_id = EXCLUDED.banned_team_id, value = EXCLUDED.value, review = EXCLUDED.review, updated_at = EXCLUDED.updated_at`
	backupCommentUpsertSuffix      = `ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, updated_at = EXCLUDED.updated_at`
	backupFieldUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, field_type = EXCLUDED.field_type, entity_type = EXCLUDED.entity_type, required = EXCLUDED.required, is_public = EXCLUDED.is_public, editable = EXCLUDED.editable, options = EXCLUDED.options, order_index = EXCLUDED.order_index`
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

// EraseAllTables truncates all backup-managed tables via EraseTables. Used at the
// start of a full backup import to reset state before replaying exported data.
func (r *BackupRepo) EraseAllTables(ctx context.Context) error {
	return r.EraseTables(ctx, backupEraseTables)
}

// EraseTables truncates a dynamic set of tables with CASCADE. Each table name is
// validated against an allowlist before inclusion to prevent SQL injection; names
// not in the allowlist are silently skipped.
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
	// TRUNCATE ... CASCADE handles FKs and is faster than DELETE; RESTART IDentity resets sequences
	// SQL is built only from precomputed quoted identifiers, no input concatenation
	sql := "TRUNCATE " + strings.Join(quoted, ", ") + " RESTART IDentity CASCADE"

	_, err := r.DB(ctx).Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("BackupRepo - EraseTables: %w", err)
	}

	return nil
}
