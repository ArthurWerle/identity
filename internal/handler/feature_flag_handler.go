package handler

import (
	"identity/internal/service"
	"identity/internal/service/dto"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FeatureFlagHandler handles HTTP requests for feature flags
type FeatureFlagHandler struct {
	featureFlagService service.FeatureFlagService
	logger             *slog.Logger
}

// NewFeatureFlagHandler creates a new feature flag handler
func NewFeatureFlagHandler(featureFlagService service.FeatureFlagService, logger *slog.Logger) *FeatureFlagHandler {
	return &FeatureFlagHandler{
		featureFlagService: featureFlagService,
		logger:             logger,
	}
}

// CreateFeatureFlag godoc
// @Summary Create a new feature flag
// @Description Create a new feature flag with the provided information
// @Tags feature-flags
// @Accept json
// @Produce json
// @Param feature_flag body dto.CreateFeatureFlagRequest true "Feature flag information"
// @Success 201 {object} dto.FeatureFlagResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/feature-flags [post]
func (h *FeatureFlagHandler) CreateFeatureFlag(c *gin.Context) {
	var req dto.CreateFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	flag, err := h.featureFlagService.CreateFeatureFlag(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create feature flag", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "creation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, flag)
}

// GetFeatureFlag godoc
// @Summary Get a feature flag by ID
// @Description Get feature flag details by feature flag ID
// @Tags feature-flags
// @Accept json
// @Produce json
// @Param id path int true "Feature Flag ID"
// @Success 200 {object} dto.FeatureFlagResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/feature-flags/{id} [get]
func (h *FeatureFlagHandler) GetFeatureFlag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid feature flag ID",
		})
		return
	}

	flag, err := h.featureFlagService.GetFeatureFlag(c.Request.Context(), uint(id))
	if err != nil {
		if err.Error() == "feature flag not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to get feature flag", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, flag)
}

// GetFeatureFlags godoc
// @Summary Get all feature flags
// @Description Get a paginated list of all feature flags
// @Tags feature-flags
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} dto.FeatureFlagListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/feature-flags [get]
func (h *FeatureFlagHandler) GetFeatureFlags(c *gin.Context) {
	var pagination dto.PaginationParams
	if err := c.ShouldBindQuery(&pagination); err != nil {
		h.logger.Error("invalid query parameters", "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_query",
			Message: err.Error(),
		})
		return
	}

	// Set defaults
	if pagination.Page == 0 {
		pagination.Page = 1
	}
	if pagination.PageSize == 0 {
		pagination.PageSize = 10
	}

	flags, err := h.featureFlagService.GetFeatureFlags(c.Request.Context(), &pagination)
	if err != nil {
		h.logger.Error("failed to get feature flags", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, flags)
}

// UpdateFeatureFlag godoc
// @Summary Update a feature flag
// @Description Update feature flag information by feature flag ID
// @Tags feature-flags
// @Accept json
// @Produce json
// @Param id path int true "Feature Flag ID"
// @Param feature_flag body dto.UpdateFeatureFlagRequest true "Feature flag information to update"
// @Success 200 {object} dto.FeatureFlagResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/feature-flags/{id} [put]
func (h *FeatureFlagHandler) UpdateFeatureFlag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid feature flag ID",
		})
		return
	}

	var req dto.UpdateFeatureFlagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	flag, err := h.featureFlagService.UpdateFeatureFlag(c.Request.Context(), uint(id), &req)
	if err != nil {
		if err.Error() == "feature flag not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to update feature flag", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, flag)
}

// DeleteFeatureFlag godoc
// @Summary Delete a feature flag
// @Description Soft delete a feature flag by feature flag ID
// @Tags feature-flags
// @Accept json
// @Produce json
// @Param id path int true "Feature Flag ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/feature-flags/{id} [delete]
func (h *FeatureFlagHandler) DeleteFeatureFlag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid feature flag ID",
		})
		return
	}

	err = h.featureFlagService.DeleteFeatureFlag(c.Request.Context(), uint(id))
	if err != nil {
		if err.Error() == "feature flag not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to delete feature flag", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "deletion_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Feature flag deleted successfully",
	})
}
