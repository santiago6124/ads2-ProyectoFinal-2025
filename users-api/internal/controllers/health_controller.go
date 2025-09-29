package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"users-api/pkg/database"
)

type HealthController struct {
	db        *database.Database
	startTime time.Time
}

func NewHealthController(db *database.Database) *HealthController {
	return &HealthController{
		db:        db,
		startTime: time.Now(),
	}
}

type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Database  string    `json:"database"`
	Uptime    string    `json:"uptime"`
	Version   string    `json:"version"`
}

type ReadinessResponse struct {
	Ready    bool                   `json:"ready"`
	Checks   map[string]CheckResult `json:"checks"`
}

type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// Health godoc
// @Summary Health check endpoint
// @Description Check if the service is healthy
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (hc *HealthController) Health(c *gin.Context) {
	uptime := time.Since(hc.startTime)

	dbStatus := "connected"
	if err := hc.db.Ping(); err != nil {
		dbStatus = "disconnected"
	}

	response := HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Database:  dbStatus,
		Uptime:    uptime.String(),
		Version:   "1.0.0",
	}

	c.JSON(http.StatusOK, response)
}

// Readiness godoc
// @Summary Readiness check endpoint
// @Description Check if the service is ready to accept requests
// @Tags health
// @Produce json
// @Success 200 {object} ReadinessResponse
// @Failure 503 {object} ReadinessResponse
// @Router /ready [get]
func (hc *HealthController) Readiness(c *gin.Context) {
	checks := make(map[string]CheckResult)
	ready := true

	if err := hc.db.Ping(); err != nil {
		checks["database"] = CheckResult{
			Status:  "fail",
			Message: err.Error(),
		}
		ready = false
	} else {
		checks["database"] = CheckResult{
			Status: "pass",
		}
	}

	response := ReadinessResponse{
		Ready:  ready,
		Checks: checks,
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, response)
}

// Liveness godoc
// @Summary Liveness check endpoint
// @Description Check if the service is alive
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /live [get]
func (hc *HealthController) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"time":   time.Now(),
	})
}