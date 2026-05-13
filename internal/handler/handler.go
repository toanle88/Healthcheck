package handler

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/toanle88/healthcheck/internal/store"
)

type Handler struct {
    store *store.Store
}

func New(s *store.Store) *Handler {
    return &Handler{store: s}
}

// GET /health
func (h *Handler) Health(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status": "ok",
        "time":   time.Now().UTC().Format(time.RFC3339),
        "service": "healthcheck-api",
    })
}

// GET /api/status
func (h *Handler) Status(c *gin.Context) {
	ctx := c.Request.Context()

	// Call the new store method which handles the grouping and latest logic
	checks, err := h.store.GetLatestChecks(ctx)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{"checks": checks, "count": len(checks)})
}

// GET /api/history
func (h *Handler) History(c *gin.Context) {
    h.Status(c) // Day 1: same as status
}