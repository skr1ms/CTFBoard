package request

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func TestSetupRequestToParamsNilRequest(t *testing.T) {
	t.Parallel()

	got := SetupRequestToParams(nil, "192.0.2.10")

	assert.Equal(t, "192.0.2.10", got.ClientIP)
	assert.Empty(t, got.CTFName)
	assert.False(t, got.EmailVerificationRequired)
}

func TestSetupRequestToParamsMapsPointersAndEnums(t *testing.T) {
	t.Parallel()

	start := time.Unix(100, 0)
	end := time.Unix(200, 0)
	freeze := time.Unix(150, 0)
	req := &openapi.SetupRequest{
		AccountVisibility:         openapi.SetupRequestAccountVisibilityPrivate,
		AdminEmail:                "admin@example.com",
		AdminPassword:             "Password12345",
		AdminUsername:             "admin",
		ChallengeVisibility:       openapi.SetupRequestChallengeVisibilityHidden,
		CtfDescription:            new("desc"),
		CtfName:                   "Astro CTF",
		EmailVerificationRequired: new(true),
		EndTime:                   &end,
		FreezeTime:                &freeze,
		MaxTeamSize:               new(7),
		Mode:                      openapi.SetupRequestModeTeamsOnly,
		RegistrationVisibility:    openapi.SetupRequestRegistrationVisibilityPublic,
		ScoreVisibility:           openapi.SetupRequestScoreVisibilityAdminsOnly,
		StartTime:                 &start,
		Timezone:                  new("UTC"),
	}

	got := SetupRequestToParams(req, "127.0.0.1")

	assert.Equal(t, "Astro CTF", got.CTFName)
	assert.Equal(t, "desc", got.CTFDescription)
	assert.Equal(t, string(openapi.SetupRequestModeTeamsOnly), got.Mode)
	assert.Equal(t, 7, got.MaxTeamSize)
	assert.Equal(t, string(openapi.SetupRequestChallengeVisibilityHidden), got.ChallengeVisibility)
	assert.Equal(t, string(openapi.SetupRequestScoreVisibilityAdminsOnly), got.ScoreVisibility)
	assert.Equal(t, string(openapi.SetupRequestAccountVisibilityPrivate), got.AccountVisibility)
	assert.Equal(t, string(openapi.SetupRequestRegistrationVisibilityPublic), got.RegistrationVisibility)
	assert.True(t, got.EmailVerificationRequired)
	assert.Equal(t, "admin", got.AdminUsername)
	assert.Equal(t, "admin@example.com", got.AdminEmail)
	assert.Equal(t, "Password12345", got.AdminPassword)
	assert.Equal(t, &start, got.StartTime)
	assert.Equal(t, &end, got.EndTime)
	assert.Equal(t, &freeze, got.FreezeTime)
	assert.Equal(t, "UTC", got.Timezone)
	assert.Equal(t, "127.0.0.1", got.ClientIP)
}

func TestValidateStoragePathAndPrefix(t *testing.T) {
	t.Parallel()

	assert.NoError(t, ValidateStoragePath("uploads/file.txt"))
	assert.NoError(t, ValidateStoragePrefix(""))
	assert.NoError(t, ValidateStoragePrefix("uploads/"))

	for _, path := range []string{"../secret", "uploads/../secret", "/absolute"} {
		t.Run("path "+path, func(t *testing.T) {
			t.Parallel()

			err := ValidateStoragePath(path)

			require.Error(t, err)

			var validationErr *apperr.ValidationError
			assert.ErrorAs(t, err, &validationErr)
		})
	}

	for _, prefix := range []string{"../secret", "uploads/../secret", "/absolute"} {
		t.Run("prefix "+prefix, func(t *testing.T) {
			t.Parallel()

			err := ValidateStoragePrefix(prefix)

			require.Error(t, err)

			var validationErr *apperr.ValidationError
			assert.ErrorAs(t, err, &validationErr)
		})
	}
}

