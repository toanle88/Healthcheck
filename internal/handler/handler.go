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

// CreateTargetInput defines the schema for target creation request body.
type CreateTargetInput struct {
	Name string `json:"name" binding:"required" example:"Google"`
	URL  string `json:"url" binding:"required,url" example:"https://google.com"`
}

// Health godoc
// @Summary Check service health
// @Description Returns the operational status, current time, and service name.
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"time":    time.Now().UTC().Format(time.RFC3339),
		"service": "healthcheck-api",
	})
}

// Status godoc
// @Summary Get latest checks status
// @Description Retrieves the most recent check results for all active targets, including their computed 24-hour SLA.
// @Tags Status
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/status [get]
// @Security EntraID
// @Security BearerAuth
func (h *Handler) Status(c *gin.Context) {
	ctx := c.Request.Context()

	// Call the new store method which handles the grouping and latest logic
	checks, err := h.store.GetLatestChecks(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"checks": checks, "count": len(checks)})
}

// History godoc
// @Summary Get historical checks
// @Description Retrieves historical ping results for a specific target URL.
// @Tags History
// @Produce json
// @Param target query string true "Target URL to filter history by"
// @Param limit query int false "Max historical records to return (default 30)"
// @Success 200 {array} store.Check
// @Router /api/history [get]
// @Security EntraID
// @Security BearerAuth
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

// GetTargets godoc
// @Summary Get all monitored targets
// @Description Retrieves the list of active URL targets being monitored by the system.
// @Tags Targets
// @Produce json
// @Success 200 {array} store.Target
// @Router /api/targets [get]
// @Security EntraID
// @Security BearerAuth
func (h *Handler) GetTargets(c *gin.Context) {
	ctx := c.Request.Context()
	targets, err := h.store.GetTargets(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, targets)
}

// CreateTarget godoc
// @Summary Create a monitored target
// @Description Adds a new target URL to the monitoring queue.
// @Tags Targets
// @Accept json
// @Produce json
// @Param target body CreateTargetInput true "Target configuration details"
// @Success 201 {object} store.Target
// @Router /api/targets [post]
// @Security EntraID
// @Security BearerAuth
func (h *Handler) CreateTarget(c *gin.Context) {
	var input CreateTargetInput

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

// DeleteTarget godoc
// @Summary Delete a monitored target
// @Description Removes a URL target from the monitor list by its database ID.
// @Tags Targets
// @Produce json
// @Param id path int true "Target Database ID"
// @Success 200 {object} map[string]string
// @Router /api/targets/{id} [delete]
// @Security EntraID
// @Security BearerAuth
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

// TestError godoc
// @Summary Trigger a chaos error
// @Description Artificially triggers a 500 Internal Server Error for testing observability and alerting.
// @Tags Chaos
// @Produce json
// @Success 500 {object} map[string]string
// @Router /api/test/error [get]
func (h *Handler) TestError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Chaos alert triggered: internal server error"})
}

// TestSlow godoc
// @Summary Trigger a slow chaos response
// @Description Deliberately delays the response by 2 seconds to test timeout handling and latency metrics.
// @Tags Chaos
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/test/slow [get]
func (h *Handler) TestSlow(c *gin.Context) {
	time.Sleep(2 * time.Second)
	c.JSON(http.StatusOK, gin.H{"message": "Chaos alert triggered: slow response"})
}
