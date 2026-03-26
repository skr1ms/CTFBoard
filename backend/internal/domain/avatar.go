package domain

import "time"

type AvatarEntityType string

const (
	AvatarEntityUser AvatarEntityType = "users"
	AvatarEntityTeam AvatarEntityType = "teams"
)

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

type ProcessedAvatar struct {
	FullImage      []byte
	ThumbnailImage []byte
	SHA256Hash     string
}

func GetDefaultAvatarConfig() AvatarConfig {
	presignedTTL := 24 * time.Hour

	return AvatarConfig{
		MaxFileSize:  5 << 20,
		FullSize:     256,
		ThumbSize:    64,
		WebPQuality:  80,
		MaxDimension: 4096,
		MinDimension: 64,
		PresignedTTL: presignedTTL,
		CacheTTL:     23 * time.Hour,
	}
}
