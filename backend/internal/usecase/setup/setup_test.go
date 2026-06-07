package setup

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type setupFakeUserUC struct {
	usecase.UserUseCase

	adminUser *domain.User
	err       error
	calls     int
}

func (u *setupFakeUserUC) AdminCreate(context.Context, string, string, string, string) (*domain.User, error) {
	u.calls++

	return u.adminUser, u.err
}

type setupFakeCompetitionUC struct {
	usecase.CompetitionUseCase

	comp        *domain.Competition
	getErr      error
	updateErr   error
	updateComp  *domain.Competition
	actorID     uuid.UUID
	clientIP    string
	getCalls    int
	updateCalls int
}

func (c *setupFakeCompetitionUC) Get(context.Context) (*domain.Competition, error) {
	c.getCalls++

	return c.comp, c.getErr
}

func (c *setupFakeCompetitionUC) Update(_ context.Context, comp *domain.Competition, _ *usecase.CompetitionUpdateOptionals, actorID uuid.UUID, clientIP string) error {
	c.updateCalls++
	c.updateComp = comp
	c.actorID = actorID
	c.clientIP = clientIP

	return c.updateErr
}

type setupFakeCompetitionParamUC struct {
	usecase.CompetitionParamUseCase

	mu            sync.Mutex
	setupComplete bool
	setBatchErr   error
	setErr        error
	setBatchCalls int
	setCalls      int
	getBoolCalls  int
	batchParams   []*domain.CompetitionParam
	setParam      usecase.CompetitionParamSetParams
	actorID       uuid.UUID
	clientIP      string
}

func (c *setupFakeCompetitionParamUC) GetBool(_ context.Context, key string, defaultVal bool) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.getBoolCalls++

	if key == "setup_complete" {
		return c.setupComplete
	}

	return defaultVal
}

func (c *setupFakeCompetitionParamUC) SetBatch(_ context.Context, params []*domain.CompetitionParam, actorID uuid.UUID, clientIP string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setBatchCalls++
	c.batchParams = params
	c.actorID = actorID
	c.clientIP = clientIP

	return c.setBatchErr
}

func (c *setupFakeCompetitionParamUC) Set(_ context.Context, params usecase.CompetitionParamSetParams) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setCalls++

	c.setParam = params
	if params.Key == "setup_complete" && params.Value == "true" && c.setErr == nil {
		c.setupComplete = true
	}

	return c.setErr
}

type setupFakeSettingsUC struct {
	usecase.SettingsUseCase

	settings    *domain.Settings
	getErr      error
	updateErr   error
	updateValue *domain.Settings
	actorID     uuid.UUID
	clientIP    string
	getCalls    int
	updateCalls int
}

func (s *setupFakeSettingsUC) Get(context.Context) (*domain.Settings, error) {
	s.getCalls++

	return s.settings, s.getErr
}

func (s *setupFakeSettingsUC) Update(_ context.Context, settings *domain.Settings, actorID uuid.UUID, clientIP string) error {
	s.updateCalls++
	s.updateValue = settings
	s.actorID = actorID
	s.clientIP = clientIP

	return s.updateErr
}

type setupFakeJWTService struct {
	jwtkit.Service

	pair  *jwtkit.TokenPair
	err   error
	calls int
}

func (j *setupFakeJWTService) GenerateTokenPair(context.Context, uuid.UUID, string) (*jwtkit.TokenPair, error) {
	j.calls++

	return j.pair, j.err
}

type setupTestDeps struct {
	user      *setupFakeUserUC
	comp      *setupFakeCompetitionUC
	compParam *setupFakeCompetitionParamUC
	settings  *setupFakeSettingsUC
	jwt       *setupFakeJWTService
}

func newSetupTestDeps() *setupTestDeps {
	adminID := uuid.New()

	return &setupTestDeps{
		user: &setupFakeUserUC{
			adminUser: &domain.User{
				ID:       adminID,
				Username: "admin",
				Email:    "admin@example.com",
				Role:     domain.RoleAdmin,
			},
		},
		comp:      &setupFakeCompetitionUC{comp: &domain.Competition{Mode: domain.ModeTeamsOnly, MaxTeamSize: 3}},
		compParam: &setupFakeCompetitionParamUC{},
		settings:  &setupFakeSettingsUC{settings: &domain.Settings{AppName: "old"}},
		jwt: &setupFakeJWTService{pair: &jwtkit.TokenPair{
			AccessToken:      "access",
			RefreshToken:     "refresh",
			AccessExpiresAt:  100,
			RefreshExpiresAt: 200,
		}},
	}
}

func (d *setupTestDeps) useCase() *SetupUseCase {
	return NewSetupUseCase(SetupDeps{
		UserUC:      d.user,
		CompUC:      d.comp,
		CompParamUC: d.compParam,
		SettingsUC:  d.settings,
		JWTService:  d.jwt,
	})
}

func validSetupUseCaseRequest() *usecase.SetupRequest {
	start := time.Unix(1000, 0)
	end := time.Unix(2000, 0)
	freeze := time.Unix(1500, 0)

	return &usecase.SetupRequest{
		CTFName:                   "Astro CTF",
		CTFDescription:            "description",
		Mode:                      string(domain.ModeTeamsOnly),
		MaxTeamSize:               5,
		ChallengeVisibility:       "private",
		ScoreVisibility:           domain.ScoreboardVisibleAdminsOnly,
		AccountVisibility:         "public",
		RegistrationVisibility:    "public",
		EmailVerificationRequired: true,
		AdminUsername:             "admin",
		AdminEmail:                "admin@example.com",
		AdminPassword:             "Password12345",
		StartTime:                 &start,
		EndTime:                   &end,
		FreezeTime:                &freeze,
		Timezone:                  "UTC",
		ClientIP:                  "127.0.0.1",
	}
}
