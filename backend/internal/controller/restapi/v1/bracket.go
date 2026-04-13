package v1

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// (GET /brackets).
func (h *Server) GetBrackets(w http.ResponseWriter, r *http.Request) {
	list, err := h.comp.BracketUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetBrackets", "GetAll") {
		return
	}

	httputil.RenderOK(w, r, response.FromBracketList(list))
}

// (POST /admin/brackets).
func (h *Server) PostAdminBrackets(w http.ResponseWriter, r *http.Request) {
	req, ok := httputil.DecodeAndValidate[openapi.CreateBracketRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if err := request.ValidateCreateBracketRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PostAdminBrackets", "Validate") {
		return
	}

	name, description, isDefault, err := request.CreateBracketRequestToParams(&req)
	if h.OnError(w, r, err, "PostAdminBrackets", "CreateBracketRequestToParams") {
		return
	}

	bracket, err := h.comp.BracketUC.Create(r.Context(), name, description, isDefault)
	if h.OnError(w, r, err, "PostAdminBrackets", "Create") {
		return
	}

	httputil.RenderCreated(w, r, response.FromBracket(bracket))
}

// (GET /admin/brackets/{ID}).
func (h *Server) GetAdminBracketsID(w http.ResponseWriter, r *http.Request, ID string) {
	bracketIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	bracket, err := h.comp.BracketUC.GetByID(r.Context(), bracketIDParsed)
	if h.OnError(w, r, err, "GetAdminBracketsID", "GetByID") {
		return
	}

	httputil.RenderOK(w, r, response.FromBracket(bracket))
}

// (PUT /admin/brackets/{ID}).
func (h *Server) PutAdminBracketsID(w http.ResponseWriter, r *http.Request, ID string) {
	bracketIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.UpdateBracketRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	if err := request.ValidateUpdateBracketRequest(&req, h.infra.Validator); h.OnError(w, r, err, "PutAdminBracketsID", "Validate") {
		return
	}

	name, description, isDefault, err := request.UpdateBracketRequestToParams(&req)
	if h.OnError(w, r, err, "PutAdminBracketsID", "UpdateBracketRequestToParams") {
		return
	}

	bracket, err := h.comp.BracketUC.Update(r.Context(), bracketIDParsed, name, description, isDefault)
	if h.OnError(w, r, err, "PutAdminBracketsID", "Update") {
		return
	}

	httputil.RenderOK(w, r, response.FromBracket(bracket))
}

// (DELETE /admin/brackets/{ID}).
func (h *Server) DeleteAdminBracketsID(w http.ResponseWriter, r *http.Request, ID string) {
	bracketIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	if h.OnError(w, r, h.comp.BracketUC.Delete(r.Context(), bracketIDParsed), "DeleteAdminBracketsID", "Delete") {
		return
	}

	httputil.RenderNoContent(w, r)
}

// (PATCH /admin/teams/{ID}/bracket).
func (h *Server) PatchAdminTeamsIDBracket(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := httputil.ParseUUID(w, r, ID)
	if !ok {
		return
	}

	req, ok := httputil.DecodeAndValidate[openapi.SetTeamBracketRequest](
		w, r, h.infra.Validator,
	)
	if !ok {
		return
	}

	bracketID, err := request.SetTeamBracketRequestToParams(&req)
	if h.OnError(w, r, err, "PatchAdminTeamsIDBracket", "SetTeamBracketRequestToParams") {
		return
	}

	if h.OnError(w, r, h.team.TeamUC.SetBracket(r.Context(), teamIDParsed, bracketID), "PatchAdminTeamsIDBracket", "SetBracket") {
		return
	}

	team, err := h.team.TeamUC.GetByID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "PatchAdminTeamsIDBracket", "GetByID") {
		return
	}

	httputil.RenderOK(w, r, response.FromTeam(team))
}
