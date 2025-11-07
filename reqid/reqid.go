package reqid

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/fyrna/mofu"
)

// RequestIDConfig optionally defines header name & generator.
type Config struct {
	Header  string // default "X-Request-ID"
	GenFunc func() string
}

func Sparkle(config ...Config) mofu.Middleware {
	cfg := Config{
		Header:  "X-Request-ID",
		GenFunc: genID,
	}

	if len(config) > 0 {
		cfg = config[0]
	}

	return mofu.MwHug(func(c *mofu.C) error {
		id := c.GetHeader(cfg.Header)
		if id == "" {
			id = cfg.GenFunc()
		}

		c.SetHeader(cfg.Header, id)
		c.Set("request_id", id)
		return c.Next()
	})
}

// genID 16 byte random → 32 char hex
func genID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
