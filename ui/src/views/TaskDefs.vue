<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()

type TaskDef = { defId: string; name: string; executor: string; targetKind?: string; targetRef?: string; labels?: Record<string,string>; createdAt?: number }

const items = ref<TaskDef[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    items.value = await res.json() as TaskDef[]
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}

async function run(defId: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs/${encodeURIComponent(defId)}?action=run`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const j = await res.json()
    ElMessage.success('已创建运行')
    router.push('/tasks')
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

const showCreate = ref(false)
const form = reactive<TaskDef>({ defId:'', name:'my.task.echo', executor:'embedded', targetKind:'', targetRef:'', labels:{} })

function openCreate() {
  form.defId=''
  form.name='my.task.echo'
  form.executor='embedded'
  form.targetKind=''
  form.targetRef=''
  form.labels={}
  showCreate.value = true
}

async function submit() {
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name: form.name, executor: form.executor, targetKind: form.targetKind, targetRef: form.targetRef, labels: form.labels }) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建')
    showCreate.value = false
    load()
  } catch (e:any) { ElMessage.error(e?.message || '创建失败') }
}

onMounted(load)
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">刷新</el-button>
      <el-button type="success" @click="openCreate">创建定义</el-button>
    </div>

    <el-table :data="items" v-loading="loading" style="width:100%; margin-top:12px;">
      <el-table-column label="DefID" width="320">
        <template #default="{ row }">{{ (row as any).defId || (row as any).DefID }}</template>
      </el-table-column>
      <el-table-column label="Name" width="220">
        <template #default="{ row }">{{ (row as any).name || (row as any).Name }}</template>
      </el-table-column>
      <el-table-column label="Executor" width="140">
        <template #default="{ row }">{{ (row as any).executor || (row as any).Executor }}</template>
      </el-table-column>
      <el-table-column label="Target">
        <template #default="{ row }">{{ ((row as any).targetKind||(row as any).TargetKind)||'' }} {{ ((row as any).targetRef||(row as any).TargetRef)||'' }}</template>
      </el-table-column>
      <el-table-column label="Action" width="220">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="run(((row as any).defId||(row as any).DefID))">Run</el-button>
          <el-button size="small" @click="router.push('/task-defs/'+((row as any).defId||(row as any).DefID))">详情</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" title="创建 TaskDefinition" width="700px">
      <el-form label-width="120px">
        <el-form-item label="Name"><el-input v-model="form.name" placeholder="task 名称，如 my.task.echo" /></el-form-item>
        <el-form-item label="Executor">
          <el-select v-model="form.executor" style="width:100%">
            <el-option label="embedded" value="embedded" />
            <el-option label="service" value="service" />
            <el-option label="os_process" value="os_process" />
          </el-select>
        </el-form-item>
        <el-form-item label="TargetKind"><el-input v-model="form.targetKind" placeholder="service/deployment/node" /></el-form-item>
        <el-form-item label="TargetRef"><el-input v-model="form.targetRef" placeholder="如 serviceName" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">取消</el-button>
        <el-button type="primary" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
