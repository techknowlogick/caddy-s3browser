# Caddy s3browser

This will provide directory listing for an S3 bucket (you are able to use minio, or other S3 providers).

Note: For performance reasons, the file listing is fetched once every 5 minutes to reduce load on S3. You can force a refresh by sending a POST request to the plugin.

## Configuration

See `Caddyfile.tmpl` for a template.

|  option   |  type  |  default   | help |
|-----------|:------:|------------|------|
| site_name | string | S3 Browser | Site display name |
| endpoint  | string |            | S3 hostname |
| region    | string |   empty    | S3 region (optional) |
| key       | string |            | S3 access key |
| secret    | string |            | S3 secret key |
| secure    |  bool  |   `true`   | Use TLS when connection to S3 |
| bucket    | string |            | S3 bucket |
| refresh   | string |    `5m`    | Time between periodic refresh |
| debug     |  bool  |   `false`  | Output debug information |


## Prior Art
* This is based on the [Browse plugin](https://github.com/mholt/caddy/tree/master/caddyhttp/browse) that is built into Caddy
* The template is based on the [browse template](https://github.com/dockhippie/caddy/blob/master/rootfs/etc/caddy/browse.tmpl) from Webhippie
* [s3server](https://github.com/jessfraz/s3server) from jessfraz
* [pretty-s3-index-html](https://github.com/nolanlawson/pretty-s3-index-html) by Nolan Lawson
