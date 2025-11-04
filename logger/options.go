package logger

type ConfigOption func(*Config)

func SetEnableColor(e bool) ConfigOption {
	return func(c *Config) { c.EnableColor = e }
}

func SetLogRequestBody(e bool) ConfigOption {
	return func(c *Config) { c.LogRequestBody = e }
}
func SetLogResponseBody(e bool) ConfigOption {
	return func(c *Config) { c.LogResponseBody = e }
}

func SetMaxBodySize(s int) ConfigOption {
	return func(c *Config) { c.MaxBodySize = s }
}

func SetSkipPaths(paths ...string) ConfigOption {
	return func(c *Config) { c.SkipPaths = paths }
}

func SetEnableIP(e bool) ConfigOption {
	return func(c *Config) { c.EnableIP = e }
}

func SetEnableUserAgent(e bool) ConfigOption {
	return func(c *Config) { c.EnableUserAgent = e }
}
