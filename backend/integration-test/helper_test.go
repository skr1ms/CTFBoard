package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/stretchr/testify/require"
)

type TestFixture struct {
	Pool                  *pgxpool.Pool
	UserRepo              *persistent.UserRepo
	TeamRepo              *persistent.TeamRepo
	ChallengeRepo         *persistent.ChallengeRepo
	SolveRepo             *persistent.SolveRepo
	HintRepo              *persistent.HintRepo
	HintUnlockRepo        *persistent.HintUnlockRepo
	AwardRepo             *persistent.AwardRepo
	TxRepo                *persistent.TxRepo
	CompetitionRepo       *persistent.CompetitionRepo
	VerificationTokenRepo *persistent.VerificationTokenRepo
	FileRepo              *persistent.FileRepository
	AuditLogRepo          *persistent.AuditLogRepo
	StatisticsRepo        *persistent.StatisticsRepository
	BackupRepo            *persistent.BackupRepo
	AppSettingsRepo       *persistent.AppSettingsRepo
}

func NewTestFixture(Pool *pgxpool.Pool) *TestFixture {
	return &TestFixture{
		Pool:                  Pool,
		UserRepo:              persistent.NewUserRepo(Pool),
		TeamRepo:              persistent.NewTeamRepo(Pool),
		ChallengeRepo:         persistent.NewChallengeRepo(Pool),
		SolveRepo:             persistent.NewSolveRepo(Pool),
		HintRepo:              persistent.NewHintRepo(Pool),
		HintUnlockRepo:        persistent.NewHintUnlockRepo(Pool),
		AwardRepo:             persistent.NewAwardRepo(Pool),
		TxRepo:                persistent.NewTxRepo(Pool),
		CompetitionRepo:       persistent.NewCompetitionRepo(Pool),
		VerificationTokenRepo: persistent.NewVerificationTokenRepo(Pool),
		FileRepo:              persistent.NewFileRepository(Pool),
		AuditLogRepo:          persistent.NewAuditLogRepo(Pool),
		StatisticsRepo:        persistent.NewStatisticsRepository(Pool),
		BackupRepo:            persistent.NewBackupRepo(Pool),
		AppSettingsRepo:       persistent.NewAppSettingsRepo(Pool),
	}
}

func (f *TestFixture) CreateUser(t *testing.T, suffix string) *entity.User {
	t.Helper()
	ctx := context.Background()
	user := &entity.User{
		Username:     "user_" + suffix,
		Email:        "user_" + suffix + "@x.com",
		PasswordHash: "hash123",
	}
	err := f.UserRepo.Create(ctx, user)
	require.NoError(t, err)

	gotUser, err := f.UserRepo.GetByEmail(ctx, user.Email)
	require.NoError(t, err)
	user.ID = gotUser.ID
	return user
}

func (f *TestFixture) CreateTeam(t *testing.T, suffix string, captainID uuid.UUID) *entity.Team {
	t.Helper()
	ctx := context.Background()
	team := &entity.Team{
		Name:        "team_" + suffix,
		InviteToken: uuid.New(),
		CaptainID:   captainID,
	}
	err := f.TeamRepo.Create(ctx, team)
	require.NoError(t, err)

	gotTeam, err := f.TeamRepo.GetByName(ctx, team.Name)
	require.NoError(t, err)
	team.ID = gotTeam.ID
	return team
}

func (f *TestFixture) CreateUserWithTeam(t *testing.T, suffix string) (*entity.User, *entity.Team) {
	t.Helper()
	user := f.CreateUser(t, suffix)
	team := f.CreateTeam(t, suffix, user.ID)
	return user, team
}

func (f *TestFixture) CreateChallenge(t *testing.T, suffix string, points int) *entity.Challenge {
	t.Helper()
	ctx := context.Background()
	challenge := &entity.Challenge{
		Title:        "Challenge " + suffix,
		Description:  "Description " + suffix,
		Category:     "Web",
		Points:       points,
		FlagHash:     "hash_" + suffix,
		IsHidden:     false,
		InitialValue: points,
		MinValue:     points,
		Decay:        0,
	}
	err := f.ChallengeRepo.Create(ctx, challenge)
	require.NoError(t, err)
	return challenge
}

