package v1

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/request"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/response"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

// Get brackets list
// (GET /brackets)
func (h *Server) GetBrackets(w http.ResponseWriter, r *http.Request) {
	list, err := h.comp.BracketUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetBrackets", "GetAll") {
		return
	}
	helper.RenderOK(w, r, response.FromBracketList(list))
}

// Create bracket
// (POST /admin/brackets)
func (h *Server) PostAdminBrackets(w http.ResponseWriter, r *http.Request) {
	req, ok := helper.DecodeAndValidate[openapi.CreateBracketRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PostAdminBrackets",
	)
	if !ok {
		return
	}
	name, description, isDefault := request.CreateBracketRequestToParams(&req)
	bracket, err := h.comp.BracketUC.Create(r.Context(), name, description, isDefault)
	if h.OnError(w, r, err, "PostAdminBrackets", "Create") {
		return
	}
	helper.RenderCreated(w, r, response.FromBracket(bracket))
}

// Get bracket by ID
// (GET /admin/brackets/{ID})
func (h *Server) GetAdminBracketsID(w http.ResponseWriter, r *http.Request, ID string) {
	bracketIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	bracket, err := h.comp.BracketUC.GetByID(r.Context(), bracketIDParsed)
	if h.OnError(w, r, err, "GetAdminBracketsID", "GetByID") {
		return
	}
	helper.RenderOK(w, r, response.FromBracket(bracket))
}

// Update bracket
// (PUT /admin/brackets/{ID})
func (h *Server) PutAdminBracketsID(w http.ResponseWriter, r *http.Request, ID string) {
	bracketIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.UpdateBracketRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PutAdminBracketsID",
	)
	if !ok {
		return
	}
	name, description, isDefault := request.UpdateBracketRequestToParams(&req)
	bracket, err := h.comp.BracketUC.Update(r.Context(), bracketIDParsed, name, description, isDefault)
	if h.OnError(w, r, err, "PutAdminBracketsID", "Update") {
		return
	}
	helper.RenderOK(w, r, response.FromBracket(bracket))
}

// Delete bracket
// (DELETE /admin/brackets/{ID})
func (h *Server) DeleteAdminBracketsID(w http.ResponseWriter, r *http.Request, ID string) {
	bracketIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	if h.OnError(w, r, h.comp.BracketUC.Delete(r.Context(), bracketIDParsed), "DeleteAdminBracketsID", "Delete") {
		return
	}
	helper.RenderNoContent(w, r)
}

// Set team bracket
// (PATCH /admin/teams/{ID}/bracket)
func (h *Server) PatchAdminTeamsIDBracket(w http.ResponseWriter, r *http.Request, ID string) {
	teamIDParsed, ok := helper.ParseUUID(w, r, ID)
	if !ok {
		return
	}
	req, ok := helper.DecodeAndValidate[openapi.SetTeamBracketRequest](
		w, r, h.infra.Validator, h.infra.Logger, "PatchAdminTeamsIDBracket",
	)
	if !ok {
		return
	}
	bracketID := request.SetTeamBracketRequestToParams(&req)
	if h.OnError(w, r, h.team.TeamUC.SetBracket(r.Context(), teamIDParsed, bracketID), "PatchAdminTeamsIDBracket", "SetBracket") {
		return
	}
	team, err := h.team.TeamUC.GetByID(r.Context(), teamIDParsed)
	if h.OnError(w, r, err, "PatchAdminTeamsIDBracket", "GetByID") {
		return
	}
	helper.RenderOK(w, r, response.FromTeam(team))
}
