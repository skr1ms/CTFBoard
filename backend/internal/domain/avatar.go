package domain

import (
	"strings"
	"time"
)

// ThumbPathFromFull derives the thumbnail storage path from a full-size avatar path.
// Returns an empty string if fullPath does not end with "_full.webp".
func ThumbPathFromFull(fullPath string) string {
	if len(fullPath) < 10 || !strings.HasSuffix(fullPath, "_full.webp") {
		return ""
	}

	return fullPath[:len(fullPath)-10] + "_thumb.webp"
}

// AvatarEntityType is a string-backed enum identifying which entity type owns an avatar.
type AvatarEntityType string

const (
	// AvatarEntityUser identifies a user avatar stored under the "users" namespace.
	AvatarEntityUser AvatarEntityType = "users"
	// AvatarEntityTeam identifies a team avatar stored under the "teams" namespace.
	AvatarEntityTeam AvatarEntityType = "teams"
)

const (
	avatarMaxFileSize  int64         = 5 << 20
	avatarFullSize                   = 256
	avatarThumbSize                  = 64
	avatarWebPQuality                = 80
	avatarMaxDimension               = 2048
	avatarMinDimension               = 64
	avatarPresignedTTL time.Duration = 24 * time.Hour
	avatarCacheTTL     time.Duration = 23 * time.Hour
)

// AvatarConfig holds processing parameters applied when an avatar image is uploaded.
type AvatarConfig struct {
	MaxFileSize  int64
	FullSize     int
	ThumbSize    int
	WebPQuality  int
	MaxDimension int
	MinDimension int
	PresignedTTL time.Duration
	CacheTTL     time.Duration
}

// ProcessedAvatar holds the result of an avatar upload: the full-size and thumbnail images
// encoded as WebP, along with the SHA-256 hash of the original.
type ProcessedAvatar struct {
	FullImage      []byte
	ThumbnailImage []byte
	SHA256Hash     string
}

// GetDefaultAvatarConfig returns the default avatar processing configuration
// 5 MB max upload, 256 px full size, 64 px thumbnail, WebP quality 80.
func GetDefaultAvatarConfig() AvatarConfig {
	return AvatarConfig{
		MaxFileSize:  avatarMaxFileSize,
		FullSize:     avatarFullSize,
		ThumbSize:    avatarThumbSize,
		WebPQuality:  avatarWebPQuality,
		MaxDimension: avatarMaxDimension,
		MinDimension: avatarMinDimension,
		PresignedTTL: avatarPresignedTTL,
		CacheTTL:     avatarCacheTTL,
	}
}
