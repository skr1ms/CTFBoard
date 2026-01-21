package entity

import (
	"time"

	"github.com/google/uuid"
)

type FileType string

const (
	FileTypeChallenge FileType = "challenge"
	FileTypeWriteup   FileType = "writeup"
)

type File struct {
	Id          uuid.UUID `json:"id"`
	Type        FileType  `json:"type"`
	ChallengeId uuid.UUID `json:"challenge_id"`
	Location    string    `json:"location"`
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	SHA256      string    `json:"sha256"`
	CreatedAt   time.Time `json:"created_at"`
}
