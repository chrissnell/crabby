FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.version=${VERSION}" \
    -o /crabby ./cmd/crabby

FROM alpine:latest
RUN apk add --no-cache ca-certificates chromium \
    && addgroup -g 10001 -S crabby \
    && adduser -u 10001 -S crabby -G crabby
COPY --from=build /crabby /crabby
USER crabby
ENTRYPOINT ["/crabby"]
