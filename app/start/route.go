package start

import (
	"bytes"
	"io"
	"log"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func GetGinEngine() *gin.Engine {
	engine := gin.New()

	// Add middlewares
	engine.Use(gin.Recovery())
	engine.Use(ProxyApiMiddleware())
	engine.Use(LoggingMiddleware())
	return engine
}

func ProxyApiMiddleware() gin.HandlerFunc {
	targetURL, _ := url.Parse("http://127.0.0.1:11235")
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next() // 非 /api/ 请求，继续后续处理
			return
		}
		// 仅转发以 /api/ 开头的请求
		request := c.Request.Clone(c.Request.Context())
		request.Header.Set("X-Forwarded-Proto", "http")
		request.Header.Set("X-Forwarded-Host", c.Request.Host)
		if request.Body != nil || request.ContentLength == 0 {
			if bodyBytes, err := io.ReadAll(request.Body); err == nil {
				request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				request.ContentLength = int64(len(bodyBytes))
				request.GetBody = func() (io.ReadCloser, error) {
					return io.NopCloser(bytes.NewReader(bodyBytes)), nil
				}
			}
		}
		// 交给反向代理处理并中止后续中间件/路由
		proxy.ServeHTTP(c.Writer, request)
		c.Abort()
	}
}

// LoggingMiddleware is a Gin middleware that logs request details
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Log request details
		log.Printf("[GIN] %s | %s | %s | %d | %s",
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			c.Writer.Status(),
			latency,
		)
	}
}
