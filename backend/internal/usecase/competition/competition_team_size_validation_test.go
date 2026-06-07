package competition

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func TestCompetitionUseCase_Update_RejectsZeroTeamSizes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		minTeamSize int
		maxTeamSize int
		want        string
	}{
		{name: "min team size", minTeamSize: 0, maxTeamSize: 5, want: "min_team_size must be >= 1"},
		{name: "max team size", minTeamSize: 1, maxTeamSize: 0, want: "max_team_size must be >= 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := newCompetitionTestDeps(t)
			uc, redisClient := d.createCompetitionUseCase()

			startTime := time.Now().Add(2 * time.Hour)
			currentNotStarted := newTestCompetitionWithTimes("CTF", &startTime, nil)
			currentNotStarted.Mode = domain.ModeTeamsOnly
			currentNotStarted.MinTeamSize = 1
			currentNotStarted.MaxTeamSize = 5

			d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
			d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).Once()

			comp := &domain.Competition{ID: 1, Name: "CTF", Mode: domain.ModeTeamsOnly}
			optionals := &usecase.CompetitionUpdateOptionals{
				MinTeamSize: &tt.minTeamSize,
				MaxTeamSize: &tt.maxTeamSize,
			}

			err := uc.Update(context.Background(), comp, optionals, uuid.New(), "127.0.0.1")

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
			d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
			assert.NoError(t, redisClient.ExpectationsWereMet())
		})
	}
}
