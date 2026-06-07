package backup

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestBackupFileZIPPath_ByFileType(t *testing.T) {
	t.Parallel()

	challengeID := uuid.New()
	pageID := uuid.New()

	tests := []struct {
		name string
		file domain.File
		want string
	}{
		{
			name: "challenge",
			file: domain.File{Type: domain.FileTypeChallenge, ChallengeID: &challengeID, Filename: "task.txt"},
			want: "files/challenge-" + challengeID.String() + "/task.txt",
		},
		{
			name: "writeup",
			file: domain.File{Type: domain.FileTypeWriteup, ChallengeID: &challengeID, Filename: "writeup.md"},
			want: "files/writeup-" + challengeID.String() + "/writeup.md",
		},
		{
			name: "page",
			file: domain.File{Type: domain.FileTypePage, PageID: &pageID, Filename: "rules.pdf"},
			want: "files/page-" + pageID.String() + "/rules.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := backupFileZIPPath(tt.file)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBackupFileZIPPath_FallsBackToSafeStorageFilename(t *testing.T) {
	t.Parallel()

	challengeID := uuid.New()
	file := domain.File{
		Type:        domain.FileTypeChallenge,
		ChallengeID: &challengeID,
		Location:    "tasks/0123456789abcdef/safe.txt",
		Filename:    "..",
	}

	got, err := backupFileZIPPath(file)

	require.NoError(t, err)
	assert.Equal(t, "files/challenge-"+challengeID.String()+"/safe.txt", got)
}

func TestBackupUseCase_PrepareImportFiles_FiltersUnsafeOrMissingPayloads(t *testing.T) {
	t.Parallel()

	uc := NewBackupUseCase(BackupDeps{})
	challengeID := uuid.New()
	pageID := uuid.New()
	validContent := []byte("valid file")
	mismatchContent := []byte("different file")

	validFile := domain.File{
		ID:          uuid.New(),
		Type:        domain.FileTypeChallenge,
		ChallengeID: &challengeID,
		Location:    "tasks/0123456789abcdef/valid.txt",
		Filename:    "valid.txt",
		SHA256:      testSHA256(validContent),
	}
	missingPayload := domain.File{
		ID:       uuid.New(),
		Type:     domain.FileTypePage,
		PageID:   &pageID,
		Location: "tasks/0123456789abcdef/page.txt",
		Filename: "page.txt",
		SHA256:   testSHA256([]byte("missing")),
	}
	invalidLocation := domain.File{
		ID:          uuid.New(),
		Type:        domain.FileTypeWriteup,
		ChallengeID: &challengeID,
		Location:    "../escape.txt",
		Filename:    "escape.txt",
		SHA256:      testSHA256([]byte("escape")),
	}
	hashMismatch := domain.File{
		ID:          uuid.New(),
		Type:        domain.FileTypeChallenge,
		ChallengeID: &challengeID,
		Location:    "tasks/0123456789abcdef/mismatch.txt",
		Filename:    "mismatch.txt",
		SHA256:      testSHA256([]byte("expected file")),
	}

	validPath, err := backupFileZIPPath(validFile)
	require.NoError(t, err)
	mismatchPath, err := backupFileZIPPath(hashMismatch)
	require.NoError(t, err)

	zr := newTestZipReader(t, map[string][]byte{
		validPath:    validContent,
		mismatchPath: mismatchContent,
	})

	prepared, warnings := uc.prepareImportFiles(zr, []domain.File{validFile, missingPayload, invalidLocation, hashMismatch}, true)

	require.Len(t, prepared, 1)
	assert.Equal(t, validFile.ID, prepared[0].ID)
	assert.Len(t, warnings, 3)
	assert.Contains(t, warnings[0], "payload not found")
	assert.Contains(t, warnings[1], "invalid storage location")
	assert.Contains(t, warnings[2], "sha256 mismatch")
}

func TestBackupUseCase_PrepareImportFiles_SkipsSymlinkPayload(t *testing.T) {
	t.Parallel()

	uc := NewBackupUseCase(BackupDeps{})
	challengeID := uuid.New()
	file := domain.File{
		ID:          uuid.New(),
		Type:        domain.FileTypeChallenge,
		ChallengeID: &challengeID,
		Location:    "tasks/0123456789abcdef/link.txt",
		Filename:    "link.txt",
		SHA256:      testSHA256([]byte("ignored")),
	}
	zipPath, err := backupFileZIPPath(file)
	require.NoError(t, err)

	zr := newTestZipReaderWithModes(t, map[string]testZipEntry{
		zipPath: {
			content: []byte("target.txt"),
			mode:    os.ModeSymlink | 0o644,
		},
	})

	prepared, warnings := uc.prepareImportFiles(zr, []domain.File{file}, true)

	assert.Empty(t, prepared)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "symlink payload is not allowed")
}

func newTestZipReader(t *testing.T, entries map[string][]byte) *zip.Reader {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for name, content := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	return zr
}

type testZipEntry struct {
	content []byte
	mode    os.FileMode
}

func newTestZipReaderWithModes(t *testing.T, entries map[string]testZipEntry) *zip.Reader {
	t.Helper()

	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	for name, entry := range entries {
		header := &zip.FileHeader{Name: name}
		header.SetMode(entry.mode)

		w, err := zw.CreateHeader(header)
		require.NoError(t, err)
		_, err = w.Write(entry.content)
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)

	return zr
}

func testSHA256(data []byte) string {
	sum := sha256.Sum256(data)

	return hex.EncodeToString(sum[:])
}
