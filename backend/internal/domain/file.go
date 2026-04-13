package domain

import (
	"time"

	"github.com/google/uuid"
)

// FileType is a string-backed enum categorizing files by their association context.
type FileType string

const (
	// FileTypeChallenge marks a file as an attachment belonging to a challenge.
	FileTypeChallenge FileType = "challenge"
	// FileTypeWriteup marks a file as a writeup attachment submitted by a team.
	FileTypeWriteup FileType = "writeup"
	// FileTypePage marks a file as an attachment belonging to a static page.
	FileTypePage FileType = "page"
)

// File represents a stored file object with its S3 location and integrity metadata.
type File struct {
	ID          uuid.UUID  `json:"id"`
	Type        FileType   `json:"type"`
	ChallengeID *uuid.UUID `json:"challenge_id,omitempty"`
	PageID      *uuid.UUID `json:"page_id,omitempty"`
	Location    string     `json:"location"`
	Filename    string     `json:"filename"`
	Size        int64      `json:"size"`
	SHA256      string     `json:"sha256"`
	CreatedAt   time.Time  `json:"created_at"`
}
