# Caddy s3browser

```
s3browser {
	key ADDKEYHERE
	secret ADDSECRETHERE
	bucket ADDBUCKETHERE
	endpoint s3.amazonaws.com
}
```

This will provide directory listing for an S3 bucket (you are able to use minio, or other S3 providers). To serve files via Caddy as well you'll need to use the `proxy` directive as well.

## Prior Art
* This is based on the Browse plugin that is built into Caddy
* The template is based on the browse template from Webhippie
* s3server from jessfraz
* s3 browser by Nolan Lawson
