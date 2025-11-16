#!/bin/bash

# 创建 Plum 部署包脚本
# 用途：提取项目中的部署相关文件和目录，生成 Plum_Deploy.tar.gz

set -e  # 遇到错误立即退出

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 脚本所在目录（项目根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 输出文件名
OUTPUT_FILE="Plum_Deploy.tar.gz"
# 解压后的目录名
DEPLOY_DIR="Plum_Deploy"

echo -e "${GREEN}开始创建 Plum 部署包...${NC}"

# 检查必要的文件和目录是否存在
echo -e "${YELLOW}检查必要的文件和目录...${NC}"

MISSING_FILES=0

# 检查目录
for dir in "docker" "ui/dist"; do
    if [ ! -d "$dir" ]; then
        echo -e "${RED}错误: 目录不存在: $dir${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $dir"
    fi
done

# 检查 .env 文件
for env_file in "agent-go/.env" "controller/.env"; do
    if [ ! -f "$env_file" ]; then
        echo -e "${RED}错误: 文件不存在: $env_file${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $env_file"
    fi
done

# 检查 env.example 文件
for env_example in "agent-go/env.example" "controller/env.example"; do
    if [ ! -f "$env_example" ]; then
        echo -e "${RED}错误: 文件不存在: $env_example${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $env_example"
    fi
done

