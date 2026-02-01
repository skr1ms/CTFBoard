package persistent

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
)

var appSettingsColumns = []string{
	"id", "app_name", "verify_emails", "frontend_url", "cors_origins",
	"resend_enabled", "resend_from_email", "resend_from_name",
	"verify_ttl_hours", "reset_ttl_hours",
	"submit_limit_per_user", "submit_limit_duration_min",
	"scoreboard_visible", "registration_open", "updated_at",
}

func scanAppSettings(row rowScanner) (*entity.AppSettings, error) {
	var s entity.AppSettings
	err := row.Scan(
		&s.ID, &s.AppName, &s.VerifyEmails, &s.FrontendURL, &s.CORSOrigins,
		&s.ResendEnabled, &s.ResendFromEmail, &s.ResendFromName,
		&s.VerifyTTLHours, &s.ResetTTLHours,
		&s.SubmitLimitPerUser, &s.SubmitLimitDurationMin,
		&s.ScoreboardVisible, &s.RegistrationOpen, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

type AppSettingsRepo struct {
	pool *pgxpool.Pool
}

func NewAppSettingsRepo(pool *pgxpool.Pool) *AppSettingsRepo {
	return &AppSettingsRepo{pool: pool}
}

func (r *AppSettingsRepo) Get(ctx context.Context) (*entity.AppSettings, error) {
	query := squirrel.Select(appSettingsColumns...).
		From("app_settings").
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("AppSettingsRepo - Get - BuildQuery: %w", err)
	}

	s, err := scanAppSettings(r.pool.QueryRow(ctx, sqlQuery, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entityError.ErrAppSettingsNotFound
		}
		return nil, fmt.Errorf("AppSettingsRepo - Get: %w", err)
	}
	return s, nil
}

func (r *AppSettingsRepo) Update(ctx context.Context, s *entity.AppSettings) error {
	query := squirrel.Update("app_settings").
		Set("app_name", s.AppName).
		Set("verify_emails", s.VerifyEmails).
		Set("frontend_url", s.FrontendURL).
		Set("cors_origins", s.CORSOrigins).
		Set("resend_enabled", s.ResendEnabled).
		Set("resend_from_email", s.ResendFromEmail).
		Set("resend_from_name", s.ResendFromName).
		Set("verify_ttl_hours", s.VerifyTTLHours).
		Set("reset_ttl_hours", s.ResetTTLHours).
		Set("submit_limit_per_user", s.SubmitLimitPerUser).
		Set("submit_limit_duration_min", s.SubmitLimitDurationMin).
		Set("scoreboard_visible", s.ScoreboardVisible).
		Set("registration_open", s.RegistrationOpen).
		Set("updated_at", time.Now()).
		Where(squirrel.Eq{"id": 1}).
		PlaceholderFormat(squirrel.Dollar)

	sqlQuery, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update - BuildQuery: %w", err)
	}

	result, err := r.pool.Exec(ctx, sqlQuery, args...)
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update: %w", err)
	}

	if result.RowsAffected() == 0 {
		return entityError.ErrAppSettingsNotFound
	}

	return nil
}
