# Community Service 独立镜像
FROM golang:1.25-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/community ./services/community

FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/community ./community
COPY --from=builder /build/services/community/config.yaml ./services/community/config.yaml
COPY --from=builder /build/config/config.docker.yaml ./config/config.yaml
COPY --from=builder /build/pkg/resource ./pkg/resource/

ENTRYPOINT ["./community"]
