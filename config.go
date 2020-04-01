package s3browser

import (
	"strconv"
	"time"

	"github.com/caddyserver/caddy"
)

type Config struct {
	Endpoint string
	Region   string
	Key      string
	Secret   string
	Secure   bool
	Bucket   string
	Refresh  time.Duration
	Debug    bool
}

func ParseConfig(c *caddy.Controller) (cfg Config, err error) {
	c.NextArg() // skip block beginning: "s3browser"

	cfg = Config{
		Secure:  true,
		Debug:   false,
		Refresh: 5 * time.Minute,
	}
	for c.NextBlock() {
		var err error
		switch c.Val() {
		case "key":
			cfg.Key, err = StringArg(c)
		case "secret":
			cfg.Secret, err = StringArg(c)
		case "endpoint":
			cfg.Endpoint, err = StringArg(c)
		case "region":
			cfg.Region, err = StringArg(c)
		case "bucket":
			cfg.Bucket, err = StringArg(c)
		case "secure":
			cfg.Secure, err = BoolArg(c)
		case "refresh":
			cfg.Refresh, err = DurationArg(c)
		case "debug":
			cfg.Debug, err = BoolArg(c)
		default:
			err = c.Errf("Unknown s3browser arg: %s", c.Val())
		}
		if err != nil {
			return cfg, c.Errf("Error parsing %s: %s", c.Val(), err)
		}
	}
	return cfg, nil
}

// Assert only one arg and return it
func StringArg(c *caddy.Controller) (string, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return "", c.ArgErr()
	}
	return args[0], nil
}

func DurationArg(c *caddy.Controller) (time.Duration, error) {
	str, err := StringArg(c)
	if err != nil {
		return 0 * time.Second, err
	}
	return time.ParseDuration(str)
}

func BoolArg(c *caddy.Controller) (bool, error) {
	args := c.RemainingArgs()
	if len(args) != 1 {
		return true, c.ArgErr()
	}
	return strconv.ParseBool(args[0])
}
