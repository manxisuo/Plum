<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const API_BASE = import.meta.env.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string
const loading = ref(false)
const task = ref<any>(null)
const assigns = ref<any[]>([])
const opLoading = ref(false)
const selectedNode = ref<string>('')

const nodesInTask = computed(() => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const row of assigns.value) {
    const n = (row.nodeId || row.NodeID) as string
    if (n && !seen.has(n)) { seen.add(n); out.push(n) }
  }
  return out
})

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/${encodeURIComponent(id)}`)
    if (!res.ok) throw new Error('HTTP '+res.status)
    const json = await res.json()
    task.value = json.task
    assigns.value = json.assignments || []
  } catch (e:any) {
    ElMessage.error(e?.message || '加载失败')
  } finally { loading.value = false }
}

onMounted(load)

async function del(row: any) {
  try {
    const id = row.instanceId || row.InstanceID
    const res = await fetch(`${API_BASE}/v1/assignments/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('HTTP '+res.status)
    ElMessage.success('已删除')
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

async function setDesired(row: any, desired: 'Running'|'Stopped') {
  const iid = row.instanceId || row.InstanceID
  const res = await fetch(`${API_BASE}/v1/assignments/${encodeURIComponent(iid)}`, { method:'PATCH', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ desired }) })
  if (!res.ok) { ElMessage.error('操作失败'); return }
  ElMessage.success('已更新')
  load()
}

async function stopAll() {
  try {
    opLoading.value = true
    await Promise.all((assigns.value || []).map(async (row:any) => {
      const desired = (row.desired || row.Desired)
      if (desired === 'Stopped') return
      const iid = row.instanceId || row.InstanceID
      await fetch(`${API_BASE}/v1/assignments/${encodeURIComponent(iid)}`, { method:'PATCH', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ desired: 'Stopped' }) })
    }))
    ElMessage.success('已下发停止')
  } finally { opLoading.value = false; load() }
}

async function stopByNode() {
  if (!selectedNode.value) { ElMessage.warning('请选择节点'); return }
  try {
    opLoading.value = true
    await Promise.all((assigns.value || []).map(async (row:any) => {
      const node = row.nodeId || row.NodeID
      const desired = (row.desired || row.Desired)
      if (node !== selectedNode.value || desired === 'Stopped') return
      const iid = row.instanceId || row.InstanceID
      await fetch(`${API_BASE}/v1/assignments/${encodeURIComponent(iid)}`, { method:'PATCH', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ desired: 'Stopped' }) })
    }))
    ElMessage.success('已下发按节点停止')
  } finally { opLoading.value = false; load() }
}
</script>

<template>
  <div>
    <h3>Task 详情</h3>
    <div style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
      <el-button type="warning" :loading="opLoading" @click="stopAll">全部停止</el-button>
      <el-select v-model="selectedNode" placeholder="选择节点" style="width:200px;">
        <el-option v-for="n in nodesInTask" :key="n" :label="n" :value="n" />
      </el-select>
      <el-button type="warning" :loading="opLoading" @click="stopByNode">按节点停止</el-button>
    </div>
    <el-descriptions v-if="task" :column="2" border style="margin-bottom:12px;">
      <el-descriptions-item label="TaskID">{{ task.taskId || task.TaskID }}</el-descriptions-item>
      <el-descriptions-item label="Name">{{ task.name || task.Name }}</el-descriptions-item>
      <el-descriptions-item label="Labels" :span="2"><code>{{ JSON.stringify(task.labels || task.Labels || {}) }}</code></el-descriptions-item>
    </el-descriptions>
    <el-table :data="assigns" v-loading="loading" style="width:100%">
      <el-table-column label="InstanceID" width="300">
        <template #default="{ row }">{{ row.instanceId || row.InstanceID }}</template>
      </el-table-column>
      <el-table-column label="NodeID" width="180">
        <template #default="{ row }">{{ row.nodeId || row.NodeID }}</template>
      </el-table-column>
      <el-table-column label="Artifact">
        <template #default="{ row }">{{ row.artifactUrl || row.ArtifactURL }}</template>
      </el-table-column>
      <el-table-column prop="startCmd" label="StartCmd" />
      <el-table-column label="Desired" width="120">
        <template #default="{ row }">{{ row.desired || row.Desired }}</template>
      </el-table-column>
      <el-table-column label="Action" width="220">
        <template #default="{ row }">
          <el-button size="small" type="primary" :disabled="(row.desired||row.Desired)==='Running'" @click="setDesired(row,'Running')">Start</el-button>
          <el-button size="small" type="warning" :disabled="(row.desired||row.Desired)==='Stopped'" @click="setDesired(row,'Stopped')">Stop</el-button>
          <el-popconfirm title="确认删除该实例分配？" @confirm="del(row)">
            <template #reference>
              <el-button type="danger" size="small">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>


