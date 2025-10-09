<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh, Monitor, CircleCheck, CircleClose, Warning, Delete } from '@element-plus/icons-vue'

type NodeDTO = { nodeId: string; ip: string; labels?: Record<string,string>; lastSeen: number; health: string }
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const nodes = ref<NodeDTO[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

// 计算属性：统计信息
const totalNodes = computed(() => nodes.value.length)
const healthyNodes = computed(() => {
  return nodes.value.filter(node => node.health === 'Healthy').length
})
const unhealthyNodes = computed(() => {
  return nodes.value.filter(node => node.health === 'Unhealthy').length
})
const unknownNodes = computed(() => {
  return nodes.value.filter(node => node.health === 'Unknown').length
})

// 计算属性：分页后的数据
const paginatedNodes = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return nodes.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(nodes.value.length / pageSize.value)
})

async function refresh() {
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`${API_BASE}/v1/nodes`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    nodes.value = await res.json() as NodeDTO[]
  } catch (e:any) {
    error.value = e?.message || '请求失败'
    ElMessage.error(e?.message || '请求失败')
  } finally {
    loading.value = false
  }
}

// 清除错误信息
function clearError() {
  error.value = null
}

// 获取健康状态显示信息
function getHealthStatus(health: string) {
  switch (health) {
    case 'Healthy':
      return { text: '健康', type: 'success' as const }
    case 'Unhealthy':
      return { text: '不健康', type: 'danger' as const }
    case 'Unknown':
      return { text: '未知', type: 'warning' as const }
    default:
      return { text: '未知', type: 'info' as const }
  }
}

async function remove(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/nodes/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (res.ok) {
      ElMessage.success('删除成功')
      refresh()
    } else if (res.status === 409) {
      error.value = `无法删除节点 ${id}：该节点上还有正在运行的部署，请先删除相关部署`
    } else {
      error.value = `删除节点 ${id} 失败：HTTP ${res.status}`
    }
  } catch (e: any) {
    error.value = e?.message || `删除节点 ${id} 失败`
    ElMessage.error(e?.message || `删除节点 ${id} 失败`)
  }
}

// 分页事件处理
function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1 // 重置到第一页
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

onMounted(refresh)
const { t } = useI18n()
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="refresh">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Monitor /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalNodes }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('nodes.stats.total') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><CircleCheck /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ healthyNodes }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('nodes.stats.healthy') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #F56C6C, #F78989); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><CircleClose /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ unhealthyNodes }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('nodes.stats.unhealthy') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Warning /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ unknownNodes }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('nodes.stats.unknown') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 错误提示 -->
    <el-alert v-if="error" type="error" :closable="true" @close="clearError" style="margin-bottom:16px;">
      <template #title>{{ t('nodes.error.title') }}</template>
      <template #default>{{ error }}</template>
    </el-alert>

    <!-- 节点列表 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('nodes.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ nodes.length }} {{ t('nodes.table.items') }}</span>
        </div>
      </template>
      
      <el-table v-loading="loading" :data="paginatedNodes" style="width:100%;" stripe>
        <el-table-column prop="nodeId" :label="t('nodes.columns.nodeId')" width="220" />
        <el-table-column prop="ip" :label="t('nodes.columns.ip')" width="180" />
        <el-table-column :label="t('nodes.columns.health')" width="140">
          <template #default="{ row }">
            <el-tag :type="getHealthStatus(row.health).type" size="small">
              <el-icon style="margin-right:4px;">
                <CircleCheck v-if="row.health === 'Healthy'" />
                <CircleClose v-else-if="row.health === 'Unhealthy'" />
                <Warning v-else />
              </el-icon>
              {{ getHealthStatus(row.health).text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('nodes.columns.lastSeen')" width="220">
          <template #default="{ row }">{{ new Date(row.lastSeen*1000).toLocaleString() }}</template>
        </el-table-column>
        <el-table-column :label="t('nodes.columns.action')" width="160" fixed="right">
          <template #default="{ row }">
            <el-popconfirm :title="t('nodes.confirmDelete')" @confirm="remove(row.nodeId)">
              <template #reference>
                <el-button type="danger" size="small">
                  <el-icon><Delete /></el-icon>
                  {{ t('common.delete') }}
                </el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
      
      <!-- 分页组件 -->
      <div style="margin-top: 16px; display: flex; justify-content: center;">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="pageSizes"
          :total="nodes.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>


