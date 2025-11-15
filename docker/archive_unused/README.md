# 已归档的脚本

本目录包含不再使用的脚本，保留用于参考。

## 归档原因

### 调试脚本
- `debug-simple.sh` - 临时调试脚本，硬编码路径，无引用
- `debug-copy-libs.sh` - 调试版本的智能库文件复制脚本，无引用

### 冗余的库复制脚本
- `copy-libs.sh` - 动态库文件复制脚本（已被 `examples-local/copy-deps.sh` 替代）
- `copy-qt5-libs.sh` - Qt5 应用程序库文件复制脚本（无引用）
- `smart-copy-libs.sh` - 智能库文件复制脚本（已被 `examples-local/copy-deps.sh` 替代）

### 无引用的脚本
- `download-openeuler-arm64.sh` - 下载 OpenEuler ARM64 镜像脚本（无引用）

## 替代方案

现在项目统一使用 `examples-local/copy-deps.sh` 来处理所有项目的依赖复制，包括 C++ 和 Python 项目。

