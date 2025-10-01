<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

type Artifact = { artifactId: string; name: string; version: string; url: string }
type NodeDTO = { nodeId: string; ip: string }

const API_BASE = import.meta.env.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string
const loading = ref(false)
const task = ref<any>(null)
const artifacts = ref<Artifact[]>([])
const nodes = ref<NodeDTO[]>([])
// entries read-only for now (future: edit)
const entries = ref<any[]>([])
const labels = ref<Record<string,string>>({})

async function load() {
  loading.value = true
  try {
    const [tRes, aRes, nRes] = await Promise.all([
      fetch(`${API_BASE}/v1/tasks/${encodeURIComponent(id)}`),
      fetch(`${API_BASE}/v1/apps`),
      fetch(`${API_BASE}/v1/nodes`)
    ])
    if (!tRes.ok) throw new Error('HTTP '+tRes.status)
    const tj = await tRes.json()
    task.value = tj.task
    labels.value = (tj.task?.labels || tj.task?.Labels || {})
    // reconstruct entries roughly from assignments (group by artifactUrl+startCmd)
    const asgs = (tj.assignments || []) as any[]
    const keyMap = new Map<string, any>()
    for (const a of asgs) {
      const art = a.artifactUrl || a.ArtifactURL
      const cmd = a.startCmd || a.StartCmd
      const node = a.nodeId || a.NodeID
      const key = `${art}|${cmd}`
      if (!keyMap.has(key)) keyMap.set(key, { artifactUrl: art, startCmd: cmd, replicas: {} as Record<string, number> })
      const entry = keyMap.get(key)
      entry.replicas[node] = (entry.replicas[node] || 0) + 1
    }
    entries.value = Array.from(keyMap.values())
    if (aRes.ok) artifacts.value = await aRes.json() as Artifact[]
    if (nRes.ok) nodes.value = await nRes.json() as NodeDTO[]
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}

async function saveLabels() {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/${encodeURIComponent(id)}`, { method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ labels: labels.value, name: task.value?.name || task.value?.Name }) })
    if (!res.ok) throw new Error('HTTP '+res.status)
    ElMessage.success('已保存')
  } catch (e:any) { ElMessage.error(e?.message || '保存失败') }
}

onMounted(load)
</script>

<template>
  <div>
    <h3>任务配置</h3>
    <el-descriptions v-if="task" :column="2" border style="margin-bottom:12px;">
      <el-descriptions-item label="TaskID">{{ task.taskId || task.TaskID }}</el-descriptions-item>
      <el-descriptions-item label="Name">{{ task.name || task.Name }}</el-descriptions-item>
    </el-descriptions>

    <h4>Entries（根据当前 assignments 推导）</h4>
    <el-table :data="entries" v-loading="loading" style="width:100%; margin-bottom:12px;">
      <el-table-column prop="artifactUrl" label="Artifact" />
      <el-table-column prop="startCmd" label="StartCmd" />
      <el-table-column label="Replicas">
        <template #default="{ row }">
          <code>{{ JSON.stringify(row.replicas) }}</code>
        </template>
      </el-table-column>
    </el-table>

    <h4>Labels</h4>
    <div v-for="(v,k) in labels" :key="k" style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
      <el-input :model-value="k" disabled style="flex:1" />
      <el-input v-model="labels[k]" style="flex:2" />
    </div>
    <el-button type="primary" @click="saveLabels">保存</el-button>
  </div>
</template>


