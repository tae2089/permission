package project

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tae2089/trace/v3"

	"github.com/tae2089/go-template/internal/apperr"
	"github.com/tae2089/go-template/internal/http/input"
)

type Handler struct {
	service             Service
	instanceAdminKeySum [sha256.Size]byte
}

type createRequest struct {
	Name string `json:"name"`
}

type projectResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type projectsResponse struct {
	Projects []projectResponse `json:"projects"`
}

func NewHandler(service Service, instanceAdminKey string) *Handler {
	return &Handler{
		service:             service,
		instanceAdminKeySum: sha256.Sum256([]byte(instanceAdminKey)),
	}
}

func (h *Handler) Create(c *gin.Context) {
	if !h.isAuthenticated(c.GetHeader("Authorization")) {
		c.Error(trace.Wrap(apperr.New(apperr.KindUnauthenticated, ""), "authenticate instance administrator"))
		c.Abort()
		return
	}

	var request createRequest
	if err := input.DecodeJSON(c.Writer, c.Request, &request); err != nil {
		c.Error(trace.Wrap(err, "decode create project request"))
		c.Abort()
		return
	}

	created, err := h.service.Create(c.Request.Context(), CreateInput{Name: request.Name})
	if err != nil {
		c.Error(trace.Wrap(err, "create project request"))
		c.Abort()
		return
	}

	c.Header("Location", "/v1/projects/"+created.ID.String())
	c.JSON(http.StatusCreated, newProjectResponse(created))
}

func (h *Handler) List(c *gin.Context) {
	if !h.isAuthenticated(c.GetHeader("Authorization")) {
		c.Error(trace.Wrap(apperr.New(apperr.KindUnauthenticated, ""), "authenticate instance administrator"))
		c.Abort()
		return
	}

	projects, err := h.service.List(c.Request.Context())
	if err != nil {
		c.Error(trace.Wrap(err, "list projects request"))
		c.Abort()
		return
	}

	response := projectsResponse{Projects: make([]projectResponse, 0, len(projects))}
	for _, project := range projects {
		response.Projects = append(response.Projects, newProjectResponse(project))
	}
	c.JSON(http.StatusOK, response)
}

func (h *Handler) isAuthenticated(authorization string) bool {
	scheme, token, ok := strings.Cut(authorization, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t") {
		return false
	}

	providedSum := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(providedSum[:], h.instanceAdminKeySum[:]) == 1
}

func newProjectResponse(project Project) projectResponse {
	return projectResponse{
		ID:        project.ID.String(),
		Name:      project.Name,
		CreatedAt: project.CreatedAt,
	}
}
