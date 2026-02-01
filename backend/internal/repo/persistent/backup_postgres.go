package persistent

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

var backupEraseTables = []string{
	"solves", "awards", "hint_unlocks", "files", "hints", "challenges", "users", "teams",
}

var (
	backupChallengeImportCols = []string{
		"id", "title", "description", "category", "flag_hash", "points",
		"initial_value", "min_value", "decay", "solve_count", "is_hidden", "is_regex", "is_case_insensitive", "flag_regex",
	}
	backupHintImportCols  = []string{"id", "challenge_id", "content", "cost", "order_index"}
	backupTeamImportCols  = []string{"id", "name", "captain_id", "invite_token", "is_solo", "is_banned", "banned_reason", "is_hidden", "created_at"}
	backupUserImportCols  = []string{"id", "username", "email", "password_hash", "role", "team_id"}
	backupAwardImportCols = []string{"id", "team_id", "value", "description", "created_by", "created_at"}
	backupSolveImportCols = []string{"id", "user_id", "team_id", "challenge_id", "solved_at"}
	backupFileImportCols  = []string{"id", "type", "challenge_id", "location", "filename", "size", "sha256", "created_at"}
)

const (
	backupChallengeUpsertSuffix = `ON CONFLICT (id) DO UPDATE SET
		title = EXCLUDED.title, description = EXCLUDED.description, category = EXCLUDED.category,
		flag_hash = EXCLUDED.flag_hash, points = EXCLUDED.points, initial_value = EXCLUDED.initial_value,
		min_value = EXCLUDED.min_value, decay = EXCLUDED.decay, solve_count = EXCLUDED.solve_count,
		is_hidden = EXCLUDED.is_hidden, is_regex = EXCLUDED.is_regex, is_case_insensitive = EXCLUDED.is_case_insensitive,
		flag_regex = EXCLUDED.flag_regex`
	backupHintUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET content = EXCLUDED.content, cost = EXCLUDED.cost, order_index = EXCLUDED.order_index`
	backupTeamUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, captain_id = EXCLUDED.captain_id, invite_token = EXCLUDED.invite_token, is_solo = EXCLUDED.is_solo, is_banned = EXCLUDED.is_banned, banned_reason = EXCLUDED.banned_reason, is_hidden = EXCLUDED.is_hidden`
	backupUserUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, email = EXCLUDED.email, role = EXCLUDED.role, team_id = EXCLUDED.team_id`
	backupUserRestoredPasswordHash = "__RESTORED__"
	backupAwardUpsertSuffix        = `ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value, description = EXCLUDED.description`
	backupSolveConflictSuffix      = `ON CONFLICT (id) DO NOTHING`
	backupFileUpsertSuffix         = `ON CONFLICT (id) DO UPDATE SET type = EXCLUDED.type, location = EXCLUDED.location, filename = EXCLUDED.filename, size = EXCLUDED.size, sha256 = EXCLUDED.sha256`
)

func execTx(ctx context.Context, tx pgx.Tx, b squirrel.Sqlizer) error {
	sqlStr, args, err := b.ToSql()
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, sqlStr, args...)
	return err
}

type BackupRepo struct {
	pool *pgxpool.Pool
}

func NewBackupRepo(pool *pgxpool.Pool) *BackupRepo {
	return &BackupRepo{pool: pool}
}

func (r *BackupRepo) EraseAllTablesTx(ctx context.Context, tx pgx.Tx) error {
	for _, table := range backupEraseTables {
		query := squirrel.Delete(table).PlaceholderFormat(squirrel.Dollar)
		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - EraseAllTablesTx - delete %s: %w", table, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportCompetitionTx(ctx context.Context, tx pgx.Tx, comp *entity.Competition) error {
	if comp == nil {
		return nil
	}

	query := squirrel.Update("competition").
		Set("name", comp.Name).
		Set("start_time", comp.StartTime).
		Set("end_time", comp.EndTime).
		Set("freeze_time", comp.FreezeTime).
		Set("is_paused", comp.IsPaused).
		Set("mode", comp.Mode).
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	if err := execTx(ctx, tx, query); err != nil {
		return fmt.Errorf("BackupRepo - ImportCompetitionTx: %w", err)
	}
	return nil
}

func (r *BackupRepo) ImportChallengesTx(ctx context.Context, tx pgx.Tx, data *entity.BackupData) error {
	for _, ch := range data.Challenges {
		query := squirrel.Insert("challenges").
			Columns(backupChallengeImportCols...).
			Values(ch.ID, ch.Title, ch.Description, ch.Category, ch.FlagHash, ch.Points,
				ch.InitialValue, ch.MinValue, ch.Decay, ch.SolveCount, ch.IsHidden, ch.IsRegex, ch.IsCaseInsensitive, ch.FlagRegex).
			Suffix(backupChallengeUpsertSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportChallengesTx - challenge %s: %w", ch.ID, err)
		}

		for _, hint := range ch.Hints {
			hintQuery := squirrel.Insert("hints").
				Columns(backupHintImportCols...).
				Values(hint.ID, hint.ChallengeID, hint.Content, hint.Cost, hint.OrderIndex).
				Suffix(backupHintUpsertSuffix).
				PlaceholderFormat(squirrel.Dollar)

			if err := execTx(ctx, tx, hintQuery); err != nil {
				return fmt.Errorf("BackupRepo - ImportChallengesTx - hint %s: %w", hint.ID, err)
			}
		}
	}
	return nil
}

func (r *BackupRepo) ImportTeamsTx(ctx context.Context, tx pgx.Tx, data *entity.BackupData, opts entity.ImportOptions) error {
	for _, t := range data.Teams {
		base := squirrel.Insert("teams").
			Columns(backupTeamImportCols...).
			Values(t.ID, t.Name, t.CaptainID, t.InviteToken, t.IsSolo, t.IsBanned, t.BannedReason, t.IsHidden, t.CreatedAt).
			PlaceholderFormat(squirrel.Dollar)

		var query squirrel.InsertBuilder
		if opts.ConflictMode == entity.ConflictModeSkip {
			query = base.Suffix("ON CONFLICT (id) DO NOTHING")
		} else {
			query = base.Suffix(backupTeamUpsertSuffix)
		}

		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportTeamsTx - team %s: %w", t.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportUsersTx(ctx context.Context, tx pgx.Tx, data *entity.BackupData, opts entity.ImportOptions) error {
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

		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportUsersTx - user %s: %w", u.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportAwardsTx(ctx context.Context, tx pgx.Tx, data *entity.BackupData) error {
	for _, a := range data.Awards {
		query := squirrel.Insert("awards").
			Columns(backupAwardImportCols...).
			Values(a.ID, a.TeamID, a.Value, a.Description, a.CreatedBy, a.CreatedAt).
			Suffix(backupAwardUpsertSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportAwardsTx - award %s: %w", a.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportSolvesTx(ctx context.Context, tx pgx.Tx, data *entity.BackupData) error {
	for _, s := range data.Solves {
		query := squirrel.Insert("solves").
			Columns(backupSolveImportCols...).
			Values(s.ID, s.UserID, s.TeamID, s.ChallengeID, s.SolvedAt).
			Suffix(backupSolveConflictSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportSolvesTx - solve %s: %w", s.ID, err)
		}
	}
	return nil
}

func (r *BackupRepo) ImportFileMetadataTx(ctx context.Context, tx pgx.Tx, data *entity.BackupData) error {
	for _, f := range data.Files {
		query := squirrel.Insert("files").
			Columns(backupFileImportCols...).
			Values(f.ID, f.Type, f.ChallengeID, f.Location, f.Filename, f.Size, f.SHA256, f.CreatedAt).
			Suffix(backupFileUpsertSuffix).
			PlaceholderFormat(squirrel.Dollar)

		if err := execTx(ctx, tx, query); err != nil {
			return fmt.Errorf("BackupRepo - ImportFileMetadataTx - file %s: %w", f.ID, err)
		}
	}
	return nil
}
