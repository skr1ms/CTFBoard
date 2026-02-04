package v1

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/controller/restapi/v1/response"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

// Get brackets list
// (GET /brackets)
func (h *Server) GetBrackets(w http.ResponseWriter, r *http.Request) {
	list, err := h.bracketUC.GetAll(r.Context())
	if h.OnError(w, r, err, "GetBrackets", "GetAll") {
		return
	}
	RenderOK(w, r, response.FromBracketList(list))
}

// Create bracket
// (POST /admin/brackets)
func (h *Server) PostAdminBrackets(w http.ResponseWriter, r *http.Request) {
	req, ok := DecodeAndValidate[openapi.RequestCreateBracketRequest](w, r, h.validator, h.logger, "PostAdminBrackets")
	if !ok {
		return
	}
	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	bracket, err := h.bracketUC.Create(r.Context(), req.Name, desc, isDefault)
	if h.OnError(w, r, err, "PostAdminBrackets", "Create") {
		return
	}
	RenderCreated(w, r, response.FromBracket(bracket))
}

// Get bracket by ID
// (GET /admin/brackets/{ID})
func (h *Server) GetAdminBracketsID(w http.ResponseWriter, r *http.Request, id string) {
	bracketID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	bracket, err := h.bracketUC.GetByID(r.Context(), bracketID)
	if h.OnError(w, r, err, "GetAdminBracketsID", "GetByID") {
		return
	}
	RenderOK(w, r, response.FromBracket(bracket))
}

// Update bracket
// (PUT /admin/brackets/{ID})
func (h *Server) PutAdminBracketsID(w http.ResponseWriter, r *http.Request, id string) {
	bracketID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := DecodeAndValidate[openapi.RequestUpdateBracketRequest](w, r, h.validator, h.logger, "PutAdminBracketsID")
	if !ok {
		return
	}
	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	}
	bracket, err := h.bracketUC.Update(r.Context(), bracketID, req.Name, desc, isDefault)
	if h.OnError(w, r, err, "PutAdminBracketsID", "Update") {
		return
	}
	RenderOK(w, r, response.FromBracket(bracket))
}

// Delete bracket
// (DELETE /admin/brackets/{ID})
func (h *Server) DeleteAdminBracketsID(w http.ResponseWriter, r *http.Request, id string) {
	bracketID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	if h.OnError(w, r, h.bracketUC.Delete(r.Context(), bracketID), "DeleteAdminBracketsID", "Delete") {
		return
	}
	RenderNoContent(w, r)
}

// Set team bracket
// (PATCH /admin/teams/{ID}/bracket)
func (h *Server) PatchAdminTeamsIDBracket(w http.ResponseWriter, r *http.Request, id string) {
	teamID, ok := ParseUUID(w, r, id)
	if !ok {
		return
	}
	req, ok := DecodeAndValidate[openapi.RequestSetTeamBracketRequest](w, r, h.validator, h.logger, "PatchAdminTeamsIDBracket")
	if !ok {
		return
	}
	var bracketID *uuid.UUID
	if req.BracketID != nil {
		u := *req.BracketID
		bracketID = &u
	}
	if h.OnError(w, r, h.teamUC.SetBracket(r.Context(), teamID, bracketID), "PatchAdminTeamsIDBracket", "SetBracket") {
		return
	}
	team, err := h.teamUC.GetByID(r.Context(), teamID)
	if h.OnError(w, r, err, "PatchAdminTeamsIDBracket", "GetByID") {
		return
	}
	RenderOK(w, r, response.FromTeam(team))
}
