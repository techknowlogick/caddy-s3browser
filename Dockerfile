# build stage
FROM abiosoft/caddy:builder as builder

# process wrapper
RUN go get -v github.com/abiosoft/parent

RUN VERSION="1.0.0" PLUGINS="s3browser" ENABLE_TELEMETRY=false /bin/sh /usr/bin/builder.sh

FROM alpine:3.9
EXPOSE 80

RUN apk add --no-cache wget mailcap ca-certificates gettext libintl && \
    mkdir /etc/caddy

ENV S3_ENDPOINT=s3.amazonaws.com \
    S3_PROTO=https \
    S3_SECURE=true \
    S3_REFRESH=5m \
    S3_DEBUG=false

COPY --from=build-env /go/bin/caddy /usr/sbin/caddy

COPY --from=build-env /go/src/github.com/techknowlogick/caddy-s3browser/Caddyfile.tmpl /etc/caddy/Caddyfile.tmpl

CMD /bin/sh -c "envsubst < /etc/caddy/Caddyfile.tmpl > /etc/caddy/Caddyfile && /usr/sbin/caddy -conf /etc/caddy/Caddyfile"
