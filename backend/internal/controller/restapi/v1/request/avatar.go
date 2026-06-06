package request

import (
	"regexp"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

const avatarPathPartCount = 3

var validAvatarPath = regexp.MustCompile(`^(users|teams)/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/[a-f0-9]+_(full|thumb)\.webp$`)

type AvatarPath struct {
	Path string
	Hash string
}

func ValidateAvatarPath(path string) bool {
	return validAvatarPath.MatchString(path)
}

func ParseAvatarPath(path string) (AvatarPath, error) {
	if !ValidateAvatarPath(path) {
		return AvatarPath{}, apperr.NewValidationErrorf("invalid avatar path")
	}

	parts := strings.Split(path, "/")
	if len(parts) != avatarPathPartCount {
		return AvatarPath{}, apperr.NewValidationErrorf("invalid avatar path")
	}

	hash, _, ok := strings.Cut(parts[2], "_")
	if !ok || hash == "" {
		return AvatarPath{}, apperr.NewValidationErrorf("invalid avatar path")
	}

	return AvatarPath{Path: path, Hash: hash}, nil
}

func (p AvatarPath) ETag() string {
	return `"` + p.Hash + `"`
}
