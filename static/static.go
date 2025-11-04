package static

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/fyrna/mofu"
)

// Config holds the configuration for static file serving middleware.
type Config struct {
	// Index specifies the default file to serve when a directory is requested.
	// Default: "index.html"
	Index string

	// Prefix defines the URL path prefix that should be stripped before
	// looking up files in the filesystem.
	// Example: If Prefix is "/static", then a request to "/static/css/style.css"
	// will look for "css/style.css" in the filesystem.
	// Default: "" (no prefix)
	Prefix string

	// CacheAge sets the max-age value in seconds for the Cache-Control header.
	// This controls how long browsers should cache static files.
	// Default: 3600 (1 hour)
	CacheAge int

	// SPA enables Single Page Application mode. When true, any missing files
	// will fall back to serving the index file, enabling client-side routing.
	// Default: false
	SPA bool

	// Root specifies the root directory within the filesystem where static
	// files are located. This is joined with the request path to form the
	// complete file path.
	// Default: "." (current directory)
	Root string

	// FS specifies an optional fs.FS filesystem to use instead of the OS
	// filesystem. When nil, os.DirFS is used with the provided root.
	// Default: nil
	FS fs.FS
}

func Sparkle(root string, opts ...ConfigOption) mofu.Middleware {
	cfg := &Config{
		Index:    "index.html",
		Prefix:   "",
		CacheAge: 3600,
		Root:     ".",
	}

	for _, opt := range opts {
		opt(cfg)
	}

	var rootFS fs.FS

	if cfg.FS != nil {
		rootFS = cfg.FS
	} else {
		rootFS = os.DirFS(root)
	}

	return mofu.MwHug(func(c *mofu.C) error {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			return c.Next()
		}

		urlPath := c.Request.URL.Path

		if cfg.Prefix != "" {
			if !strings.HasPrefix(urlPath, cfg.Prefix) {
				return c.Next()
			}

			urlPath = strings.TrimPrefix(urlPath, cfg.Prefix)
			urlPath = path.Clean(urlPath)

			if urlPath == "." {
				urlPath = "/"
			}
		}

		urlPath = path.Clean("/" + urlPath)
		if urlPath == "/" {
			urlPath = "."
		}

		filePath := path.Join(cfg.Root, urlPath)
		file, err := rootFS.Open(filePath)
		if err != nil {
			if cfg.SPA && urlPath != "." {
				file, err = rootFS.Open(path.Join(cfg.Root, cfg.Index))
				if err != nil {
					return c.Next()
				}
				defer file.Close()
				return serveFile(c, file, cfg.Index, cfg.CacheAge)
			}
			return c.Next()
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return c.Next()
		}

		if stat.IsDir() {
			indexPath := path.Join(filePath, cfg.Index)
			indexFile, err := rootFS.Open(indexPath)
			if err != nil {
				if cfg.SPA {
					indexFile, err = rootFS.Open(path.Join(cfg.Root, cfg.Index))
					if err != nil {
						return c.Next()
					}
					defer indexFile.Close()
					return serveFile(c, indexFile, cfg.Index, cfg.CacheAge)
				}
				return c.Next()
			}
			defer indexFile.Close()
			return serveFile(c, indexFile, cfg.Index, cfg.CacheAge)
		}

		return serveFile(c, file, stat.Name(), cfg.CacheAge)
	})
}

func serveFile(c *mofu.C, file fs.File, name string, maxAge int) error {
	stat, err := file.Stat()
	if err != nil {
		return c.Next()
	}

	c.SetHeader("Content-Type", detectContentType(name))
	c.SetHeader("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))
	c.SetHeader("X-Content-Type-Options", "nosniff")

	http.ServeContent(c.Writer, c.Request, name, stat.ModTime(), file.(io.ReadSeeker))
	c.Abort()
	return nil
}

var contentTypes = map[string]string{
	".css":  "text/css; charset=utf-8",
	".js":   "application/javascript; charset=utf-8",
	".html": "text/html; charset=utf-8",
	".json": "application/json; charset=utf-8",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".svg":  "image/svg+xml",
}

func detectContentType(name string) string {
	ext := strings.ToLower(path.Ext(name))
	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
