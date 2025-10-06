<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Refresh, View, Delete } from '@element-plus/icons-vue'

// 类型定义
type EmbeddedWorker = {
  WorkerID: string
  NodeID: string
  InstanceID: string
  AppName: string
  AppVersion: string
  GRPCAddress: string
  Tasks: string[]
  Labels: Record<string, string>
  LastSeen: number
}

type HTTPWorker = {
  WorkerID: string
  NodeID: string
  URL: string
  Tasks: string[]
  Labels: Record<string, string>
  Capacity: number
  LastSeen: number
}

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const { t } = useI18n()

// 响应式数据
const loading = ref(false)
const activeTab = ref<'embedded' | 'http'>('embedded')
const embeddedWorkers = ref<EmbeddedWorker[]>([])
const httpWorkers = ref<HTTPWorker[]>([])

// 过滤条件
const searchText = ref('')
const selectedNode = ref('')
const selectedStatus = ref('')

// 分页
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

// 详情弹窗
const showDetailsDialog = ref(false)
const selectedWorker = ref<EmbeddedWorker | HTTPWorker | null>(null)

// 计算属性
const allNodes = computed(() => {
  const nodes = new Set<string>()
  embeddedWorkers.value.forEach(w => nodes.add(w.NodeID))
  httpWorkers.value.forEach(w => nodes.add(w.NodeID))
  return Array.from(nodes).sort()
})

const allApps = computed(() => {
  const apps = new Set<string>()
  embeddedWorkers.value.forEach(w => apps.add(w.AppName))
  return Array.from(apps).sort()
})

// 过滤后的数据
const filteredEmbeddedWorkers = computed(() => {
  let result = embeddedWorkers.value

  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    result = result.filter(w => 
      w.AppName.toLowerCase().includes(search) ||
      w.WorkerID.toLowerCase().includes(search) ||
      w.Tasks.some(task => task.toLowerCase().includes(search))
    )
  }

  if (selectedNode.value) {
    result = result.filter(w => w.NodeID === selectedNode.value)
  }

  if (selectedStatus.value) {
    result = result.filter(w => getWorkerStatus(w) === selectedStatus.value)
  }

  return result
})

const filteredHTTPWorkers = computed(() => {
  let result = httpWorkers.value

  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    result = result.filter(w => 
      w.WorkerID.toLowerCase().includes(search) ||
      w.Tasks.some(task => task.toLowerCase().includes(search))
    )
  }

  if (selectedNode.value) {
    result = result.filter(w => w.NodeID === selectedNode.value)
  }

  if (selectedStatus.value) {
    result = result.filter(w => getWorkerStatus(w) === selectedStatus.value)
  }

  return result
})

// 当前显示的worker列表
const currentWorkers = computed(() => {
  return activeTab.value === 'embedded' ? filteredEmbeddedWorkers.value : filteredHTTPWorkers.value
})

// 分页后的数据
const paginatedWorkers = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return currentWorkers.value.slice(start, end)
})

const totalPages = computed(() => {
  return Math.ceil(currentWorkers.value.length / pageSize.value)
})

// 统计数据
const stats = computed(() => {
  const totalWorkers = embeddedWorkers.value.length + httpWorkers.value.length
  const activeApps = allApps.value.length
  const supportedServices = [...new Set([
    ...embeddedWorkers.value.flatMap(w => w.Tasks),
    ...httpWorkers.value.flatMap(w => w.Tasks)
  ])].length
  
  const healthyWorkers = [
    ...embeddedWorkers.value.filter(w => getWorkerStatus(w) === 'healthy'),
    ...httpWorkers.value.filter(w => getWorkerStatus(w) === 'healthy')
  ].length
  
  const healthRate = totalWorkers > 0 ? Math.round((healthyWorkers / totalWorkers) * 100) : 100

  return {
    totalWorkers,
    activeApps,
    supportedServices,
    healthRate
  }
})

// 方法
function getWorkerStatus(worker: EmbeddedWorker | HTTPWorker): 'healthy' | 'warning' | 'offline' {
  const now = Date.now() / 1000
  const timeDiff = now - worker.LastSeen
  
  if (timeDiff < 30) return 'healthy'
  if (timeDiff < 300) return 'warning' // 5分钟
  return 'offline'
}

function getStatusColor(status: string): string {
  switch (status) {
    case 'healthy': return 'success'
    case 'warning': return 'warning'
    case 'offline': return 'danger'
    default: return 'info'
  }
}

function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  return date.toLocaleString()
}

