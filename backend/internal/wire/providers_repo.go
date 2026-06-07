package wire

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

func ProvideUserRepo(pool *pgxpool.Pool) *persistent.UserRepo {
	return persistent.NewUserRepo(pool)
}

func ProvideChallengeRepo(pool *pgxpool.Pool) *persistent.ChallengeRepo {
	return persistent.NewChallengeRepo(pool)
}

func ProvideSolveRepo(pool *pgxpool.Pool) *persistent.SolveRepo {
	return persistent.NewSolveRepo(pool)
}

func ProvideTeamRepo(pool *pgxpool.Pool) *persistent.TeamRepo {
	return persistent.NewTeamRepo(pool)
}

func ProvideCompetitionRepo(pool *pgxpool.Pool) *persistent.CompetitionRepo {
	return persistent.NewCompetitionRepo(pool)
}

func ProvideHintRepo(pool *pgxpool.Pool) *persistent.HintRepo {
	return persistent.NewHintRepo(pool)
}

func ProvideTrackingRepo(pool *pgxpool.Pool) *persistent.TrackingRepo {
	return persistent.NewTrackingRepo(pool)
}

func ProvideAwardRepo(pool *pgxpool.Pool) *persistent.AwardRepo {
	return persistent.NewAwardRepo(pool)
}

func ProvideAuditLogRepo(pool *pgxpool.Pool) *persistent.AuditLogRepo {
	return persistent.NewAuditLogRepo(pool)
}

func ProvideStatisticsRepo(pool *pgxpool.Pool) *persistent.StatisticsRepo {
	return persistent.NewStatisticsRepo(pool)
}

func ProvideFileRepo(pool *pgxpool.Pool) *persistent.FileRepo {
	return persistent.NewFileRepo(pool)
}

func ProvideTransactionManager(pool *pgxpool.Pool) *persistent.TransactionManager {
	return persistent.NewTransactionManager(pool)
}

func ProvideBackupRepo(pool *pgxpool.Pool) *persistent.BackupRepo {
	return persistent.NewBackupRepo(pool)
}

func ProvideSubmissionRepo(pool *pgxpool.Pool) *persistent.SubmissionRepo {
	return persistent.NewSubmissionRepo(pool)
}

func ProvideTagRepo(pool *pgxpool.Pool) *persistent.TagRepo {
	return persistent.NewTagRepo(pool)
}

func ProvideTopicRepo(pool *pgxpool.Pool) *persistent.TopicRepo {
	return persistent.NewTopicRepo(pool)
}

func ProvideFieldRepo(pool *pgxpool.Pool) *persistent.FieldRepo {
	return persistent.NewFieldRepo(pool)
}

func ProvideFieldValueRepo(pool *pgxpool.Pool) *persistent.FieldValueRepo {
	return persistent.NewFieldValueRepo(pool)
}

func ProvideNotificationRepo(pool *pgxpool.Pool) *persistent.NotificationRepo {
	return persistent.NewNotificationRepo(pool)
}

func ProvidePageRepo(pool *pgxpool.Pool) *persistent.PageRepo {
	return persistent.NewPageRepo(pool)
}

func ProvideCommentRepo(pool *pgxpool.Pool) *persistent.CommentRepo {
	return persistent.NewCommentRepo(pool)
}

func ProvideRatingRepo(pool *pgxpool.Pool) *persistent.RatingRepo {
	return persistent.NewRatingRepo(pool)
}

func ProvideSettingsRepo(pool *pgxpool.Pool) *persistent.SettingsRepo {
	return persistent.NewSettingsRepo(pool)
}

func ProvideBanAppealRepo(pool *pgxpool.Pool) *persistent.BanAppealRepo {
	return persistent.NewBanAppealRepo(pool)
}

func ProvideCompetitionParamRepo(pool *pgxpool.Pool) *persistent.CompetitionParamRepo {
	return persistent.NewCompetitionParamRepo(pool)
}

func ProvideVerificationTokenRepo(pool *pgxpool.Pool) *persistent.VerificationTokenRepo {
	return persistent.NewVerificationTokenRepo(pool)
}

func ProvideOAuthRepo(pool *pgxpool.Pool, cryptoService crypto.Service) *persistent.OAuthRepo {
	return persistent.NewOAuthRepo(pool, cryptoService)
}

func ProvideBracketRepo(pool *pgxpool.Pool) *persistent.BracketRepo {
	return persistent.NewBracketRepo(pool)
}

func ProvideAPITokenRepo(pool *pgxpool.Pool) *persistent.APITokenRepo {
	return persistent.NewAPITokenRepo(pool)
}
