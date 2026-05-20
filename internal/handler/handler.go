package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/healthcheck/internal/store"
)

// Storer defines the methods our database layer must implement.
// This allows us to "mock" the database during unit tests.
type Storer interface {
	GetLatestChecks(ctx context.Context) ([]store.Check, error)
	GetTargets(ctx context.Context) ([]store.Target, error)
	InsertTarget(ctx context.Context, name, url string) (store.Target, error)
	DeleteTarget(ctx context.Context, id int) error
	GetHistoricalChecks(ctx context.Context, target string, limit int) ([]store.Check, error)
	GetPreviousCheckStatus(ctx context.Context, target string) (string, error)
}

type Handler struct {
	store Storer
}

func New(s Storer) *Handler {
	return &Handler{store: s}
}

// GET /health
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"time":    time.Now().UTC().Format(time.RFC3339),
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
	target := c.Query("target")
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target query parameter is required"})
		return
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "30"))
	if err != nil || limit <= 0 {
		limit = 30
	}

	ctx := c.Request.Context()
	checks, err := h.store.GetHistoricalChecks(ctx, target, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, checks)
}

// GET /api/targets
func (h *Handler) GetTargets(c *gin.Context) {
	ctx := c.Request.Context()
	targets, err := h.store.GetTargets(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, targets)
}

// POST /api/targets
func (h *Handler) CreateTarget(c *gin.Context) {
	var input struct {
		Name string `json:"name" binding:"required"`
		URL  string `json:"url" binding:"required,url"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	target, err := h.store.InsertTarget(ctx, input.Name, input.URL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, target)
}

// DELETE /api/targets/:id
func (h *Handler) DeleteTarget(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.store.DeleteTarget(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "target not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "target deleted"})
}

// GET /api/test/error (Chaos Testing)
func (h *Handler) TestError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Chaos alert triggered: internal server error"})
}

// GET /api/test/slow (Chaos Testing)
func (h *Handler) TestSlow(c *gin.Context) {
	time.Sleep(2 * time.Second)
	c.JSON(http.StatusOK, gin.H{"message": "Chaos alert triggered: slow response"})
}
