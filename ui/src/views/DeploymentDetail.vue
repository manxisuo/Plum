<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Refresh, DataBoard, Monitor, CircleCheck, VideoPlay, VideoPause, Delete } from '@element-plus/icons-vue'
import IdDisplay from '../components/IdDisplay.vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string
const loading = ref(false)
const deployment = ref<any>(null)
const assigns = ref<any[]>([])
const opLoading = ref(false)
const selectedNode = ref<string>('')

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

const nodesInDeployment = computed(() => {
  const seen = new Set<string>()
  const out: string[] = []
  for (const row of assigns.value) {
    const n = (row.nodeId || row.NodeID) as string
    if (n && !seen.has(n)) { seen.add(n); out.push(n) }
  }
  return out
})

// 计算属性：统计信息
const totalInstances = computed(() => assigns.value.length)
const runningCount = computed(() => {
  return assigns.value.filter(item => (item.desired || item.Desired) === 'Running').length
})
const stoppedCount = computed(() => {
  return assigns.value.filter(item => (item.desired || item.Desired) === 'Stopped').length
})
const healthyCount = computed(() => {
  return assigns.value.filter(item => item.healthy).length
})

// 计算属性：分页后的数据
const paginatedAssigns = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return assigns.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(assigns.value.length / pageSize.value)
})

// 分页事件处理
function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1 // 重置到第一页
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

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

