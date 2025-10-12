package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PortfolioController struct {
	logger *logrus.Logger
}

func NewPortfolioController(logger *logrus.Logger) *PortfolioController {
	return &PortfolioController{
		logger: logger,
	}
}

func (c *PortfolioController) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/health", c.Health)
}

func (c *PortfolioController) Health(ctx *gin.Context) {
	ctx.JSON(200, gin.H{"status": "ok"})
}
