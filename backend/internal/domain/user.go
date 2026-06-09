package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Role is a string-backed enum representing the privilege level of a user account
// Valid values: user, admin.
type Role string

const (
	// RoleUser is the default role for regular participants.
	RoleUser Role = "user"
	// RoleAdmin grants full administrative access to the platform.
	RoleAdmin Role = "admin"
)

// OAuthOnlyPasswordSentinel is stored in PasswordHash for accounts created via OAuth
// that have no local password set.
const OAuthOnlyPasswordSentinel = "oauth-only"

// User represents a registered user account on the platform.
type User struct {
	ID              uuid.UUID  `json:"id"`
	TeamID          *uuid.UUID `json:"team_id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Role            Role       `json:"role"`
	IsVerified      bool       `json:"is_verified"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	IsBanned        bool       `json:"is_banned"`
	BannedAt        *time.Time `json:"banned_at,omitempty"`
	BannedReason    *string    `json:"banned_reason,omitempty"`
	WasInBannedTeam bool       `json:"was_in_banned_team"`
	AvatarURL       *string    `json:"avatar_url,omitempty"`
}

// HasLocalPassword reports whether the account can authenticate with a local password.
func (u *User) HasLocalPassword() bool {
	return u != nil && u.PasswordHash != "" && u.PasswordHash != OAuthOnlyPasswordSentinel
}

// SortUsersByID sorts users in-place by their ID in lexicographic order.
// Used when acquiring row-level locks to ensure a consistent lock order and prevent deadlocks.
func SortUsersByID(users []*User) {
	slices.SortFunc(users, func(a, b *User) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})
}

// SortUUIDs sorts a UUID slice in-place in lexicographic order.
// Used when acquiring row-level locks by ID to ensure consistent lock ordering and prevent deadlocks.
func SortUUIDs(ids []uuid.UUID) {
	slices.SortFunc(ids, func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	})
}

// UniqueUUIDs returns a deduplicated copy of ids preserving first-seen order.
func UniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))

	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}
