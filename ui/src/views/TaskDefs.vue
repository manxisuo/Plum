<script setup lang="ts">
import { ref, reactive, onMounted, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

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
const form = reactive<TaskDef>({ defId:'', name:'', executor:'embedded', targetKind:'', targetRef:'', labels:{} })
const defaultPayloadText = ref<string>('')

function openCreate() {
  form.defId=''
  form.name=''
  form.executor='embedded'
  form.targetKind=''
  form.targetRef=''
  form.labels={}
  showCreate.value = true
}

// Executor ↔ TargetKind linkage
const ALL_KINDS: string[] = ['service','deployment','node']
const allowedKinds = computed<string[]>(() => {
  if (form.executor === 'service') return ['service']
  if (form.executor === 'os_process') return ['node']
  return ALL_KINDS
})
watch(() => form.executor, () => {
  if (!allowedKinds.value.includes((form.targetKind||'') as string)) {
    form.targetKind = ''
  }
})

async function submit() {
  if (!form.name || !String(form.name).trim()) {
    ElMessage.warning('请填写任务名称')
    return
  }
  try {
    let defaultPayload: any = undefined
    if (defaultPayloadText.value && defaultPayloadText.value.trim()) {
      try { defaultPayload = JSON.parse(defaultPayloadText.value) } catch { ElMessage.error('默认 Payload 不是合法 JSON'); return }
    }
    const body: any = { name: form.name, executor: form.executor, targetKind: form.targetKind, targetRef: form.targetRef, labels: form.labels }
    if (defaultPayload !== undefined) body.defaultPayload = defaultPayload
    const res = await fetch(`${API_BASE}/v1/task-defs`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建')
    showCreate.value = false
    load()
  } catch (e:any) { ElMessage.error(e?.message || '创建失败') }
}

async function onDel(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs?id=${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (res.status === 204) { ElMessage.success('已删除'); load(); return }
    if (res.status === 409) {
      const j = await res.json().catch(()=>({}))
      const n = (j && (j as any).referenced) || 0
      ElMessage.error(`有 ${n} 个任务引用该定义，无法删除`)
      return
    }
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
  } catch (e:any) { ElMessage.error(e?.message || '删除失败') }
}

onMounted(load)
const { t } = useI18n()
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">{{ t('taskDefs.buttons.refresh') }}</el-button>
      <el-button type="success" @click="openCreate">{{ t('taskDefs.buttons.create') }}</el-button>
    </div>

    <el-table :data="items" v-loading="loading" style="width:100%; margin-top:12px;">
      <el-table-column :label="t('taskDefs.columns.defId')" width="320">
        <template #default="{ row }">{{ (row as any).defId || (row as any).DefID }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.name')" width="220">
        <template #default="{ row }">{{ (row as any).name || (row as any).Name }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.executor')" width="140">
        <template #default="{ row }">{{ (row as any).executor || (row as any).Executor }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.target')">
        <template #default="{ row }">{{ ((row as any).targetKind||(row as any).TargetKind)||'' }} {{ ((row as any).targetRef||(row as any).TargetRef)||'' }}</template>
      </el-table-column>
      <el-table-column :label="t('common.action')" width="300">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="run(((row as any).defId||(row as any).DefID))">{{ t('taskDefs.buttons.run') }}</el-button>
          <el-button size="small" @click="router.push('/task-defs/'+((row as any).defId||(row as any).DefID))">{{ t('taskDefs.buttons.details') }}</el-button>
          <el-popconfirm :title="'确认删除该定义？'" @confirm="onDel(((row as any).defId||(row as any).DefID))">
            <template #reference>
              <el-button size="small" type="danger">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" :title="t('taskDefs.dialog.title')" width="700px">
      <el-form label-width="120px">
        <el-form-item :label="t('taskDefs.dialog.form.name')"><el-input v-model="form.name" placeholder="task 名称，如 my.task.echo" /></el-form-item>
        <el-form-item :label="t('taskDefs.dialog.form.executor')">
          <el-select v-model="form.executor" style="width:100%">
            <el-option label="embedded" value="embedded" />
            <el-option label="service" value="service" />
            <el-option label="os_process" value="os_process" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('taskDefs.dialog.form.targetKind')">
          <el-select v-model="form.targetKind" clearable :placeholder="allowedKinds.join(' / ')">
            <el-option v-for="k in allowedKinds" :key="k" :label="k" :value="k" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('taskDefs.dialog.form.targetRef')"><el-input v-model="form.targetRef" placeholder="如 serviceName" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">取消</el-button>
        <el-button type="primary" :disabled="!form.name || !String(form.name).trim().length" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
