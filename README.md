# Caddy s3browser

```
s3browser {
	key ADDKEYHERE
	secret ADDSECRETHERE
	bucket ADDBUCKETHERE
	endpoint nyc3.digitaloceanspaces.com
}
proxy / https://examplebucket.nyc3.digitaloceanspaces.com {
	header_upstream Host examplebucket.nyc3.digitaloceanspaces.com
}
```

This will provide directory listing for an S3 bucket (you are able to use minio, or other S3 providers). To serve files via Caddy as well you'll need to use the `proxy` directive as well. The server must be able to have public access to the files in the bucket.

## Prior Art
* This is based on the Browse plugin that is built into Caddy
* The template is based on the browse template from Webhippie
* s3server from jessfraz
* s3 browser by Nolan Lawson
