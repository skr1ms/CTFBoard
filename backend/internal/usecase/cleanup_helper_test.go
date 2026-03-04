package usecase

import (
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
)

type CleanupTestHelper struct {
	t        *testing.T
	TeamRepo *mocks.MockTeamRepository
}

func NewCleanupTestHelper(t *testing.T) *CleanupTestHelper {
	t.Helper()
	return &CleanupTestHelper{
		t:        t,
		TeamRepo: mocks.NewMockTeamRepository(t),
	}
}

func (h *CleanupTestHelper) CreateUseCase() *CleanupUseCase {
	h.t.Helper()
	return NewCleanupUseCase(CleanupDeps{TeamRepo: h.TeamRepo})
}

func (h *CleanupTestHelper) DefaultOlderThan() time.Duration {
	return 24 * time.Hour
}
