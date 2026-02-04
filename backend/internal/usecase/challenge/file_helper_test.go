package challenge

import (
	"time"
)

func (h *ChallengeTestHelper) CreateFileUseCase() *FileUseCase {
	h.t.Helper()
	return NewFileUseCase(h.deps.fileRepo, h.deps.s3Provider, time.Hour)
}
