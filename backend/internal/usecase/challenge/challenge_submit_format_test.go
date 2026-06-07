package challenge

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestChallengeUseCase_SubmitFlag_InvalidFormat(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	challenge := &domain.Challenge{
		ID:              challengeID,
		FlagHash:        "hash",
		IsRegex:         false,
		FlagFormatRegex: nil,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	regex := "^GoCTF\\{.+\\}$"
	comp := newActiveCompetition()
	comp.FlagRegex = &regex
	d.compRepo.On("Get", mock.Anything).Return(comp, nil)

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, "InvalidFlag", uuid.New(), &teamID))

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrInvalidFlagFormat)
	assert.False(t, valid)
}
