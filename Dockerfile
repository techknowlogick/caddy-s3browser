# build stage
FROM abiosoft/caddy:builder as abiosoft-builder

FROM golang:1.13-alpine as builder

RUN apk add --no-cache git gcc musl-dev

COPY --from=abiosoft-builder /usr/bin/builder.sh /usr/bin/builder.sh

CMD ["/bin/sh", "/usr/bin/builder.sh"]

# process wrapper
RUN go get -v github.com/abiosoft/parent

RUN VERSION="1.0.4" PLUGINS="github.com/techknowlogick/caddy-s3browser" ENABLE_TELEMETRY=false /bin/sh /usr/bin/builder.sh

FROM alpine:3.11
EXPOSE 80

RUN apk add --no-cache wget mailcap ca-certificates gettext libintl && \
    mkdir /etc/caddy

ENV S3_ENDPOINT=s3.amazonaws.com \
    S3_PROTO=https \
    S3_SECURE=true \
    S3_REFRESH=5m \
    S3_DEBUG=false

COPY --from=builder /install/caddy /usr/sbin/caddy

COPY Caddyfile.tmpl /etc/caddy/Caddyfile.tmpl

CMD /bin/sh -c "envsubst < /etc/caddy/Caddyfile.tmpl > /etc/caddy/Caddyfile && /usr/sbin/caddy -conf /etc/caddy/Caddyfile"
