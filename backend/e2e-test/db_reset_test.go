package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
)

//nolint:thelper // test helper: truncates and re-seeds all tables; t can be nil so t.Helper() is guarded
func truncateE2EDB(ctx context.Context, t *testing.T) error {
	if t != nil {
		t.Helper()
	}

	truncateAndSeed := func() error {
		if TestPool == nil {
			return fmt.Errorf("TestPool is not initialized")
		}

		_, err := TestPool.Exec(ctx, `TRUNCATE TABLE
			configs, comments, api_tokens, backup_import_jobs,
			field_values, fields, brackets, pages, user_notifications, notifications,
			submissions, challenge_tags, tags, audit_logs, team_audit_log, app_settings,
			solutions, files, verification_tokens, awards, hint_unlocks, hints, solves,
			ratings, challenges, teams, users, competition, ban_appeals
			RESTART IDentity CASCADE`)
		if err != nil {
			return err
		}

		_, err = TestPool.Exec(ctx, `INSERT INTO competition (id, name, is_paused, is_public, mode, allow_team_switch, min_team_size, max_team_size, start_time, end_time)
			VALUES (1, 'CTF Competition', FALSE, TRUE, 'teams_only', TRUE, 1, 10, NULL, NULL)
			ON CONFLICT (id) DO UPDATE set name = EXCLUDED.name, is_paused = EXCLUDED.is_paused, is_public = EXCLUDED.is_public, mode = EXCLUDED.mode, allow_team_switch = EXCLUDED.allow_team_switch, min_team_size = EXCLUDED.min_team_size, max_team_size = EXCLUDED.max_team_size, start_time = EXCLUDED.start_time, end_time = EXCLUDED.end_time, updated_at = NOW()`)
		if err != nil {
			return err
		}

		_, err = TestPool.Exec(ctx, `INSERT INTO app_settings (
				id, app_name, verify_emails, frontend_url, cors_origins,
				resend_enabled, resend_from_email, resend_from_name,
				verify_ttl_hours, reset_ttl_hours, submit_limit_per_user, submit_limit_duration_min,
				registration_open,
				rate_limit_login_per_minute, rate_limit_register_per_minute,
				rate_limit_forgot_password_per_minute, rate_limit_reset_password_per_minute,
				rate_limit_logout_per_minute, rate_limit_refresh_per_minute,
				rate_limit_scoreboard_per_minute, rate_limit_general_ip_per_minute,
				rate_limit_verify_email_per_minute, rate_limit_oauth_callback_per_minute,
				updated_at
			) VALUES (
				1, 'CTF Platform', FALSE, 'http://localhost:3000', 'http://localhost:3000,http://localhost:5173',
				TRUE, 'noreply@ctf-platform.local', 'CTF Platform',
				24, 1, 500000, 1,
				TRUE,
				10000, 10000,
				100000, 10000,
				10000, 10000,
				10000, 10000,
				10000, 10000,
				now()
			) ON CONFLICT (id) DO UPDATE SET
				submit_limit_per_user = 500000,
				submit_limit_duration_min = 1,
				rate_limit_login_per_minute = 10000,
				rate_limit_register_per_minute = 10000,
				rate_limit_forgot_password_per_minute = 100000,
				rate_limit_reset_password_per_minute = 10000,
				rate_limit_logout_per_minute = 10000,
				rate_limit_refresh_per_minute = 10000,
				rate_limit_scoreboard_per_minute = 10000,
				rate_limit_general_ip_per_minute = 10000,
				rate_limit_verify_email_per_minute = 10000,
				rate_limit_oauth_callback_per_minute = 10000,
				updated_at = NOW()`)
		if err != nil {
			return err
		}

		if TestRedis != nil {
			_ = TestRedis.Del(ctx, cache.KeyAppSettings)

			for _, pattern := range []string{"limiter:*", "e2e:*"} {
				var cursor uint64

				for {
					keys, next, scanErr := TestRedis.Scan(ctx, cursor, pattern, 100).Result()
					if scanErr != nil {
						break
					}

					if len(keys) > 0 {
						_ = TestRedis.Del(ctx, keys...)
					}

					cursor = next
					if cursor == 0 {
						break
					}
				}
			}
		}

		return nil
	}
	bo := backoff.NewExponentialBackOff()
	bo.InitialInterval = 50 * time.Millisecond
	bo.MaxElapsedTime = 10 * time.Second

	err := backoff.Retry(func() error {
		err := truncateAndSeed()
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "40P01" {
				return err // retry on deadlock
			}

			return backoff.Permanent(err)
		}

		return nil
	}, backoff.WithContext(bo, ctx))
	if err != nil {
		if t != nil {
			t.Fatalf("truncate db: %v", err)
		}

		return err
	}

	return nil
}

