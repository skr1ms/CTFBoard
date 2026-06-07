package wire

import (
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

func ProvideChallengeUseCase(
	challengeRepo repo.ChallengeRepository,
	tagRepo repo.TagRepository,
	solveRepo repo.SolveRepository,
	submissionRepo repo.SubmissionRepository,
	TM repo.TransactionManager,
	compRepo repo.CompetitionRepository,
	settingsRepo repo.SettingsRepository,
	compUC *competition.CompetitionUseCase,
	compParamUC *competition.CompetitionParamUseCase,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	scoreboardCache *cache.ScoreboardCacheService,
	c *cachekit.Cache,
	broadcaster *iws.Broadcaster,
	auditLogRepo repo.AuditLogRepository,
	cryptoService crypto.Service,
	fileRepo repo.FileRepository,
	storageProvider storage.Provider,
	hintUC *challenge.HintUseCase,
	l logkit.Logger,
) *challenge.ChallengeUseCase {
	var statsInvalidator competition.StatisticsCacheInvalidator

	if c != nil {
		statsInvalidator = &competition.StatsCacheInvalidatorImpl{Cache: c}
	}

	return challenge.NewChallengeUseCase(challenge.ChallengeDeps{
		ChallengeRepo:   challengeRepo,
		TagRepo:         tagRepo,
		SolveRepo:       solveRepo,
		SubmissionRepo:  submissionRepo,
		TM:              TM,
		CompRepo:        compRepo,
		SettingsRepo:    settingsRepo,
		CompUC:          compUC,
		CompParamUC:     compParamUC,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		ScoreboardCache: scoreboardCache,
		StatsCache:      statsInvalidator,
		ListCache:       c,
		Broadcaster:     broadcaster,
		AuditLogRepo:    auditLogRepo,
		Crypto:          cryptoService,
		FileRepo:        fileRepo,
		Storage:         storageProvider,
		HintUC:          hintUC,
		SolveRecord:     competition.RecordSolveInTx,
		Logger:          l,
	})
}

func ProvideHintUseCase(
	hintRepo repo.HintRepository,
	awardRepo repo.AwardRepository,
	TM repo.TransactionManager,
	solveRepo repo.SolveRepository,
	compRepo repo.CompetitionRepository,
	competitionUC *competition.CompetitionUseCase,
	teamRepo repo.TeamRepository,
	userRepo repo.UserRepository,
	challengeRepo repo.ChallengeRepository,
	scoreboardCache *cache.ScoreboardCacheService,
	l logkit.Logger,
) *challenge.HintUseCase {
	return challenge.NewHintUseCase(challenge.HintDeps{
		HintRepo:        hintRepo,
		AwardRepo:       awardRepo,
		TM:              TM,
		SolveRepo:       solveRepo,
		CompRepo:        compRepo,
		CompUC:          competitionUC,
		TeamRepo:        teamRepo,
		UserRepo:        userRepo,
		ChallengeRepo:   challengeRepo,
		ScoreboardCache: scoreboardCache,
		Logger:          l,
	})
}

func ProvideTagUseCase(tagRepo repo.TagRepository, challengeRepo repo.ChallengeRepository) *challenge.TagUseCase {
	return challenge.NewTagUseCase(challenge.TagDeps{TagRepo: tagRepo, ChallengeRepo: challengeRepo})
}

func ProvideTopicUseCase(topicRepo repo.TopicRepository, challengeRepo repo.ChallengeRepository, tm repo.TransactionManager) *challenge.TopicUseCase {
	return challenge.NewTopicUseCase(challenge.TopicDeps{TopicRepo: topicRepo, ChallengeRepo: challengeRepo, TM: tm})
}

func ProvideCommentUseCase(commentRepo repo.CommentRepository, challengeRepo repo.ChallengeRepository, userRepo repo.UserRepository, teamRepo repo.TeamRepository, tm repo.TransactionManager) *challenge.CommentUseCase {
	return challenge.NewCommentUseCase(challenge.CommentDeps{
		CommentRepo:   commentRepo,
		ChallengeRepo: challengeRepo,
		UserRepo:      userRepo,
		TeamRepo:      teamRepo,
		TM:            tm,
	})
}

func ProvideRatingUseCase(challengeRepo repo.ChallengeRepository, solveRepo repo.SolveRepository, ratingRepo repo.RatingRepository, userRepo repo.UserRepository, teamRepo repo.TeamRepository, tm repo.TransactionManager) *challenge.RatingUseCase {
	return challenge.NewRatingUseCase(challenge.RatingDeps{
		ChallengeRepo: challengeRepo,
		SolveRepo:     solveRepo,
		RatingRepo:    ratingRepo,
		UserRepo:      userRepo,
		TeamRepo:      teamRepo,
		TM:            tm,
	})
}

func ProvideFileUseCase(
	fileRepo repo.FileRepository,
	challengeRepo repo.ChallengeRepository,
	pageRepo challenge.PageReader,
	solveRepo repo.SolveRepository,
	compRepo repo.CompetitionRepository,
	settingsRepo repo.SettingsRepository,
	storageProvider storage.Provider,
	cfg *config.Config,
) *challenge.FileUseCase {
	return challenge.NewFileUseCase(challenge.FileDeps{
		FileRepo:       fileRepo,
		ChallengeRepo:  challengeRepo,
		PageRepo:       pageRepo,
		SolveRepo:      solveRepo,
		CompRepo:       compRepo,
		SettingsRepo:   settingsRepo,
		Storage:        storageProvider,
		Expiry:         cfg.PresignedExpiry,
		DownloadSecret: cfg.DownloadSecret,
		BaseURL:        cfg.BaseURL,
	})
}
