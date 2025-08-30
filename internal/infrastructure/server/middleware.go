// internal/infrastructure/server/middleware.go
package server

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"

	"github.com/taskmaster/core/internal/application/services"
	"github.com/taskmaster/core/internal/domain/entities"
	"github.com/taskmaster/core/internal/ports"
)

// authMiddleware validates JWT tokens
func (s *Server) authMiddleware(authService *services.AuthService) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				s.logger.LogSecurityEvent("invalid_token", "", c.RealIP(), map[string]interface{}{
					"error": err.Error(),
				})
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			// Set user claims in context
			c.Set("user", claims.UserID)
			c.Set("user_role", claims.Role)
			c.Set("user_email", claims.Email)

			return next(c)
		}
	}
}

// requireRole checks if user has required role
func (s *Server) requireRole(roles ...entities.UserRole) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, ok := c.Get("user_role").(entities.UserRole)
			if !ok {
				return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get user role from context")
			}

			userID := c.Get("user").(string)

			// Check if user has required role
			for _, requiredRole := range roles {
				if userRole == requiredRole {
					return next(c)
				}
			}

			s.logger.LogSecurityEvent("insufficient_permissions", 
				userID, 
				c.RealIP(), 
				map[string]interface{}{
					"required_roles": roles,
					"user_role": userRole,
					"endpoint": c.Request().URL.Path,
				})

			return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
		}
	}
}

// getUserIDFromContext extracts user ID from context with proper error handling
func getUserIDFromContext(c echo.Context) uuid.UUID {
	userIDStr, ok := c.Get("user").(string)
	if !ok {
		return uuid.Nil
	}
	
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil
	}
	
	return userID
}

// getUserRoleFromContext extracts user role from context
func getUserRoleFromContext(c echo.Context) entities.UserRole {
	role, ok := c.Get("user_role").(entities.UserRole)
	if !ok {
		return entities.UserRoleViewer // Default role
	}
	
	return role
}

// getUserEmailFromContext extracts user email from context
func getUserEmailFromContext(c echo.Context) string {
	email, ok := c.Get("user_email").(string)
	if !ok {
		return ""
	}
	
	return email
}
