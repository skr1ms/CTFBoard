package errmap

import (
	"errors"
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httperr"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

type mapping struct {
	Status int
	Code   string
}

//nolint:gochecknoglobals // central error -> HTTP mapping table, intentionally global
var table = map[error]mapping{
	// user
	apperr.ErrUserNotFound:                {http.StatusNotFound, "USER_NOT_FOUND"},
	apperr.ErrUserAlreadyExists:           {http.StatusConflict, "USER_ALREADY_EXISTS"},
	apperr.ErrUsernameTaken:               {http.StatusConflict, "USERNAME_TAKEN"},
	apperr.ErrRegistrationClosed:          {http.StatusForbidden, "REGISTRATION_CLOSED"},
	apperr.ErrRegistrationCodeRequired:    {http.StatusForbidden, "REGISTRATION_CODE_REQUIRED"},
	apperr.ErrInvalidRegistrationCode:     {http.StatusForbidden, "INVALID_REGISTRATION_CODE"},
	apperr.ErrMaxUsersReached:             {http.StatusConflict, "MAX_USERS_REACHED"},
	apperr.ErrInvalidCredentials:          {http.StatusUnauthorized, "INVALID_CREDENTIALS"},
	apperr.ErrAuthorizationHeaderRequired: {http.StatusUnauthorized, "AUTHORIZATION_REQUIRED"},
	apperr.ErrInvalidAuthorizationHeader:  {http.StatusUnauthorized, "INVALID_AUTHORIZATION_HEADER"},
	apperr.ErrInvalidToken:                {http.StatusUnauthorized, "INVALID_TOKEN"},
	apperr.ErrAccessDenied:                {http.StatusForbidden, "ACCESS_DENIED"},
	apperr.ErrUserBanned:                  {http.StatusForbidden, "USER_BANNED"},
	apperr.ErrCaptainCannotBeDeleted:      {http.StatusConflict, "CAPTAIN_CANNOT_BE_DELETED"},
	apperr.ErrAppealNotFound:              {http.StatusNotFound, "APPEAL_NOT_FOUND"},
	apperr.ErrAppealRateLimited:           {http.StatusTooManyRequests, "APPEAL_RATE_LIMITED"},
	apperr.ErrAnimatedImageNotAllowed:     {http.StatusUnprocessableEntity, "ANIMATED_IMAGE_NOT_ALLOWED"},

	// team
	apperr.ErrTeamNotFound:            {http.StatusNotFound, "TEAM_NOT_FOUND"},
	apperr.ErrTeamAlreadyExists:       {http.StatusConflict, "TEAM_ALREADY_EXISTS"},
	apperr.ErrUserAlreadyInTeam:       {http.StatusConflict, "USER_ALREADY_IN_TEAM"},
	apperr.ErrTeamFull:                {http.StatusConflict, "TEAM_FULL"},
	apperr.ErrMaxTeamsReached:         {http.StatusConflict, "MAX_TEAMS_REACHED"},
	apperr.ErrNotCaptain:              {http.StatusForbidden, "NOT_CAPTAIN"},
	apperr.ErrCaptainCannotLeave:      {http.StatusForbidden, "CAPTAIN_CANNOT_LEAVE"},
	apperr.ErrCannotLeaveAsOnlyMember: {http.StatusConflict, "CANNOT_LEAVE_AS_ONLY_MEMBER"},
	apperr.ErrTeamBelowMinSize:        {http.StatusConflict, "TEAM_BELOW_MIN_SIZE"},
	apperr.ErrNewCaptainNotInTeam:     {http.StatusBadRequest, "NEW_CAPTAIN_NOT_IN_TEAM"},
	apperr.ErrCannotTransferToSelf:    {http.StatusBadRequest, "CANNOT_TRANSFER_TO_SELF"},
	apperr.ErrNoTeamSelected:          {http.StatusBadRequest, "NO_TEAM_SELECTED"},
	apperr.ErrSoloModeNotAllowed:      {http.StatusForbidden, "SOLO_MODE_NOT_ALLOWED"},
	apperr.ErrTeamsNotAllowed:         {http.StatusForbidden, "TEAMS_NOT_ALLOWED"},
	apperr.ErrTeamModeRequired:        {http.StatusForbidden, "TEAM_MODE_REQUIRED"},
	apperr.ErrSoloModeRequired:        {http.StatusForbidden, "SOLO_MODE_REQUIRED"},
	apperr.ErrConfirmationRequired:    {http.StatusBadRequest, "CONFIRMATION_REQUIRED"},
	apperr.ErrRosterFrozen:            {http.StatusForbidden, "ROSTER_FROZEN"},
	apperr.ErrEmailNotVerified:        {http.StatusForbidden, "EMAIL_NOT_VERIFIED"},
	apperr.ErrTeamBanned:              {http.StatusForbidden, "TEAM_BANNED"},
	apperr.ErrUserWasInBannedTeam:     {http.StatusForbidden, "USER_WAS_IN_BANNED_TEAM"},
	apperr.ErrInviteExpired:           {http.StatusGone, "INVITE_EXPIRED"},
	apperr.ErrTeamConflict:            {http.StatusConflict, "TEAM_CONFLICT"},
	apperr.ErrCannotAddToSoloTeam:     {http.StatusBadRequest, "CANNOT_ADD_TO_SOLO_TEAM"},
	apperr.ErrTeamMemberNotFound:      {http.StatusNotFound, "TEAM_MEMBER_NOT_FOUND"},
	apperr.ErrCannotKickSelf:          {http.StatusBadRequest, "CANNOT_KICK_SELF"},
	apperr.ErrCannotKickCaptain:       {http.StatusForbidden, "CANNOT_KICK_CAPTAIN"},
	apperr.ErrUserNotInTeam:           {http.StatusNotFound, "USER_NOT_IN_TEAM"},
	apperr.ErrCannotLeaveSoloTeam:     {http.StatusForbidden, "CANNOT_LEAVE_SOLO_TEAM"},
	apperr.ErrCannotDisbandSoloTeam:   {http.StatusForbidden, "CANNOT_DISBAND_SOLO_TEAM"},

	// challenge
	apperr.ErrChallengeNotFound:                      {http.StatusNotFound, "CHALLENGE_NOT_FOUND"},
	apperr.ErrInvalidFlagFormat:                      {http.StatusBadRequest, "INVALID_FLAG_FORMAT"},
	apperr.ErrInvalidScoringRange:                    {http.StatusBadRequest, "INVALID_SCORING_RANGE"},
	apperr.ErrChallengeFlagRequiredWhenSwitchingMode: {http.StatusBadRequest, "CHALLENGE_FLAG_REQUIRED"},
	apperr.ErrRequirementsNotMet:                     {http.StatusForbidden, "REQUIREMENTS_NOT_MET"},
	apperr.ErrMaxAttemptsReached:                     {http.StatusTooManyRequests, "MAX_ATTEMPTS_REACHED"},
	apperr.ErrChallengeLocked:                        {http.StatusForbidden, "CHALLENGE_LOCKED"},
	apperr.ErrTopicNotFound:                          {http.StatusNotFound, "TOPIC_NOT_FOUND"},
	apperr.ErrTopicNameRequired:                      {http.StatusBadRequest, "TOPIC_NAME_REQUIRED"},

	// competition
	apperr.ErrCompetitionNotFound:           {http.StatusNotFound, "COMPETITION_NOT_FOUND"},
	apperr.ErrCompetitionNotStarted:         {http.StatusForbidden, "COMPETITION_NOT_STARTED"},
	apperr.ErrCompetitionEnded:              {http.StatusForbidden, "COMPETITION_ENDED"},
	apperr.ErrCompetitionPaused:             {http.StatusForbidden, "COMPETITION_PAUSED"},
	apperr.ErrSubmissionNotAllowed:          {http.StatusForbidden, "SUBMISSION_NOT_ALLOWED"},
	apperr.ErrCommentsAvailableAfterEnd:     {http.StatusForbidden, "COMPETITION_NOT_ENDED"},
	apperr.ErrCompetitionActiveCannotUpdate: {http.StatusForbidden, "COMPETITION_ACTIVE_CANNOT_UPDATE"},

	// misc
	apperr.ErrVisibilityForbidden:               {http.StatusNotFound, "NOT_FOUND"},
	apperr.ErrDebugNotEnabled:                   {http.StatusNotFound, "NOT_FOUND"},
	apperr.ErrScoreboardHidden:                  {http.StatusForbidden, "SCOREBOARD_HIDDEN"},
	apperr.ErrScoreboardAdminsOnly:              {http.StatusForbidden, "SCOREBOARD_ADMINS_ONLY"},
	apperr.ErrScoreboardAccessDenied:            {http.StatusForbidden, "SCOREBOARD_ACCESS_DENIED"},
	apperr.ErrWebsocketOriginNotConfigured:      {http.StatusForbidden, "WEBSOCKET_ORIGIN_NOT_CONFIGURED"},
	apperr.ErrWebsocketWildcardOriginNotAllowed: {http.StatusForbidden, "WEBSOCKET_WILDCARD_ORIGIN_NOT_ALLOWED"},

	// file
	apperr.ErrFileNotFound:        {http.StatusNotFound, "FILE_NOT_FOUND"},
	apperr.ErrWriteupAccessDenied: {http.StatusForbidden, "WRITEUP_ACCESS_DENIED"},
	apperr.ErrWriteupsDisabled:    {http.StatusForbidden, "WRITEUPS_DISABLED"},
	apperr.ErrFileIDMismatch:      {http.StatusBadRequest, "FILE_ID_MISMATCH"},
	apperr.ErrFileInvalidToken:    {http.StatusBadRequest, "FILE_INVALID_TOKEN"},
	apperr.ErrFileTokenExpired:    {http.StatusBadRequest, "FILE_TOKEN_EXPIRED"},

	// hint
	apperr.ErrHintNotFound:        {http.StatusNotFound, "HINT_NOT_FOUND"},
	apperr.ErrHintAlreadyUnlocked: {http.StatusConflict, "HINT_ALREADY_UNLOCKED"},
	apperr.ErrInsufficientPoints:  {http.StatusPaymentRequired, "INSUFFICIENT_POINTS"},
	apperr.ErrHintOrderRequired:   {http.StatusBadRequest, "HINT_ORDER_REQUIRED"},

	// settings
	apperr.ErrAppSettingsNotFound:                   {http.StatusNotFound, "APP_SETTINGS_NOT_FOUND"},
	apperr.ErrSettingsCannotChangeDuringCompetition: {http.StatusForbidden, "SETTINGS_CANNOT_CHANGE_DURING_COMPETITION"},
	apperr.ErrSettingsConflict:                      {http.StatusConflict, "SETTINGS_CONFLICT"},

	// backup
	apperr.ErrBackupJSONNotFound:         {http.StatusBadRequest, "BACKUP_JSON_NOT_FOUND"},
	apperr.ErrBackupVersionUnsupported:   {http.StatusBadRequest, "BACKUP_VERSION_UNSUPPORTED"},
	apperr.ErrBackupTableUnsupported:     {http.StatusBadRequest, "BACKUP_TABLE_UNSUPPORTED"},
	apperr.ErrBackupCSVEmpty:             {http.StatusBadRequest, "BACKUP_CSV_EMPTY"},
	apperr.ErrBackupImportJobNotFound:    {http.StatusNotFound, "BACKUP_IMPORT_JOB_NOT_FOUND"},
	apperr.ErrBackupImportAlreadyRunning: {http.StatusConflict, "BACKUP_IMPORT_ALREADY_RUNNING"},

	// page
	apperr.ErrPageNotFound:      {http.StatusNotFound, "PAGE_NOT_FOUND"},
	apperr.ErrPageSlugConflict:  {http.StatusConflict, "PAGE_SLUG_CONFLICT"},
	apperr.ErrPageSlugRequired:  {http.StatusBadRequest, "PAGE_SLUG_REQUIRED"},
	apperr.ErrPageSlugInvalid:   {http.StatusBadRequest, "PAGE_SLUG_INVALID"},
	apperr.ErrPageTitleRequired: {http.StatusBadRequest, "PAGE_TITLE_REQUIRED"},

	// oauth
	apperr.ErrOAuthAccountNotFound:      {http.StatusNotFound, "OAUTH_ACCOUNT_NOT_FOUND"},
	apperr.ErrOAuthProviderDisabled:     {http.StatusBadRequest, "OAUTH_PROVIDER_DISABLED"},
	apperr.ErrOAuthUnsupportedProvider:  {http.StatusBadRequest, "OAUTH_UNSUPPORTED_PROVIDER"},
	apperr.ErrOAuthStateMissing:         {http.StatusBadRequest, "OAUTH_STATE_MISSING"},
	apperr.ErrOAuthStateMismatch:        {http.StatusBadRequest, "OAUTH_STATE_MISMATCH"},
	apperr.ErrOAuthAccountAlreadyLinked: {http.StatusConflict, "OAUTH_ACCOUNT_ALREADY_LINKED"},

	// solve
	apperr.ErrSolveNotFound:        {http.StatusNotFound, "SOLVE_NOT_FOUND"},
	apperr.ErrAlreadySolved:        {http.StatusConflict, "ALREADY_SOLVED"},
	apperr.ErrSolutionAccessDenied: {http.StatusForbidden, "SOLUTION_ACCESS_DENIED"},
	apperr.ErrShareNotFound:        {http.StatusNotFound, "SHARE_NOT_FOUND"},
	apperr.ErrSharesDisabled:       {http.StatusForbidden, "SHARES_DISABLED"},

	// avatar
	apperr.ErrInvalidAvatar:          {http.StatusBadRequest, "INVALID_AVATAR"},
	apperr.ErrFileTooLarge:           {http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE"},
	apperr.ErrUnsupportedMediaType:   {http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE"},
	apperr.ErrAvatarProcessingFailed: {http.StatusInternalServerError, "AVATAR_PROCESSING_FAILED"},
	apperr.ErrNotTeamCaptain:         {http.StatusForbidden, "NOT_TEAM_CAPTAIN"},

	// competition_params
	apperr.ErrCompetitionParamNotFound:         {http.StatusNotFound, "COMPETITION_PARAM_NOT_FOUND"},
	apperr.ErrCompetitionParamKeyRequired:      {http.StatusBadRequest, "COMPETITION_PARAM_KEY_REQUIRED"},
	apperr.ErrCompetitionParamInvalidValueType: {http.StatusBadRequest, "COMPETITION_PARAM_INVALID_VALUE_TYPE"},

	// bracket
	apperr.ErrBracketNotFound:     {http.StatusNotFound, "BRACKET_NOT_FOUND"},
	apperr.ErrBracketNameConflict: {http.StatusConflict, "BRACKET_NAME_CONFLICT"},
	apperr.ErrBracketNameRequired: {http.StatusBadRequest, "BRACKET_NAME_REQUIRED"},

	// comment
	apperr.ErrCommentNotFound:        {http.StatusNotFound, "COMMENT_NOT_FOUND"},
	apperr.ErrCommentForbidden:       {http.StatusForbidden, "COMMENT_FORBIDDEN"},
	apperr.ErrCommentContentRequired: {http.StatusBadRequest, "COMMENT_CONTENT_REQUIRED"},

	// field
	apperr.ErrFieldNotFound:       {http.StatusNotFound, "FIELD_NOT_FOUND"},
	apperr.ErrFieldInvalidNumber:  {http.StatusBadRequest, "FIELD_INVALID_NUMBER"},
	apperr.ErrFieldInvalidBoolean: {http.StatusBadRequest, "FIELD_INVALID_BOOLEAN"},
	apperr.ErrFieldTextTooLong:    {http.StatusBadRequest, "FIELD_TEXT_TOO_LONG"},

	// tag
	apperr.ErrTagNotFound:     {http.StatusNotFound, "TAG_NOT_FOUND"},
	apperr.ErrTagNameRequired: {http.StatusBadRequest, "TAG_NAME_REQUIRED"},

	// submission
	apperr.ErrSubmissionNotFound: {http.StatusNotFound, "SUBMISSION_NOT_FOUND"},

	// notification
	apperr.ErrNotificationNotFound:             {http.StatusNotFound, "NOTIFICATION_NOT_FOUND"},
	apperr.ErrNotificationTitleContentRequired: {http.StatusBadRequest, "NOTIFICATION_TITLE_CONTENT_REQUIRED"},

	// rating
	apperr.ErrRatingNotFound:         {http.StatusNotFound, "RATING_NOT_FOUND"},
	apperr.ErrSolveRequiredForRating: {http.StatusForbidden, "SOLVE_REQUIRED_FOR_RATING"},

	// api_token
	apperr.ErrAPITokenNotFound: {http.StatusNotFound, "API_TOKEN_NOT_FOUND"},

	// award
	apperr.ErrAwardNotFound:          {http.StatusNotFound, "AWARD_NOT_FOUND"},
	apperr.ErrAwardTeamIDRequired:    {http.StatusBadRequest, "AWARD_TEAM_ID_REQUIRED"},
	apperr.ErrAwardValueCannotBeZero: {http.StatusBadRequest, "AWARD_VALUE_CANNOT_BE_ZERO"},

	// setup wizard
	apperr.ErrSetupAlreadyComplete: {http.StatusConflict, "SETUP_ALREADY_COMPLETE"},
	apperr.ErrSetupRequired:        {http.StatusServiceUnavailable, "SETUP_REQUIRED"},

	// auth / rate-limit - domain sentinels returned by usecase (e.g. login lockout)
	apperr.ErrNotAuthenticated: {http.StatusUnauthorized, "NOT_AUTHENTICATED"},
	apperr.ErrTooManyRequests:  {http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED"},

	// verification_token
	apperr.ErrTokenRequired:    {http.StatusBadRequest, "TOKEN_REQUIRED"},
	apperr.ErrTokenNotFound:    {http.StatusNotFound, "TOKEN_NOT_FOUND"},
	apperr.ErrTokenExpired:     {http.StatusGone, "TOKEN_EXPIRED"},
	apperr.ErrTokenAlreadyUsed: {http.StatusConflict, "TOKEN_ALREADY_USED"},
}

// MapAppError wraps a domain error from apperr into *httperr.HTTPError
// for JSON rendering by go-httpkit's HandleError.
// Errors that are already *HTTPError pass through unchanged.
//
// Lookup strategy: O(1) direct map hit for bare sentinels, then O(n)
// errors.Is scan for wrapped errors (fmt.Errorf("…: %w", sentinel)).
func MapAppError(err error) error {
	if err == nil {
		return nil
	}

	var he *httperr.HTTPError
	if errors.As(err, &he) {
		return err
	}

	var ve *apperr.ValidationError
	if errors.As(err, &ve) {
		return httperr.New(ve, http.StatusBadRequest, "VALIDATION_ERROR")
	}

	if m, ok := table[err]; ok {
		return httperr.New(err, m.Status, m.Code)
	}

	for sentinel, m := range table {
		if errors.Is(err, sentinel) {
			return httperr.New(err, m.Status, m.Code)
		}
	}

	return httperr.New(err, http.StatusInternalServerError, "INTERNAL_ERROR")
}

// NewHTTPError constructs a transport error in the backend's central HTTP error
// mapping package, keeping handlers and router glue off local pkg aliases.
func NewHTTPError(err error, status int, code string) error {
	return httperr.New(err, status, code)
}

// Code returns the mapped HTTP error code for an application error.
func Code(err error) string {
	mapped := MapAppError(err)

	var he *httperr.HTTPError
	if errors.As(mapped, &he) {
		return he.GetCode()
	}

	return "INTERNAL_ERROR"
}
