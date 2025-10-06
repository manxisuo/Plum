<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Refresh, ArrowLeft, Files, Check, Close, Loading, Clock, View, Delete } from '@element-plus/icons-vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const router = useRouter()

const workflowId = route.params.workflowId as string
const runs = ref<any[]>([])
const loading = ref(false)

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

// 计算属性：统计信息
const totalRuns = computed(() => runs.value.length)
const succeededCount = computed(() => {
  return runs.value.filter(run => (run.state || run.State) === 'Succeeded').length
})
const failedCount = computed(() => {
  return runs.value.filter(run => (run.state || run.State) === 'Failed').length
})
const runningCount = computed(() => {
  return runs.value.filter(run => (run.state || run.State) === 'Running').length
})

// 计算属性：分页后的数据
const paginatedRuns = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return runs.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(runs.value.length / pageSize.value)
})

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/workflow-runs?workflowId=${encodeURIComponent(workflowId)}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    runs.value = await res.json()
  } catch (e: any) { 
    ElMessage.error(e?.message || '加载失败') 
  }
  finally { 
    loading.value = false 
  }
}

async function deleteRun(runId: string) {
  try {
    await ElMessageBox.confirm('确定要删除这个运行记录吗？删除后无法恢复。', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const res = await fetch(`${API_BASE}/v1/workflow-runs/${encodeURIComponent(runId)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e: any) { 
    if (e !== 'cancel') {
      ElMessage.error(e?.message || '删除失败') 
    }
  }
}

function viewRun(runId: string) {
  router.push(`/workflow-runs/${runId}`)
}

function formatTime(timestamp: number) {
  return new Date(timestamp * 1000).toLocaleString()
}

function getStateTagType(state: string) {
  switch (state) {
    case 'Succeeded': return 'success'
    case 'Failed': return 'danger'
    case 'Running': return 'warning'
    default: return 'info'
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

onMounted(load)
const { t } = useI18n()
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
        <el-button @click="router.push('/workflows')">
          <el-icon><ArrowLeft /></el-icon>
          {{ t('workflowRuns.buttons.back') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Files /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalRuns }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workflowRuns.stats.total') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Check /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ succeededCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workflowRuns.stats.succeeded') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Loading /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ runningCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workflowRuns.stats.running') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #F56C6C, #F78989); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Close /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ failedCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workflowRuns.stats.failed') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 工作流运行历史 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('workflowRuns.title', { workflowId }) }}</span>
          <span style="font-size:14px; color:#909399;">{{ runs.length }} {{ t('workflowRuns.table.items') }}</span>
        </div>
      </template>

      <el-table :data="paginatedRuns" v-loading="loading" style="width:100%;" stripe>
        <el-table-column :label="t('workflowRuns.columns.runId')" width="320">
          <template #default="{ row }">{{ (row as any).runId || (row as any).RunID }}</template>
        </el-table-column>
        <el-table-column :label="t('workflowRuns.columns.state')" width="120">
          <template #default="{ row }">
            <el-tag :type="getStateTagType((row as any).state || (row as any).State)" size="small">
              <el-icon style="margin-right:4px;">
                <Check v-if="((row as any).state || (row as any).State) === 'Succeeded'" />
                <Close v-else-if="((row as any).state || (row as any).State) === 'Failed'" />
                <Loading v-else-if="((row as any).state || (row as any).State) === 'Running'" />
                <Clock v-else />
              </el-icon>
              {{ (row as any).state || (row as any).State }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('workflowRuns.columns.createdAt')" width="200">
          <template #default="{ row }">{{ formatTime((row as any).createdAt || (row as any).CreatedAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('workflowRuns.columns.startedAt')" width="200">
          <template #default="{ row }">
            {{ (row as any).startedAt || (row as any).StartedAt ? formatTime((row as any).startedAt || (row as any).StartedAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column :label="t('workflowRuns.columns.finishedAt')" width="200">
          <template #default="{ row }">
            {{ (row as any).finishedAt || (row as any).FinishedAt ? formatTime((row as any).finishedAt || (row as any).FinishedAt) : '-' }}
          </template>
        </el-table-column>
        <el-table-column :label="t('common.action')" width="200" fixed="right">
          <template #default="{ row }">
            <div style="display:flex; gap:6px; flex-wrap:wrap;">
              <el-button size="small" type="primary" @click="viewRun((row as any).runId || (row as any).RunID)">
                <el-icon><View /></el-icon>
                {{ t('workflowRuns.buttons.view') }}
              </el-button>
              <el-button size="small" type="danger" @click="deleteRun((row as any).runId || (row as any).RunID)">
                <el-icon><Delete /></el-icon>
                {{ t('common.delete') }}
              </el-button>
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
          :total="runs.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>
