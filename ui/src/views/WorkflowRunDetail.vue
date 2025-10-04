<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import WorkflowDAG from '../components/WorkflowDAG.vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string

const run = ref<any>(null)
const steps = ref<any[]>([])
const stepRuns = ref<any[]>([])
const loading = ref(false)
const refreshTimer = ref<number | null>(null)

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/workflow-runs/${encodeURIComponent(id)}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const j = await res.json()
    run.value = j.run
    steps.value = j.steps||[]
    stepRuns.value = j.stepRuns||[]
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}

// 为DAG组件准备数据 - 显示所有步骤，包括未执行的
const dagSteps = computed(() => {
  return steps.value.map(step => {
    // 查找对应的stepRun
    const stepRun = stepRuns.value.find((sr: any) => (sr.stepId || sr.StepID) === (step.stepId || step.StepID))
    
    return {
      stepId: step.stepId || step.StepID,
      name: step.name || step.Name || step.stepId || step.StepID,
      state: stepRun ? (stepRun.state || stepRun.State) : 'Pending', // 如果没有stepRun，显示为Pending
      ord: step.ord || step.Ord || 0,
      startedAt: stepRun ? (stepRun.startedAt || stepRun.StartedAt) : undefined,
      finishedAt: stepRun ? (stepRun.finishedAt || stepRun.FinishedAt) : undefined
    }
  }).sort((a, b) => a.ord - b.ord)
})

// 检查是否需要继续刷新
const shouldContinueRefresh = computed(() => {
  if (!run.value) return false
  const state = run.value.state || run.value.State
  return state === 'Pending' || state === 'Running'
})

// 自动刷新函数
function startAutoRefresh() {
  if (refreshTimer.value) return
  
  refreshTimer.value = setInterval(() => {
    if (shouldContinueRefresh.value) {
      load()
    } else {
      stopAutoRefresh()
    }
  }, 2000) // 每2秒刷新一次
}

function stopAutoRefresh() {
  if (refreshTimer.value) {
    clearInterval(refreshTimer.value)
    refreshTimer.value = null
  }
}

onMounted(() => {
  load().then(() => {
    if (shouldContinueRefresh.value) {
      startAutoRefresh()
    }
  })
})

onUnmounted(() => {
  stopAutoRefresh()
})

const { t } = useI18n()
</script>

<template>
  <div>
    <!-- 工作流运行详情 -->
    <el-card class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('workflowRun.title') }}</span>
        </div>
      </template>
      
      <el-descriptions v-if="run" :column="2" border>
        <el-descriptions-item :label="t('workflowRun.desc.runId')">{{ run.runId || run.RunID }}</el-descriptions-item>
        <el-descriptions-item :label="t('workflowRun.desc.workflowId')">{{ run.workflowId || run.WorkflowID }}</el-descriptions-item>
        <el-descriptions-item :label="t('workflowRun.desc.state')">{{ run.state || run.State }}</el-descriptions-item>
        <el-descriptions-item :label="t('workflowRun.desc.created')">{{ new Date(((run.createdAt||run.CreatedAt)||0)*1000).toLocaleString() }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 工作流DAG可视化 -->
    <el-card v-if="dagSteps.length > 0" class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>工作流执行流程图</span>
        </div>
      </template>
      
      <WorkflowDAG :steps="dagSteps" :workflow-state="run?.state || run?.State" />
    </el-card>

    <!-- 步骤运行详情 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>步骤运行详情</span>
        </div>
      </template>
      
      <el-table v-loading="loading" :data="stepRuns" style="width:100%">
      <el-table-column :label="t('workflowRun.columns.ord')" width="80">
        <template #default="{ row }">{{ row.ord ?? row.Ord }}</template>
      </el-table-column>
      <el-table-column :label="t('workflowRun.columns.step')">
        <template #default="{ row }">
          {{ (steps.find((s:any)=> (s.stepId||s.StepID)===(row.stepId||row.StepID))||{}).name || (row.stepId||row.StepID) }}
        </template>
      </el-table-column>
      <el-table-column :label="t('workflowRun.columns.taskId')" width="320">
        <template #default="{ row }">{{ row.taskId || row.TaskID }}</template>
      </el-table-column>
        <el-table-column :label="t('workflowRun.columns.state')" width="120">
          <template #default="{ row }">{{ row.state || row.State }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
