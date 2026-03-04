package competition

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
)

func (h *CompetitionTestHelper) CreateSubmissionUseCase() *SubmissionUseCase {
	h.t.Helper()
	return NewSubmissionUseCase(SubmissionDeps{SubmissionRepo: h.deps.submissionRepo})
}

func (h *CompetitionTestHelper) NewSubmission(userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, flag string, isCorrect bool) *entity.Submission {
	h.t.Helper()
	return &entity.Submission{
		ID:            uuid.New(),
		UserID:        userID,
		TeamID:        teamID,
		ChallengeID:   challengeID,
		SubmittedFlag: flag,
		IsCorrect:     isCorrect,
	}
}
