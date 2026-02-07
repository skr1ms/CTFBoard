package competition

import (
	"context"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
)

type SubmissionUseCase struct {
	submissionRepo repo.SubmissionRepository
}

func NewSubmissionUseCase(submissionRepo repo.SubmissionRepository) *SubmissionUseCase {
	return &SubmissionUseCase{
		submissionRepo: submissionRepo,
	}
}

func (uc *SubmissionUseCase) LogSubmission(ctx context.Context, sub *entity.Submission) error {
	if err := uc.submissionRepo.Create(ctx, sub); err != nil {
		return usecaseutil.Wrap(err, "SubmissionUseCase - LogSubmission")
	}
	return nil
}

func (uc *SubmissionUseCase) GetByChallenge(ctx context.Context, challengeID uuid.UUID, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	submissions, err := uc.submissionRepo.GetByChallenge(ctx, challengeID, perPage, offset)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetByChallenge")
	}

	total, err := uc.submissionRepo.CountByChallenge(ctx, challengeID)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetByChallenge count")
	}

	return submissions, total, nil
}

func (uc *SubmissionUseCase) GetByUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	submissions, err := uc.submissionRepo.GetByUser(ctx, userID, perPage, offset)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetByUser")
	}

	total, err := uc.submissionRepo.CountByUser(ctx, userID)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetByUser count")
	}

	return submissions, total, nil
}

func (uc *SubmissionUseCase) GetByTeam(ctx context.Context, teamID uuid.UUID, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	submissions, err := uc.submissionRepo.GetByTeam(ctx, teamID, perPage, offset)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetByTeam")
	}

	total, err := uc.submissionRepo.CountByTeam(ctx, teamID)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetByTeam count")
	}

	return submissions, total, nil
}

func (uc *SubmissionUseCase) GetAll(ctx context.Context, page, perPage int) ([]*entity.SubmissionWithDetails, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	submissions, err := uc.submissionRepo.GetAll(ctx, perPage, offset)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetAll")
	}

	total, err := uc.submissionRepo.CountAll(ctx)
	if err != nil {
		return nil, 0, usecaseutil.Wrap(err, "SubmissionUseCase - GetAll count")
	}

	return submissions, total, nil
}

func (uc *SubmissionUseCase) GetStats(ctx context.Context, challengeID uuid.UUID) (*entity.SubmissionStats, error) {
	stats, err := uc.submissionRepo.GetStats(ctx, challengeID)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "SubmissionUseCase - GetStats")
	}
	return stats, nil
}
