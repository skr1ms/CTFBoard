package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type AppSettingsRepo struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewAppSettingsRepo(db *pgxpool.Pool) *AppSettingsRepo {
	return &AppSettingsRepo{db: db, q: sqlc.New(db)}
}

func toEntityAppSettings(s sqlc.AppSetting) *entity.AppSettings {
	return &entity.AppSettings{
		ID:                     int(s.ID),
		AppName:                s.AppName,
		VerifyEmails:           s.VerifyEmails,
		FrontendURL:            s.FrontendUrl,
		CORSOrigins:            s.CorsOrigins,
		ResendEnabled:          s.ResendEnabled,
		ResendFromEmail:        s.ResendFromEmail,
		ResendFromName:         s.ResendFromName,
		VerifyTTLHours:         int(s.VerifyTtlHours),
		ResetTTLHours:          int(s.ResetTtlHours),
		SubmitLimitPerUser:     int(s.SubmitLimitPerUser),
		SubmitLimitDurationMin: int(s.SubmitLimitDurationMin),
		ScoreboardVisible:      s.ScoreboardVisible,
		RegistrationOpen:       s.RegistrationOpen,
		UpdatedAt:              s.UpdatedAt,
	}
}

func (r *AppSettingsRepo) Get(ctx context.Context) (*entity.AppSettings, error) {
	s, err := r.q.GetAppSettings(ctx)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrAppSettingsNotFound
		}
		return nil, fmt.Errorf("AppSettingsRepo - Get: %w", err)
	}
	return toEntityAppSettings(s), nil
}

func (r *AppSettingsRepo) Update(ctx context.Context, s *entity.AppSettings) error {
	verifyTtl, err := intToInt32Safe(s.VerifyTTLHours)
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update VerifyTTLHours: %w", err)
	}
	resetTtl, err := intToInt32Safe(s.ResetTTLHours)
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update ResetTTLHours: %w", err)
	}
	submitLimit, err := intToInt32Safe(s.SubmitLimitPerUser)
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update SubmitLimitPerUser: %w", err)
	}
	submitDuration, err := intToInt32Safe(s.SubmitLimitDurationMin)
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update SubmitLimitDurationMin: %w", err)
	}
	err = r.q.UpdateAppSettings(ctx, sqlc.UpdateAppSettingsParams{
		AppName:                s.AppName,
		VerifyEmails:           s.VerifyEmails,
		FrontendUrl:            s.FrontendURL,
		CorsOrigins:            s.CORSOrigins,
		ResendEnabled:          s.ResendEnabled,
		ResendFromEmail:        s.ResendFromEmail,
		ResendFromName:         s.ResendFromName,
		VerifyTtlHours:         verifyTtl,
		ResetTtlHours:          resetTtl,
		SubmitLimitPerUser:     submitLimit,
		SubmitLimitDurationMin: submitDuration,
		ScoreboardVisible:      s.ScoreboardVisible,
		RegistrationOpen:       s.RegistrationOpen,
		UpdatedAt:              time.Now(),
	})
	if err != nil {
		return fmt.Errorf("AppSettingsRepo - Update: %w", err)
	}
	return nil
}
