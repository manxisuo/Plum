# UI组件使用说明

## IdDisplay - ID显示组件

### 功能
- 显示缩短版的长ID（默认前8个字符）
- 鼠标悬停显示完整ID的tooltip
- 点击复制完整ID到剪贴板
- 悬停时高亮显示

### 使用方法

```vue
<script setup>
import IdDisplay from '@/components/IdDisplay.vue'
</script>

<template>
  <!-- 默认显示8个字符 -->
  <IdDisplay :id="taskId" />
  
  <!-- 自定义显示长度 -->
  <IdDisplay :id="deploymentId" :length="12" />
</template>
```

### Props
- `id` (string, required): 要显示的完整ID
- `length` (number, optional): 显示的字符长度，默认8

### 特性
- 等宽字体显示（monospace）
- 点击复制时显示成功/失败提示
- 悬停时变色并显示下划线
- 完整ID通过tooltip展示

### 适用场景
- 任务ID
- 部署ID
- 实例ID
- 工作流ID
- 节点ID
- 任何长度超过8个字符的唯一标识符

