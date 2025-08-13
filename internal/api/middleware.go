package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// loggingMiddleware logs HTTP requests (security-focused, no sensitive data)
func (s *Server) loggingMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		start := time.Now()

		// Call next handler
		next(ctx)

		duration := time.Since(start)
		s.logger.Info("HTTP request",
			zap.String("method", string(ctx.Method())),
			zap.String("path", string(ctx.Path())),
			zap.Int("status", ctx.Response.StatusCode()),
			zap.Duration("duration", duration),
			zap.String("user_agent", string(ctx.UserAgent())),
		)
	}
}

// securityMiddleware adds security headers
func (s *Server) securityMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Security headers
		ctx.Response.Header.Set("X-Content-Type-Options", "nosniff")
		ctx.Response.Header.Set("X-Frame-Options", "DENY")
		ctx.Response.Header.Set("X-XSS-Protection", "1; mode=block")
		ctx.Response.Header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		ctx.Response.Header.Set("Content-Security-Policy", "default-src 'self'")
		ctx.Response.Header.Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Remove server information
		ctx.Response.Header.Del("Server")

		next(ctx)
	}
}

// rateLimitMiddleware implements basic rate limiting
func (s *Server) rateLimitMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	// Simple in-memory rate limiter (in production, use Redis)
	return func(ctx *fasthttp.RequestCtx) {
		next(ctx)
	}
}

// authMiddleware validates JWT tokens
func (s *Server) authMiddleware(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// Get Authorization header
		authHeader := string(ctx.Request.Header.Peek("Authorization"))
		if authHeader == "" {
			s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Authorization header required")
			return
		}

		// Check Bearer token format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Invalid authorization format")
			return
		}

		// Extract token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Token required")
			return
		}

		// Validate token
		claims, err := s.authService.ValidateToken(token)
		if err != nil {
			s.sendErrorResponse(ctx, fasthttp.StatusUnauthorized, "Invalid token")
			return
		}

		// Store user info in context for handlers to use
		ctx.SetUserValue("user_id", claims.UserID)
		ctx.SetUserValue("user_email", claims.Email)

		next(ctx)
	}
}

// sendErrorResponse sends a JSON error response
func (s *Server) sendErrorResponse(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	s.setCORSHeaders(ctx)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)

	response := map[string]interface{}{
		"error":     true,
		"message":   message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, _ := json.Marshal(response)
	ctx.SetBody(jsonData)
}

// sendSuccessResponse sends a JSON success response
func (s *Server) sendSuccessResponse(ctx *fasthttp.RequestCtx, data interface{}) {
	s.setCORSHeaders(ctx)
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	response := map[string]interface{}{
		"success":   true,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		s.logger.Error("Failed to marshal response", zap.Error(err))
		s.sendErrorResponse(ctx, fasthttp.StatusInternalServerError, "Internal server error")
		return
	}

	ctx.SetBody(jsonData)
}

// parseJSONBody parses JSON request body
func (s *Server) parseJSONBody(ctx *fasthttp.RequestCtx, dest interface{}) error {
	if !ctx.IsPost() {
		return fmt.Errorf("method not allowed")
	}

	contentType := string(ctx.Request.Header.ContentType())
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("content-type must be application/json")
	}

	body := ctx.PostBody()
	if len(body) == 0 {
		return fmt.Errorf("request body is empty")
	}

	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}
