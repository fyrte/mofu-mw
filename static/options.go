package static

import "io/fs"

// ConfigOption defines a function type for configuring the static file server
// using the functional options pattern.
type ConfigOption func(*Config)

// SetIndex returns a ConfigOption that sets the default index file name.
// This file is served when a directory is requested or when using SPA mode.
func SetIndex(file string) ConfigOption {
	return func(c *Config) { c.Index = file }
}

// SetPrefix returns a ConfigOption that sets the URL prefix to strip from requests.
// The prefix is removed from the request path before file lookup.
func SetPrefix(prefix string) ConfigOption {
	return func(c *Config) { c.Prefix = prefix }
}

// SetSPAMode returns a ConfigOption that enables or disables SPA mode.
// When enabled, missing files fall back to the index file for client-side routing.
func SetSPAMode(enable bool) ConfigOption {
	return func(c *Config) { c.SPA = enable }
}

// SetRoot returns a ConfigOption that sets the root directory for static files.
// This is the base directory where file lookups start.
func SetRoot(root string) ConfigOption {
	return func(c *Config) { c.Root = root }
}

// SetFS returns a ConfigOption that sets a custom filesystem implementation.
// Use this to serve files from embedded filesystems or other sources.
func SetFS(fs fs.FS) ConfigOption {
	return func(c *Config) { c.FS = fs }
}

// SetCacheAge returns a ConfigOption that sets the cache duration in seconds.
// Controls the max-age value in the Cache-Control header for static assets.
func SetCacheAge(a int) ConfigOption {
	return func(c *Config) { c.CacheAge = a }
}
