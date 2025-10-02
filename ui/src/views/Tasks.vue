<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, reactive, computed } from 'vue'
import { ElMessage } from 'element-plus'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''

type Task = {
  TaskID: string
  Name: string
  Executor: string
  TargetKind: string
  TargetRef: string
  State: string
  CreatedAt: number
}

const items = ref<Task[]>([])
const grouped = computed(() => {
  const map = new Map<string, Task[]>()
  const src = Array.isArray(items.value) ? items.value : []
  for (const t of src) {
    const key = (t as any).OriginTaskID && (t as any).OriginTaskID.length > 0 ? (t as any).OriginTaskID : t.TaskID
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(t)
  }
  // 排序组内按创建时间倒序
  const out: { originId: string; runs: Task[] }[] = []
  for (const [k, arr] of map.entries()) {
    arr.sort((a,b)=> (b.CreatedAt||0)-(a.CreatedAt||0))
    out.push({ originId: k, runs: arr })
  }
  // 组排序：按最近一次创建时间倒序
  out.sort((a,b)=> (b.runs[0]?.CreatedAt||0)-(a.runs[0]?.CreatedAt||0))
  return out
})
const loading = ref(false)
let es: EventSource | null = null

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/tasks`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    items.value = await res.json() as Task[]
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

// create dialog
const showCreate = ref(false)
const form = reactive<{ name: string; executor: string; targetKind: string; targetRef: string; payload: string; timeoutSec: number; maxRetries: number; autoStart: boolean }>({ name: '', executor: 'service', targetKind: 'service', targetRef: '', payload: '{}', timeoutSec: 300, maxRetries: 0, autoStart: true })

function resetForm() {
  form.name = ''
  form.executor = 'service'
  form.targetKind = 'service'
  form.targetRef = ''
  form.payload = '{}'
  form.timeoutSec = 300
  form.maxRetries = 0
  form.autoStart = true
}

function openCreate() {
  resetForm()
  showCreate.value = true
}

async function submit() {
  try {
    const payloadObj = JSON.parse(form.payload || '{}')
    const res = await fetch(`${API_BASE}/v1/tasks`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: form.name,
        executor: form.executor,
        targetKind: form.targetKind,
        targetRef: form.targetRef,
        payload: payloadObj,
        timeoutSec: form.timeoutSec,
        maxRetries: form.maxRetries,
        autoStart: form.autoStart,
      })
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建')
    showCreate.value = false
    resetForm()
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || '创建失败')
  }
}
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">刷新</el-button>
      <el-button type="success" @click="openCreate">创建任务</el-button>
    </div>
    <!-- 分组渲染：每组显示一行头，展开看历史 -->
    <el-table v-loading="loading" :data="grouped" style="width:100%; margin-top:12px;">
      <el-table-column type="expand">
        <template #default="{ row }">
          <div>
            <el-table :data="row.runs" size="small" style="width:80%">
              <el-table-column prop="TaskID" label="TaskID" width="300" />
              <el-table-column prop="Name" label="Name" width="180" />
              <el-table-column prop="Executor" label="Executor" width="100" />
              <el-table-column prop="TargetKind" label="TargetKind" width="100" />
              <el-table-column prop="TargetRef" label="TargetRef" />
              <el-table-column prop="State" label="State" width="120" />
              <el-table-column label="Created" width="160">
                <template #default="{ row: rr }">{{ new Date((rr.CreatedAt||0)*1000).toLocaleString() }}</template>
              </el-table-column>
            </el-table>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="Origin" width="160">
        <template #default="{ row }">{{ row.originId }}</template>
      </el-table-column>
      <el-table-column label="Latest Name" width="120">
        <template #default="{ row }">{{ row.runs[0]?.Name }}</template>
      </el-table-column>
      <el-table-column label="Latest TaskID" width="160">
        <template #default="{ row }">{{ row.runs[0]?.TaskID }}</template>
      </el-table-column>
      <el-table-column label="Executor" width="100">
        <template #default="{ row }">{{ row.runs[0]?.Executor }}</template>
      </el-table-column>
      <el-table-column label="TargetKind" width="100">
        <template #default="{ row }">{{ row.runs[0]?.TargetKind }}</template>
      </el-table-column>
      <el-table-column label="TargetRef">
        <template #default="{ row }">{{ row.runs[0]?.TargetRef }}</template>
      </el-table-column>
      <el-table-column label="Latest State" width="100">
        <template #default="{ row }">{{ row.runs[0]?.State }}</template>
      </el-table-column>
      <el-table-column label="Created" width="150">
        <template #default="{ row }">{{ new Date((row.runs[0]?.CreatedAt||0)*1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="Runs" width="60">
        <template #default="{ row }">{{ row.runs.length }}</template>
      </el-table-column>
      <el-table-column label="Action" width="340">
        <template #default="{ row }">
          <el-button size="small" type="primary" :disabled="row.runs[0]?.State!=='Queued'" @click="startTask(row.runs[0]?.TaskID)">Start</el-button>
          <el-button size="small" @click="rerunTask(row.runs[0]?.TaskID)">Rerun</el-button>
          <el-button size="small" type="warning" :disabled="!(row.runs[0]?.State==='Running'||row.runs[0]?.State==='Queued')" @click="cancelTask(row.runs[0]?.TaskID)">Cancel</el-button>
          <el-popconfirm title="确认删除该任务？" @confirm="delTask(row.runs[0]?.TaskID)">
            <template #reference>
              <el-button type="danger" size="small">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" title="创建任务" width="600px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="form.name" placeholder="任务名称" /></el-form-item>
        <el-form-item label="Executor">
          <el-select v-model="form.executor" style="width:100%">
            <el-option label="service" value="service" />
            <el-option label="embedded" value="embedded" />
            <el-option label="os_process" value="os_process" />
          </el-select>
        </el-form-item>
        <el-form-item label="TargetKind">
          <el-select v-model="form.targetKind" style="width:100%">
            <el-option label="service" value="service" />
            <el-option label="deployment" value="deployment" />
            <el-option label="node" value="node" />
          </el-select>
        </el-form-item>
        <el-form-item label="TargetRef"><el-input v-model="form.targetRef" placeholder="如 serviceName 或 deploymentId" /></el-form-item>
        <el-form-item label="Payload(JSON)"><el-input type="textarea" v-model="form.payload" rows="4" /></el-form-item>
        <el-form-item label="TimeoutSec"><el-input v-model.number="form.timeoutSec" /></el-form-item>
        <el-form-item label="MaxRetries"><el-input v-model.number="form.maxRetries" /></el-form-item>
        <el-form-item label="创建后立即执行"><el-checkbox v-model="form.autoStart">Auto Start</el-checkbox></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
