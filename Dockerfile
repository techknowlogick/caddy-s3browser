# build stage
FROM golang:1.14-alpine as builder

RUN apk add --no-cache git gcc musl-dev wget

# process wrapper
RUN go get -v github.com/abiosoft/parent

COPY . /tmp/caddy-s3browser

RUN mkdir -p /go/src/github.com/caddyserver/caddy && \
    cd /go/src/github.com/caddyserver/caddy && \
    wget https://raw.githubusercontent.com/caddyserver/caddy/v2.0.0/cmd/caddy/main.go && \
    sed -i '/plug in Caddy modules here/a _ "github.com/techknowlogick/caddy-s3browser"' main.go && \
    go mod init caddy && \
    go get github.com/caddyserver/caddy/v2@v2.0.0 && \
    go mod edit -replace github.com/techknowlogick/caddy-s3browser=/tmp/caddy-s3browser && \
    go install -v . && \
    /go/bin/caddy version && \
    mkdir -p /install && \
    cp /go/bin/caddy /install/caddy
# last copy command is for backwards compatibility

FROM alpine:3.11
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
    S3_SIGNED_URL_REDIRECT=false

COPY --from=builder /install/caddy /usr/sbin/caddy

COPY Caddyfile.tmpl /etc/caddy/Caddyfile.tmpl

CMD /bin/sh -c "envsubst < /etc/caddy/Caddyfile.tmpl > /etc/caddy/Caddyfile && /usr/sbin/caddy run -conf /etc/caddy/Caddyfile"
