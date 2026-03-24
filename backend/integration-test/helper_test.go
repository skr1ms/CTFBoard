package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent"
)

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
		OAuthRepo:             persistent.NewOAuthRepo(Pool),
	}
}

func (f *TestFixture) CreateUser(t *testing.T, suffix string) *domain.User {
	t.Helper()
	// Username and email must fit varchar(50): "user_" = 5, "@x.com" = 6, so unique at most 39.
	unique := suffix + "_" + uuid.NewString()[:8]
	if len(unique) > 39 {
		unique = unique[:39]
	}
	ctx := context.Background()
	user := &domain.User{
		Username:     "user_" + unique,
		Email:        "user_" + unique + "@x.com",
		PasswordHash: "hash123",
	}
	err := f.UserRepo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.ID = gotUser.ID
	return user
}

func (f *TestFixture) CreateTeam(t *testing.T, suffix string, captainID uuid.UUID) *domain.Team {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	team := &domain.Team{
		Name:        "team_" + unique,
		InviteToken: uuid.New(),
		CaptainID:   captainID,
	}
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.TeamRepo.Create(ctx, team)
	})
	require.NoError(t, err)
	return team
}

func (f *TestFixture) CreateUserWithTeam(t *testing.T, suffix string) (*domain.User, *domain.Team) {
	t.Helper()
	user := f.CreateUser(t, suffix)
	team := f.CreateTeam(t, suffix, user.ID)
	err := f.UserRepo.UpdateTeamID(context.Background(), user.ID, &team.ID)
	require.NoError(t, err)
	user.TeamID = &team.ID
	return user, team
}

func (f *TestFixture) CreateChallenge(t *testing.T, suffix string, points int) *domain.Challenge {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	challenge := &domain.Challenge{
		Title:        "Challenge " + unique,
		Description:  "Description " + unique,
		Category:     "Web",
		Points:       points,
		FlagHash:     "hash_" + unique,
		State:        domain.ChallengeStateVisible,
		InitialValue: points,
		MinValue:     points,
		Decay:        0,
	}
	challenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.ChallengeRepo.Create(ctx, challenge)
	})
	require.NoError(t, err)
	return challenge
}

func (f *TestFixture) CreateDynamicChallenge(t *testing.T, suffix string, initial, minValue, decay int) *domain.Challenge {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	challenge := &domain.Challenge{
		Title:        "Dynamic " + unique,
		Description:  "Description " + unique,
		Category:     "Pwn",
		Points:       initial,
		FlagHash:     "hash_" + unique,
		State:        domain.ChallengeStateVisible,
		InitialValue: initial,
		MinValue:     minValue,
		Decay:        decay,
	}
	challenge.ID = uuid.New()
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.ChallengeRepo.Create(ctx, challenge)
	})
	require.NoError(t, err)
	return challenge
}

func (f *TestFixture) CreateHint(t *testing.T, challengeID uuid.UUID, cost, order int) *domain.Hint {
	t.Helper()
	ctx := context.Background()
	hint := &domain.Hint{
		ChallengeID: challengeID,
		Content:     "Hint content",
		Cost:        cost,
		OrderIndex:  order,
	}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)
	return hint
}

func (f *TestFixture) CreateSolve(t *testing.T, userID, teamID, challengeID uuid.UUID) *domain.Solve {
	t.Helper()
	ctx := context.Background()
	challenge, err := f.ChallengeRepo.GetByID(ctx, challengeID)
	require.NoError(t, err)
	solve := &domain.Solve{
		UserID:        userID,
		TeamID:        teamID,
		ChallengeID:   challengeID,
		PointsAtSolve: challenge.Points,
	}
	err = f.TM.Run(ctx, func(ctx context.Context) error {
		return f.SolveRepo.Create(ctx, solve)
	})
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, teamID, challengeID)
	require.NoError(t, err)
	solve.ID = gotSolve.ID
	solve.SolvedAt = gotSolve.SolvedAt
	return solve
}

func (f *TestFixture) CreateAwardTx(t *testing.T, ctx context.Context, teamID uuid.UUID, value int, desc string) *domain.Award {
	t.Helper()
	award := &domain.Award{
		TeamID:      teamID,
		Value:       value,
		Description: desc,
	}
	err := f.AwardRepo.Create(ctx, award)
	require.NoError(t, err)
	return award
}

// CreateAward creates an award inside a transaction (production path). Use in tests.
func (f *TestFixture) CreateAward(t *testing.T, teamID uuid.UUID, value int, desc string, createdBy *uuid.UUID) *domain.Award {
	t.Helper()
	ctx := context.Background()
	award := &domain.Award{
		TeamID:      teamID,
		Value:       value,
		Description: desc,
		CreatedBy:   createdBy,
	}
	err := f.TM.Run(ctx, func(ctx context.Context) error {
		return f.AwardRepo.Create(ctx, award)
	})
	require.NoError(t, err)
	return award
}

func (f *TestFixture) AddUserToTeam(t *testing.T, userID, teamID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, "UPDATE users SET team_id = $1 WHERE id = $2", teamID, userID)
	require.NoError(t, err)
}

