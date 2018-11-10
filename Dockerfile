# build stage
FROM golang:alpine AS build-env

RUN apk add --no-cache git 
RUN go get -d -v github.com/mholt/caddy/caddy github.com/techknowlogick/caddy-s3browser
WORKDIR /go/src/github.com/mholt/caddy/caddy

RUN sed -i '/This is where other plugins get plugged in (imported)/a _ "github.com/techknowlogick/caddy-s3browser"' caddymain/run.go \
 && sed -i '/"cors",/a "s3browser", ' ../caddyhttp/httpserver/plugin.go \
 && go install -v . \
 && /go/bin/caddy -version

FROM alpine:edge
EXPOSE 80

RUN apk add --no-cache wget mailcap ca-certificates gettext libintl && \
    mkdir /etc/caddy

COPY --from=build-env /go/bin/caddy /usr/sbin/caddy

CMD ["/usr/sbin/caddy", "-conf", "/etc/caddy/Caddyfile"]
