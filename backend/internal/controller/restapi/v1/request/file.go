package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

const errMsgFileTypeMustBe = `type must be "challenge" or "writeup"`

func MultipartFileType(t *openapi.PostAdminChallengesChallengeIDFilesMultipartBodyType) (domain.FileType, error) {
	if t == nil || *t == "" || *t == openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeChallenge {
		return domain.FileTypeChallenge, nil
	}

	if *t == openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeWriteup {
		return domain.FileTypeWriteup, nil
	}

	return "", apperr.NewValidationErrorf(errMsgFileTypeMustBe)
}

func ChallengeFileTypeFromParams(t *openapi.GetChallengesChallengeIDFilesParamsType) (domain.FileType, error) {
	if t == nil || *t == openapi.GetChallengesChallengeIDFilesParamsTypeChallenge {
		return domain.FileTypeChallenge, nil
	}

	if *t == openapi.GetChallengesChallengeIDFilesParamsTypeWriteup {
		return domain.FileTypeWriteup, nil
	}

	return "", apperr.NewValidationErrorf(errMsgFileTypeMustBe)
}

func ValidateChallengeUploadFilename(filename string) error {
	if !validator.ValidateChallengeUploadFilename(filename) {
		return apperr.NewValidationErrorf("file type not allowed")
	}

	return nil
}
