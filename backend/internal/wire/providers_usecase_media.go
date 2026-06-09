package wire

import (
	"context"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	wscontroller "github.com/TakuyaYagam1/AstroCTFb/internal/controller/websocket/v1"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/avatar"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cleanup"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/storageadmin"
)

func ProvideAvatarUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	storageProvider storage.Provider,
	keyValueStore cachekit.KeyValueStore,
	TM repo.TransactionManager,
	auditLogRepo repo.AuditLogRepository,
	l logkit.Logger,
) *avatar.AvatarUseCase {
	return avatar.NewAvatarUseCase(avatar.AvatarDeps{
		UserRepo:     userRepo,
		TeamRepo:     teamRepo,
		Storage:      storageProvider,
		Cache:        keyValueStore,
		TM:           TM,
		AuditLogRepo: auditLogRepo,
		Config:       domain.GetDefaultAvatarConfig(),
		Logger:       l,
	})
}

func ProvideStorageAdminUseCase(storageProvider storage.Provider, auditLogRepo repo.AuditLogRepository) *storageadmin.UseCase {
	return storageadmin.NewUseCase(storageadmin.Deps{Storage: storageProvider, AuditLog: auditLogRepo})
}

func ProvideCleanupUseCase(
	userRepo repo.UserRepository,
	teamRepo repo.TeamRepository,
	fileRepo repo.FileRepository,
	trackingRepo repo.TrackingRepository,
	storageProvider storage.Provider,
) *cleanup.CleanupUseCase {
	return cleanup.NewCleanupUseCase(cleanup.CleanupDeps{
		UserRepo:     userRepo,
		TeamRepo:     teamRepo,
		FileRepo:     fileRepo,
		TrackingRepo: trackingRepo,
		Storage:      storageProvider,
	})
}

func ProvideBackupUseCase(
	ctx context.Context,
	competitionRepo repo.CompetitionRepository,
	challengeRepo repo.ChallengeRepository,
	pageRepo backup.PageRepository,
	tagRepo repo.TagRepository,
	topicRepo repo.TopicRepository,
	hintRepo repo.HintRepository,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	awardRepo repo.AwardRepository,
	solveRepo repo.SolveRepository,
	submissionRepo repo.SubmissionRepository,
	fileRepo repo.FileRepository,
	backupRepo repo.BackupRepository,
	settingsRepo repo.SettingsRepository,
	auditLogRepo repo.AuditLogRepository,
	bracketRepo repo.BracketRepository,
	commentRepo repo.CommentRepository,
	fieldRepo repo.FieldRepository,
	fieldValueRepo repo.FieldValueRepository,
	ratingRepo repo.RatingRepository,
	storageProvider storage.Provider,
	TM repo.TransactionManager,
	l logkit.Logger,
) *backup.BackupUseCase {
	return backup.NewBackupUseCase(backup.BackupDeps{
		StopContext:     ctx,
		CompetitionRepo: competitionRepo,
		ChallengeRepo:   challengeRepo,
		PageRepo:        pageRepo,
		TagRepo:         tagRepo,
		TopicRepo:       topicRepo,
		HintRepo:        hintRepo,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		AwardRepo:       awardRepo,
		SolveRepo:       solveRepo,
		SubmissionRepo:  submissionRepo,
		FileRepo:        fileRepo,
		BackupRepo:      backupRepo,
		SettingsRepo:    settingsRepo,
		AuditLogRepo:    auditLogRepo,
		BracketRepo:     bracketRepo,
		CommentRepo:     commentRepo,
		FieldRepo:       fieldRepo,
		FieldValueRepo:  fieldValueRepo,
		RatingRepo:      ratingRepo,
		Storage:         storageProvider,
		TM:              TM,
		Logger:          l,
	})
}

func ProvideWsController(wsHub *wskit.Hub, l logkit.Logger, cfg *config.Config) *wscontroller.Controller {
	return wscontroller.NewController(wsHub, l, cfg.CORSOrigins)
}
