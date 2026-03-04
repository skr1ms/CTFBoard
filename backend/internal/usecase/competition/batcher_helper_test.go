package competition

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/google/uuid"
)

func (h *CompetitionTestHelper) CreateSubmissionBatcher() *SubmissionBatcher {
	h.t.Helper()
	return NewSubmissionBatcher(h.deps.submissionRepo)
}

func (h *CompetitionTestHelper) NewBatcherSub() *entity.Submission {
	h.t.Helper()
	return h.NewSubmission(
		uuid.New(),
		nil,
		uuid.New(),
		"flag{batcher}",
		false,
	)
}