func (f *TestFixture) CreateDynamicChallenge(t *testing.T, suffix string, initial, minValue, decay int) *entity.Challenge {
	t.Helper()
	ctx := context.Background()
	challenge := &entity.Challenge{
		Title:        "Dynamic " + suffix,
		Description:  "Description " + suffix,
		Category:     "Pwn",
		Points:       initial,
		FlagHash:     "hash_" + suffix,
		IsHidden:     false,
		InitialValue: initial,
		MinValue:     minValue,
		Decay:        decay,
	}
	err := f.ChallengeRepo.Create(ctx, challenge)
	require.NoError(t, err)
	return challenge
}

func (f *TestFixture) CreateHint(t *testing.T, challengeID uuid.UUID, cost, order int) *entity.Hint {
	t.Helper()
	ctx := context.Background()
	hint := &entity.Hint{
		ChallengeID: challengeID,
		Content:     "Hint content",
		Cost:        cost,
		OrderIndex:  order,
	}
	err := f.HintRepo.Create(ctx, hint)
	require.NoError(t, err)
	return hint
}

func (f *TestFixture) CreateSolve(t *testing.T, userID, teamID, challengeID uuid.UUID) *entity.Solve {
	t.Helper()
	ctx := context.Background()
	solve := &entity.Solve{
		UserID:      userID,
		TeamID:      teamID,
		ChallengeID: challengeID,
	}
	err := f.SolveRepo.Create(ctx, solve)
	require.NoError(t, err)

	gotSolve, err := f.SolveRepo.GetByTeamAndChallenge(ctx, teamID, challengeID)
	require.NoError(t, err)
	solve.ID = gotSolve.ID
	solve.SolvedAt = gotSolve.SolvedAt
	return solve
}

func (f *TestFixture) CreateAwardTx(t *testing.T, tx pgx.Tx, teamID uuid.UUID, value int, desc string) *entity.Award {
	t.Helper()
	ctx := context.Background()
	award := &entity.Award{
		TeamID:      teamID,
		Value:       value,
		Description: desc,
	}
	err := f.TxRepo.CreateAwardTx(ctx, tx, award)
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

func (f *TestFixture) NewMinimalBackupData(t *testing.T) *entity.BackupData {
	t.Helper()
	comp, err := f.CompetitionRepo.Get(context.Background())
	require.NoError(t, err)
	return &entity.BackupData{
		Version:     entity.BackupVersion,
		ExportedAt:  time.Now().UTC(),
		Competition: comp,
		Challenges:  []entity.ChallengeExport{},
		Teams:       []entity.TeamExport{},
		Users:       []entity.UserExport{},
		Awards:      []entity.Award{},
		Solves:      []entity.Solve{},
		Files:       []entity.File{},
	}
}

func (f *TestFixture) GetDefaultAppSettings(t *testing.T) *entity.AppSettings {
	t.Helper()
	ctx := context.Background()
	settings, err := f.AppSettingsRepo.Get(ctx)
	require.NoError(t, err)
	return settings
}

func (f *TestFixture) ResetAppSettings(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	_, err := f.Pool.Exec(ctx, `
		UPDATE app_settings SET 
			app_name = 'CTFBoard',
			verify_emails = TRUE,
			frontend_url = 'http://localhost:3000',
			cors_origins = 'http://localhost:3000,http://localhost:5173',
			resend_enabled = FALSE,
			resend_from_email = 'noreply@ctfboard.local',
			resend_from_name = 'CTFBoard',
			verify_ttl_hours = 24,
			reset_ttl_hours = 1,
			submit_limit_per_user = 10,
			submit_limit_duration_min = 1,
			scoreboard_visible = 'public',
			registration_open = TRUE,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`)
	require.NoError(t, err)
}