func (f *TestFixture) BackdateTeamDeletedAt(t *testing.T, teamID uuid.UUID, deletedAt time.Time) {
	t.Helper()
	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, "UPDATE teams SET deleted_at = $1 WHERE id = $2", deletedAt, teamID)
	require.NoError(t, err)
}

func (f *TestFixture) NewMinimalBackupData(t *testing.T) *domain.BackupData {
	t.Helper()
	comp, err := f.CompetitionRepo.Get(context.Background())
	require.NoError(t, err)
	return &domain.BackupData{
		Version:     domain.BackupVersion,
		ExportedAt:  time.Now().UTC(),
		Competition: comp,
		Challenges:  []domain.ChallengeExport{},
		Teams:       []domain.TeamExport{},
		Users:       []domain.UserExport{},
		Awards:      []domain.Award{},
		Solves:      []domain.Solve{},
		Files:       []domain.File{},
	}
}

func (f *TestFixture) GetDefaultAppSettings(t *testing.T) *domain.Settings {
	t.Helper()
	ctx := context.Background()
	settings, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)
	return settings
}

func (f *TestFixture) CreateTag(t *testing.T, suffix string) *domain.Tag {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	tag := &domain.Tag{
		Name:  "tag_" + unique,
		Color: "#ff0000",
	}
	err := f.TagRepo.Create(ctx, tag)
	require.NoError(t, err)
	gotTag, err := f.TagRepo.GetByName(ctx, tag.Name)
	require.NoError(t, err)
	tag.ID = gotTag.ID
	return tag
}

func (f *TestFixture) CreateComment(t *testing.T, userID, challengeID uuid.UUID, content string) *domain.Comment {
	t.Helper()
	ctx := context.Background()
	comment := &domain.Comment{
		UserID:      userID,
		ChallengeID: challengeID,
		Content:     content,
	}
	err := f.CommentRepo.Create(ctx, comment)
	require.NoError(t, err)
	return comment
}

func (f *TestFixture) CreateBracket(t *testing.T, suffix string) *domain.Bracket {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	bracket := &domain.Bracket{
		Name:        "bracket_" + unique,
		Description: "desc",
		IsDefault:   false,
	}
	err := f.BracketRepo.Create(ctx, bracket)
	require.NoError(t, err)
	got, err := f.BracketRepo.GetByName(ctx, bracket.Name)
	require.NoError(t, err)
	bracket.ID = got.ID
	return bracket
}

func (f *TestFixture) CreatePage(t *testing.T, suffix string, isDraft bool) *domain.Page {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	page := &domain.Page{
		Title:      "Page " + unique,
		Slug:       "page-" + unique,
		Content:    "content",
		IsDraft:    isDraft,
		OrderIndex: 0,
	}
	err := f.PageRepo.Create(ctx, page)
	require.NoError(t, err)
	got, err := f.PageRepo.GetBySlug(ctx, page.Slug)
	require.NoError(t, err)
	page.ID = got.ID
	return page
}

func (f *TestFixture) CreateNotification(t *testing.T, suffix string) *domain.Notification {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	notif := &domain.Notification{
		Title:    "Notif " + unique,
		Content:  "content",
		Type:     domain.NotificationInfo,
		IsPinned: false,
		IsGlobal: true,
	}
	err := f.NotificationRepo.Create(ctx, notif)
	require.NoError(t, err)
	return notif
}

func (f *TestFixture) CreateVerificationToken(t *testing.T, userID uuid.UUID, tokenType domain.TokenType) *domain.VerificationToken {
	t.Helper()
	ctx := context.Background()
	tok := &domain.VerificationToken{
		UserID:    userID,
		Token:     "token_" + uuid.New().String(),
		Type:      tokenType,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	err := f.VerificationTokenRepo.Create(ctx, tok)
	require.NoError(t, err)
	return tok
}

func (f *TestFixture) CreateField(t *testing.T, suffix string, entityType domain.EntityType) *domain.Field {
	t.Helper()
	unique := suffix + "_" + uuid.NewString()[:8]
	ctx := context.Background()
	field := &domain.Field{
		Name:       "field_" + unique,
		FieldType:  domain.FieldTypeText,
		EntityType: entityType,
		Required:   false,
		OrderIndex: 0,
	}
	err := f.FieldRepo.Create(ctx, field)
	require.NoError(t, err)
	return field
}

func (f *TestFixture) ResetAppSettings(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, `
		UPDATE app_settings SET 
			app_name = 'AstroCTFb',
			verify_emails = TRUE,
			frontend_url = 'http://localhost:3000',
			cors_origins = 'http://localhost:3000,http://localhost:5173',
			resend_enabled = FALSE,
			resend_from_email = 'noreply@astroctfb.local',
			resend_from_name = 'AstroCTFb',
			verify_ttl_hours = 24,
			reset_ttl_hours = 1,
			submit_limit_per_user = 10,
			submit_limit_duration_min = 1,
			scoreboard_visible = 'public',
			registration_open = TRUE,
			writeup_enabled = TRUE,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	require.NoError(t, err)
}
