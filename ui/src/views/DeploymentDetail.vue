<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string
const loading = ref(false)
const deployment = ref<any>(null)
const assigns = ref<any[]>([])
const opLoading = ref(false)
const selectedNode = ref<string>('')

const nodesInDeployment = computed(() => {
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
    const res = await fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}`)
    if (!res.ok) throw new Error('HTTP '+res.status)
    const json = await res.json()
    deployment.value = json.deployment
    assigns.value = json.assignments || []
  } catch (e:any) {
    ElMessage.error(e?.message || '加载失败')
  } finally { loading.value = false }
}

onMounted(load)
const { t } = useI18n()

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
    <!-- 部署详情 -->
    <el-card class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deploymentDetail.title') }}</span>
          <div style="display:flex; gap:8px; align-items:center;">
            <el-button type="warning" :loading="opLoading" @click="stopAll">{{ t('deploymentDetail.buttons.stopAll') }}</el-button>
            <el-select v-model="selectedNode" :placeholder="t('common.selectNode')" style="width:200px;">
              <el-option v-for="n in nodesInDeployment" :key="n" :label="n" :value="n" />
            </el-select>
            <el-button type="warning" :loading="opLoading" @click="stopByNode">{{ t('deploymentDetail.buttons.stopByNode') }}</el-button>
          </div>
        </div>
      </template>
      
      <el-descriptions v-if="deployment" :column="2" border style="margin-bottom:16px;">
        <el-descriptions-item :label="t('deploymentDetail.desc.deploymentId')">{{ deployment.deploymentId }}</el-descriptions-item>
        <el-descriptions-item :label="t('deploymentDetail.desc.name')">{{ deployment.name || deployment.Name }}</el-descriptions-item>
        <el-descriptions-item :label="t('deploymentDetail.desc.labels')" :span="2"><code>{{ JSON.stringify(deployment.labels || deployment.Labels || {}) }}</code></el-descriptions-item>
      </el-descriptions>
      
      <el-table :data="assigns" v-loading="loading" style="width:100%">
      <el-table-column :label="t('deploymentDetail.columns.instanceId')" width="300">
        <template #default="{ row }">{{ row.instanceId || row.InstanceID }}</template>
      </el-table-column>
      <el-table-column :label="t('deploymentDetail.columns.nodeId')" width="180">
        <template #default="{ row }">{{ row.nodeId || row.NodeID }}</template>
      </el-table-column>
      <el-table-column :label="t('deploymentDetail.columns.artifact')">
        <template #default="{ row }">{{ row.artifactUrl || row.ArtifactURL }}</template>
      </el-table-column>
      <el-table-column prop="startCmd" :label="t('deploymentDetail.columns.startCmd')" />
      <el-table-column :label="t('deploymentDetail.columns.desired')" width="120">
        <template #default="{ row }">{{ row.desired || row.Desired }}</template>
      </el-table-column>
        <el-table-column :label="t('deploymentDetail.columns.action')" width="220">
          <template #default="{ row }">
            <el-button size="small" type="primary" :disabled="(row.desired||row.Desired)==='Running'" @click="setDesired(row,'Running')">{{ t('common.start') }}</el-button>
            <el-button size="small" type="warning" :disabled="(row.desired||row.Desired)==='Stopped'" @click="setDesired(row,'Stopped')">{{ t('common.stop') }}</el-button>
            <el-popconfirm :title="t('common.confirmDelete')" @confirm="del(row)">
              <template #reference>
                <el-button type="danger" size="small">{{ t('common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>


