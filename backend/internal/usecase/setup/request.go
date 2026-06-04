package setup

import "time"

// SetupRequest carries all data required to complete the first-run setup wizard.
type SetupRequest struct {
	// General step
	CTFName        string
	CTFDescription string

	// Mode step
	Mode        string // "teams_only" | "solo_only" | "flexible"
	MaxTeamSize int

	// Settings step
	ChallengeVisibility       string // "public" | "private" | "admins"
	ScoreVisibility           string // "public" | "private" | "hidden" | "admins"
	AccountVisibility         string // "public" | "private" | "admins"
	RegistrationVisibility    string // "public" | "private"
	EmailVerificationRequired bool

	// Administration step
	AdminUsername string
	AdminEmail    string
	AdminPassword string

	// Date & Time step
	StartTime  *time.Time
	EndTime    *time.Time
	FreezeTime *time.Time
	Timezone   string

	// Internal - set by controller from request context.
	ClientIP string
}
