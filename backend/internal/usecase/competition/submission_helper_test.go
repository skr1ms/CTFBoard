package competition

import (
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *CompetitionTestHelper) CreateSubmissionUseCase() *SubmissionUseCase {
	h.t.Helper()
	return NewSubmissionUseCase(h.deps.submissionRepo)
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