// resetAppSettingsFull resets all app_settings fields that tests may mutate.
// Use in t.Cleanup after tests that change these global settings.
func resetAppSettingsFull() {
	ctx := context.Background()

	_, err := TestPool.Exec(ctx, `UPDATE app_settings SET
		verify_emails = FALSE, registration_open = TRUE, max_teams = 0,
		submit_limit_per_user = 500000, submit_limit_duration_min = 1,
		rate_limit_login_per_minute = 10000, rate_limit_register_per_minute = 10000,
		rate_limit_forgot_password_per_minute = 100000, rate_limit_reset_password_per_minute = 10000,
		rate_limit_logout_per_minute = 10000, rate_limit_refresh_per_minute = 10000,
		rate_limit_scoreboard_per_minute = 10000, rate_limit_general_ip_per_minute = 10000,
		rate_limit_verify_email_per_minute = 10000, rate_limit_oauth_callback_per_minute = 10000,
		updated_at = NOW() WHERE id = 1`)
	if err != nil {
		panic("resetAppSettingsFull: " + err.Error())
	}

	if TestRedis != nil {
		_ = TestRedis.Del(ctx, "app_settings")
	}

	if testRateLimitCache != nil {
		testRateLimitCache.Invalidate()
	}
}

// resetCompetitionToActive sets competition id=1 to the default active teams-only baseline.
// Use in t.Cleanup for tests that mutate global competition state.
func resetCompetitionToActive() {
	ctx := context.Background()
	now := time.Now().UTC()

	_, err := TestPool.Exec(ctx, `UPDATE competition SET is_paused = FALSE, mode = 'teams_only', start_time = $1, end_time = $2, freeze_time = NULL, updated_at = now() WHERE id = 1`,
		now.Add(-1*time.Hour), now.Add(24*time.Hour))
	if err != nil {
		panic("resetCompetitionToActive: " + err.Error())
	}

	_ = TestRedis.Del(ctx, "competition")

	if testCompetitionUC != nil {
		testCompetitionUC.InvalidateLocalCache()
	}
}

// setCompetitionTimes sets competition id=1 times directly in DB, bypassing API restrictions
// Use when parallel tests may have activated the competition between resetCompetitionToNotStarted
// and the API PUT call. Pass nil for freezeTime to clear it.
func setCompetitionTimes(startTime, endTime time.Time, freezeTime *time.Time) {
	ctx := context.Background()

	_, err := TestPool.Exec(ctx, `UPDATE competition SET start_time = $1, end_time = $2, freeze_time = $3, is_paused = FALSE, updated_at = now() WHERE id = 1`,
		startTime, endTime, freezeTime)
	if err != nil {
		panic("setCompetitionTimes: " + err.Error())
	}

	_ = TestRedis.Del(ctx, "competition")

	if testCompetitionUC != nil {
		testCompetitionUC.InvalidateLocalCache()
	}
}

func WithCompetitionTimes(t *testing.T, start, end time.Time, freeze *time.Time) {
	t.Helper()
	t.Cleanup(resetCompetitionToActive)
	setCompetitionTimes(start, end, freeze)
}

// invalidateScoreboardCache clears scoreboard and frozen scoreboard keys in Redis.
func invalidateScoreboardCache(ctx context.Context) {
	if TestRedis == nil {
		return
	}

	c := cachekit.New(TestRedis)

	err := c.Del(ctx, cache.KeyScoreboard)
	if err != nil {
		return
	}

	err = c.DeleteByPrefix(ctx, cache.KeyScoreboardFrozenPrefix)
	if err != nil {
		return
	}

	err = c.DeleteByPrefix(ctx, cache.KeyScoreboardBracketPrefix)
	if err != nil {
		return
	}
}

// Infrastructure Setup
