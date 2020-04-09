package s3browser

import (
	"strconv"
	"time"

	"github.com/caddyserver/caddy"
)

type Config struct {
	SiteName          string
	Endpoint          string
	Region            string
	Key               string
	Secret            string
	Secure            bool
	Bucket            string
	APISecret         string
	Refresh           time.Duration
	Debug             bool
	SignedURLRedirect bool
}

func ParseConfig(c *caddy.Controller) (cfg Config, err error) {
	c.NextArg() // skip block beginning: "s3browser"

	cfg = Config{
		SiteName: "S3 Browser",
		Secure:   true,
		Debug:    false,
		Refresh:  5 * time.Minute,
	}

	for c.NextBlock() {
		var err error
		switch c.Val() {
		case "site_name":
			cfg.SiteName, err = parseStringArg(c)
		case "key":
			cfg.Key, err = parseStringArg(c)
		case "secret":
			cfg.Secret, err = parseStringArg(c)
		case "endpoint":
			cfg.Endpoint, err = parseStringArg(c)
		case "region":
			cfg.Region, err = parseStringArg(c)
		case "bucket":
			cfg.Bucket, err = parseStringArg(c)
		case "secure":
			cfg.Secure, err = parseBoolArg(c)
		case "refresh_api_secret":
			cfg.APISecret, err = parseStringArg(c)
		case "refresh":
			cfg.Refresh, err = parseDurationArg(c)
		case "debug":
			cfg.Debug, err = parseBoolArg(c)
		case "signed_url_redirect":
			cfg.SignedURLRedirect, err = parseBoolArg(c)
		default:
			err = c.Errf("not a valid s3browser option")
		}
		if err != nil {
			return cfg, c.Errf("Error parsing %s: %s", c.Val(), err)
		}
	}

	return cfg, nil
}

func parseBoolArg(c *caddy.Controller) (bool, error) {
	if !c.NextArg() {
		return true, c.ArgErr()
	}
	return strconv.ParseBool(c.Val())
}

func parseDurationArg(c *caddy.Controller) (time.Duration, error) {
	str, err := parseStringArg(c)
	if err != nil {
		return 0 * time.Second, err
	}
	return time.ParseDuration(str)
}

func parseStringArg(c *caddy.Controller) (string, error) {
	if !c.NextArg() {
		return "", c.ArgErr()
	}
	return c.Val(), nil
}
