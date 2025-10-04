<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string

const defn = ref<any>(null)
const runs = ref<any[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const [dRes, tRes] = await Promise.all([
      fetch(`${API_BASE}/v1/task-defs/${encodeURIComponent(id)}`),
      fetch(`${API_BASE}/v1/tasks`)
    ])
    if (!dRes.ok) throw new Error('HTTP '+dRes.status)
    defn.value = await dRes.json()
    if (tRes.ok) {
      const arr = await tRes.json() as any[]
      runs.value = (arr||[]).filter(t => (t.originTaskId||t.OriginTaskID) === id)
    }
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}

onMounted(load)
const { t } = useI18n()

async function startTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/start/${encodeURIComponent(id)}`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已开始')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function cancelTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/cancel/${encodeURIComponent(id)}`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已取消')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function deleteTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '删除失败') }
}
</script>

<template>
  <div>
    <!-- 任务定义详情 -->
    <el-card class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('taskDefDetail.title') }}</span>
        </div>
      </template>
      
      <el-descriptions v-if="defn" :column="2" border>
        <el-descriptions-item :label="t('taskDefDetail.desc.defId')">{{ defn.defId || defn.DefID }}</el-descriptions-item>
        <el-descriptions-item :label="t('taskDefDetail.desc.name')">{{ defn.name || defn.Name }}</el-descriptions-item>
        <el-descriptions-item :label="t('taskDefDetail.desc.executor')">{{ defn.executor || defn.Executor }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 运行历史 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('taskDefDetail.runsTitle') }}</span>
        </div>
      </template>
      
      <el-table :data="runs" v-loading="loading" style="width:100%">
      <el-table-column :label="t('taskDefDetail.columns.taskId')" width="320">
        <template #default="{ row }">{{ row.taskId || row.TaskID }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefDetail.columns.state')" width="140">
        <template #default="{ row }">{{ row.state || row.State }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefDetail.columns.created')" width="160">
        <template #default="{ row }">{{ new Date(((row.createdAt||row.CreatedAt)||0)*1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefDetail.columns.result')">
        <template #default="{ row }">
          {{ (()=>{ const s = (row.resultJson||row.ResultJSON)||''; return String(s).length>200 ? String(s).slice(0,200)+'…' : String(s) })() }}
        </template>
      </el-table-column>
        <el-table-column :label="t('common.action')" width="300">
          <template #default="{ row }">
            <el-button size="small" type="primary" :disabled="(row.state||row.State)!=='Queued'" @click="startTask(row.taskId||row.TaskID)">{{ t('taskDefDetail.buttons.start') }}</el-button>
            <el-button size="small" type="warning" :disabled="!((row.state||row.State)==='Running' || (row.state||row.State)==='Queued')" @click="cancelTask(row.taskId||row.TaskID)">{{ t('taskDefDetail.buttons.cancel') }}</el-button>
            <el-popconfirm :title="t('taskDefDetail.confirmDelete')" @confirm="deleteTask(row.taskId||row.TaskID)">
              <template #reference>
                <el-button size="small" type="danger">{{ t('taskDefDetail.buttons.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>
