<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh, Monitor, List, CircleCheck, CircleClose, VideoPlay, VideoPause } from '@element-plus/icons-vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
type Assignment = { instanceId: string; deploymentId?: string; desired: string; artifactUrl: string; startCmd: string; healthy?: boolean; phase?: string; lastReportAt?: number }
type Assignments = { items: Assignment[] }
type NodeDTO = { nodeId: string; ip: string }
const nodeId = ref('nodeA')
const loading = ref(false)
const error = ref<string | null>(null)
const data = ref<Assignments>({ items: [] })
const nodes = ref<NodeDTO[]>([])
const url = computed(() => `${API_BASE}/v1/assignments?nodeId=${encodeURIComponent(nodeId.value)}`)

// 计算属性：统计信息
const runningCount = computed(() => {
  return data.value.items.filter(item => item.desired === 'Running').length
})

const stoppedCount = computed(() => {
  return data.value.items.filter(item => item.desired === 'Stopped').length
})

const healthyCount = computed(() => {
  return data.value.items.filter(item => item.healthy).length
})

async function fetchAssignments(){
  loading.value=true; error.value=null
  try{ 
    const res=await fetch(url.value); 
    if(!res.ok) throw new Error('HTTP '+res.status); 
    data.value=await res.json() 
  } catch(e:any){ 
    error.value=e?.message||'请求失败' 
    ElMessage.error(e?.message || '请求失败')
  } finally{ 
    loading.value=false 
  }
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
  try {
    const res = await fetch(`${API_BASE}/v1/assignments/${encodeURIComponent(id)}`, { 
      method:'PATCH', 
      headers:{'Content-Type':'application/json'}, 
      body: JSON.stringify({ desired }) 
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(`${desired === 'Running' ? '启动' : '停止'}成功`)
    fetchAssignments()
  } catch (e: any) {
    ElMessage.error(e?.message || '操作失败')
  }
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
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮和节点选择 -->
      <div style="display:flex; gap:12px; align-items:center; flex-shrink:0;">
        <el-select v-model="nodeId" :placeholder="t('common.selectNode')" filterable style="min-width:240px;">
          <el-option v-for="n in nodes" :key="n.nodeId" :label="`${n.nodeId} (${n.ip})`" :value="n.nodeId" />
        </el-select>
        <el-button type="primary" :loading="loading" @click="fetchAssignments">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><List /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ data.items.length }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('assignments.stats.total') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPlay /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ runningCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('assignments.stats.running') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPause /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ stoppedCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('assignments.stats.stopped') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><CircleCheck /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ healthyCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('assignments.stats.healthy') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 错误提示 -->
    <el-alert v-if="error" type="error" :closable="true" @close="error = null" style="margin-bottom:16px;">
      <template #title>{{ t('assignments.error.title') }}</template>
      <template #default>{{ error }}</template>
    </el-alert>

    <!-- 实例分配表格 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('assignments.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ data.items.length }} {{ t('assignments.table.items') }}</span>
        </div>
      </template>
      
      <el-table v-loading="loading" :data="data.items" style="width:100%;" stripe>
        <el-table-column prop="deploymentId" :label="t('assignments.columns.deployment')" width="200" />
        <el-table-column prop="instanceId" :label="t('assignments.columns.instance')" width="200" />
        <el-table-column :label="t('assignments.columns.desired')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.desired === 'Running' ? 'success' : 'warning'" size="small">
              <el-icon style="margin-right:4px;">
                <VideoPlay v-if="row.desired === 'Running'" />
                <VideoPause v-else />
              </el-icon>
              {{ row.desired }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="phase" :label="t('assignments.columns.phase')" width="100" />
        <el-table-column :label="t('assignments.columns.healthy')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.healthy ? 'success' : 'danger'" size="small">
              <el-icon style="margin-right:4px;">
                <CircleCheck v-if="row.healthy" />
                <CircleClose v-else />
              </el-icon>
              {{ row.healthy ? 'true' : 'false' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('assignments.columns.lastReportAt')" width="160">
          <template #default="{ row }">{{ row.lastReportAt ? new Date(row.lastReportAt*1000).toLocaleString() : '-' }}</template>
        </el-table-column>
        <el-table-column prop="startCmd" :label="t('assignments.columns.startCmd')" />
        <el-table-column prop="artifactUrl" :label="t('assignments.columns.artifact')" />
        <el-table-column :label="t('common.action')" width="200" fixed="right">
          <template #default="{ row }">
            <div style="display:flex; gap:6px; flex-wrap:wrap;">
              <el-button size="small" type="success" :disabled="row.desired==='Running'" @click="setDesired(row.instanceId, 'Running')">
                <el-icon><VideoPlay /></el-icon>
                {{ t('common.start') }}
              </el-button>
              <el-button size="small" type="warning" :disabled="row.desired==='Stopped'" @click="setDesired(row.instanceId, 'Stopped')">
                <el-icon><VideoPause /></el-icon>
                {{ t('common.stop') }}
              </el-button>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>


