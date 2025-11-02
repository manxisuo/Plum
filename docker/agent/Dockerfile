# Plum Agent Dockerfile
# 多阶段构建，优化镜像大小

# 构建阶段
# 使用 go1.23-alpine，通过 GOTOOLCHAIN=auto 自动下载 go1.24 工具链（go.mod 要求 go >= 1.24.0）
FROM golang:1.23-alpine AS builder

# 设置 GOTOOLCHAIN 允许自动下载所需工具链
# go.mod 要求 go >= 1.24.0，但 Docker 镜像可能只有 1.23，通过此设置自动下载 1.24 工具链
ENV GOTOOLCHAIN=auto

# 设置工作目录
WORKDIR /app

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 复制go模块文件
COPY agent-go/go.mod agent-go/go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY agent-go/ ./

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o plum-agent .

# 运行阶段
FROM alpine:3.18

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata curl

# 创建非root用户
RUN addgroup -g 1001 -S plum && \
    adduser -u 1001 -S plum -G plum

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/plum-agent ./plum-agent

# 复制配置文件
COPY agent-go/env.example ./.env

# 创建数据目录
RUN mkdir -p /app/data && \
    chown -R plum:plum /app

# 切换到非root用户
USER plum

# 设置环境变量
ENV AGENT_NODE_ID=nodeA
ENV CONTROLLER_BASE=http://plum-controller:8080
ENV AGENT_DATA_DIR=/app/data

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep plum-agent > /dev/null || exit 1

# 启动命令
CMD ["./plum-agent"]