async function startDeployment() {
  try {
    opLoading.value = true
    const res = await fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}?action=start`, { method: 'POST' })
    if (!res.ok) throw new Error('HTTP ' + res.status)
    ElMessage.success(t('deployments.startSuccess'))
    load()
  } catch (e: any) {
    ElMessage.error(e?.message || t('deployments.startFailed'))
  } finally {
    opLoading.value = false
  }
}

async function stopDeployment() {
  try {
    opLoading.value = true
    const res = await fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}?action=stop`, { method: 'POST' })
    if (!res.ok) throw new Error('HTTP ' + res.status)
    ElMessage.success(t('deployments.stopSuccess'))
    load()
  } catch (e: any) {
    ElMessage.error(e?.message || t('deployments.stopFailed'))
  } finally {
    opLoading.value = false
  }
}
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
        <el-button 
          v-if="deployment?.status === 'Stopped'"
          type="success" 
          :loading="opLoading" 
          @click="startDeployment">
          <el-icon><VideoPlay /></el-icon>
          {{ t('deployments.buttons.start') }}
        </el-button>
        <el-button 
          v-else
          type="warning" 
          :loading="opLoading" 
          @click="stopDeployment">
          <el-icon><VideoPause /></el-icon>
          {{ t('deployments.buttons.stop') }}
        </el-button>
        <el-divider direction="vertical" />
        <el-button type="warning" :loading="opLoading" @click="stopAll">
          <el-icon><VideoPause /></el-icon>
          {{ t('deploymentDetail.buttons.stopAll') }}
        </el-button>
        <el-select v-model="selectedNode" :placeholder="t('common.selectNode')" style="width:180px;" size="default">
          <el-option v-for="n in nodesInDeployment" :key="n" :label="n" :value="n" />
        </el-select>
        <el-button type="warning" :loading="opLoading" @click="stopByNode">
          <el-icon><VideoPause /></el-icon>
          {{ t('deploymentDetail.buttons.stopByNode') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><DataBoard /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalInstances }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentDetail.stats.instances') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPlay /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ runningCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentDetail.stats.running') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPause /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ stoppedCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentDetail.stats.stopped') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><CircleCheck /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ healthyCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentDetail.stats.healthy') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 部署详情 -->
    <el-card class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deploymentDetail.title') }}</span>
        </div>
      </template>
      
      <el-descriptions v-if="deployment" :column="2" border style="margin-bottom:16px;">
        <el-descriptions-item :label="t('deploymentDetail.desc.deploymentId')">{{ deployment.deploymentId || deployment.DeploymentID }}</el-descriptions-item>
        <el-descriptions-item :label="t('deploymentDetail.desc.name')">{{ deployment.name || deployment.Name }}</el-descriptions-item>
        <el-descriptions-item :label="t('deploymentDetail.desc.labels')" :span="2"><code>{{ JSON.stringify(deployment.labels || deployment.Labels || {}) }}</code></el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 实例列表 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deploymentDetail.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ assigns.length }} {{ t('deploymentDetail.table.items') }}</span>
        </div>
      </template>
      
      <el-table :data="paginatedAssigns" v-loading="loading" style="width:100%" stripe>
        <el-table-column :label="t('deploymentDetail.columns.app')" width="200" min-width="150">
          <template #default="{ row }">
            <span v-if="row.appName || row.appVersion || row.AppName || row.AppVersion">
              {{ (row.appName || row.AppName || '-') }}:{{ (row.appVersion || row.AppVersion || '-') }}
            </span>
            <span v-else style="color: #909399;">-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('deploymentDetail.columns.instanceId')" width="120">
          <template #default="{ row }">
            <IdDisplay :id="row.instanceId || row.InstanceID" :length="8" />
          </template>
        </el-table-column>
        <el-table-column :label="t('deploymentDetail.columns.nodeId')" width="160">
          <template #default="{ row }">{{ row.nodeId || row.NodeID }}</template>
        </el-table-column>
        <el-table-column :label="t('deploymentDetail.columns.artifact')" width="320">
          <template #default="{ row }">
            <span v-if="(row.artifactType || row.ArtifactType) === 'image' && (row.imageRepository || row.ImageRepository) && (row.imageTag || row.ImageTag)">
              <el-tag type="info" size="small" style="margin-right: 4px;">镜像</el-tag>
              {{ (row.imageRepository || row.ImageRepository) }}:{{ (row.imageTag || row.ImageTag) }}
            </span>
            <span v-else-if="(row.artifactUrl || row.ArtifactURL) && (row.artifactUrl || row.ArtifactURL).startsWith('image://')">
              <el-tag type="info" size="small" style="margin-right: 4px;">镜像</el-tag>
              <span style="color: #909399; font-family: monospace;">{{ row.artifactUrl || row.ArtifactURL }}</span>
            </span>
            <span v-else>{{ row.artifactUrl || row.ArtifactURL || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="startCmd" :label="t('deploymentDetail.columns.startCmd')" />
        <el-table-column :label="t('deploymentDetail.columns.desired')" width="130">
          <template #default="{ row }">
            <el-tag :type="(row.desired || row.Desired) === 'Running' ? 'success' : 'warning'" size="small">
              <el-icon style="margin-right:4px;">
                <VideoPlay v-if="(row.desired || row.Desired) === 'Running'" />
                <VideoPause v-else />
              </el-icon>
              {{ row.desired || row.Desired }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('deploymentDetail.columns.action')" width="280" fixed="right">
          <template #default="{ row }">
            <div style="display:flex; gap:6px; flex-wrap:wrap;">
              <el-button size="small" type="success" :disabled="(row.desired||row.Desired)==='Running'" @click="setDesired(row,'Running')">
                <el-icon><VideoPlay /></el-icon>
                {{ t('common.start') }}
              </el-button>
              <el-button size="small" type="warning" :disabled="(row.desired||row.Desired)==='Stopped'" @click="setDesired(row,'Stopped')">
                <el-icon><VideoPause /></el-icon>
                {{ t('common.stop') }}
              </el-button>
              <el-popconfirm :title="t('common.confirmDelete')" @confirm="del(row)">
                <template #reference>
                  <el-button type="danger" size="small">
                    <el-icon><Delete /></el-icon>
                    {{ t('common.delete') }}
                  </el-button>
                </template>
              </el-popconfirm>
            </div>
          </template>
        </el-table-column>
      </el-table>
      
      <!-- 分页组件 -->
      <div style="margin-top: 16px; display: flex; justify-content: center;">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="pageSizes"
          :total="assigns.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>


