package integration_test

import (
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

var (
	itestCryptoOnce sync.Once
	itestCrypto     crypto.Service
)

func itestCryptoService() crypto.Service {
	itestCryptoOnce.Do(func() {
		svc, err := crypto.NewCryptoService("1234567890123456789012345678901212345678901234567890123456789012")
		if err != nil {
			panic(err)
		}

		itestCrypto = svc
	})

	return itestCrypto
}

type TestFixture struct {
	Pool                  *pgxpool.Pool
	UserRepo              *persistent.UserRepo
	TeamRepo              *persistent.TeamRepo
	ChallengeRepo         *persistent.ChallengeRepo
	SolveRepo             *persistent.SolveRepo
	HintRepo              *persistent.HintRepo
	AwardRepo             *persistent.AwardRepo
	TM                    *persistent.TransactionManager
	CompetitionRepo       *persistent.CompetitionRepo
	VerificationTokenRepo *persistent.VerificationTokenRepo
	FileRepo              *persistent.FileRepo
	AuditLogRepo          *persistent.AuditLogRepo
	StatisticsRepo        *persistent.StatisticsRepo
	BackupRepo            *persistent.BackupRepo
	SettingsRepo          *persistent.SettingsRepo
	TagRepo               *persistent.TagRepo
	CommentRepo           *persistent.CommentRepo
	BracketRepo           *persistent.BracketRepo
	CompetitionParamRepo  *persistent.CompetitionParamRepo
	FieldRepo             *persistent.FieldRepo
	FieldValueRepo        *persistent.FieldValueRepo
	NotificationRepo      *persistent.NotificationRepo
	PageRepo              *persistent.PageRepo
	SubmissionRepo        *persistent.SubmissionRepo
	APITokenRepo          *persistent.APITokenRepo
	OAuthRepo             *persistent.OAuthRepo
	RatingRepo            *persistent.RatingRepo
	BanAppealRepo         *persistent.BanAppealRepo
}

func NewTestFixture(Pool *pgxpool.Pool) *TestFixture {
	tm := persistent.NewTransactionManager(Pool)

	return &TestFixture{
		Pool:                  Pool,
		UserRepo:              persistent.NewUserRepo(Pool),
		TeamRepo:              persistent.NewTeamRepo(Pool),
		ChallengeRepo:         persistent.NewChallengeRepo(Pool),
		SolveRepo:             persistent.NewSolveRepo(Pool),
		HintRepo:              persistent.NewHintRepo(Pool),
		AwardRepo:             persistent.NewAwardRepo(Pool),
		TM:                    tm,
		CompetitionRepo:       persistent.NewCompetitionRepo(Pool),
		VerificationTokenRepo: persistent.NewVerificationTokenRepo(Pool),
		FileRepo:              persistent.NewFileRepo(Pool),
		AuditLogRepo:          persistent.NewAuditLogRepo(Pool),
		StatisticsRepo:        persistent.NewStatisticsRepo(Pool),
		BackupRepo:            persistent.NewBackupRepo(Pool),
		SettingsRepo:          persistent.NewSettingsRepo(Pool),
		TagRepo:               persistent.NewTagRepo(Pool),
		CommentRepo:           persistent.NewCommentRepo(Pool),
		BracketRepo:           persistent.NewBracketRepo(Pool),
		CompetitionParamRepo:  persistent.NewCompetitionParamRepo(Pool),
		FieldRepo:             persistent.NewFieldRepo(Pool),
		FieldValueRepo:        persistent.NewFieldValueRepo(Pool),
		NotificationRepo:      persistent.NewNotificationRepo(Pool),
		PageRepo:              persistent.NewPageRepo(Pool),
		SubmissionRepo:        persistent.NewSubmissionRepo(Pool),
		APITokenRepo:          persistent.NewAPITokenRepo(Pool),
		OAuthRepo:             persistent.NewOAuthRepo(Pool, itestCryptoService()),
		RatingRepo:            persistent.NewRatingRepo(Pool),
		BanAppealRepo:         persistent.NewBanAppealRepo(Pool),
	}
}
