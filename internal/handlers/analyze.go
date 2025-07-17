package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"

	"github.com/webpage-analyser-server/internal/constants"
	"github.com/webpage-analyser-server/internal/models"
	"github.com/webpage-analyser-server/internal/services"
)

// AnalyzeHandler handles webpage analysis requests
type AnalyzeHandler struct {
	logger    *zap.Logger
	analyzer  *services.Analyzer
	validator *validator.Validate
}

// NewAnalyzeHandler creates a new AnalyzeHandler instance
func NewAnalyzeHandler(logger *zap.Logger, analyzer *services.Analyzer) *AnalyzeHandler {
	return &AnalyzeHandler{
		logger:    logger,
		analyzer:  analyzer,
		validator: validator.New(),
	}
}

// Handle processes webpage analysis requests
func (h *AnalyzeHandler) Handle(c *gin.Context) {
	var req models.AnalyzeRequest

	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(constants.StatusBadRequest, models.ErrorResponse{
			Code:    constants.StatusBadRequest,
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(constants.StatusBadRequest, models.ErrorResponse{
			Code:    constants.StatusBadRequest,
			Message: "Validation failed",
			Details: err.Error(),
		})
		return
	}

	// Custom validation
	if err := req.Validate(); err != nil {
		c.JSON(constants.StatusBadRequest, models.ErrorResponse{
			Code:    constants.StatusBadRequest,
			Message: "Validation failed",
			Details: err.Error(),
		})
		return
	}

	// Analyze webpage
	result, err := h.analyzer.Analyze(c.Request.Context(), req.URL)
	if err != nil {
		h.logger.Error("Failed to analyze webpage",
			zap.String("url", req.URL),
			zap.Error(err),
		)

		c.JSON(constants.StatusInternalServerError, models.ErrorResponse{
			Code:    constants.StatusInternalServerError,
			Message: "Failed to analyze webpage",
			Details: err.Error(),
		})
		return
	}

	c.JSON(constants.StatusOK, result)
} 