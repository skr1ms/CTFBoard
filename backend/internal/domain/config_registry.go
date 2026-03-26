package domain

import "sync"

type ConfigDef struct {
	Key          string
	DefaultValue string
	ValueType    CompetitionParamValueType
	Category     string
	Description  string
}

var configRegistryMu sync.RWMutex

var configRegistry = map[string]ConfigDef{
	"ctf_name": {
		Key: "ctf_name", DefaultValue: "AstroCTFb", ValueType: CompetitionParamTypeString,
		Category: "general", Description: "CTF competition name",
	},
	"ctf_description": {
		Key: "ctf_description", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "general", Description: "CTF competition description (Markdown)",
	},
	"ctf_logo": {
		Key: "ctf_logo", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "general", Description: "CTF logo URL",
	},
	"user_mode": {
		Key: "user_mode", DefaultValue: "teams", ValueType: CompetitionParamTypeString,
		Category: "general", Description: "Participation mode: teams or users",
	},
	"theme_color_primary": {
		Key: "theme_color_primary", DefaultValue: "#6366f1", ValueType: CompetitionParamTypeString,
		Category: "theme", Description: "Primary theme color (hex)",
	},
	"theme_color_secondary": {
		Key: "theme_color_secondary", DefaultValue: "#4f46e5", ValueType: CompetitionParamTypeString,
		Category: "theme", Description: "Secondary theme color (hex)",
	},
	"theme_header_html": {
		Key: "theme_header_html", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "theme", Description: "Custom HTML for <head>",
	},
	"theme_footer_html": {
		Key: "theme_footer_html", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "theme", Description: "Custom HTML for footer",
	},
	"theme_dark_mode": {
		Key: "theme_dark_mode", DefaultValue: "true", ValueType: CompetitionParamTypeBool,
		Category: "theme", Description: "Enable dark theme by default",
	},
	"challenge_visibility": {
		Key: "challenge_visibility", DefaultValue: "private", ValueType: CompetitionParamTypeString,
		Category: "visibility", Description: "Challenge visibility: public, private, admins",
	},
	"score_visibility": {
		Key: "score_visibility", DefaultValue: "public", ValueType: CompetitionParamTypeString,
		Category: "visibility", Description: "Scoreboard visibility: public, hidden, admins",
	},
	"registration_visibility": {
		Key: "registration_visibility", DefaultValue: "public", ValueType: CompetitionParamTypeString,
		Category: "visibility", Description: "Registration visibility: public, private",
	},
	"view_after_ctf": {
		Key: "view_after_ctf", DefaultValue: "true", ValueType: CompetitionParamTypeBool,
		Category: "visibility", Description: "Show challenges after CTF ends",
	},
	"scoring_type": {
		Key: "scoring_type", DefaultValue: "dynamic", ValueType: CompetitionParamTypeString,
		Category: "scoring", Description: "Scoring type: static or dynamic",
	},
	"decay_function": {
		Key: "decay_function", DefaultValue: "linear", ValueType: CompetitionParamTypeString,
		Category: "scoring", Description: "Decay function: linear or logarithmic",
	},
	"first_blood_bonus": {
		Key: "first_blood_bonus", DefaultValue: "0", ValueType: CompetitionParamTypeInt,
		Category: "scoring", Description: "Bonus points for first blood",
	},
	"mail_verification_subject": {
		Key: "mail_verification_subject", DefaultValue: "Verify your email — {ctf_name}", ValueType: CompetitionParamTypeString,
		Category: "email", Description: "Verification email subject (supports {ctf_name})",
	},
	"mail_verification_body": {
		Key: "mail_verification_body", DefaultValue: "Follow the link to verify: {url}", ValueType: CompetitionParamTypeString,
		Category: "email", Description: "Verification email body (supports {url}, {ctf_name})",
	},
	"mail_reset_subject": {
		Key: "mail_reset_subject", DefaultValue: "Password reset — {ctf_name}", ValueType: CompetitionParamTypeString,
		Category: "email", Description: "Password reset email subject",
	},
	"mail_reset_body": {
		Key: "mail_reset_body", DefaultValue: "Follow the link to reset password: {url}", ValueType: CompetitionParamTypeString,
		Category: "email", Description: "Password reset email body",
	},
	"social_github": {
		Key: "social_github", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "social", Description: "CTF GitHub account URL",
	},
	"social_discord": {
		Key: "social_discord", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "social", Description: "Discord server URL",
	},
	"social_twitter": {
		Key: "social_twitter", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "social", Description: "Twitter/X account URL",
	},
	"social_website": {
		Key: "social_website", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "social", Description: "Organizers website URL",
	},
	"tos_url": {
		Key: "tos_url", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "legal", Description: "Terms of service URL (external link)",
	},
	"tos_text": {
		Key: "tos_text", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "legal", Description: "Terms of service text (Markdown)",
	},
	"privacy_url": {
		Key: "privacy_url", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "legal", Description: "Privacy policy URL",
	},
	"privacy_text": {
		Key: "privacy_text", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "legal", Description: "Privacy policy text (Markdown)",
	},
	"html_sanitization": {
		Key: "html_sanitization", DefaultValue: "true", ValueType: CompetitionParamTypeBool,
		Category: "advanced", Description: "Sanitize HTML in challenge descriptions",
	},
	"csv_delimiter": {
		Key: "csv_delimiter", DefaultValue: ",", ValueType: CompetitionParamTypeString,
		Category: "advanced", Description: "CSV export delimiter",
	},
	"max_content_length": {
		Key: "max_content_length", DefaultValue: "10485760", ValueType: CompetitionParamTypeInt,
		Category: "advanced", Description: "Max content size in bytes (10MB)",
	},
	"registration_code": {
		Key: "registration_code", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "advanced", Description: "Registration access code (empty = disabled)",
	},
	"entity_whitelist": {
		Key: "entity_whitelist", DefaultValue: "", ValueType: CompetitionParamTypeString,
		Category: "advanced", Description: "Allowed email domains (comma-separated, empty = all)",
	},
	"password_min_length": {
		Key: "password_min_length", DefaultValue: "8", ValueType: CompetitionParamTypeInt,
		Category: "advanced", Description: "Minimum password length",
	},
}

func GetConfigDef(key string) (ConfigDef, bool) {
	configRegistryMu.RLock()
	defer configRegistryMu.RUnlock()

	def, ok := configRegistry[key]

	return def, ok
}

func ConfigRegistryCount() int {
	configRegistryMu.RLock()
	defer configRegistryMu.RUnlock()

	return len(configRegistry)
}

func RangeConfigRegistry(f func(key string, def ConfigDef) bool) {
	configRegistryMu.RLock()
	defer configRegistryMu.RUnlock()

	for k, d := range configRegistry {
		if !f(k, d) {
			return
		}
	}
}

func GetConfigDefault(key string) (string, bool) {
	def, ok := GetConfigDef(key)
	if !ok {
		return "", false
	}

	return def.DefaultValue, true
}