async function loadEmbeddedWorkers() {
  try {
    const res = await fetch(`${API_BASE}/v1/embedded-workers`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    embeddedWorkers.value = Array.isArray(data) ? data : []
  } catch (e: any) {
    console.error('Failed to load embedded workers:', e)
    embeddedWorkers.value = []
  }
}

async function loadHTTPWorkers() {
  try {
    const res = await fetch(`${API_BASE}/v1/workers`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json()
    httpWorkers.value = Array.isArray(data) ? data : []
  } catch (e: any) {
    console.error('Failed to load HTTP workers:', e)
    httpWorkers.value = []
  }
}

async function load() {
  loading.value = true
  try {
    await Promise.all([loadEmbeddedWorkers(), loadHTTPWorkers()])
  } catch (e: any) {
    ElMessage.error(e?.message || t('workers.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

function showDetails(worker: EmbeddedWorker | HTTPWorker) {
  selectedWorker.value = worker
  showDetailsDialog.value = true
}

async function deleteWorker(worker: EmbeddedWorker | HTTPWorker) {
  try {
    await ElMessageBox.confirm(
      t('workers.confirmDelete'),
      t('common.confirm'),
      {
        confirmButtonText: t('common.ok'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )

    const isEmbedded = 'GRPCAddress' in worker
    const url = isEmbedded 
      ? `${API_BASE}/v1/embedded-workers/${worker.WorkerID}`
      : `${API_BASE}/v1/workers/${worker.WorkerID}`

    const res = await fetch(url, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)

    ElMessage.success(t('workers.messages.deleteSuccess'))
    await load()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e?.message || t('workers.messages.deleteFailed'))
    }
  }
}

function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

function resetFilters() {
  searchText.value = ''
  selectedNode.value = ''
  selectedStatus.value = ''
  currentPage.value = 1
}

onMounted(load)
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          {{ t('workers.buttons.refresh') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Refresh /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ stats.totalWorkers }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workers.stats.totalWorkers') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><View /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ stats.activeApps }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workers.stats.activeApps') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #409EFF); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Delete /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ stats.supportedServices }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workers.stats.supportedServices') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #F56C6C, #E6A23C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Refresh /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ stats.healthRate }}%</span>
          <span style="font-size:12px; color:#909399;">{{ t('workers.stats.healthRate') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 主内容区域 -->
    <el-card>
      <!-- 标签页 -->
      <div style="margin-bottom: 20px;">
        <el-tabs v-model="activeTab" @tab-change="resetFilters">
          <el-tab-pane :label="t('workers.tabs.embedded')" name="embedded" />
          <el-tab-pane :label="t('workers.tabs.http')" name="http" />
        </el-tabs>
      </div>

      <!-- 过滤条件 -->
      <div style="display: flex; gap: 16px; margin-bottom: 20px; flex-wrap: wrap;">
        <el-input
          v-model="searchText"
          :placeholder="t('workers.filters.search')"
          style="width: 200px;"
          clearable
        />
        <el-select
          v-model="selectedNode"
          :placeholder="t('workers.filters.node')"
          style="width: 120px;"
          clearable
        >
          <el-option
            v-for="node in allNodes"
            :key="node"
            :label="node"
            :value="node"
          />
        </el-select>
        <el-select
          v-model="selectedStatus"
          :placeholder="t('workers.filters.status')"
          style="width: 120px;"
          clearable
        >
          <el-option :label="t('workers.status.healthy')" value="healthy" />
          <el-option :label="t('workers.status.warning')" value="warning" />
          <el-option :label="t('workers.status.offline')" value="offline" />
        </el-select>
        <el-button @click="resetFilters">{{ t('common.reset') }}</el-button>
      </div>

      <!-- 工作器列表表格 -->
      <el-table v-loading="loading" :data="paginatedWorkers" style="width: 100%;" stripe>
        <!-- 应用信息列 -->
        <el-table-column :label="t('workers.columns.appInfo')" min-width="160">
          <template #default="{ row }">
            <div v-if="activeTab === 'embedded'">
              <div style="font-weight: 600; color: #303133;">{{ (row as EmbeddedWorker).AppName }}</div>
              <div style="font-size: 12px; color: #909399;">{{ (row as EmbeddedWorker).AppVersion }}</div>
              <div style="font-size: 11px; color: #C0C4CC;">{{ row.WorkerID }}</div>
            </div>
            <div v-else>
              <div style="font-weight: 600; color: #303133;">{{ row.WorkerID }}</div>
              <div style="font-size: 12px; color: #909399;">HTTP Worker</div>
            </div>
          </template>
        </el-table-column>

        <!-- 节点列 -->
        <el-table-column :label="t('workers.columns.node')" width="100">
          <template #default="{ row }">
            <el-tag size="small">{{ row.NodeID }}</el-tag>
          </template>
        </el-table-column>

        <!-- 支持的任务列 -->
        <el-table-column :label="t('workers.columns.supportedTasks')" min-width="200">
          <template #default="{ row }">
            <div style="display: flex; flex-wrap: wrap; gap: 4px; align-items: flex-start; min-height: 32px;">
              <el-tag
                v-for="task in row.Tasks"
                :key="`${activeTab}-${row.WorkerID}-${task}`"
                size="small"
                type="info"
                style="margin: 2px 0; flex-shrink: 0; transition: none;"
              >
                {{ task }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <!-- 状态列 -->
        <el-table-column :label="t('workers.columns.status')" width="100">
          <template #default="{ row }">
            <el-tag
              :type="getStatusColor(getWorkerStatus(row))"
              size="small"
            >
              {{ t(`workers.status.${getWorkerStatus(row)}`) }}
            </el-tag>
          </template>
        </el-table-column>

        <!-- 最后心跳列 -->
        <el-table-column :label="t('workers.columns.lastSeen')" width="150">
          <template #default="{ row }">
            <div style="font-size: 12px; color: #606266;">
              {{ formatTimestamp(row.LastSeen) }}
            </div>
          </template>
        </el-table-column>

        <!-- 操作列 -->
        <el-table-column :label="t('workers.columns.actions')" width="160" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="showDetails(row)">
              {{ t('workers.buttons.details') }}
            </el-button>
            <el-button size="small" type="danger" @click="deleteWorker(row)">
              {{ t('workers.buttons.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div style="margin-top: 16px; display: flex; justify-content: center;">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="pageSizes"
          :total="currentWorkers.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 详情弹窗 -->
    <el-dialog
      v-model="showDetailsDialog"
      :title="t('workers.details.title')"
      width="600px"
    >
      <div v-if="selectedWorker" style="padding: 16px;">
        <!-- 基本信息 -->
        <h4 style="margin: 0 0 12px 0; color: #303133;">{{ t('workers.details.basicInfo') }}</h4>
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="t('workers.details.workerId')">
            {{ selectedWorker.WorkerID }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('workers.details.node')">
            {{ selectedWorker.NodeID }}
          </el-descriptions-item>
          <template v-if="activeTab === 'embedded'">
            <el-descriptions-item :label="t('workers.details.appName')">
              {{ (selectedWorker as EmbeddedWorker).AppName }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('workers.details.version')">
              {{ (selectedWorker as EmbeddedWorker).AppVersion }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('workers.details.instanceId')">
              {{ (selectedWorker as EmbeddedWorker).InstanceID }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('workers.details.grpcAddress')">
              {{ (selectedWorker as EmbeddedWorker).GRPCAddress }}
            </el-descriptions-item>
          </template>
          <template v-else>
            <el-descriptions-item :label="t('workers.details.httpUrl')">
              {{ (selectedWorker as HTTPWorker).URL }}
            </el-descriptions-item>
            <el-descriptions-item :label="t('workers.details.capacity')">
              {{ (selectedWorker as HTTPWorker).Capacity }}
            </el-descriptions-item>
          </template>
          <el-descriptions-item :label="t('workers.details.lastHeartbeat')">
            {{ formatTimestamp(selectedWorker.LastSeen) }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('workers.columns.status')">
            <el-tag :type="getStatusColor(getWorkerStatus(selectedWorker))">
              {{ t(`workers.status.${getWorkerStatus(selectedWorker)}`) }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>

        <!-- 支持的任务 -->
        <h4 style="margin: 20px 0 12px 0; color: #303133;">{{ t('workers.details.supportedTasks') }}</h4>
        <div style="display: flex; flex-wrap: wrap; gap: 8px;">
          <el-tag
            v-for="task in selectedWorker.Tasks"
            :key="task"
            type="success"
          >
            {{ task }}
          </el-tag>
        </div>

        <!-- 标签信息 -->
        <h4 style="margin: 20px 0 12px 0; color: #303133;">{{ t('workers.details.labels') }}</h4>
        <el-descriptions :column="2" border>
          <el-descriptions-item
            v-for="(value, key) in selectedWorker.Labels"
            :key="key"
            :label="key"
          >
            {{ value }}
          </el-descriptions-item>
        </el-descriptions>
      </div>

      <template #footer>
        <el-button @click="showDetailsDialog = false">{{ t('common.close') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.el-card {
  border-radius: 8px;
}

.el-tag {
  border-radius: 4px;
}
</style>
