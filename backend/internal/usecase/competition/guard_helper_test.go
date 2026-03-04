package competition

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
)

type GuardTestHelper struct {
	t    *testing.T
	Repo *mocks.MockCompetitionRepository
}

func NewGuardTestHelper(t *testing.T) *GuardTestHelper {
	t.Helper()
	return &GuardTestHelper{
		t:    t,
		Repo: mocks.NewMockCompetitionRepository(t),
	}
}

func (h *GuardTestHelper) CreateGuard() *Guard {
	h.t.Helper()
	return NewGuard(h.Repo)
}

func (h *GuardTestHelper) NewCompetition(mode string, allowTeamSwitch bool) *entity.Competition {
	h.t.Helper()
	return &entity.Competition{
		Name:            "CTF",
		Mode:            entity.CompetitionMode(mode),
		AllowTeamSwitch: allowTeamSwitch,
	}
}
