package middleware

import (
	"identity/internal/service/dto"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery returns a gin middleware for recovering from panics
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, dto.ErrorResponse{
					Error:   "internal_server_error",
					Message: "An unexpected error occurred",
				})
			}
		}()

		c.Next()
	}
}