# 检查文件
for file in "docker-compose.agent.yml" "docker-compose.main.yml" "export-docker-images.sh" "import-docker-images.sh"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}错误: 文件不存在: $file${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $file"
    fi
done

# 检查 docker 子目录（仅检查需要的目录）
for dir in "docker/agent" "docker/controller" "docker/nginx"; do
    if [ ! -d "$dir" ]; then
        echo -e "${RED}错误: 目录不存在: $dir${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $dir"
    fi
done

# 检查 docker 根目录下的文件
for file in "docker/build-docker.sh" "docker/build-static-offline.sh" "docker/start-agent.sh" "docker/start-controller.sh" "docker/stop-agent.sh" "docker/stop-controller.sh"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}错误: 文件不存在: $file${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $file"
    fi
done

# 检查 docker/nginx 下的文件
for file in "docker/nginx/Dockerfile" "docker/nginx/nginx.conf" "docker/nginx/nginx.conf.host"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}错误: 文件不存在: $file${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $file"
    fi
done

# 检查 docker/agent 和 docker/controller 的 Dockerfile
for file in "docker/agent/Dockerfile" "docker/controller/Dockerfile"; do
    if [ ! -f "$file" ]; then
        echo -e "${RED}错误: 文件不存在: $file${NC}"
        MISSING_FILES=1
    else
        echo -e "${GREEN}✓${NC} $file"
    fi
done

if [ $MISSING_FILES -eq 1 ]; then
    echo -e "${RED}错误: 缺少必要的文件或目录，请检查后重试${NC}"
    exit 1
fi

# 如果输出文件已存在，询问是否覆盖
if [ -f "$OUTPUT_FILE" ]; then
    echo -e "${YELLOW}警告: $OUTPUT_FILE 已存在${NC}"
    read -p "是否覆盖? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}已取消${NC}"
        exit 0
    fi
    rm -f "$OUTPUT_FILE"
fi

echo -e "${YELLOW}正在打包...${NC}"

# 创建临时目录结构
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# 在临时目录中创建 Plum_Deploy 目录结构
mkdir -p "$TEMP_DIR/$DEPLOY_DIR"

# 复制文件到临时目录，保持目录结构
# agent-go 和 controller 包含 .env 和 env.example 文件
mkdir -p "$TEMP_DIR/$DEPLOY_DIR/agent-go"
cp "agent-go/.env" "$TEMP_DIR/$DEPLOY_DIR/agent-go/.env"
cp "agent-go/env.example" "$TEMP_DIR/$DEPLOY_DIR/agent-go/env.example"

mkdir -p "$TEMP_DIR/$DEPLOY_DIR/controller"
cp "controller/.env" "$TEMP_DIR/$DEPLOY_DIR/controller/.env"
cp "controller/env.example" "$TEMP_DIR/$DEPLOY_DIR/controller/env.example"

# 复制 docker 目录下的特定文件（不复制整个目录）
mkdir -p "$TEMP_DIR/$DEPLOY_DIR/docker/agent"
cp "docker/agent/Dockerfile" "$TEMP_DIR/$DEPLOY_DIR/docker/agent/Dockerfile"

mkdir -p "$TEMP_DIR/$DEPLOY_DIR/docker/controller"
cp "docker/controller/Dockerfile" "$TEMP_DIR/$DEPLOY_DIR/docker/controller/Dockerfile"

mkdir -p "$TEMP_DIR/$DEPLOY_DIR/docker/nginx"
cp "docker/nginx/Dockerfile" "$TEMP_DIR/$DEPLOY_DIR/docker/nginx/Dockerfile"
cp "docker/nginx/nginx.conf" "$TEMP_DIR/$DEPLOY_DIR/docker/nginx/nginx.conf"
cp "docker/nginx/nginx.conf.host" "$TEMP_DIR/$DEPLOY_DIR/docker/nginx/nginx.conf.host"

cp "docker/build-docker.sh" "$TEMP_DIR/$DEPLOY_DIR/docker/build-docker.sh"
cp "docker/build-static-offline.sh" "$TEMP_DIR/$DEPLOY_DIR/docker/build-static-offline.sh"
cp "docker/start-agent.sh" "$TEMP_DIR/$DEPLOY_DIR/docker/start-agent.sh"
cp "docker/start-controller.sh" "$TEMP_DIR/$DEPLOY_DIR/docker/start-controller.sh"
cp "docker/stop-agent.sh" "$TEMP_DIR/$DEPLOY_DIR/docker/stop-agent.sh"
cp "docker/stop-controller.sh" "$TEMP_DIR/$DEPLOY_DIR/docker/stop-controller.sh"

# 复制根目录文件
cp "docker-compose.agent.yml" "$TEMP_DIR/$DEPLOY_DIR/"
cp "docker-compose.main.yml" "$TEMP_DIR/$DEPLOY_DIR/"
cp "export-docker-images.sh" "$TEMP_DIR/$DEPLOY_DIR/"
cp "import-docker-images.sh" "$TEMP_DIR/$DEPLOY_DIR/"

# 复制 ui/dist 目录（包括所有子目录）
mkdir -p "$TEMP_DIR/$DEPLOY_DIR/ui"
cp -r "ui/dist" "$TEMP_DIR/$DEPLOY_DIR/ui/"

# 进入临时目录，打包 Plum_Deploy 目录
cd "$TEMP_DIR"

# 使用 tar 打包，排除不需要的文件
if ! tar -czf "$SCRIPT_DIR/$OUTPUT_FILE" \
    --exclude='*.db' \
    --exclude='*.db-shm' \
    --exclude='*.db-wal' \
    --exclude='.git' \
    --exclude='.gitignore' \
    --exclude='node_modules' \
    --exclude='*.log' \
    --exclude='*.tmp' \
    --exclude='__pycache__' \
    --exclude='*.pyc' \
    --exclude='*.swp' \
    --exclude='*.swo' \
    --exclude='*~' \
    "$DEPLOY_DIR" \
    2>&1; then
    echo -e "${RED}错误: 打包失败${NC}"
    echo -e "${YELLOW}请检查文件列表和权限${NC}"
    exit 1
fi

# 返回原目录
cd "$SCRIPT_DIR"

# 检查输出文件
if [ -f "$OUTPUT_FILE" ]; then
    FILE_SIZE=$(du -h "$OUTPUT_FILE" | cut -f1)
    echo -e "${GREEN}✓ 打包成功!${NC}"
    echo -e "${GREEN}输出文件: $OUTPUT_FILE${NC}"
    echo -e "${GREEN}文件大小: $FILE_SIZE${NC}"
    echo ""
    echo -e "${YELLOW}包内容结构:${NC}"
    echo "$DEPLOY_DIR/"
    echo "├── agent-go/"
    echo "│   ├── .env"
    echo "│   └── env.example"
    echo "├── controller/"
    echo "│   ├── .env"
    echo "│   └── env.example"
    echo "├── docker/"
    echo "│   ├── agent/"
    echo "│   │   └── Dockerfile"
    echo "│   ├── build-docker.sh"
    echo "│   ├── build-static-offline.sh"
    echo "│   ├── controller/"
    echo "│   │   └── Dockerfile"
    echo "│   ├── nginx/"
    echo "│   │   ├── Dockerfile"
    echo "│   │   ├── nginx.conf"
    echo "│   │   └── nginx.conf.host"
    echo "│   ├── start-agent.sh"
    echo "│   ├── start-controller.sh"
    echo "│   ├── stop-agent.sh"
    echo "│   └── stop-controller.sh"
    echo "├── docker-compose.agent.yml"
    echo "├── docker-compose.main.yml"
    echo "├── export-docker-images.sh"
    echo "├── import-docker-images.sh"
    echo "└── ui/"
    echo "    └── dist/"
    echo ""
    echo -e "${YELLOW}解压方式:${NC}"
    echo "  tar -xzvf $OUTPUT_FILE"
    echo -e "${YELLOW}解压后将得到: $DEPLOY_DIR/ 目录${NC}"
else
    echo -e "${RED}错误: 打包失败，输出文件不存在${NC}"
    exit 1
fi

echo -e "${GREEN}完成!${NC}"

