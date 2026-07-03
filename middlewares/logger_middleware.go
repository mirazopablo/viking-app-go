package middlewares

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

// bodyLogWriter wraps the standard gin.ResponseWriter to capture response payloads.
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w bodyLogWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// CustomLoggerMiddleware filters HTTP access logs.
// It prints a clean summary for 2XX responses and comprehensive diagnostic details for non-2XX responses.
func CustomLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		// Wrap response writer to intercept error payloads
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Process request
		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if rawQuery != "" {
			path = path + "?" + rawQuery
		}

		// Rule 1: For 2XX responses, log only the minimal Gin standard info
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			log.Printf("[GIN] %s | %3d | %v | %s | %-7s %q",
				time.Now().Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				method,
				path,
			)
			return
		}

		// Rule 2: For non-2XX responses, provide comprehensive diagnostic info at a glance
		errorDetails := c.Errors.ByType(gin.ErrorTypeAny).String()
		if errorDetails == "" {
			errorDetails = fmt.Sprintf("HTTP %d", statusCode)
		}

		respBody := blw.body.String()
		if len(respBody) > 500 {
			respBody = respBody[:500] + "... [truncated]"
		}

		userAgent := c.Request.UserAgent()
		if userAgent == "" {
			userAgent = "Unknown-Client"
		}

		log.Printf("[ERROR] %s | %3d | %v | %s | %-7s %q | Client: %s | Details: %s | Response: %s",
			time.Now().Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			userAgent,
			errorDetails,
			respBody,
		)
	}
}

// CustomRecoveryMiddleware intercepts runtime panics, logs the backlog/stack trace,
// and returns a clean 500 error without crashing the API server.
func CustomRecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())

				// Log the panic error and exact backlog
				log.Printf("[PANIC RECOVERY] %s | Error: %v\nBacklog Stack:\n%s",
					time.Now().Format("2006/01/02 - 15:04:05"),
					err,
					stack,
				)

				// Abort gracefully without breaking the API process
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal Server Error - Execution Aborted Gracefully",
				})
			}
		}()
		c.Next()
	}
}
