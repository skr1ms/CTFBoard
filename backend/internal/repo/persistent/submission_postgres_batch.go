package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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

			q := sqlBuilder().Insert("submissions").Columns(cols...).
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
