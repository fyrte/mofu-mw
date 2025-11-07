package timeout

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/fyrna/mofu"
)

type Config struct {
	Duration time.Duration // default: 5s
	Code     int           // default: 408 Request Timeout
	Message  string        // default: "Request timeout"
}

func Sparkle(config ...Config) mofu.Middleware {
	cfg := Config{
		Duration: 5 * time.Second,
		Code:     http.StatusRequestTimeout,
		Message:  "Request timeout",
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	return mofu.MwHug(func(c *mofu.C) error {
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Duration)
		defer cancel()

		// Replace request context
		c.Request = c.Request.WithContext(ctx)

		// Guard to prevent double response
		var once sync.Once
		var timeoutErr error

		// AfterFunc: run once after timeout
		context.AfterFunc(ctx, func() {
			once.Do(func() {
				// Check if response already started
				rc := http.NewResponseController(c.Writer)
				if !isHeaderWritten(rc) {
					timeoutErr = c.String(cfg.Code, cfg.Message)
					c.Abort()
				}
			})
		})

		// Run handler
		err := c.Next()

		// Cancel early if handler finished
		cancel()

		// If timeout already fired, return timeout error
		if timeoutErr != nil {
			return timeoutErr
		}

		return err
	})
}

func isHeaderWritten(rc *http.ResponseController) bool {
	// Trick: try to set a dummy header
	// If it fails, headers already sent
	return rc.SetWriteDeadline(time.Now().Add(0)) != nil
}
