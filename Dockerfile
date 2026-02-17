FROM golang:1.22-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o edgeclaw ./cmd/edgeclaw

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /build/edgeclaw /usr/local/bin/edgeclaw
COPY workspace/ /etc/edgeclaw/workspace/
COPY configs/config.example.json /etc/edgeclaw/config.json

ENTRYPOINT ["edgeclaw"]
CMD ["-config", "/etc/edgeclaw/config.json"]
