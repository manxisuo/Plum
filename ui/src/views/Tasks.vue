<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, reactive } from 'vue'
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

// create dialog
const showCreate = ref(false)
const form = reactive<{ name: string; executor: string; targetKind: string; targetRef: string; payload: string; timeoutSec: number; maxRetries: number }>({ name: '', executor: 'service', targetKind: 'service', targetRef: '', payload: '{}', timeoutSec: 300, maxRetries: 0 })

function resetForm() {
  form.name = ''
  form.executor = 'service'
  form.targetKind = 'service'
  form.targetRef = ''
  form.payload = '{}'
  form.timeoutSec = 300
  form.maxRetries = 0
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
    <el-table v-loading="loading" :data="items" style="width:100%; margin-top:12px;">
      <el-table-column prop="TaskID" label="TaskID" width="320" />
      <el-table-column prop="Name" label="Name" width="200" />
      <el-table-column prop="Executor" label="Executor" width="140" />
      <el-table-column prop="TargetKind" label="TargetKind" width="140" />
      <el-table-column prop="TargetRef" label="TargetRef" />
      <el-table-column prop="State" label="State" width="140" />
      <el-table-column prop="CreatedAt" label="Created" width="160">
        <template #default="{ row }">{{ new Date((row.CreatedAt||0)*1000).toLocaleString() }}</template>
      </el-table-column>
    <el-table-column label="Action" width="140">
      <template #default="{ row }">
        <el-popconfirm title="确认删除该任务？" @confirm="delTask(row.TaskID)">
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
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
