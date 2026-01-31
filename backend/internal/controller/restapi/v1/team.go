package v1

import (
	"errors"
	"net/http"

	"github.com/go-chi/render"
	"github.com/google/uuid"
	restapimiddleware "github.com/skr1ms/CTFBoard/internal/controller/restapi/middleware"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/request"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

// Create team
// (POST /teams)
func (h *Server) PostTeams(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.CreateTeamRequest](
		w, r, h.validator, h.logger, "PostTeams",
	)
	if !ok {
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	team, err := h.teamUC.Create(r.Context(), req.Name, user.ID, false, req.ConfirmReset)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostTeams")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	httputil.RenderCreated(w, r, res)
}

// Join team
// (POST /teams/join)
func (h *Server) PostTeamsJoin(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.JoinTeamRequest](
		w, r, h.validator, h.logger, "PostTeamsJoin",
	)
	if !ok {
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	inviteTokenuuid, err := uuid.Parse(req.InviteToken)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, openapi.V1ErrorResponse{Error: ptr("invalid invite token format")})
		return
	}

	team, err := h.teamUC.Join(r.Context(), inviteTokenuuid, user.ID, req.ConfirmReset)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostTeamsJoin")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// Leave team
// (POST /teams/leave)
func (h *Server) PostTeamsLeave(w http.ResponseWriter, r *http.Request) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.teamUC.Leave(r.Context(), user.ID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostTeamsLeave")
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Successfully left the team"})
}

// Disband team
// (DELETE /teams/me)
func (h *Server) DeleteTeamsMe(w http.ResponseWriter, r *http.Request) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	if err := h.teamUC.DisbandTeam(r.Context(), user.ID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - DeleteTeamsMe")
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Team disbanded successfully"})
}

// Kick member
// (DELETE /teams/members/{ID})
func (h *Server) DeleteTeamsMembersID(w http.ResponseWriter, r *http.Request, ID string) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	targetuserID, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	if err := h.teamUC.KickMember(r.Context(), user.ID, targetuserID); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - DeleteTeamsMembersID")
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Member kicked successfully"})
}

// Get my team
// (GET /teams/my)
func (h *Server) GetTeamsMy(w http.ResponseWriter, r *http.Request) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	team, members, err := h.teamUC.GetMyTeam(r.Context(), user.ID)
	if err != nil {
		if errors.Is(err, entityError.ErrTeamNotFound) {
			httputil.RenderError(w, r, http.StatusNotFound, "user is not in a team")
			return
		}
		h.logger.WithError(err).Error("restapi - v1 - GetTeamsMy")
		handleError(w, r, err)
		return
	}

	res := response.FromTeamWithMembers(team, members)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}

// Create solo team
// (POST /teams/solo)
func (h *Server) PostTeamsSolo(w http.ResponseWriter, r *http.Request) {
	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req request.CreateTeamRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.RenderError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	team, err := h.teamUC.CreateSoloTeam(r.Context(), user.ID, req.ConfirmReset)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostTeamsSolo")
		handleError(w, r, err)
		return
	}

	res := response.FromTeam(team)

	httputil.RenderCreated(w, r, res)
}

// Transfer captainship
// (POST /teams/transfer-captain)
func (h *Server) PostTeamsTransferCaptain(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[request.TransferCaptainRequest](
		w, r, h.validator, h.logger, "PostTeamsTransferCaptain",
	)
	if !ok {
		return
	}

	user, ok := restapimiddleware.GetUser(r.Context())
	if !ok {
		httputil.RenderError(w, r, http.StatusUnauthorized, "not authenticated")
		return
	}

	newCaptainuuid, err := uuid.Parse(req.NewCaptainID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	if err := h.teamUC.TransferCaptain(r.Context(), user.ID, newCaptainuuid); err != nil {
		h.logger.WithError(err).Error("restapi - v1 - PostTeamsTransferCaptain")
		handleError(w, r, err)
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, map[string]string{"message": "Captainship transferred successfully"})
}

// Get team by ID
// (GET /teams/{ID})
func (h *Server) GetTeamsID(w http.ResponseWriter, r *http.Request, ID string) {
	teamuuid, err := uuid.Parse(ID)
	if err != nil {
		httputil.RenderInvalidID(w, r)
		return
	}

	team, err := h.teamUC.GetByID(r.Context(), teamuuid)
	if err != nil {
		h.logger.WithError(err).Error("restapi - v1 - GetTeamsID")
		handleError(w, r, err)
		return
	}

	res := response.FromTeamWithoutToken(team)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, res)
}
