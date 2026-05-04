# phase 1: build images
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 复制依赖文件，利用Docker缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制全部源码
COPY . .

# 编译为静态二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o ticket_project main.go

# phase2: run images
FROM alpine:3.20

RUN apk add --no-cache bash

WORKDIR /app

# 复制编译好的二进制文件
COPY --from=builder /app/ticket_project .

# 复制配置文件
COPY --from=builder /app/config ./config

# 开放端口（根据你的服务端口修改）
EXPOSE 8080

# 启动服务
CMD ["/app/ticket_project"]