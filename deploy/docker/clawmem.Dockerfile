FROM golang:1.26-alpine AS build
ENV CLAWMEM_ADDR=0.0.0.0:8088
WORKDIR /src
RUN apk add --no-cache ca-certificates git

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

COPY go.mod ./
RUN go mod download

COPY cmd/ ./cmd/
COPY configs/ ./configs/
COPY internal/ ./internal/

RUN go build -ldflags "-X 'clawmem/internal/version.Version=${VERSION}' -X 'clawmem/internal/version.Commit=${COMMIT}' -X 'clawmem/internal/version.Date=${BUILD_DATE}'" -o /out/clawmem ./cmd/clawmem

FROM alpine:3.21

ARG OCI_SOURCE="https://github.com/clawbot-platform/clawmem"
ARG OCI_DESCRIPTION="Reusable Go-first memory, replay, and historical context service."
ARG OCI_LICENSES="Apache-2.0"

LABEL org.opencontainers.image.source="${OCI_SOURCE}" \
      org.opencontainers.image.description="${OCI_DESCRIPTION}" \
      org.opencontainers.image.licenses="${OCI_LICENSES}"

RUN apk add --no-cache ca-certificates wget && adduser -D -u 10001 clawmem

COPY --from=build /out/clawmem /usr/local/bin/clawmem
COPY --from=build /src/configs /app/configs

RUN mkdir -p /app/var /data/clawmem && chown -R clawmem:clawmem /app /data

WORKDIR /app
USER clawmem

EXPOSE 8088
ENTRYPOINT ["/usr/local/bin/clawmem"]
