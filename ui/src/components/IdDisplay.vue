<script setup lang="ts">
import { ElMessage } from 'element-plus'
import { formatId, copyToClipboard } from '../utils/formatters'

const props = defineProps<{
  id: string
  length?: number
}>()

async function handleClick() {
  const success = await copyToClipboard(props.id)
  if (success) {
    ElMessage.success('ID已复制到剪贴板')
  } else {
    ElMessage.error('复制失败')
  }
}
</script>

<template>
  <el-tooltip :content="id" placement="top">
    <span 
      style="cursor: pointer; user-select: none;"
      @click="handleClick"
    >
      {{ formatId(id, length) }}
    </span>
  </el-tooltip>
</template>

<style scoped>
span:hover {
  color: #409eff;
  text-decoration: underline;
}
</style>

