package domain

import (
	"time"

	"github.com/google/uuid"
)

// BackupVersion is the current version string embedded in every exported backup file.
const BackupVersion = "1.0"

// ConflictMode controls how records that already exist in the database are handled during import.
type ConflictMode string

const (
	// ConflictModeOverwrite replaces existing records with imported data.
	ConflictModeOverwrite ConflictMode = "overwrite"
	// ConflictModeSkip retains existing records and ignores conflicting imported entries.
	ConflictModeSkip ConflictMode = "skip"
)

type ChallengeRequirementPair struct {
	ChallengeID         uuid.UUID `json:"challenge_id"`
	RequiredChallengeID uuid.UUID `json:"required_challenge_id"`
}

type SolutionBackup struct {
	ID          uuid.UUID `json:"id"`
	ChallengeID uuid.UUID `json:"challenge_id"`
	Content     string    `json:"content"`
}

// BackupData is the root structure of the JSON backup format, versioned and timestamped.
type BackupData struct {
	Version               string                     `json:"version"`
	ExportedAt            time.Time                  `json:"exported_at"`
	Competition           *Competition               `json:"competition"`
	Tags                  []Tag                      `json:"tags,omitempty"`
	Challenges            []ChallengeExport          `json:"challenges"`
	Brackets              []Bracket                  `json:"brackets,omitempty"`
	ChallengeRequirements []ChallengeRequirementPair `json:"challenge_requirements,omitempty"`
	Solutions             []SolutionBackup           `json:"solutions,omitempty"`
	Teams                 []TeamExport               `json:"teams,omitempty"`
	Users                 []UserExport               `json:"users,omitempty"`
	Awards                []Award                    `json:"awards,omitempty"`
	Solves                []Solve                    `json:"solves,omitempty"`
	HintUnlocks           []HintUnlock               `json:"hint_unlocks,omitempty"`
	Files                 []File                     `json:"files,omitempty"`
	Comments              []Comment                  `json:"comments,omitempty"`
	Fields                []Field                    `json:"fields,omitempty"`
	FieldValues           []FieldValue               `json:"field_values,omitempty"`
	Ratings               []Rating                   `json:"ratings,omitempty"`
}

// ChallengeExport extends Challenge with admin-only fields and associated data
// needed to fully reconstruct a challenge from a backup.
type ChallengeExport struct {
	Challenge

	State          string      `json:"state"`
	FlagHash       string      `json:"flag_hash"`
	FlagRegex      string      `json:"flag_regex"`
	ConnectionInfo string      `json:"connection_info"`
	MaxAttempts    int         `json:"max_attempts"`
	Position       int         `json:"position"`
	Hints          []Hint      `json:"hints,omitempty"`
	TagIDs         []uuid.UUID `json:"tag_ids,omitempty"`
}

// TeamExport extends Team with its member list for backup and import purposes.
type TeamExport struct {
	Team

	InviteToken          uuid.UUID   `json:"invite_token"`
	InviteTokenExpiresAt *time.Time  `json:"invite_token_expires_at,omitempty"`
	MemberIDs            []uuid.UUID `json:"member_ids,omitempty"`
}

// UserExport is a portable representation of a user account for backup and import.
type UserExport struct {
	ID           uuid.UUID  `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email,omitempty"`
	Role         string     `json:"role"`
	TeamID       *uuid.UUID `json:"team_id,omitempty"`
	IsVerified   bool       `json:"is_verified"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty"`
	IsBanned     bool       `json:"is_banned"`
	BannedAt     *time.Time `json:"banned_at,omitempty"`
	BannedReason *string    `json:"banned_reason,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ExportOptions controls which optional data categories are included in a backup export.
type ExportOptions struct {
	IncludeUsers       bool
	IncludeTeams       bool
	IncludeSolves      bool
	IncludeHintUnlocks bool
	IncludeAwards      bool
	IncludeFiles       bool
}

// ImportOptions controls backup import behavior, including whether to erase existing data
// first and how to resolve conflicts between imported and existing records.
type ImportOptions struct {
	EraseExisting      bool         `json:"erase_existing"`
	ConflictMode       ConflictMode `json:"conflict_mode"`
	ValidateFiles      bool         `json:"validate_files"`
	PreserveAdminRoles bool         `json:"preserve_admin_roles"`
	AdminUserID        *uuid.UUID   `json:"-"`
	AdminIP            string       `json:"-"`
}

// ImportResult summarizes the outcome of a backup import operation.
type ImportResult struct {
	Success      bool     `json:"success"`
	Errors       []string `json:"errors,omitempty"`
	SkippedCount int      `json:"skipped_count,omitempty"`
}

// AdminResetOptions specifies which data categories to wipe during an admin reset operation.
type AdminResetOptions struct {
	Pages         bool
	Notifications bool
	Challenges    bool
	Accounts      bool
	Submissions   bool
}
