#!/bin/bash

# 打包应用脚本
# 使用方法: tools/pkg_app.sh <app_dir>
# 功能: 将 app_dir 下的所有文件打包成 {appName}_{appVersion}.zip，放到 apps 目录下

set -e

# 检查参数
if [ $# -lt 1 ]; then
    echo "用法: $0 <app_dir>"
    echo "示例: $0 examples/worker-demo"
    exit 1
fi

APP_DIR="$1"

# 检查 app_dir 是否存在
if [ ! -d "$APP_DIR" ]; then
    echo "错误: 目录不存在: $APP_DIR"
    exit 1
fi

# 获取绝对路径
APP_DIR=$(cd "$APP_DIR" && pwd)

# 查找 meta.ini 文件（必须在当前目录或父目录中）
# 前提条件：指定的目录路径中必须包含 meta.ini
META_INI="$APP_DIR/meta.ini"
PACK_SUBDIR_ONLY=false

if [ ! -f "$META_INI" ]; then
    # 如果当前目录没有 meta.ini，尝试在父目录查找
    PARENT_DIR=$(dirname "$APP_DIR")
    META_INI="$PARENT_DIR/meta.ini"
    if [ ! -f "$META_INI" ]; then
        echo "错误: meta.ini 文件不存在（已检查: $APP_DIR/meta.ini 和 $PARENT_DIR/meta.ini）"
        echo "提示: 指定的目录路径中必须包含 meta.ini 文件"
        exit 1
    fi
    # 在父目录找到 meta.ini，说明传入的是子目录，只打包子目录内容
    echo "提示: 在父目录找到 meta.ini: $META_INI"
    echo "提示: 将只打包 $APP_DIR 目录下的文件（不包含目录本身）"
    PACK_SUBDIR_ONLY=true
else
    # 当前目录有 meta.ini，检查传入的路径是否以常见子目录名结尾
    # 如果是，说明用户想只打包子目录内容，不包含目录本身
    DIR_BASENAME=$(basename "$APP_DIR")
    if [[ "$DIR_BASENAME" =~ ^(bin|dist|build|output|release|target|out)$ ]]; then
        echo "提示: 检测到子目录（$DIR_BASENAME），将只打包目录下的文件（不包含目录本身）"
        PACK_SUBDIR_ONLY=true
    fi
    # 如果当前目录有 meta.ini 且不是子目录，则正常打包整个目录（包含目录本身）
fi

# 解析 meta.ini 文件，提取 name 和 version
APP_NAME=""
APP_VERSION=""

while IFS= read -r line || [ -n "$line" ]; do
    # 移除前后空白
    line=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    
    # 跳过空行和注释
    if [[ -z "$line" || "$line" =~ ^# || "$line" =~ ^\; ]]; then
        continue
    fi
    
    # 解析 key=value 格式
    if [[ "$line" =~ ^[[:space:]]*([^=:]+)[=:](.+)$ ]]; then
        key=$(echo "${BASH_REMATCH[1]}" | tr '[:upper:]' '[:lower:]' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        value=$(echo "${BASH_REMATCH[2]}" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        case "$key" in
            name)
                APP_NAME="$value"
                ;;
            version)
                APP_VERSION="$value"
                ;;
        esac
    fi
done < "$META_INI"

# 检查是否成功读取 name 和 version
if [ -z "$APP_NAME" ]; then
    echo "错误: 无法从 meta.ini 中读取 name 字段"
    exit 1
fi

if [ -z "$APP_VERSION" ]; then
    echo "错误: 无法从 meta.ini 中读取 version 字段"
    exit 1
fi

echo "应用名称: $APP_NAME"
echo "应用版本: $APP_VERSION"

# 确定 Plum 项目根目录（脚本在 tools 目录下，所以上一级是根目录）
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
APPS_DIR="$PROJECT_ROOT/apps"

# 创建 apps 目录（如果不存在）
mkdir -p "$APPS_DIR"

# 生成 zip 文件名
ZIP_NAME="${APP_NAME}_${APP_VERSION}.zip"
ZIP_PATH="$APPS_DIR/$ZIP_NAME"

# 如果 zip 文件已存在，询问是否覆盖
if [ -f "$ZIP_PATH" ]; then
    echo "警告: 文件已存在: $ZIP_PATH"
    read -p "是否覆盖? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "已取消"
        exit 0
    fi
    rm -f "$ZIP_PATH"
fi

# 打包文件
if [ "$PACK_SUBDIR_ONLY" = "true" ]; then
    # 只打包子目录下的文件，不包含目录本身
    echo "正在打包: $APP_DIR/* -> $ZIP_PATH"
    cd "$APP_DIR"
    
    # 使用 find 查找所有文件，然后逐个添加到 zip
    # 关键：cd 到目录后，使用相对路径（去掉 ./ 前缀），zip 会自动处理路径
    # zip 会保留子目录结构但不包含顶层目录名
    find . -type f ! -path "*/\.*" ! -path "*/.git/*" ! -path "*/.svn/*" \
        ! -path "*/build/*" ! -name "*.o" ! -name "*.a" ! -name "*.so" \
        ! -name "*.dylib" ! -name "*.dll" ! -name "*.exe" | while IFS= read -r file; do
        # 去掉开头的 ./
        rel_path="${file#./}"
        # 确保路径不为空
        if [ -n "$rel_path" ]; then
            # 直接使用相对路径，zip 会保留子目录结构但不包含顶层目录
            zip -q "$ZIP_PATH" "$rel_path" || {
                echo "警告: 添加文件失败: $rel_path" >&2
            }
        fi
    done
    
    if [ ! -f "$ZIP_PATH" ] || [ ! -s "$ZIP_PATH" ]; then
        echo "错误: 打包失败或生成的文件为空"
        exit 1
    fi
else
    # 打包整个目录（包含目录本身）
    APP_PARENT_DIR=$(dirname "$APP_DIR")
    APP_BASE_NAME=$(basename "$APP_DIR")
    
    echo "正在打包: $APP_DIR -> $ZIP_PATH"
    cd "$APP_PARENT_DIR"
    
    # 使用 zip 命令打包（排除 .git 等隐藏文件，但保留所有其他文件）
    zip -r "$ZIP_PATH" "$APP_BASE_NAME" \
        -x "*/\.*" \
        -x "*/.git/*" \
        -x "*/.svn/*" \
        -x "*/build/*" \
        -x "*/*.o" \
        -x "*/*.a" \
        -x "*/*.so" \
        -x "*/*.dylib" \
        -x "*/*.dll" \
        -x "*/*.exe" \
        > /dev/null 2>&1 || {
        echo "错误: 打包失败"
        exit 1
    }
fi

echo "✅ 打包完成: $ZIP_PATH"
echo "文件大小: $(du -h "$ZIP_PATH" | cut -f1)"

