<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const router = useRouter()

const workflowId = route.params.workflowId as string
const runs = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/workflow-runs?workflowId=${encodeURIComponent(workflowId)}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    runs.value = await res.json()
  } catch (e: any) { 
    ElMessage.error(e?.message || '加载失败') 
  }
  finally { 
    loading.value = false 
  }
}

async function deleteRun(runId: string) {
  try {
    await ElMessageBox.confirm('确定要删除这个运行记录吗？删除后无法恢复。', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const res = await fetch(`${API_BASE}/v1/workflow-runs/${encodeURIComponent(runId)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e: any) { 
    if (e !== 'cancel') {
      ElMessage.error(e?.message || '删除失败') 
    }
  }
}

function viewRun(runId: string) {
  router.push(`/workflow-runs/${runId}`)
}

function formatTime(timestamp: number) {
  return new Date(timestamp * 1000).toLocaleString()
}

onMounted(load)
const { t } = useI18n()
</script>

<template>
  <div>
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>工作流运行历史 ({{ workflowId }})</span>
          <el-button @click="router.push('/workflows')">← 返回工作流列表</el-button>
        </div>
      </template>

      <el-table :data="runs" v-loading="loading" style="width:100%;">
      <el-table-column label="运行ID" width="320">
        <template #default="{ row }">{{ (row as any).runId || (row as any).RunID }}</template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <el-tag :type="((row as any).state || (row as any).State) === 'Succeeded' ? 'success' : ((row as any).state || (row as any).State) === 'Failed' ? 'danger' : 'warning'">
            {{ (row as any).state || (row as any).State }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="200">
        <template #default="{ row }">{{ formatTime((row as any).createdAt || (row as any).CreatedAt) }}</template>
      </el-table-column>
      <el-table-column label="开始时间" width="200">
        <template #default="{ row }">
          {{ (row as any).startedAt || (row as any).StartedAt ? formatTime((row as any).startedAt || (row as any).StartedAt) : '-' }}
        </template>
      </el-table-column>
      <el-table-column label="结束时间" width="200">
        <template #default="{ row }">
          {{ (row as any).finishedAt || (row as any).FinishedAt ? formatTime((row as any).finishedAt || (row as any).FinishedAt) : '-' }}
        </template>
      </el-table-column>
        <el-table-column label="操作" width="200">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="viewRun((row as any).runId || (row as any).RunID)">查看详情</el-button>
            <el-button size="small" type="danger" @click="deleteRun((row as any).runId || (row as any).RunID)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
