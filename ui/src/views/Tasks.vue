<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()

type TaskDef = { defId: string; name: string; executor: string; targetKind?: string; targetRef?: string; labels?: Record<string,string>; createdAt?: number }
type TaskRun = { TaskID: string; OriginTaskID?: string; State?: string; CreatedAt?: number }

// 定义视图：defs 列表 + 最近一次运行
const defs = ref<TaskDef[]>([])
const latestByDef = ref<Record<string, { state: string; createdAt: number; taskId: string }>>({})
const loading = ref(false)
let es: EventSource | null = null

async function load() {
  loading.value = true
  try {
    const [dRes, tRes] = await Promise.all([
      fetch(`${API_BASE}/v1/task-defs`),
      fetch(`${API_BASE}/v1/tasks`)
    ])
    if (dRes.ok) defs.value = await dRes.json() as TaskDef[]
    if (tRes.ok) {
      const runs = await tRes.json() as any[]
      const map: Record<string, { state: string; createdAt: number; taskId: string }>= {}
      for (const r of (runs||[])) {
        const defId = r.originTaskId || r.OriginTaskID || ''
        if (!defId) continue
        const created = r.createdAt || r.CreatedAt || 0
        if (!map[defId] || created > map[defId].createdAt) {
          map[defId] = { state: r.state || r.State || '', createdAt: created, taskId: r.taskId || r.TaskID }
        }
      }
      latestByDef.value = map
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function connectSSE() {
  try {
    es?.close()
    es = new EventSource(`${API_BASE}/v1/tasks/stream`)
    es.addEventListener('update', () => load())
  } catch {}
}

onMounted(() => { load(); connectSSE() })
onBeforeUnmount(() => { try { es?.close() } catch {} })

async function delTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

async function startTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/start/${encodeURIComponent(id)}`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已开始')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function rerunTask(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/tasks/rerun/${encodeURIComponent(id)}`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已重跑')
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

// 创建定义（取代创建任务）
const showCreate = ref(false)
const form = reactive<TaskDef>({ defId:'', name:'my.task.echo', executor:'embedded', targetKind:'', targetRef:'', labels:{} })

function resetForm() { form.defId=''; form.name='my.task.echo'; form.executor='embedded'; form.targetKind=''; form.targetRef=''; form.labels={} }
function openCreate() { resetForm(); showCreate.value = true }

async function submit() {
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name: form.name, executor: form.executor, targetKind: form.targetKind, targetRef: form.targetRef, labels: form.labels }) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建定义')
    showCreate.value = false
    load()
  } catch (e:any) { ElMessage.error(e?.message || '创建失败') }
}

async function runDef(defId: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs/${encodeURIComponent(defId)}?action=run`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已触发运行')
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">刷新</el-button>
      <el-button type="success" @click="openCreate">创建任务定义</el-button>
    </div>
    <el-table v-loading="loading" :data="defs" style="width:100%; margin-top:12px;">
      <el-table-column label="DefID" width="320">
        <template #default="{ row }">{{ (row as any).defId || (row as any).DefID }}</template>
      </el-table-column>
      <el-table-column label="Name" width="220">
        <template #default="{ row }">{{ (row as any).name || (row as any).Name }}</template>
      </el-table-column>
      <el-table-column label="Executor" width="120">
        <template #default="{ row }">{{ (row as any).executor || (row as any).Executor }}</template>
      </el-table-column>
      <el-table-column label="Target">
        <template #default="{ row }">{{ ((row as any).targetKind||(row as any).TargetKind)||'' }} {{ ((row as any).targetRef||(row as any).TargetRef)||'' }}</template>
      </el-table-column>
      <el-table-column label="Latest State" width="140">
        <template #default="{ row }">
          {{ latestByDef[(row as any).defId || (row as any).DefID]?.state || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="Latest Time" width="180">
        <template #default="{ row }">
          {{ new Date(((latestByDef[(row as any).defId || (row as any).DefID]?.createdAt)||0)*1000).toLocaleString() }}
        </template>
      </el-table-column>
      <el-table-column label="Action" width="260">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="runDef((row as any).defId || (row as any).DefID)">Run</el-button>
          <el-button size="small" @click="router.push('/tasks/defs/'+((row as any).defId || (row as any).DefID))">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" title="创建任务定义" width="600px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="form.name" placeholder="任务名称（如 my.task.echo）" /></el-form-item>
        <el-form-item label="Executor">
          <el-select v-model="form.executor" style="width:100%">
            <el-option label="embedded" value="embedded" />
            <el-option label="service" value="service" />
            <el-option label="os_process" value="os_process" />
          </el-select>
        </el-form-item>
        <el-form-item label="TargetKind"><el-input v-model="form.targetKind" placeholder="service/deployment/node（可选）" /></el-form-item>
        <el-form-item label="TargetRef"><el-input v-model="form.targetRef" placeholder="如 serviceName（可选）" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
