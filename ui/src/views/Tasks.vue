<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, List, Loading, Check, Close, Search, VideoPlay, View, Delete, Warning, Clock, InfoFilled } from '@element-plus/icons-vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()

type TaskDef = { 
  defId?: string; DefID?: string; 
  name?: string; Name?: string; 
  executor?: string; Executor?: string; 
  targetKind?: string; TargetKind?: string; 
  targetRef?: string; TargetRef?: string; 
  labels?: Record<string,string>; 
  createdAt?: number; 
  defaultPayloadJSON?: string; DefaultPayloadJSON?: string 
}
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
const { t } = useI18n()

// 搜索和筛选
const searchText = ref('')
const selectedExecutor = ref('')
const selectedState = ref('')

// 计算属性：过滤后的任务定义
const filteredDefs = computed(() => {
  let result = defs.value
  
  // 按搜索文本过滤
  if (searchText.value.trim()) {
    const search = searchText.value.toLowerCase()
    result = result.filter(def => {
      const name = (def.name || def.Name || '').toLowerCase()
      const defId = (def.defId || def.DefID || '').toLowerCase()
      const executor = (def.executor || def.Executor || '').toLowerCase()
      return name.includes(search) || defId.includes(search) || executor.includes(search)
    })
  }
  
  // 按执行器过滤
  if (selectedExecutor.value) {
    result = result.filter(def => (def.executor || def.Executor) === selectedExecutor.value)
  }
  
  // 按状态过滤
  if (selectedState.value) {
    result = result.filter(def => {
      const defId = def.defId || def.DefID
      if (!defId) return false
      const state = latestByDef.value[defId]?.state
      return state === selectedState.value
    })
  }
  
  return result
})

// 统计计算
const runningCount = computed(() => {
  return Object.values(latestByDef.value).filter(item => item.state === 'Running').length
})

const completedCount = computed(() => {
  return Object.values(latestByDef.value).filter(item => item.state === 'Completed').length
})

const failedCount = computed(() => {
  return Object.values(latestByDef.value).filter(item => item.state === 'Failed').length
})

// 状态标签类型
function getStateTagType(state: string) {
  switch (state) {
    case 'Running': return 'warning'
    case 'Completed': return 'success'
    case 'Succeeded': return 'success'
    case 'Failed': return 'danger'
    case 'Cancelled': return 'info'
    case 'Pending': return 'info'
    default: return ''
  }
}

// 时间格式化
function formatTime(timestamp: number) {
  if (!timestamp) return ''
  return new Date(timestamp * 1000).toLocaleTimeString()
}

function formatDate(timestamp: number) {
  if (!timestamp) return ''
  return new Date(timestamp * 1000).toLocaleDateString()
}

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
const form = reactive<TaskDef>({ defId:'', name:'', executor:'embedded', targetKind:'', targetRef:'', labels:{} })
const defaultPayloadText = ref<string>('')

function resetForm() { form.defId=''; form.name=''; form.executor='embedded'; form.targetKind=''; form.targetRef=''; form.labels={}; defaultPayloadText.value='' }
function openCreate() { resetForm(); showCreate.value = true }