func TestFileTypeMappings(t *testing.T) {
	t.Parallel()

	uploadChallenge := openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeChallenge
	uploadWriteup := openapi.PostAdminChallengesChallengeIDFilesMultipartBodyTypeWriteup
	uploadInvalid := openapi.PostAdminChallengesChallengeIDFilesMultipartBodyType("invalid")

	got, err := MultipartFileType(nil)
	require.NoError(t, err)
	assert.Equal(t, domain.FileTypeChallenge, got)

	got, err = MultipartFileType(&uploadChallenge)
	require.NoError(t, err)
	assert.Equal(t, domain.FileTypeChallenge, got)

	got, err = MultipartFileType(&uploadWriteup)
	require.NoError(t, err)
	assert.Equal(t, domain.FileTypeWriteup, got)

	got, err = MultipartFileType(&uploadInvalid)
	require.Error(t, err)
	assert.Empty(t, got)

	paramsChallenge := openapi.GetChallengesChallengeIDFilesParamsTypeChallenge
	paramsWriteup := openapi.GetChallengesChallengeIDFilesParamsTypeWriteup
	paramsInvalid := openapi.GetChallengesChallengeIDFilesParamsType("invalid")

	got, err = ChallengeFileTypeFromParams(nil)
	require.NoError(t, err)
	assert.Equal(t, domain.FileTypeChallenge, got)

	got, err = ChallengeFileTypeFromParams(&paramsChallenge)
	require.NoError(t, err)
	assert.Equal(t, domain.FileTypeChallenge, got)

	got, err = ChallengeFileTypeFromParams(&paramsWriteup)
	require.NoError(t, err)
	assert.Equal(t, domain.FileTypeWriteup, got)

	got, err = ChallengeFileTypeFromParams(&paramsInvalid)
	require.Error(t, err)
	assert.Empty(t, got)
}

func TestValidateUploadFilenameRejectsDangerousExtensions(t *testing.T) {
	t.Parallel()

	assert.NoError(t, ValidateUploadFilename("archive.tar.gz"))

	err := ValidateUploadFilename("shell.php.txt")
	require.Error(t, err)

	var validationErr *apperr.ValidationError
	assert.ErrorAs(t, err, &validationErr)
}

func TestAdminUsersSearchParams(t *testing.T) {
	t.Parallel()

	field, err := AdminUsersFieldFromParams(openapi.GetAdminUsersParams{})
	require.NoError(t, err)
	assert.Equal(t, UserSearchFieldUsername, field)

	ipField := openapi.IP
	field, err = AdminUsersFieldFromParams(openapi.GetAdminUsersParams{Field: &ipField})
	require.NoError(t, err)
	assert.Equal(t, UserSearchFieldIP, field)

	invalidField := openapi.GetAdminUsersParamsField("email")
	_, err = AdminUsersFieldFromParams(openapi.GetAdminUsersParams{Field: &invalidField})
	require.Error(t, err)

	assert.NoError(t, ValidateAdminUsersSearch(UserSearchFieldUsername, new("not an ip")))
	assert.NoError(t, ValidateAdminUsersSearch(UserSearchFieldIP, new("192.0.2.1")))
	assert.Error(t, ValidateAdminUsersSearch(UserSearchFieldIP, new("not an ip")))
}

func TestAdminUserRoleParams(t *testing.T) {
	t.Parallel()

	adminRole := "admin"
	username, email, password, role, err := AdminCreateUserRequestToParams(&openapi.AdminCreateUserRequest{
		Username: "admin",
		Email:    "admin@example.com",
		Password: "Password12345",
		Role:     &adminRole,
	})
	require.NoError(t, err)
	assert.Equal(t, "admin", username)
	assert.Equal(t, "admin@example.com", email)
	assert.Equal(t, "Password12345", password)
	assert.Equal(t, "admin", role)

	invalidRole := "owner"
	_, _, _, _, err = AdminCreateUserRequestToParams(&openapi.AdminCreateUserRequest{Role: &invalidRole})
	require.Error(t, err)

	_, _, _, _, _, err = AdminUpdateUserRequestToParams(&openapi.AdminUpdateUserRequest{Role: &invalidRole})
	require.Error(t, err)
}
