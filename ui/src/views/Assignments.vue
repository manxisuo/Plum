<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
type Assignment = { instanceId: string; deploymentId?: string; desired: string; artifactUrl: string; startCmd: string }
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

// SSE subscribe to updates for current node
let es: EventSource | null = null
function openSSE(){
  if (!nodeId.value) return
  closeSSE()
  try {
    es = new EventSource(`${API_BASE}/v1/stream?nodeId=${encodeURIComponent(nodeId.value)}`)
    es.addEventListener('update', () => { fetchAssignments() })
  } catch {}
}
function closeSSE(){
  if (es) { es.close(); es = null }
}
watch(nodeId, () => { openSSE(); fetchAssignments() })
onMounted(() => { openSSE() })
onBeforeUnmount(() => { closeSSE() })
const { t } = useI18n()
</script>

<template>
  <div>
    <el-form inline>
      <el-form-item :label="t('assignments.form.nodeId')">
        <el-select v-model="nodeId" :placeholder="t('common.selectNode')" filterable style="min-width:240px;">
          <el-option v-for="n in nodes" :key="n.nodeId" :label="`${n.nodeId} (${n.ip})`" :value="n.nodeId" />
        </el-select>
      </el-form-item>
      <el-form-item><el-button type="primary" :loading="loading" @click="fetchAssignments">{{ t('common.refresh') }}</el-button></el-form-item>
    </el-form>
    <el-alert v-if="error" type="error" :closable="false" :title="`错误：${error}`" />
    <el-table v-loading="loading" :data="data.items" style="width:100%; margin-top:12px;">
      <el-table-column prop="deploymentId" :label="t('assignments.columns.deployment')" width="200" />
      <el-table-column prop="instanceId" :label="t('assignments.columns.instance')" width="200" />
      <el-table-column prop="desired" :label="t('assignments.columns.desired')" width="80" />
      <el-table-column prop="phase" :label="t('assignments.columns.phase')" width="80" />
      <el-table-column :label="t('assignments.columns.healthy')" width="80">
        <template #default="{ row }">
          <el-tag :type="row.healthy ? 'success' : 'danger'">{{ row.healthy ? 'true' : 'false' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('assignments.columns.lastReportAt')" width="160">
        <template #default="{ row }">{{ row.lastReportAt ? new Date(row.lastReportAt*1000).toLocaleString() : '-' }}</template>
      </el-table-column>
      <el-table-column prop="startCmd" :label="t('assignments.columns.startCmd')" />
      <el-table-column prop="artifactUrl" :label="t('assignments.columns.artifact')" />
      <el-table-column :label="t('common.action')" width="160">
        <template #default="{ row }">
          <el-button size="small" type="primary" :disabled="row.desired==='Running'" @click="setDesired(row.instanceId, 'Running')">{{ t('common.start') }}</el-button>
          <el-button size="small" type="warning" :disabled="row.desired==='Stopped'" @click="setDesired(row.instanceId, 'Stopped')">{{ t('common.stop') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>


