# Qt 项目命令行构建指南

本文档说明如何使用命令行构建 `examples-local` 目录下的 Qt 项目。

## 前置要求

1. **Qt 开发环境**：已安装 Qt 5.12.8 或更高版本
2. **qmake**：Qt 的构建工具（通常在 Qt 安装目录的 `bin` 目录下）
3. **make**：GNU Make 或兼容工具
4. **C++ 编译器**：g++ 或 clang++

## 构建步骤

### 通用构建方法

对于每个 Qt 项目（SimRoutePlan、SimNaviControl、SimSonar），构建步骤如下：

```bash
# 1. 进入项目目录
cd examples-local/SimRoutePlan  # 或 SimNaviControl、SimSonar

# 2. 清理之前的构建产物（可选）
make clean
# 或手动删除
rm -rf build/ Makefile

# 3. 运行 qmake 生成 Makefile
qmake SimRoutePlan.pro           # 根据项目名称调整
# 或者如果 qmake 不在 PATH 中：
# /path/to/Qt/5.12.8/gcc_64/bin/qmake SimRoutePlan.pro

# 4. 运行 make 编译
make

# 5. 可执行文件会在 bin/ 目录下
ls bin/
```

### 详细示例

#### SimRoutePlan

```bash
cd examples-local/SimRoutePlan
qmake SimRoutePlan.pro
make
# 可执行文件: bin/SimRoutePlan
```

#### SimNaviControl

```bash
cd examples-local/SimNaviControl
qmake SimNaviControl.pro
make
# 可执行文件: bin/SimNaviControl
```

#### SimSonar

```bash
cd examples-local/SimSonar
qmake SimSonar.pro
make
# 可执行文件: bin/SimSonar
```

## 构建选项

### Debug 构建（默认）

```bash
qmake SimRoutePlan.pro CONFIG+=debug
make
```

### Release 构建

```bash
qmake SimRoutePlan.pro CONFIG+=release
make
```

### 指定输出目录

在 `.pro` 文件中已经设置了 `DESTDIR = bin`，所以可执行文件会自动输出到 `bin/` 目录。

## 常见问题

### 1. qmake 命令未找到

如果 `qmake` 不在系统的 PATH 中，需要：
- 使用完整路径：`/path/to/Qt/5.12.8/gcc_64/bin/qmake`
- 或者设置环境变量：
  ```bash
  export PATH=/path/to/Qt/5.12.8/gcc_64/bin:$PATH
  ```

### 2. 找不到 Qt 库

确保 Qt 库路径正确：
```bash
export LD_LIBRARY_PATH=/path/to/Qt/5.12.8/gcc_64/lib:$LD_LIBRARY_PATH
```

### 3. 清理构建

```bash
make clean          # 清理编译产物
rm -rf build/       # 删除构建目录
rm -f Makefile      # 删除 Makefile
```

### 4. 重新构建

```bash
make clean          # 清理
qmake               # 重新生成 Makefile
make                # 重新编译
```

## 一键构建脚本

可以创建一个脚本批量构建所有项目：

```bash
#!/bin/bash
# build-all.sh

projects=("SimRoutePlan" "SimNaviControl" "SimSonar")

for project in "${projects[@]}"; do
    echo "构建 $project..."
    cd "examples-local/$project"
    qmake "$project.pro"
    make
    cd ../..
    echo "$project 构建完成"
    echo ""
done

echo "所有项目构建完成！"
```

## 验证构建

构建完成后，可以运行可执行文件验证：

```bash
# SimRoutePlan
cd examples-local/SimRoutePlan
./bin/SimRoutePlan

# SimNaviControl
cd examples-local/SimNaviControl
./bin/SimNaviControl

# SimSonar
cd examples-local/SimSonar
./bin/SimSonar
```

## 注意事项

1. **依赖文件**：确保 `httplib.h` 和 `json.hpp` 在项目目录中
2. **Qt 版本**：确保使用的 Qt 版本与项目配置兼容
3. **编译选项**：`.pro` 文件中已配置 `CONFIG += c++11`，确保编译器支持 C++11
4. **输出目录**：所有可执行文件都输出到 `bin/` 目录，与 `start.sh` 脚本配合使用

