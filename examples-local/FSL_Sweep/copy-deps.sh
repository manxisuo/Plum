#!/bin/bash
# 复制 FSL_Sweep 的依赖库到指定目录，用于 Docker 镜像构建
# 使用方法: ./copy-deps.sh <target_dir>

set -e

if [ $# -lt 1 ]; then
    echo "用法: $0 <target_dir>"
    echo "示例: $0 /tmp/fsl-sweep-deps"
    exit 1
fi

TARGET_DIR="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

cd "$PROJECT_ROOT"

# 检查 FSL_Sweep 是否已编译
if [ ! -f "examples-local/FSL_Sweep/bin/FSL_Sweep" ]; then
    echo "错误: FSL_Sweep 未编译，请先执行: make examples_FSL_Sweep"
    exit 1
fi

echo "📦 复制 FSL_Sweep 依赖库到 $TARGET_DIR..."

# 创建目标目录
mkdir -p "$TARGET_DIR/lib"
mkdir -p "$TARGET_DIR/bin"

# 复制可执行文件和脚本
echo "复制可执行文件..."
cp examples-local/FSL_Sweep/bin/FSL_Sweep "$TARGET_DIR/bin/"
cp examples-local/FSL_Sweep/bin/start.sh "$TARGET_DIR/bin/"
cp examples-local/FSL_Sweep/bin/meta.ini "$TARGET_DIR/bin/"
chmod +x "$TARGET_DIR/bin/FSL_Sweep" "$TARGET_DIR/bin/start.sh"

# 复制 SDK 库（.so 或 .a）
echo "复制 SDK 库..."
# 查找并复制 plumworker 库（.so 或 .a）
if [ -f "sdk/cpp/build/plumworker/libplumworker.so" ]; then
    cp sdk/cpp/build/plumworker/libplumworker.so* "$TARGET_DIR/lib/" 2>/dev/null || true
elif [ -f "sdk/cpp/build/plumworker/libplumworker.a" ]; then
    echo "  注意: plumworker 是静态库 (.a)，不需要复制"
fi

# 查找并复制 grpc_proto 库（.so 或 .a）
if [ -f "sdk/cpp/build/grpc_proto/libgrpc_proto.so" ]; then
    cp sdk/cpp/build/grpc_proto/libgrpc_proto.so* "$TARGET_DIR/lib/" 2>/dev/null || true
elif [ -f "sdk/cpp/build/grpc_proto/libgrpc_proto.a" ]; then
    echo "  注意: grpc_proto 是静态库 (.a)，不需要复制"
fi

# 使用 ldd 查找并复制系统依赖库
echo "查找系统依赖库..."
DEPS_FILE=$(mktemp)
ldd examples-local/FSL_Sweep/bin/FSL_Sweep 2>/dev/null | grep -E "\.so" | awk '{print $3}' > "$DEPS_FILE" || true

# 复制函数：复制库文件及其所有符号链接
copy_lib_with_symlinks() {
    local lib_path="$1"
    if [ ! -f "$lib_path" ] && [ ! -L "$lib_path" ]; then
        return 1
    fi
    
    local lib_name=$(basename "$lib_path")
    local lib_dir=$(dirname "$lib_path")
    
    # 如果是符号链接，找到真实文件并复制
    local real_lib="$lib_path"
    if [ -L "$lib_path" ]; then
        real_lib=$(readlink -f "$lib_path" 2>/dev/null || readlink "$lib_path")
        # 如果是相对路径，转换为绝对路径
        if [ "${real_lib#/}" = "$real_lib" ]; then
            real_lib="$lib_dir/$real_lib"
        fi
    fi
    
    # 复制真实文件（如果存在且是文件）
    if [ -f "$real_lib" ]; then
        local real_name=$(basename "$real_lib")
        if [ ! -f "$TARGET_DIR/lib/$real_name" ]; then
            cp "$real_lib" "$TARGET_DIR/lib/" 2>/dev/null || return 1
        fi
    fi
    
    # 复制符号链接（在目标目录中重新创建）
    if [ -L "$lib_path" ]; then
        local link_target=$(readlink "$lib_path")
        # 如果链接目标是相对路径，需要找到绝对路径
        if [ "${link_target#/}" = "$link_target" ]; then
            link_target=$(readlink -f "$lib_path" 2>/dev/null || readlink "$lib_path")
            if [ "${link_target#/}" = "$link_target" ]; then
                link_target="$lib_dir/$link_target"
            fi
        fi
        local target_name=$(basename "$link_target")
        # 在目标目录创建符号链接
        if [ ! -e "$TARGET_DIR/lib/$lib_name" ]; then
            (cd "$TARGET_DIR/lib" && ln -sf "$target_name" "$lib_name" 2>/dev/null || true)
        fi
    fi
    
    # 复制同一目录下所有相关的符号链接和文件
    local base_name=$(echo "$lib_name" | sed 's/\.[0-9].*$//' | sed 's/\.so$//')
    find "$lib_dir" -maxdepth 1 \( -name "${base_name}*.so*" -o -name "${base_name}*.so" \) 2>/dev/null | while read -r related_lib; do
        if [ "$related_lib" != "$lib_path" ]; then
            related_name=$(basename "$related_lib")
            if [ ! -e "$TARGET_DIR/lib/$related_name" ]; then
                if [ -L "$related_lib" ]; then
                    # 符号链接：找到目标并复制，然后创建链接
                    local rel_target=$(readlink "$related_lib")
                    if [ "${rel_target#/}" = "$rel_target" ]; then
                        rel_target=$(readlink -f "$related_lib" 2>/dev/null || readlink "$related_lib")
                        if [ "${rel_target#/}" = "$rel_target" ]; then
                            rel_target="$lib_dir/$rel_target"
                        fi
                    fi
                    if [ -f "$rel_target" ]; then
                        local rel_target_name=$(basename "$rel_target")
                        if [ ! -f "$TARGET_DIR/lib/$rel_target_name" ]; then
                            cp "$rel_target" "$TARGET_DIR/lib/" 2>/dev/null || true
                        fi
                        (cd "$TARGET_DIR/lib" && ln -sf "$rel_target_name" "$related_name" 2>/dev/null || true)
                    fi
                elif [ -f "$related_lib" ]; then
                    # 普通文件：直接复制
                    cp "$related_lib" "$TARGET_DIR/lib/" 2>/dev/null || true
                fi
            fi
        fi
    done
    
    return 0
}

if [ -s "$DEPS_FILE" ]; then
    copied_libs=()
    while IFS= read -r lib_path; do
        if [ -n "$lib_path" ] && [ -f "$lib_path" ]; then
            lib_name=$(basename "$lib_path")
            # 复制 gRPC、protobuf、absl 以及 gRPC 的依赖库（gpr, cares, re2, upb, address_sorting 等）
            # 同时复制系统库（libc, libstdc++）以解决 GLIBC 版本不匹配问题
            if echo "$lib_name" | grep -qE "(grpc|protobuf|absl|gpr|cares|re2|upb|address_sorting|ssl|crypto|libc\.so|libstdc\+\+\.so|libm\.so|libgcc_s\.so|libpthread\.so)"; then
                # 避免重复复制
                if [[ ! " ${copied_libs[@]} " =~ " ${lib_name} " ]]; then
                    echo "  复制系统库: $lib_name"
                    if copy_lib_with_symlinks "$lib_path"; then
                        copied_libs+=("$lib_name")
                    fi
                fi
            fi
        fi
    done < "$DEPS_FILE"
else
    echo "  警告: 无法获取依赖库列表，尝试手动查找 gRPC 库..."
    # 手动查找 gRPC 库
    for lib_dir in /usr/lib /usr/local/lib /usr/lib/x86_64-linux-gnu /usr/lib/aarch64-linux-gnu; do
        if [ -d "$lib_dir" ]; then
            find "$lib_dir" -name "libgrpc++*.so*" -o -name "libgrpc*.so*" -o -name "libprotobuf*.so*" 2>/dev/null | while read -r lib_path; do
                lib_name=$(basename "$lib_path")
                if [[ ! " ${copied_libs[@]} " =~ " ${lib_name} " ]]; then
                    echo "  复制系统库: $lib_name (从 $lib_dir)"
                    copy_lib_with_symlinks "$lib_path" && copied_libs+=("$lib_name")
                fi
            done
        fi
    done
fi
rm -f "$DEPS_FILE"

# 验证复制的库
echo ""
echo "已复制的库文件："
ls -lh "$TARGET_DIR/lib/" 2>/dev/null | tail -n +2 || echo "  (无)"

echo "✅ 依赖库复制完成"
echo "   可执行文件: $TARGET_DIR/bin/"
echo "   库文件: $TARGET_DIR/lib/"
echo ""
echo "现在可以使用以下命令构建镜像："
echo "  docker build -f examples-local/FSL_Sweep/Dockerfile.local -t fsl-sweep:1.0.0 $TARGET_DIR"

