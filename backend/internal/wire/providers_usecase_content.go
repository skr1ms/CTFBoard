package wire

import (
	"context"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/page"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

func ProvideFieldUseCase(fieldRepo settings.FieldRepository) *settings.FieldUseCase {
	return settings.NewFieldUseCase(settings.FieldDeps{FieldRepo: fieldRepo})
}

func ProvideFieldValidator(fieldRepo settings.FieldRepository) *settings.FieldValidator {
	return settings.NewFieldValidator(fieldRepo)
}

func ProvideNotificationUseCase(notifRepo notification.NotificationRepository, broadcaster *iws.Broadcaster, l logkit.Logger) *notification.NotificationUseCase {
	return notification.NewNotificationUseCase(notification.NotificationDeps{
		NotifRepo:   notifRepo,
		Broadcaster: broadcaster,
		Logger:      l,
	})
}

func ProvidePageUseCase(pageRepo page.PageRepository, l logkit.Logger) *page.PageUseCase {
	return page.NewPageUseCase(page.PageDeps{PageRepo: pageRepo, Logger: l})
}

func ProvideSettingsUseCase(
	ctx context.Context,
	settingsRepo settings.SettingsRepository,
	auditLogRepo settings.AuditLogRepository,
	TM settings.TransactionManager,
	kv cachekit.KeyValueStore,
	competitionRepo settings.CompetitionRepository,
	competitionParamUC *competition.CompetitionParamUseCase,
	pubsub cachekit.PubSubStore,
	runtimeInvalidator *runtimeSettingsInvalidator,
	l logkit.Logger,
) *settings.SettingsUseCase {
	return settings.NewSettingsUseCase(settings.SettingsDeps{
		Repo:               settingsRepo,
		AuditLogRepo:       auditLogRepo,
		TM:                 TM,
		Redis:              kv,
		CompRepo:           competitionRepo,
		ConfigUC:           competitionParamUC,
		PubSub:             pubsub,
		StopContext:        ctx,
		RuntimeInvalidator: runtimeInvalidator,
		Logger:             l,
	})
}
