package handler

import (
	"identity/internal/service"
	"identity/internal/service/dto"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// UserHandler handles HTTP requests for users
type UserHandler struct {
	userService service.UserService
	logger      *slog.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger,
	}
}

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user with the provided information
// @Tags users
// @Accept json
// @Produce json
// @Param user body dto.CreateUserRequest true "User information"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create user", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "creation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, user)
}

// GetUser godoc
// @Summary Get a user by ID
// @Description Get user details by user ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), uint(id))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to get user", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// GetUsers godoc
// @Summary Get all users
// @Description Get a paginated list of all users
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(10)
// @Success 200 {object} dto.UserListResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
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

	users, err := h.userService.GetUsers(c.Request.Context(), &pagination)
	if err != nil {
		h.logger.Error("failed to get users", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, users)
}

// UpdateUser godoc
// @Summary Update a user
// @Description Update user information by user ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body dto.UpdateUserRequest true "User information to update"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), uint(id), &req)
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to update user", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Soft delete a user by user ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	err = h.userService.DeleteUser(c.Request.Context(), uint(id))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to delete user", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "deletion_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "User deleted successfully",
	})
}

// GetUserFeatureFlags godoc
// @Summary Get user's feature flags
// @Description Get all feature flags assigned to a user
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {array} dto.FeatureFlagResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id}/feature-flags [get]
func (h *UserHandler) GetUserFeatureFlags(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	flags, err := h.userService.GetUserFeatureFlags(c.Request.Context(), uint(id))
	if err != nil {
		if err.Error() == "user not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to get user feature flags", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "retrieval_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, flags)
}

// AssignFeatureFlagToUser godoc
// @Summary Assign feature flag to user
// @Description Assign a feature flag to a user by feature flag key
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param key path string true "Feature Flag Key"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id}/feature-flags/{key} [post]
func (h *UserHandler) AssignFeatureFlagToUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_key",
			Message: "Feature flag key is required",
		})
		return
	}

	err = h.userService.AssignFeatureFlagToUser(c.Request.Context(), uint(id), key)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "feature flag not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to assign feature flag", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "assignment_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Feature flag assigned successfully",
	})
}

// UnassignFeatureFlagFromUser godoc
// @Summary Unassign feature flag from user
// @Description Remove a feature flag from a user by feature flag key
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param key path string true "Feature Flag Key"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/users/{id}/feature-flags/{key} [delete]
func (h *UserHandler) UnassignFeatureFlagFromUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid user ID",
		})
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid_key",
			Message: "Feature flag key is required",
		})
		return
	}

	err = h.userService.UnassignFeatureFlagFromUser(c.Request.Context(), uint(id), key)
	if err != nil {
		if err.Error() == "user not found" || err.Error() == "feature flag not found" {
			c.JSON(http.StatusNotFound, dto.ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
			return
		}
		h.logger.Error("failed to unassign feature flag", "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "unassignment_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SuccessResponse{
		Message: "Feature flag unassigned successfully",
	})
}
