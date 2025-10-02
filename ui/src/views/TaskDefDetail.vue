<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string

const defn = ref<any>(null)
const runs = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const [dRes, tRes] = await Promise.all([
      fetch(`${API_BASE}/v1/task-defs/${encodeURIComponent(id)}`),
      fetch(`${API_BASE}/v1/tasks`)
    ])
    if (!dRes.ok) throw new Error('HTTP '+dRes.status)
    defn.value = await dRes.json()
    if (tRes.ok) {
      const arr = await tRes.json() as any[]
      runs.value = (arr||[]).filter(t => (t.originTaskId||t.OriginTaskID) === id)
    }
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}

onMounted(load)

async function startTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/start/${encodeURIComponent(id)}`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已开始')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function cancelTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/cancel/${encodeURIComponent(id)}`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已取消')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function deleteTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '删除失败') }
}
</script>

<template>
  <div>
    <h3>TaskDefinition 详情</h3>
    <el-descriptions v-if="defn" :column="2" border style="margin-bottom:12px;">
      <el-descriptions-item label="DefID">{{ defn.defId || defn.DefID }}</el-descriptions-item>
      <el-descriptions-item label="Name">{{ defn.name || defn.Name }}</el-descriptions-item>
      <el-descriptions-item label="Executor">{{ defn.executor || defn.Executor }}</el-descriptions-item>
    </el-descriptions>

    <h4>运行历史</h4>
    <el-table :data="runs" v-loading="loading" style="width:100%">
      <el-table-column label="TaskID" width="320">
        <template #default="{ row }">{{ row.taskId || row.TaskID }}</template>
      </el-table-column>
      <el-table-column label="State" width="140">
        <template #default="{ row }">{{ row.state || row.State }}</template>
      </el-table-column>
      <el-table-column label="Created" width="160">
        <template #default="{ row }">{{ new Date(((row.createdAt||row.CreatedAt)||0)*1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="Action" width="300">
        <template #default="{ row }">
          <el-button size="small" type="primary" :disabled="(row.state||row.State)!=='Queued'" @click="startTask(row.taskId||row.TaskID)">Start</el-button>
          <el-button size="small" type="warning" :disabled="!((row.state||row.State)==='Running' || (row.state||row.State)==='Queued')" @click="cancelTask(row.taskId||row.TaskID)">Cancel</el-button>
          <el-popconfirm title="确认删除该任务？" @confirm="deleteTask(row.taskId||row.TaskID)">
            <template #reference>
              <el-button size="small" type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
