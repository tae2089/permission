package apikey

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/http/input"
	"github.com/tae2089/trace/v3"
)

type Handler struct {
	service             Service
	instanceAdminKeySum [sha256.Size]byte
}

func NewHandler(service Service, instanceAdminKey string) *Handler {
	return &Handler{service: service, instanceAdminKeySum: sha256.Sum256([]byte(instanceAdminKey))}
}

func (h *Handler) Issue(c *gin.Context) {
	projectID, ok := h.projectID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	var request issueRequest
	if err := input.DecodeJSON(c.Writer, c.Request, &request); err != nil {
		h.fail(c, err, "decode API key issue request")
		return
	}
	issued, err := h.service.Issue(c.Request.Context(), actor, IssueInput{ProjectID: projectID, Kind: request.Kind})
	if err != nil {
		h.fail(c, err, "issue API key request")
		return
	}
	c.Header("Location", "/v1/projects/"+projectID.String()+"/api-keys/"+issued.Key.ID.String())
	c.JSON(http.StatusCreated, newIssuedKeyResponse(issued))
}

func (h *Handler) List(c *gin.Context) {
	projectID, ok := h.projectID(c)
	if !ok {
		return
	}
	actor, ok := h.actor(c)
	if !ok {
		return
	}
	keys, err := h.service.List(c.Request.Context(), actor, projectID)
	if err != nil {
		h.fail(c, err, "list API keys request")
		return
	}
	response := keysResponse{Keys: make([]keyResponse, 0, len(keys))}
	for _, key := range keys {
		response.Keys = append(response.Keys, newKeyResponse(key))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) Rotate(c *gin.Context) {
	projectID, keyID, ok := h.projectAndKeyID(c)
	if !ok {
		return
	}
	actor, ok := h.projectAdministrator(c)
	if !ok {
		return
	}
	issued, err := h.service.Rotate(c.Request.Context(), actor, projectID, keyID)
	if err != nil {
		h.fail(c, err, "rotate API key request")
		return
	}
	c.JSON(http.StatusCreated, newIssuedKeyResponse(issued))
}

func (h *Handler) Revoke(c *gin.Context) {
	projectID, keyID, ok := h.projectAndKeyID(c)
	if !ok {
		return
	}
	actor, ok := h.projectAdministrator(c)
	if !ok {
		return
	}
	if err := h.service.Revoke(c.Request.Context(), actor, projectID, keyID); err != nil {
		h.fail(c, err, "revoke API key request")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) projectID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("project_id"))
	if err != nil {
		h.fail(c, apperr.New(apperr.KindBadParameter, "invalid project ID"), "parse project ID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) projectAndKeyID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	projectID, ok := h.projectID(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	keyID, err := uuid.Parse(c.Param("key_id"))
	if err != nil {
		h.fail(c, apperr.New(apperr.KindBadParameter, "invalid API key ID"), "parse API key ID")
		return uuid.Nil, uuid.Nil, false
	}
	return projectID, keyID, true
}

func (h *Handler) actor(c *gin.Context) (any, bool) {
	token, ok := bearerToken(c.GetHeader("Authorization"))
	if !ok {
		h.fail(c, apperr.New(apperr.KindUnauthenticated, ""), "authenticate API key manager")
		return nil, false
	}
	provided := sha256.Sum256([]byte(token))
	if subtle.ConstantTimeCompare(provided[:], h.instanceAdminKeySum[:]) == 1 {
		return InstanceAdministrator{}, true
	}
	key, err := h.service.Authenticate(c.Request.Context(), token)
	if err != nil {
		h.fail(c, err, "authenticate API key manager")
		return nil, false
	}
	if key.Kind != KindProjectAdmin {
		h.fail(c, apperr.New(apperr.KindAccessDenied, ""), "authorize API key manager")
		return nil, false
	}
	return ProjectAdministrator{ProjectID: key.ProjectID, KeyID: key.ID}, true
}

func (h *Handler) projectAdministrator(c *gin.Context) (ProjectAdministrator, bool) {
	actor, ok := h.actor(c)
	if !ok {
		return ProjectAdministrator{}, false
	}
	projectAdministrator, ok := actor.(ProjectAdministrator)
	if !ok {
		h.fail(c, apperr.New(apperr.KindAccessDenied, ""), "authorize project API key manager")
		return ProjectAdministrator{}, false
	}
	return projectAdministrator, true
}

func bearerToken(authorization string) (string, bool) {
	scheme, token, ok := strings.Cut(authorization, " ")
	return token, ok && strings.EqualFold(scheme, "Bearer") && token != "" && !strings.ContainsAny(token, " \t")
}

func (h *Handler) fail(c *gin.Context, err error, operation string) {
	c.Error(trace.Wrap(err, operation))
	c.Abort()
}

func newIssuedKeyResponse(issued IssuedKey) issuedKeyResponse {
	key := newKeyResponse(issued.Key)
	return issuedKeyResponse{ID: key.ID, ProjectID: key.ProjectID, Kind: key.Kind, Status: key.Status, CreatedAt: key.CreatedAt, Secret: issued.Secret}
}

func newKeyResponse(key Key) keyResponse {
	return keyResponse{ID: key.ID.String(), ProjectID: key.ProjectID.String(), Kind: key.Kind, Status: key.Status, CreatedAt: key.CreatedAt, RevokedAt: key.RevokedAt}
}
