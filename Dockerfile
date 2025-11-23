FROM golang:1.25-alpine AS build-env

ARG ADDITIONAL_BUILD_TAGS=""

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk add --no-cache build-base

WORKDIR /maddy

COPY . ./

COPY . ./
RUN mkdir -p /pkg/data && \
    cp maddy.conf.docker /pkg/data/maddy.conf && \
    ./build.sh --builddir /tmp --destdir /pkg/ --tags "docker ${ADDITIONAL_BUILD_TAGS}" build install

FROM alpine:3.22
LABEL maintainer="@deshke"
LABEL org.opencontainers.image.source=https://github.com/foxcpp/maddy
LABEL org.opencontainers.image.fork=https://github.com/Deshke/maddy

RUN set -ex && \
    apk upgrade --no-cache --available && \
    apk --no-cache add ca-certificates
COPY --from=build-env /pkg/data/maddy.conf /data/maddy.conf
COPY --from=build-env /pkg/usr/local/bin/maddy /bin/

EXPOSE 25 143 993 587 465
VOLUME ["/data"]
ENTRYPOINT ["/bin/maddy", "-config", "/data/maddy.conf"]
CMD ["run"]
