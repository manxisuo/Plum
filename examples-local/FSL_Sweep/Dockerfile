# FSL_Sweep Docker 镜像
# 多阶段构建：先编译，再打包运行时镜像

# ========== 构建阶段 ==========
FROM ubuntu:22.04 AS builder

# 安装构建依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    cmake \
    qmake \
    qtbase5-dev \
    qtbase5-dev-tools \
    pkg-config \
    libprotobuf-dev \
    protobuf-compiler \
    git \
    curl \
    ninja-build \
    && rm -rf /var/lib/apt/lists/*

# 安装 gRPC（从源码构建，因为 Ubuntu 22.04 默认仓库可能没有预编译包）
# 注意：这会花费较长时间，可以考虑使用预构建的 gRPC 镜像或缓存
RUN mkdir -p /tmp/grpc && \
    cd /tmp/grpc && \
    git clone --depth 1 --branch v1.60.0 https://github.com/grpc/grpc.git . && \
    git submodule update --init --recursive && \
    mkdir -p cmake/build && \
    cd cmake/build && \
    cmake -DgRPC_INSTALL=ON \
          -DgRPC_BUILD_TESTS=OFF \
          -DgRPC_PROTOBUF_PROVIDER=package \
          -DCMAKE_INSTALL_PREFIX=/usr/local \
          -DCMAKE_BUILD_TYPE=Release \
          ../.. && \
    make -j$(nproc) && \
    make install && \
    cd / && \
    rm -rf /tmp/grpc && \
    ldconfig

# 设置工作目录
WORKDIR /build

# 复制整个项目（需要 SDK 库）
# 注意：这里假设从项目根目录构建，使用 -f 参数指定 Dockerfile
# COPY 路径是相对于构建上下文（项目根目录）的
COPY sdk /build/sdk
COPY examples-local/FSL_Common /build/FSL_Common
COPY examples-local/FSL_Sweep /build/FSL_Sweep

# 先构建 SDK 库
WORKDIR /build/sdk/cpp
RUN mkdir -p build && \
    cd build && \
    cmake .. && \
    make plumworker grpc_proto

# 构建 FSL_Sweep
WORKDIR /build/FSL_Sweep
RUN mkdir -p build && \
    cd build && \
    qmake CONFIG+=release .. && \
    make

# ========== 运行时阶段 ==========
FROM ubuntu:22.04

# 安装运行时依赖
RUN apt-get update && apt-get install -y \
    libqt5core5a \
    libprotobuf23 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# 从构建阶段复制 gRPC 运行时库
RUN mkdir -p /usr/local/lib && \
    cp --from=builder /usr/local/lib/libgrpc*.so* /usr/local/lib/ && \
    cp --from=builder /usr/local/lib/libabsl*.so* /usr/local/lib/ 2>/dev/null || true && \
    ldconfig

# 创建应用目录
WORKDIR /app

# 从构建阶段复制可执行文件和必要文件
COPY --from=builder /build/FSL_Sweep/bin/FSL_Sweep /app/FSL_Sweep
COPY --from=builder /build/FSL_Sweep/bin/start.sh /app/start.sh
COPY --from=builder /build/FSL_Sweep/bin/meta.ini /app/meta.ini

# 创建 lib 目录
RUN mkdir -p /app/lib

# 从构建阶段复制 SDK 库文件（如果存在）
# 注意：如果库是静态链接的，可能不需要这些库文件
# 使用 shell 脚本来安全地复制库文件
RUN --mount=from=builder,source=/build/sdk/cpp/build,target=/tmp/build \
    if [ -f /tmp/build/plumworker/libplumworker.so ]; then \
        cp /tmp/build/plumworker/libplumworker.so* /app/lib/; \
    fi && \
    if [ -f /tmp/build/grpc_proto/libgrpc_proto.so ]; then \
        cp /tmp/build/grpc_proto/libgrpc_proto.so* /app/lib/; \
    fi || true

# 设置可执行权限
RUN chmod +x /app/FSL_Sweep /app/start.sh

# 设置环境变量
ENV LD_LIBRARY_PATH=/app/lib
ENV QT_QPA_PLATFORM=offscreen

# 使用 start.sh 作为默认命令
CMD ["./start.sh"]

