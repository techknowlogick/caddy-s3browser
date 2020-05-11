# build stage
FROM caddy:2.2.1-builder-alpine as builder

RUN apk add --no-cache git gcc musl-dev wget

COPY . /tmp/caddy-s3browser

RUN \
    xcaddy build --with github.com/techknowlogick/caddy-s3browser=/tmp/caddy-s3browser && \
    /usr/bin/caddy version && \
    /usr/bin/caddy list-modules | grep s3browser && \
    mkdir -p /install && \
    cp /usr/bin/caddy /install/caddy
# last copy command is for backwards compatibility

FROM alpine:3.12
EXPOSE 80

RUN apk add --no-cache wget mailcap ca-certificates gettext libintl && \
    mkdir /etc/caddy

ENV S3_ENDPOINT=s3.amazonaws.com \
    S3_PROTO=https \
    S3_SECURE=true \
    S3_REFRESH=5m \
    S3_REFRESH_SECRET='changeme' \
    S3_DEBUG=false \
    S3_SITENAME="S3 Browser" \
    S3_REGION="us-east-1" \
    S3_SIGNED_URL_REDIRECT=false \
    S3_SEMANTIC_SORT=false

COPY --from=builder /install/caddy /usr/sbin/caddy

COPY Caddyfile.tmpl /etc/caddy/Caddyfile.tmpl

CMD /bin/sh -c "envsubst < /etc/caddy/Caddyfile.tmpl > /etc/caddy/Caddyfile && /usr/sbin/caddy run -config /etc/caddy/Caddyfile"
