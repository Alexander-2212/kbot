# syntax=docker/dockerfile:1
# Мультиархітектурна збірка kbot: бінарник крос-компілюється під $TARGETOS/$TARGETARCH,
# фінальний образ - scratch із самим бінарником і CA-сертифікатами.
ARG GO_VERSION=1.26

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# Без значень за замовчуванням: BuildKit підставляє їх із --platform,
# а явний --build-arg (Makefile, GitHub Actions) має пріоритет.
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN apk add --no-cache ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
      -ldflags "-s -w -X github.com/Alexander-2212/kbot/cmd.appVersion=${VERSION}" \
      -o /out/kbot .

FROM scratch

ARG VERSION=dev

# OCI-мітки: source лінкує пакет у ghcr.io з цим репозиторієм
LABEL org.opencontainers.image.source="https://github.com/Alexander-2212/kbot" \
      org.opencontainers.image.url="https://github.com/Alexander-2212/kbot" \
      org.opencontainers.image.title="kbot" \
      org.opencontainers.image.description="Telegram bot written in Go (GlobalLogic DEVOPS101)" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/kbot /kbot

USER 65532:65532
ENTRYPOINT ["/kbot"]
CMD ["start"]
