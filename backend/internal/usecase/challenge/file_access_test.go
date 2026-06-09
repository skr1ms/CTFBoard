package challenge

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestFileUseCase_GetByChallengeIDWithAccess_WriteupSolvedOnlyAllowed(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()
	files := []*domain.File{{ID: uuid.New(), Type: domain.FileTypeWriteup, ChallengeID: &challengeID}}

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(newTestChallenge(challengeID, "Web", "web", 100, "hash"), nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(&domain.ChallengeSolution{
		ChallengeID: challengeID,
		State:       domain.SolutionStateSolvedOnly,
	}, nil)
	d.solveRepo.On("GetByTeamAndChallenge", mock.Anything, teamID, challengeID).Return(&domain.Solve{TeamID: teamID, ChallengeID: challengeID}, nil)
	d.fileRepo.On("GetByChallengeID", mock.Anything, challengeID, domain.FileTypeWriteup).Return(files, nil)

	result, err := uc.GetByChallengeIDWithAccess(context.Background(), challengeID, domain.FileTypeWriteup, &teamID, false)

	assert.NoError(t, err)
	assert.Equal(t, files, result)
}

func TestFileUseCase_GetByChallengeIDWithAccess_WriteupHiddenDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	challengeID := uuid.New()
	teamID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(newTestChallenge(challengeID, "Web", "web", 100, "hash"), nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(&domain.ChallengeSolution{
		ChallengeID: challengeID,
		State:       domain.SolutionStateHidden,
	}, nil)

	result, err := uc.GetByChallengeIDWithAccess(context.Background(), challengeID, domain.FileTypeWriteup, &teamID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrWriteupAccessDenied)
}

func TestFileUseCase_GetByChallengeIDWithAccess_WriteupsDisabledDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()
	uc.deps.SettingsRepo = &solutionTestSettingsRepo{settings: &domain.Settings{WriteupEnabled: false}}

	challengeID := uuid.New()

	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(newTestChallenge(challengeID, "Web", "web", 100, "hash"), nil)
	d.challengeRepo.On("GetSolution", mock.Anything, challengeID).Return(&domain.ChallengeSolution{
		ChallengeID: challengeID,
		State:       domain.SolutionStateAfterEvent,
	}, nil)

	result, err := uc.GetByChallengeIDWithAccess(context.Background(), challengeID, domain.FileTypeWriteup, nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrWriteupsDisabled)
}

func TestFileUseCase_GetDownloadURLWithAccess_PageFilePublishedAllowed(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	pageID := uuid.New()
	fileID := uuid.New()
	uc.deps.PageRepo = fakePageReader{pages: map[uuid.UUID]*domain.Page{
		pageID: {ID: pageID, IsDraft: false},
	}}

	d.fileRepo.On("GetByID", mock.Anything, fileID).Return(&domain.File{
		ID:       fileID,
		Type:     domain.FileTypePage,
		PageID:   &pageID,
		Location: "tasks/0123456789abcdef/rules.pdf",
	}, nil).Once()

	url, err := uc.GetDownloadURLWithAccess(context.Background(), fileID, nil, false)

	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "http://localhost:8080/api/v1/files/download/tasks/0123456789abcdef/rules.pdf?token="))
}

func TestFileUseCase_GetDownloadURLWithAccess_InvalidStorageLocationDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	fileID := uuid.New()
	d.fileRepo.On("GetByID", mock.Anything, fileID).Return(&domain.File{
		ID:       fileID,
		Type:     domain.FileTypeChallenge,
		Location: "pages/rules.pdf",
	}, nil).Once()

	url, err := uc.GetDownloadURLWithAccess(context.Background(), fileID, nil, true)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrFileNotFound)
	assert.Empty(t, url)
}

func TestFileUseCase_GetDownloadURLWithAccess_PageFileDraftDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	pageID := uuid.New()
	fileID := uuid.New()
	uc.deps.PageRepo = fakePageReader{pages: map[uuid.UUID]*domain.Page{
		pageID: {ID: pageID, IsDraft: true},
	}}

	d.fileRepo.On("GetByID", mock.Anything, fileID).Return(&domain.File{
		ID:       fileID,
		Type:     domain.FileTypePage,
		PageID:   &pageID,
		Location: "tasks/0123456789abcdef/draft.pdf",
	}, nil).Once()

	url, err := uc.GetDownloadURLWithAccess(context.Background(), fileID, nil, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrFileNotFound)
	assert.Empty(t, url)
}

func TestFileUseCase_GetDownloadURLWithAccess_PageFileWithoutPageDenied(t *testing.T) {
	t.Parallel()
	d := newChallengeTestDeps(t)
	uc := d.createFileUseCase()

	fileID := uuid.New()
	d.fileRepo.On("GetByID", mock.Anything, fileID).Return(&domain.File{
		ID:       fileID,
		Type:     domain.FileTypePage,
		Location: "pages/orphan.pdf",
	}, nil).Once()

	url, err := uc.GetDownloadURLWithAccess(context.Background(), fileID, nil, false)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrFileNotFound)
	assert.Empty(t, url)
}

type fakePageReader struct {
	pages map[uuid.UUID]*domain.Page
}

func (r fakePageReader) GetByID(_ context.Context, ID uuid.UUID) (*domain.Page, error) {
	page, ok := r.pages[ID]
	if !ok {
		return nil, apperr.ErrPageNotFound
	}

	return page, nil
}
