<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string

const run = ref<any>(null)
const steps = ref<any[]>([])
const stepRuns = ref<any[]>([])
const loading = ref(false)

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

onMounted(load)
</script>

<template>
  <div>
    <h3>Workflow Run 详情</h3>
    <el-descriptions v-if="run" :column="2" border style="margin-bottom:12px;">
      <el-descriptions-item label="RunID">{{ run.runId || run.RunID }}</el-descriptions-item>
      <el-descriptions-item label="WorkflowID">{{ run.workflowId || run.WorkflowID }}</el-descriptions-item>
      <el-descriptions-item label="State">{{ run.state || run.State }}</el-descriptions-item>
      <el-descriptions-item label="Created">{{ new Date(((run.createdAt||run.CreatedAt)||0)*1000).toLocaleString() }}</el-descriptions-item>
    </el-descriptions>

    <el-table v-loading="loading" :data="stepRuns" style="width:100%">
      <el-table-column label="#" width="80">
        <template #default="{ row }">{{ row.ord ?? row.Ord }}</template>
      </el-table-column>
      <el-table-column label="Step">
        <template #default="{ row }">
          {{ (steps.find((s:any)=> (s.stepId||s.StepID)===(row.stepId||row.StepID))||{}).name || (row.stepId||row.StepID) }}
        </template>
      </el-table-column>
      <el-table-column label="TaskID" width="320">
        <template #default="{ row }">{{ row.taskId || row.TaskID }}</template>
      </el-table-column>
      <el-table-column label="State" width="120">
        <template #default="{ row }">{{ row.state || row.State }}</template>
      </el-table-column>
    </el-table>
  </div>
</template>