// Executor ↔ TargetKind 约束
const ALL_KINDS: string[] = ['service','deployment','node']
const allowedKinds = computed<string[]>(() => {
  if (form.executor === 'service') return ['service']
  if (form.executor === 'os_process') return ['node']
  // embedded 默认不限：可选 service/deployment/node
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
  if (!form.targetRef || !String(form.targetRef).trim()) {
    ElMessage.warning('请填写目标引用')
    return
  }
  try {
    let defaultPayload: any = undefined
    if (defaultPayloadText.value && defaultPayloadText.value.trim()) {
      try { defaultPayload = JSON.parse(defaultPayloadText.value) } catch { ElMessage.error('默认 Payload 不是合法 JSON'); return }
    }
    const body: any = { name: form.name, executor: form.executor, targetKind: form.targetKind, targetRef: form.targetRef, labels: { ...(form.labels||{}) } }
    if (form.executor === 'service') {
      const sv = (form as any).serviceVersion as string | undefined
      const sp = (form as any).serviceProtocol as string | undefined
      const port = (form as any).servicePort as string | undefined
      const path = (form as any).servicePath as string | undefined
      if (sv && sv.trim()) body.labels.serviceVersion = sv.trim()
      if (sp && sp.trim()) body.labels.serviceProtocol = sp.trim()
      if (port && port.trim()) body.labels.servicePort = port.trim()
      if (path && path.trim()) body.labels.servicePath = path.trim()
    }
    if (defaultPayload !== undefined) body.defaultPayload = defaultPayload
    const res = await fetch(`${API_BASE}/v1/task-defs`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建定义')
    showCreate.value = false
    load()
  } catch (e:any) { ElMessage.error(e?.message || '创建失败') }
}

async function runDef(defId: string) {
  openRun(defId)
}

// Run dialog with payload
const showRun = ref(false)
const runDefId = ref('')
const runPayloadText = ref<string>('{}')
function openRun(defId: string) {
  runDefId.value = defId
  try {
    const def = (defs.value||[]).find((d:any)=> ((d as any).defId||(d as any).DefID) === defId)
    let raw = ''
    const d: any = def as any
    if (d) {
      raw = (d.defaultPayloadJSON || d.DefaultPayloadJSON || '') as string
    }
    if (raw && String(raw).trim().length) {
      try {
        const obj = JSON.parse(String(raw))
        runPayloadText.value = JSON.stringify(obj, null, 2)
      } catch {
        runPayloadText.value = String(raw)
      }
    } else {
      runPayloadText.value = '{}'
    }
  } catch {
    runPayloadText.value = '{}'
  }
  showRun.value = true
}
async function submitRun() {
  let payload: any = {}
  try {
    payload = runPayloadText.value ? JSON.parse(runPayloadText.value) : {}
  } catch {
    ElMessage.error('Payload 不是合法 JSON')
    return
  }
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs/${encodeURIComponent(runDefId.value)}?action=run`, {
      method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ payload })
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已触发运行')
    showRun.value = false
    load()
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
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
</script>

<template>
  <div>
    <!-- 页面标题、统计和操作 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <h2 style="margin:0; font-size:20px; flex-shrink:0;">{{ t('taskDefs.title') }}</h2>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><List /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ defs.length }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('taskDefs.stats.total') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Loading /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ runningCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('taskDefs.stats.running') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Check /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ completedCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('taskDefs.stats.completed') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #F56C6C, #F78989); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Close /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ failedCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('taskDefs.stats.failed') }}</span>
        </div>
      </div>
      
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          {{ t('taskDefs.buttons.refresh') }}
        </el-button>
        <el-button type="success" @click="openCreate">
          <el-icon><Plus /></el-icon>
          {{ t('taskDefs.buttons.create') }}
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div style="display:flex; gap:12px; align-items:center; margin-bottom:16px;">
      <el-input
        v-model="searchText"
        :placeholder="t('taskDefs.search.placeholder')"
        style="width:300px;"
        clearable>
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
      <el-select v-model="selectedExecutor" :placeholder="t('taskDefs.filter.executor')" clearable style="width:150px;">
        <el-option :label="t('taskDefs.filter.all')" value="" />
        <el-option label="embedded" value="embedded" />
        <el-option label="service" value="service" />
        <el-option label="os_process" value="os_process" />
      </el-select>
      <el-select v-model="selectedState" :placeholder="t('taskDefs.filter.state')" clearable style="width:150px;">
        <el-option :label="t('taskDefs.filter.all')" value="" />
        <el-option label="Running" value="Running" />
        <el-option label="Completed" value="Completed" />
        <el-option label="Failed" value="Failed" />
        <el-option label="Cancelled" value="Cancelled" />
      </el-select>
    </div>

    <!-- 任务定义表格 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('taskDefs.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ filteredDefs.length }} {{ t('taskDefs.table.items') }}</span>
        </div>
      </template>
      
      <el-table v-loading="loading" :data="filteredDefs" style="width:100%;" stripe>
      <el-table-column :label="t('taskDefs.columns.defId')" width="280">
        <template #default="{ row }">{{ (row as any).defId || (row as any).DefID }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.name')" width="200">
        <template #default="{ row }">{{ (row as any).name || (row as any).Name }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.executor')" width="120">
        <template #default="{ row }">{{ (row as any).executor || (row as any).Executor }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.target')">
        <template #default="{ row }">{{ ((row as any).targetKind||(row as any).TargetKind)||'' }} {{ ((row as any).targetRef||(row as any).TargetRef)||'' }}</template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.latestState')" width="140">
        <template #default="{ row }">
          <el-tag :type="getStateTagType(latestByDef[(row as any).defId || (row as any).DefID]?.state)" size="small">
            <el-icon style="margin-right:4px;">
              <Loading v-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Running'" />
              <Check v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Completed'" />
              <Check v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Succeeded'" />
              <Close v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Failed'" />
              <Warning v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Cancelled'" />
              <Clock v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Pending'" />
              <InfoFilled v-else />
            </el-icon>
            {{ latestByDef[(row as any).defId || (row as any).DefID]?.state || t('taskDefs.status.neverRun') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.latestTime')" width="120">
        <template #default="{ row }">
          <div v-if="latestByDef[(row as any).defId || (row as any).DefID]?.createdAt">
            <div style="font-size:13px;">{{ formatTime(latestByDef[(row as any).defId || (row as any).DefID]?.createdAt) }}</div>
            <div style="font-size:12px; color:#909399;">{{ formatDate(latestByDef[(row as any).defId || (row as any).DefID]?.createdAt) }}</div>
          </div>
          <span v-else style="color:#C0C4CC;">{{ t('taskDefs.status.neverRun') }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.action')" width="240" fixed="right">
        <template #default="{ row }">
          <div style="display:flex; gap:6px; flex-wrap:wrap;">
            <el-button size="small" type="primary" @click="runDef((row as any).defId || (row as any).DefID)">
              <el-icon><VideoPlay /></el-icon>
              {{ t('taskDefs.buttons.run') }}
            </el-button>
            <el-button size="small" @click="router.push('/tasks/defs/'+((row as any).defId || (row as any).DefID))">
              <el-icon><View /></el-icon>
              {{ t('taskDefs.buttons.details') }}
            </el-button>
            <el-popconfirm :title="t('taskDefs.confirm.delete')" @confirm="onDel(((row as any).defId || (row as any).DefID))">
              <template #reference>
                <el-button size="small" type="danger">
                  <el-icon><Delete /></el-icon>
                  {{ t('common.delete') }}
                </el-button>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </el-table-column>
    </el-table>
    </el-card>

    <el-dialog v-model="showCreate" :title="t('taskDefs.dialog.title')" width="600px">
      <el-form label-width="120px">
        <el-form-item :label="t('taskDefs.dialog.form.name')"><el-input v-model="form.name" placeholder="任务名称（如 my.task.echo）" /></el-form-item>
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
        <el-form-item :label="t('taskDefs.dialog.form.targetRef')" required><el-input v-model="form.targetRef" placeholder="如 serviceName（必填）" /></el-form-item>
        <template v-if="form.executor==='service'">
          <el-form-item :label="t('taskDefs.dialog.form.serviceVersion')"><el-input v-model="(form as any).serviceVersion" placeholder="如 1.0.0（可选）" /></el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.serviceProtocol')"><el-input v-model="(form as any).serviceProtocol" placeholder="http 或 https（可选）" /></el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.servicePort')"><el-input v-model="(form as any).servicePort" placeholder="如 8080（可选）" /></el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.servicePath')"><el-input v-model="(form as any).servicePath" placeholder="如 /task 或 /tasks/execute（可选）" /></el-form-item>
        </template>
        <el-form-item label="默认Payload(JSON，可选)">
          <el-input type="textarea" v-model="defaultPayloadText" :rows="6" placeholder="{}" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">{{ t('taskDefs.dialog.footer.cancel') }}</el-button>
        <el-button type="primary" :disabled="!form.name || !String(form.name).trim().length" @click="submit">{{ t('taskDefs.dialog.footer.submit') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showRun" title="运行任务" width="600px">
      <el-form label-width="120px">
        <el-form-item label="Payload(JSON)">
          <el-input type="textarea" v-model="runPayloadText" :rows="8" placeholder="{}" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRun=false">取消</el-button>
        <el-button type="primary" @click="submitRun">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
