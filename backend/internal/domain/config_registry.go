package domain

import "sync"

// ConfigDef describes a registered dynamic configuration key, including its expected
// value type, default value, and organizational metadata.
type ConfigDef struct {
	Key          string
	DefaultValue string
	ValueType    CompetitionParamValueType
	Category     string
	Description  string
}

const (
	ConfigCategoryGeneral    = "general"
	ConfigCategoryTheme      = "theme"
	ConfigCategoryVisibility = "visibility"
	ConfigCategoryScoring    = "scoring"
	ConfigCategoryEmail      = "email"
	ConfigCategorySocial     = "social"
	ConfigCategoryLegal      = "legal"
	ConfigCategoryAdvanced   = "advanced"
)

var configRegistryMu sync.RWMutex

var configRegistry = map[string]ConfigDef{
	"ctf_name": {
		Key: "ctf_name", DefaultValue: "CTF Platform", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryGeneral, Description: "CTF competition name",
	},
	"ctf_description": {
		Key: "ctf_description", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryGeneral, Description: "CTF competition description (Markdown)",
	},
	"ctf_logo": {
		Key: "ctf_logo", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryGeneral, Description: "CTF logo URL",
	},
	"user_mode": {
		Key: "user_mode", DefaultValue: string(DefaultCompetitionMode), ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryGeneral, Description: "Participation mode: teams_only or solo_only",
	},
	"theme_color_primary": {
		Key: "theme_color_primary", DefaultValue: "#6366f1", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryTheme, Description: "Primary theme color (hex)",
	},
	"theme_color_secondary": {
		Key: "theme_color_secondary", DefaultValue: "#4f46e5", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryTheme, Description: "Secondary theme color (hex)",
	},
	"theme_header_html": {
		Key: "theme_header_html", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryTheme, Description: "Custom HTML for <head>",
	},
	"theme_footer_html": {
		Key: "theme_footer_html", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryTheme, Description: "Custom HTML for footer",
	},
	"theme_dark_mode": {
		Key: "theme_dark_mode", DefaultValue: "true", ValueType: CompetitionParamTypeBool,
		Category: ConfigCategoryTheme, Description: "Enable dark theme by default",
	},
	"challenge_visibility": {
		Key: "challenge_visibility", DefaultValue: "private", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryVisibility, Description: "Challenge visibility: public, private, admins",
	},
	"score_visibility": {
		Key: "score_visibility", DefaultValue: "public", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryVisibility, Description: "Scoreboard visibility: public, private, hidden, admins",
	},
	"account_visibility": {
		Key: "account_visibility", DefaultValue: "public", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryVisibility, Description: "User/team account visibility: public, private, admins",
	},
	"registration_visibility": {
		Key: "registration_visibility", DefaultValue: "public", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryVisibility, Description: "Registration visibility: public, private",
	},
	"view_after_ctf": {
		Key: "view_after_ctf", DefaultValue: "true", ValueType: CompetitionParamTypeBool,
		Category: ConfigCategoryVisibility, Description: "Show challenges after CTF ends",
	},
	"scoring_type": {
		Key: "scoring_type", DefaultValue: "dynamic", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryScoring, Description: "Scoring type: static or dynamic",
	},
	"decay_function": {
		Key: "decay_function", DefaultValue: "linear", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryScoring, Description: "Decay function: linear or logarithmic",
	},
	"first_blood_bonus": {
		Key: "first_blood_bonus", DefaultValue: "0", ValueType: CompetitionParamTypeInt,
		Category: ConfigCategoryScoring, Description: "Bonus points for first blood",
	},
	"mail_verification_subject": {
		Key: "mail_verification_subject", DefaultValue: "Verify your email - {ctf_name}", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryEmail, Description: "Verification email subject (supports {ctf_name})",
	},
	"mail_verification_body": {
		Key: "mail_verification_body", DefaultValue: "Follow the link to verify: {url}", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryEmail, Description: "Verification email body (supports {url}, {ctf_name})",
	},
	"mail_reset_subject": {
		Key: "mail_reset_subject", DefaultValue: "Password reset - {ctf_name}", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryEmail, Description: "Password reset email subject",
	},
	"mail_reset_body": {
		Key: "mail_reset_body", DefaultValue: "Follow the link to reset password: {url}", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryEmail, Description: "Password reset email body",
	},
	"social_github": {
		Key: "social_github", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategorySocial, Description: "CTF GitHub account URL",
	},
	"social_discord": {
		Key: "social_discord", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategorySocial, Description: "Discord server URL",
	},
	"social_twitter": {
		Key: "social_twitter", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategorySocial, Description: "Twitter/X account URL",
	},
	"social_website": {
		Key: "social_website", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategorySocial, Description: "Organizers website URL",
	},
	"tos_url": {
		Key: "tos_url", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryLegal, Description: "Terms of service URL (external link)",
	},
	"tos_text": {
		Key: "tos_text", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryLegal, Description: "Terms of service text (Markdown)",
	},
	"privacy_url": {
		Key: "privacy_url", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryLegal, Description: "Privacy policy URL",
	},
	"privacy_text": {
		Key: "privacy_text", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryLegal, Description: "Privacy policy text (Markdown)",
	},
	"challenge_prerequisite_anonymize": {
		Key: "challenge_prerequisite_anonymize", DefaultValue: "false", ValueType: CompetitionParamTypeBool,
		Category: ConfigCategoryVisibility, Description: "Show locked-by-requirements challenges as '???' instead of hiding",
	},
	"html_sanitization": {
		Key: "html_sanitization", DefaultValue: "true", ValueType: CompetitionParamTypeBool,
		Category: ConfigCategoryAdvanced, Description: "Sanitize HTML in challenge descriptions",
	},
	"csv_delimiter": {
		Key: "csv_delimiter", DefaultValue: ",", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryAdvanced, Description: "CSV export delimiter",
	},
	"max_content_length": {
		Key: "max_content_length", DefaultValue: "10485760", ValueType: CompetitionParamTypeInt,
		Category: ConfigCategoryAdvanced, Description: "Max content size in bytes (10MB)",
	},
	"registration_code": {
		Key: "registration_code", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryAdvanced, Description: "Registration access code (empty = disabled)",
	},
	"entity_whitelist": {
		Key: "entity_whitelist", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryAdvanced, Description: "Allowed email domains (comma-separated, empty = all)",
	},
	"password_min_length": {
		Key: "password_min_length", DefaultValue: "8", ValueType: CompetitionParamTypeInt,
		Category: ConfigCategoryAdvanced, Description: "Minimum password length",
	},
	"setup_complete": {
		Key: "setup_complete", DefaultValue: "false", ValueType: CompetitionParamTypeBool,
		Category: ConfigCategoryGeneral, Description: "Whether the initial setup wizard has been completed",
	},
	"email_verification_required": {
		Key: "email_verification_required", DefaultValue: "false", ValueType: CompetitionParamTypeBool,
		Category: ConfigCategoryGeneral, Description: "Require email verification on signup",
	},
	"timezone": {
		Key: "timezone", DefaultValue: "UTC", ValueType: CompetitionParamTypeString,
		Category: ConfigCategoryGeneral, Description: "Display timezone for competition times",
	},
}

// GetConfigDef performs a thread-safe lookup of a config definition by key in the global registry.
func GetConfigDef(key string) (ConfigDef, bool) {
	configRegistryMu.RLock()
	defer configRegistryMu.RUnlock()

	def, ok := configRegistry[key]

	return def, ok
}

// ConfigRegistryCount returns the number of registered config keys in a thread-safe manner.
func ConfigRegistryCount() int {
	configRegistryMu.RLock()
	defer configRegistryMu.RUnlock()

	return len(configRegistry)
}

// RangeConfigRegistry iterates over all registered config definitions in a thread-safe manner
// The callback f receives each key and its definition; returning false stops the iteration.
func RangeConfigRegistry(f func(key string, def ConfigDef) bool) {
	configRegistryMu.RLock()
	defer configRegistryMu.RUnlock()

	for k, d := range configRegistry {
		if !f(k, d) {
			return
		}
	}
}

// GetConfigDefault returns the default value for a registered config key, or false if the key is unknown.
func GetConfigDefault(key string) (string, bool) {
	def, ok := GetConfigDef(key)
	if !ok {
		return "", false
	}

	return def.DefaultValue, true
}
