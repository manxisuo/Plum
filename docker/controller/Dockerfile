# Plum Controller Dockerfile
# 多阶段构建，优化镜像大小

# 构建阶段
# 使用 go1.23-alpine，通过 GOTOOLCHAIN=auto 自动下载所需工具链
FROM golang:1.23-alpine AS builder

# 设置 GOTOOLCHAIN 允许自动下载所需工具链
# go.mod 可能要求更高版本，通过此设置自动下载所需工具链
ENV GOTOOLCHAIN=auto

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 复制go模块文件
COPY controller/go.mod controller/go.sum ./
COPY proto/ ./proto/

# 下载依赖
RUN go mod download

# 复制源代码
COPY controller/ ./

# 生成proto代码（如果需要）
RUN if [ -f "proto/*.proto" ]; then \
        apk add --no-cache protobuf-dev protoc-gen-go protoc-gen-go-grpc && \
        go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
        go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
        make proto; \
    fi

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o bin/controller ./cmd/server

# 运行阶段
FROM alpine:3.18

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 -S plum && \
    adduser -u 1001 -S plum -G plum

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin/controller ./bin/controller
# 复制Swagger静态资源
COPY --from=builder /app/static ./controller/static

# 复制配置文件
COPY controller/env.example ./.env

# 创建数据目录
RUN mkdir -p /app/data && \
    chown -R plum:plum /app

# 切换到非root用户
USER plum

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV CONTROLLER_ADDR=:8080
ENV CONTROLLER_DB=file:/app/data/controller.db?_pragma=busy_timeout(5000)
ENV CONTROLLER_DATA_DIR=/app/data

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/v1/health || exit 1

# 启动命令
CMD ["./bin/controller"]
