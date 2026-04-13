package competition_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"
)

func BenchmarkBatcherEnqueue(b *testing.B) {
	repo := compMock.NewMockSubmissionRepository(b)
	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(nil).Maybe()
	// fallback path when channel is full: batcher calls Create directly
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()

	batcher := competition.NewSubmissionBatcher(repo)
	defer batcher.Stop()

	teamID := uuid.New()
	sub := &domain.Submission{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		TeamID:        &teamID,
		ChallengeID:   uuid.New(),
		SubmittedFlag: "flag{bench}",
	}

	b.ReportAllocs()

	for b.Loop() {
		batcher.Enqueue(sub)
	}
}

func BenchmarkBatcherEnqueue_Parallel(b *testing.B) {
	repo := compMock.NewMockSubmissionRepository(b)
	repo.EXPECT().CreateBatch(mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Maybe()

	batcher := competition.NewSubmissionBatcher(repo)

	defer func() {
		// allow background goroutine to flush before Stop
		time.Sleep(200 * time.Millisecond)
		batcher.Stop()
	}()

	teamID := uuid.New()
	sub := &domain.Submission{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		TeamID:        &teamID,
		ChallengeID:   uuid.New(),
		SubmittedFlag: "flag{bench_parallel}",
	}

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			batcher.Enqueue(sub)
		}
	})
}
