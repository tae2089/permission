package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tae2089/trace/v3"

	"github.com/tae2089/go-template/internal/http/input"
)

type Handler struct {
	service Service
}

type createRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(c *gin.Context) {
	var request createRequest
	if err := input.DecodeJSON(c.Writer, c.Request, &request); err != nil {
		c.Error(trace.Wrap(err, "decode create user request"))
		c.Abort()
		return
	}

	created, err := h.service.Create(c.Request.Context(), CreateInput{
		Name:  request.Name,
		Email: request.Email,
	})
	if err != nil {
		c.Error(trace.Wrap(err, "create user request"))
		c.Abort()
		return
	}

	c.Header("Location", "/users/"+created.ID.String())
	c.Status(http.StatusCreated)
}
