package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/fyrna/mofu"
	"github.com/fyrna/x/color"
)

type statusRecorder struct {
	http.ResponseWriter
	status      int
	size        int
	body        *bytes.Buffer
	captureBody bool
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size

	if r.captureBody && r.body != nil && size > 0 {
		r.body.Write(b)
	}
	return size, err
}

type Config struct {
	// EnableColor enables colored output in logs to improve readability
	// Default: true
	EnableColor bool

	// LogRequestBody enables logging of HTTP request bodies
	// If enabled, the request body will be shown in the logs (limited by MaxBodySize)
	// Default: false
	LogRequestBody bool

	// LogResponseBody enables logging of HTTP response bodies
	// If enabled, the response body will be shown in the logs (limited by MaxBodySize)
	// Default: false
	LogResponseBody bool

	// MaxBodySize specifies the maximum size of the request/response body to be logged
	// Bodies exceeding this size will be truncated
	// Default: 1024 (1KB)
	MaxBodySize int

	// SkipPaths contains a list of paths that will be excluded from logging
	// Useful for endpoints like health checks or metrics that don’t need to be logged
	// Default: []string{"/health", "/metrics"}
	SkipPaths []string

	// EnableIP enables logging of the client’s IP address
	// If enabled, the client’s IP will be shown in the logs
	// Default: true
	EnableIP bool

	// EnableUserAgent enables logging of the User-Agent header
	// If enabled, the client’s User-Agent will be shown in the logs
	// Default: true
	EnableUserAgent bool
}

func Sparkle(opts ...ConfigOption) mofu.Middleware {
	cfg := &Config{
		EnableColor:     true,
		LogRequestBody:  false,
		LogResponseBody: false,
		MaxBodySize:     1024, // 1KB
		SkipPaths:       []string{"/health", "/metrics"},
		EnableIP:        true,
		EnableUserAgent: true,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.MaxBodySize < 0 {
		cfg.MaxBodySize = 1024
	}

	return mofu.MwHug(func(c *mofu.C) error {
		// Skip logging for certain paths
		if slices.Contains(cfg.SkipPaths, c.Request.URL.Path) {
			return c.Next()
		}

		start := time.Now()

		// Setup response recorder
		recorder := &statusRecorder{
			ResponseWriter: c.Writer,
			status:         200,
		}

		// Capture request body if enabled
		var requestBody []byte
		if cfg.LogRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			if len(requestBody) > cfg.MaxBodySize {
				requestBody = requestBody[:cfg.MaxBodySize]
			}
		}

		// Capture response body if enabled
		if cfg.LogResponseBody {
			recorder.captureBody = true
			recorder.body = &bytes.Buffer{}
		}

		c.Writer = recorder

		// Process request
		err := c.Next()
		dur := time.Since(start)

		// Format log entry
		logEntry := formatLogEntry(c, recorder, dur, requestBody, recorder.body, cfg)

		fmt.Print(logEntry)
		return err
	})
}

func formatLogEntry(c *mofu.C, recorder *statusRecorder, dur time.Duration, reqBody []byte, respBody *bytes.Buffer, config *Config) string {
	estSize := 200
	if config.EnableIP {
		estSize += 50
	}
	if config.EnableUserAgent {
		estSize += 100
	}

	var sb strings.Builder
	sb.Grow(estSize)

	// Timestamp
	sb.WriteString(fmt.Sprintf("[%s] ", time.Now().Format("2006-01-02 15:04:05")))

	// Status code with color
	if config.EnableColor {
		sb.WriteString(fmt.Sprintf("%s%3d%s ",
			getStatusColor(recorder.status), recorder.status, color.Reset))
	} else {
		sb.WriteString(fmt.Sprintf("%3d ", recorder.status))
	}

	// Method
	method := fmt.Sprintf("%-7s", c.Request.Method)
	if config.EnableColor {
		sb.WriteString(fmt.Sprintf("%s%s%s ", color.Magenta, method, color.Reset))
	} else {
		sb.WriteString(fmt.Sprintf("%s ", method))
	}

	// Path
	sb.WriteString(c.Request.URL.Path)

	// Query parameters
	if c.Request.URL.RawQuery != "" {
		sb.WriteString("?" + c.Request.URL.RawQuery)
	}

	// Client IP
	if config.EnableIP {
		sb.WriteString(fmt.Sprintf(" | %s", getClientIP(c.Request)))
	}

	// User Agent
	if config.EnableUserAgent {
		if ua := c.Request.UserAgent(); ua != "" {
			// Shorten long user agents
			if len(ua) > 50 {
				ua = ua[:47] + "..."
			}
			sb.WriteString(fmt.Sprintf(" | %s", ua))
		}
	}

	// Duration with color based on performance
	durationStr := formatDuration(dur)
	if config.EnableColor {
		sb.WriteString(fmt.Sprintf(" | %s%s%s",
			getDurationColor(dur), durationStr, color.Reset))
	} else {
		sb.WriteString(fmt.Sprintf(" | %s", durationStr))
	}

	// Response size
	sb.WriteString(fmt.Sprintf(" | %dB", recorder.size))

	// Request body (if enabled)
	if config.LogRequestBody && len(reqBody) > 0 {
		bodyStr := string(reqBody)
		if isJSON(bodyStr) {
			sb.WriteString(" | req:")
			sb.WriteString(truncate(string(reqBody), 100))
		} else {
			sb.WriteString(fmt.Sprintf(" | req:%q", truncate(string(reqBody), 100)))
		}
	}

	// Response body (if enabled)
	if config.LogResponseBody && respBody != nil && respBody.Len() > 0 {
		body := respBody.Bytes()
		if len(body) > config.MaxBodySize {
			body = body[:config.MaxBodySize]
		}

		bodyStr := string(body)
		if isJSON(bodyStr) {
			sb.WriteString(" | resp:")
			sb.WriteString(truncate(bodyStr, 100))
		} else {
			sb.WriteString(fmt.Sprintf(" | resp:%q", truncate(bodyStr, 100)))
		}
	}

	sb.WriteString(" nyaa~\n")
	return sb.String()
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Microsecond:
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case d < time.Millisecond:
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	case d < time.Second:
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

func getClientIP(r *http.Request) string {
	// Check forwarded headers first
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func getStatusColor(status int) string {
	switch {
	case status < 200:
		return color.Cyan
	case status < 300:
		return color.Green
	case status < 400:
		return color.Yellow
	case status < 500:
		return color.Magenta
	default:
		return color.Red
	}
}

func getDurationColor(d time.Duration) string {
	switch {
	case d < 100*time.Millisecond:
		return color.Green
	case d < 500*time.Millisecond:
		return color.Yellow
	default:
		return color.Red
	}
}

func isJSON(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func truncate(s string, length int) string {
	if len(s) > length {
		return s[:length] + "..."
	}
	return s
}
