package challenge

import (
	"time"
)

func (h *ChallengeTestHelper) CreateFileUseCase() *FileUseCase {
	h.t.Helper()
	return NewFileUseCase(FileDeps{
		FileRepo:       h.deps.fileRepo,
		ChallengeRepo:  h.deps.challengeRepo,
		SolveRepo:      h.deps.solveRepo,
		Storage:        h.deps.s3Provider,
		Expiry:         time.Hour,
		DownloadSecret: "test-secret",
		BaseURL:        "http://localhost:8080",
	})
}
