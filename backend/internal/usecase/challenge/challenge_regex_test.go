package challenge

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestChallengeUseCase_Create_Regex_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	encryptedFlag := "encrypted_regex"
	d.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		err := fn(ctx)
		if err != nil {
			return
		}
	})
	d.challengeRepo.On("Create", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.IsRegex && c.FlagRegex != nil && *c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil).Run(func(args mock.Arguments) {
		c, ok := args.Get(1).(*domain.Challenge)
		if ok && c != nil {
			c.ID = uuid.New()
		}
	})
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Maybe().Return(nil)

	challenge, err := uc.Create(context.Background(), challengeCreateParams("Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, true))

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotNil(t, challenge.FlagRegex)
	assert.Equal(t, encryptedFlag, *challenge.FlagRegex)
	assert.True(t, challenge.IsRegex)
}

func TestChallengeUseCase_Create_Regex_EncryptionError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	flag := "^flag{test}$"
	expectedError := errors.New("encryption failed")
	d.crypto.On("Encrypt", flag).Return("", expectedError)

	challenge, err := uc.Create(context.Background(), challengeCreateParams("Regex Challenge", "Desc", "Crypto", 100, 0, 0, 0, flag, true))

	assert.Error(t, err)
	assert.Nil(t, challenge)
	assert.Contains(t, err.Error(), "Encrypt")
	assert.Contains(t, err.Error(), expectedError.Error())
}

func TestChallengeUseCase_Update_Regex_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	existingChallenge := &domain.Challenge{
		ID:       challengeID,
		Title:    "Old Challenge",
		IsRegex:  false,
		FlagHash: "somehash",
	}

	flag := "^flag{new}$"
	encryptedFlag := "encrypted_new_regex"

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.crypto.On("Encrypt", flag).Return(encryptedFlag, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		err := fn(ctx)
		if err != nil {
			return
		}
	})
	d.challengeRepo.On("Update", mock.Anything, mock.MatchedBy(func(c *domain.Challenge) bool {
		return c.IsRegex && c.FlagRegex != nil && *c.FlagRegex == encryptedFlag && c.FlagHash == "REGEX_CHALLENGE"
	})).Return(nil)
	d.challengeRepo.On("SetTags", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	iv, mv, dc := 0, 0, 0
	ci, ma, pos := "", 0, 0
	ir, ic := true, false
	challenge, err := uc.Update(context.Background(), challengeID, challengeUpdateParams("Updated", "Desc", "Crypto", 100, &iv, &mv, &dc, flag, &ci, &ma, nil, &pos, "visible", &ir, &ic))

	assert.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.NotNil(t, challenge.FlagRegex)
	assert.Equal(t, encryptedFlag, *challenge.FlagRegex)
}

func TestChallengeUseCase_Update_Regex_EncryptionError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	existingChallenge := &domain.Challenge{
		ID:    challengeID,
		Title: "Old Challenge",
	}

	flag := "^flag{new}$"
	expectedError := errors.New("encryption failed")

	d.tm.On("Run", mock.Anything, mock.Anything).Return(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) })
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(existingChallenge, nil)
	d.crypto.On("Encrypt", flag).Return("", expectedError)

	iv, mv, dc := 0, 0, 0
	ci, ma, pos := "", 0, 0
	ir, ic := true, false
	challenge, err := uc.Update(context.Background(), challengeID, challengeUpdateParams("Updated", "Desc", "Crypto", 100, &iv, &mv, &dc, flag, &ci, &ma, nil, &pos, "visible", &ir, &ic))

	assert.Error(t, err)
	assert.Nil(t, challenge)
}

func TestChallengeUseCase_SubmitFlag_Regex_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test_regex_match}"
	regexPattern := "^flag\\{test_regex_match\\}$"
	encryptedRegex := "encrypted_regex_pattern"

	challenge := &domain.Challenge{
		ID:        challengeID,
		Title:     "Regex Challenge",
		IsRegex:   true,
		FlagRegex: &encryptedRegex,
		Points:    100,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.crypto.On("Decrypt", encryptedRegex).Return(regexPattern, nil)
	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		_ = fn(ctx)
	})
	d.challengeRepo.On("GetByIDForUpdate", mock.Anything, challengeID).Return(challenge, nil)
	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestChallengeUseCase_SubmitFlag_Regex_DecryptionError(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCaseWithCompAndCrypto()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "flag{test}"
	encryptedRegex := "encrypted_regex_pattern"

	challenge := &domain.Challenge{
		ID:        challengeID,
		IsRegex:   true,
		FlagRegex: &encryptedRegex,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)
	d.crypto.On("Decrypt", encryptedRegex).Return("", errors.New("decryption failed"))

	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.Error(t, err)
	assert.False(t, valid)
}

func TestChallengeUseCase_SubmitFlag_CaseInsensitive_Success(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc, _ := d.createChallengeUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	userID := uuid.New()
	flag := "FLAG{CaSe_InSeNsItIvE}"
	normalizedFlag := "flag{case_insensitive}"
	flagHash := challengeTestSha256Hash(normalizedFlag)

	challenge := &domain.Challenge{
		ID:                challengeID,
		IsCaseInsensitive: true,
		FlagHash:          flagHash,
		Points:            100,
	}
	team := newTestTeam(teamID)

	d.teamRepo.On("GetByID", mock.Anything, teamID).Return(team, nil)
	d.teamRepo.On("Lock", mock.Anything, teamID).Return(nil)

	d.compRepo.On("Get", mock.Anything).Return(newActiveCompetition(), nil)
	d.compRepo.On("GetForUpdate", mock.Anything).Return(newActiveCompetition(), nil)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.challengeRepo.On("GetRequirements", mock.Anything, challengeID).Return([]*repo.ChallengeRequirement{}, nil)

	d.tm.On("Run", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx, ok := args.Get(0).(context.Context)
		if !ok {
			return
		}

		fn, ok := args.Get(1).(func(context.Context) error)
		if !ok {
			return
		}

		_ = fn(ctx)
	})
	d.challengeRepo.On("GetByIDForUpdate", mock.Anything, challengeID).Return(challenge, nil)
	valid, err := uc.SubmitFlag(context.Background(), submitFlagParams(challengeID, flag, userID, &teamID))

	assert.NoError(t, err)
	assert.True(t, valid)
}
