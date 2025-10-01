<script setup lang="ts">
import { ref, computed } from 'vue'
const API_BASE = import.meta.env.VITE_API_BASE || ''
type Assignment = { instanceId: string; taskId?: string; desired: string; artifactUrl: string; startCmd: string }
type Assignments = { items: Assignment[] }
type NodeDTO = { nodeId: string; ip: string }
const nodeId = ref('nodeA')
const loading = ref(false)
const error = ref<string | null>(null)
const data = ref<Assignments>({ items: [] })
const nodes = ref<NodeDTO[]>([])
const url = computed(() => `${API_BASE}/v1/assignments?nodeId=${encodeURIComponent(nodeId.value)}`)
async function fetchAssignments(){
  loading.value=true; error.value=null
  try{ const res=await fetch(url.value); if(!res.ok) throw new Error('HTTP '+res.status); data.value=await res.json() }catch(e:any){ error.value=e?.message||'请求失败' } finally{ loading.value=false }
}
fetchAssignments()

async function loadNodes(){
  try {
    const res = await fetch(`${API_BASE}/v1/nodes`)
    if (res.ok) {
      nodes.value = await res.json() as NodeDTO[]
    }
  } catch {}
}
loadNodes()

async function setDesired(id: string, desired: 'Running'|'Stopped') {
  const res = await fetch(`${API_BASE}/v1/assignments/${encodeURIComponent(id)}`, { method:'PATCH', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ desired }) })
  if (!res.ok) return
  fetchAssignments()
}
</script>

<template>
  <div>
    <el-form inline>
      <el-form-item label="Node ID">
        <el-select v-model="nodeId" placeholder="选择节点" filterable style="min-width:240px;">
          <el-option v-for="n in nodes" :key="n.nodeId" :label="`${n.nodeId} (${n.ip})`" :value="n.nodeId" />
        </el-select>
      </el-form-item>
      <el-form-item><el-button type="primary" :loading="loading" @click="fetchAssignments">刷新</el-button></el-form-item>
    </el-form>
    <el-alert v-if="error" type="error" :closable="false" :title="`错误：${error}`" />
    <el-table v-loading="loading" :data="data.items" style="width:100%; margin-top:12px;">
      <el-table-column prop="taskId" label="Task" width="200" />
      <el-table-column prop="instanceId" label="Instance" width="200" />
      <el-table-column prop="desired" label="Desired" width="80" />
      <el-table-column prop="phase" label="Phase" width="80" />
      <el-table-column label="Healthy" width="80">
        <template #default="{ row }">
          <el-tag :type="row.healthy ? 'success' : 'danger'">{{ row.healthy ? 'true' : 'false' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="LastReportAt" width="160">
        <template #default="{ row }">{{ row.lastReportAt ? new Date(row.lastReportAt*1000).toLocaleString() : '-' }}</template>
      </el-table-column>
      <el-table-column prop="startCmd" label="StartCmd" />
      <el-table-column prop="artifactUrl" label="Artifact" />
      <el-table-column label="Action" width="160">
        <template #default="{ row }">
          <el-button size="small" type="primary" :disabled="row.desired==='Running'" @click="setDesired(row.instanceId, 'Running')">Start</el-button>
          <el-button size="small" type="warning" :disabled="row.desired==='Stopped'" @click="setDesired(row.instanceId, 'Stopped')">Stop</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>


