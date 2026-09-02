# syntax=docker/dockerfile:1
# Мультиархітектурна збірка kbot: бінарник крос-компілюється під $TARGETOS/$TARGETARCH,
# фінальний образ - scratch із самим бінарником і CA-сертифікатами.
ARG GO_VERSION=1.26

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64
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

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /out/kbot /kbot

USER 65532:65532
ENTRYPOINT ["/kbot"]
CMD ["start"]
