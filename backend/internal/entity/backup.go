package entity

import (
	"time"

	"github.com/google/uuid"
)

const BackupVersion = "1.0"

type ConflictMode string

const (
	ConflictModeMerge     ConflictMode = "merge"
	ConflictModeOverwrite ConflictMode = "overwrite"
	ConflictModeSkip      ConflictMode = "skip"
)

type BackupData struct {
	Version     string            `json:"version"`
	ExportedAt  time.Time         `json:"exported_at"`
	Competition *Competition      `json:"competition"`
	Challenges  []ChallengeExport `json:"challenges"`
	Teams       []TeamExport      `json:"teams,omitempty"`
	Users       []UserExport      `json:"users,omitempty"`
	Awards      []Award           `json:"awards,omitempty"`
	Solves      []Solve           `json:"solves,omitempty"`
	Files       []File            `json:"files,omitempty"`
}

type ChallengeExport struct {
	Challenge
	Hints []Hint `json:"hints,omitempty"`
}

type TeamExport struct {
	Team
	MemberIDs []uuid.UUID `json:"member_ids,omitempty"`
}

type UserExport struct {
	ID       uuid.UUID  `json:"id"`
	Username string     `json:"username"`
	Email    string     `json:"email,omitempty"`
	Role     string     `json:"role"`
	TeamID   *uuid.UUID `json:"team_id,omitempty"`
}

type ExportOptions struct {
	IncludeUsers  bool
	IncludeTeams  bool
	IncludeSolves bool
	IncludeAwards bool
	IncludeFiles  bool
}

type ImportOptions struct {
	EraseExisting bool         `json:"erase_existing"`
	ConflictMode  ConflictMode `json:"conflict_mode"`
	ValidateFiles bool         `json:"validate_files"`
}

type ImportResult struct {
	Success      bool     `json:"success"`
	Errors       []string `json:"errors,omitempty"`
	SkippedCount int      `json:"skipped_count,omitempty"`
}
