package challenge

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
)

const downloadTokenPartCount = 4

func (uc *FileUseCase) VerifyDownloadTokenAndGetFile(ctx context.Context, path, token string, teamID *uuid.UUID) (*domain.File, error) {
	fileID, err := uc.VerifyDownloadToken(token, teamID)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - VerifyDownloadTokenAndGetFile - JWT.Validate: %w", err)
	}

	file, err := uc.deps.FileRepo.GetByLocation(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("FileUseCase - VerifyDownloadTokenAndGetFile - FileRepo.GetByLocation: %w", err)
	}

	if file.ID != fileID {
		return nil, apperr.ErrFileIDMismatch
	}

	return file, nil
}

// GenerateDownloadToken produces a base64-encoded HMAC-SHA256 signed download token
// encoding fileID, optional teamID, and expiry Unix timestamp.
func (uc *FileUseCase) GenerateDownloadToken(fileID uuid.UUID, teamID *uuid.UUID, expiry time.Time) string {
	expirySec := expiry.Unix()
	teamStr := ""

	if teamID != nil {
		teamStr = teamID.String()
	}

	message := fmt.Sprintf("%s:%s:%d", fileID.String(), teamStr, expirySec)
	signature := crypto.HMACSign([]byte(uc.deps.DownloadSecret), []byte(message))
	token := fmt.Sprintf("%s:%s:%d:%s", fileID.String(), teamStr, expirySec, base64.URLEncoding.EncodeToString(signature))

	return base64.URLEncoding.EncodeToString([]byte(token))
}

// VerifyDownloadToken decodes a base64url download token, parses its four colon-separated
// fields (fileID, teamID string, expiry unix timestamp, base64url HMAC signature), and
// validates them in order: format, expiry, team binding (the token's team field must match
// the caller's teamID - both empty string for anonymous tokens), and finally HMAC
// authenticity. Returns the fileID on success, or one of ErrFileInvalidToken /
// ErrFileTokenExpired on failure.
func (uc *FileUseCase) VerifyDownloadToken(token string, teamID *uuid.UUID) (uuid.UUID, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	parts := strings.Split(string(tokenBytes), ":")
	if len(parts) != downloadTokenPartCount {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	fileID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	tokenTeamStr := parts[1]

	expirySec, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	if time.Now().Unix() > expirySec {
		return uuid.Nil, apperr.ErrFileTokenExpired
	}

	callerTeamStr := ""

	if teamID != nil {
		callerTeamStr = teamID.String()
	}

	if tokenTeamStr != callerTeamStr {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	signature, err := base64.URLEncoding.DecodeString(parts[3])
	if err != nil {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	message := fmt.Sprintf("%s:%s:%d", fileID.String(), tokenTeamStr, expirySec)
	if !crypto.HMACVerify([]byte(uc.deps.DownloadSecret), []byte(message), signature) {
		return uuid.Nil, apperr.ErrFileInvalidToken
	}

	return fileID, nil
}

func escapeLocationPath(location string) string {
	if location == "" {
		return ""
	}

	parts := strings.Split(location, "/")

	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}

		out = append(out, url.PathEscape(p))
	}

	return strings.Join(out, "/")
}
