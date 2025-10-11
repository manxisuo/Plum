<script setup lang="ts">
import { onMounted, onBeforeUnmount, ref, reactive, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, List, Loading, Check, Close, Search, VideoPlay, View, Delete, Warning, Clock, InfoFilled } from '@element-plus/icons-vue'
import IdDisplay from '../components/IdDisplay.vue'

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

// 下拉框数据源
const availableNodes = ref<string[]>([])
const availableApps = ref<Array<{ name: string; online: boolean }>>([])
const availableServices = ref<string[]>([])

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

async function load() {
  loading.value = true
  try {
    const [dRes, tRes] = await Promise.all([
      fetch(`${API_BASE}/v1/task-defs`),
      fetch(`${API_BASE}/v1/tasks`)
    ])
    if (dRes.ok) {
      const data = await dRes.json()
      defs.value = Array.isArray(data) ? data : []
    }
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
    // 确保在错误情况下也重置为安全值
    defs.value = []
    latestByDef.value = {}
  } finally {
    loading.value = false
  }
}

// 加载下拉框数据
async function loadDropdownData() {
  try {
    // 加载节点列表
    const nodesRes = await fetch(`${API_BASE}/v1/nodes`)
    if (nodesRes.ok) {
      const nodes = await nodesRes.json()
      availableNodes.value = nodes.map((n: any) => n.nodeId).filter(Boolean)
    }

    // 加载应用列表（混合方案：应用包 + Worker在线状态）
    const [appsRes, workersRes] = await Promise.all([
      fetch(`${API_BASE}/v1/apps`),
      fetch(`${API_BASE}/v1/embedded-workers`)
    ])
    
    // 获取所有已上传的应用名称
    const appNames = new Set<string>()
    if (appsRes.ok) {
      const apps = await appsRes.json()
      apps.forEach((app: any) => {
        if (app.name) appNames.add(app.name)
      })
    }
    
    // 获取在线Worker的应用名称
    const onlineApps = new Set<string>()
    if (workersRes.ok) {
      const workers = await workersRes.json()
      workers.forEach((w: any) => {
        if (w.AppName || w.appName) {
          onlineApps.add(w.AppName || w.appName)
        }
      })
    }
    
    // 合并信息：所有应用 + 在线标记
    availableApps.value = Array.from(appNames)
      .sort()
      .map(name => ({
        name,
        online: onlineApps.has(name)
      }))

    // 加载服务列表
    const servicesRes = await fetch(`${API_BASE}/v1/services/list`)
    if (servicesRes.ok) {
      availableServices.value = await servicesRes.json()
    }
  } catch (e) {
    console.warn('Failed to load dropdown data:', e)
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
  let result = defs.value || []
  
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

// 计算属性：分页后的数据
const paginatedDefs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredDefs.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(filteredDefs.value.length / pageSize.value)
})

// 统计计算
const runningCount = computed(() => {
  return Object.values(latestByDef.value).filter(item => item.state === 'Running').length
})

const completedCount = computed(() => {
  return Object.values(latestByDef.value).filter(item => item.state === 'Succeeded').length
})

const failedCount = computed(() => {
  return Object.values(latestByDef.value).filter(item => item.state === 'Failed').length
})

// 状态标签类型
function getStateTagType(state: string) {
  switch (state) {
    case 'Running': return 'warning'
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

// ID格式化：显示缩短版
function formatId(id: string, length: number = 8): string {
  if (!id) return ''
  return id.length > length ? id.substring(0, length) : id
}

// 状态翻译函数
function getStateText(state: string) {
  if (!state) return t('taskDefs.status.neverRun')
  switch (state) {
    case 'Running': return t('taskDefs.status.running')
    case 'Succeeded': return t('taskDefs.status.succeeded')
    case 'Failed': return t('taskDefs.status.failed')
    case 'Cancelled': return t('taskDefs.status.cancelled')
    case 'Pending': return t('taskDefs.status.pending')
    default: return state
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

// 更新环境变量名
function updateEnvKey(oldKey: string, newKey: string) {
  if (oldKey === newKey) return
  const value = envVars.value[oldKey]
  delete envVars.value[oldKey]
  envVars.value[newKey] = value
}

function getTargetRefPlaceholder() {
  if (form.executor === 'service') {
    return '如 serviceName（必填）'
  } else if (form.executor === 'os_process') {
    return '节点ID（可选，留空则在controller本地执行）'
  } else if (form.executor === 'embedded') {
    if (form.targetKind === 'node') {
      return '节点ID（可选，留空则选择任意可用节点）'
    } else if (form.targetKind === 'app') {
      return '应用名称（可选，留空则选择任意可用应用）'
    }
    return '目标引用（可选）'
  }
  return '目标引用（可选）'
}

function getTargetKindHelp() {
  if (form.executor === 'embedded' && form.targetKind === 'node') {
    return t('taskDefs.dialog.help.embeddedNode')
  } else if (form.executor === 'embedded' && form.targetKind === 'app') {
    return t('taskDefs.dialog.help.embeddedApp')
  } else if (form.executor === 'service' && form.targetKind === 'service') {
    return t('taskDefs.dialog.help.service')
  } else if (form.executor === 'os_process' && form.targetKind === 'node') {
    return t('taskDefs.dialog.help.osProcessNode')
  }
  return ''
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
const form = reactive<TaskDef>({ defId:'', name:'', executor:'embedded', targetKind:'node', targetRef:'', labels:{} })
const defaultPayloadText = ref<string>('')
const command = ref<string>('')
const envVars = ref<Record<string, string>>({})

function resetForm() { form.defId=''; form.name=''; form.executor='embedded'; form.targetKind='node'; form.targetRef=''; form.labels={}; defaultPayloadText.value=''; command.value=''; envVars.value={} }
function openCreate() { resetForm(); showCreate.value = true; loadDropdownData() }

// Executor ↔ TargetKind 约束
const ALL_KINDS: string[] = ['service','deployment','node','app']
const allowedKinds = computed<string[]>(() => {
  if (form.executor === 'service') return ['service']
  if (form.executor === 'os_process') return ['node']
  if (form.executor === 'embedded') return ['node', 'app'] // embedded只支持node和app
  return ALL_KINDS
})
watch(() => form.executor, () => {
  if (!allowedKinds.value.includes((form.targetKind||'') as string)) {
    form.targetKind = ''
  }
})

// 计算目标引用的选项
const targetRefOptions = computed(() => {
  if (form.executor === 'service' && form.targetKind === 'service') {
    return availableServices.value
  } else if (form.executor === 'embedded' && form.targetKind === 'node') {
    return availableNodes.value
  } else if (form.executor === 'embedded' && form.targetKind === 'app') {
    return availableApps.value
  } else if (form.executor === 'os_process' && form.targetKind === 'node') {
    return availableNodes.value
  }
  return []
})

async function submit() {
  if (!form.name || !String(form.name).trim()) {
    ElMessage.warning('请填写任务名称')
    return
  }
  
  // 禁止使用 builtin.* 前缀
  if (form.name.trim().startsWith('builtin.')) {
    ElMessage.warning('任务名称不能以 "builtin." 开头（保留给系统内置任务）')
    return
  }
  
  // 检查任务名称是否已存在
  const existingDef = defs.value.find(d => 
    (d.name || d.Name) === form.name.trim()
  )
  if (existingDef) {
    ElMessage.warning('任务名称已存在，请使用其他名称')
    return
  }
  
  if (form.executor === 'service' && (!form.targetRef || !String(form.targetRef).trim())) {
    ElMessage.warning('请填写目标引用（服务名称）')
    return
  }
  try {
    let defaultPayload: any = undefined
    if (form.executor === 'os_process' && command.value.trim()) {
      // 为 os_process 自动生成 payload
      defaultPayload = {
        command: command.value.trim()
      }
      // 添加环境变量
      if (Object.keys(envVars.value).length > 0) {
        defaultPayload.env = { ...envVars.value }
      }
    } else if (defaultPayloadText.value && defaultPayloadText.value.trim()) {
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
    if (form.executor === 'os_process') {
      if (command.value && command.value.trim()) {
        body.labels.command = command.value.trim()
      }
    }
    if (defaultPayload !== undefined) body.defaultPayload = defaultPayload
    const res = await fetch(`${API_BASE}/v1/task-defs`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) })
    if (!res.ok) {
      if (res.status === 409) {
        ElMessage.error('任务名称已存在')
      } else {
        throw new Error(`HTTP ${res.status}`)
      }
      return
    }
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
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
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
          <span style="font-size:12px; color:#909399;">{{ t('taskDefs.stats.succeeded') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #F56C6C, #F78989); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Close /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ failedCount }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('taskDefs.stats.failed') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
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
        <el-option label="Pending" value="Pending" />
        <el-option label="Running" value="Running" />
        <el-option label="Succeeded" value="Succeeded" />
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
      
      <el-table v-loading="loading" :data="paginatedDefs" style="width:100%;" stripe>
      <el-table-column :label="t('taskDefs.columns.defId')" width="120">
        <template #default="{ row }">
          <IdDisplay :id="(row as any).defId || (row as any).DefID" :length="8" />
        </template>
      </el-table-column>
      <el-table-column :label="t('taskDefs.columns.name')" width="220">
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
              <Check v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Succeeded'" />
              <Close v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Failed'" />
              <Warning v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Cancelled'" />
              <Clock v-else-if="latestByDef[(row as any).defId || (row as any).DefID]?.state === 'Pending'" />
              <InfoFilled v-else />
            </el-icon>
            {{ getStateText(latestByDef[(row as any).defId || (row as any).DefID]?.state) }}
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
      
      <!-- 分页组件 -->
      <div style="margin-top: 16px; display: flex; justify-content: center;">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="pageSizes"
          :total="filteredDefs.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
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
          <div v-if="form.executor && form.targetKind" style="font-size:12px; color:#909399; margin-top:4px;">
            {{ getTargetKindHelp() }}
          </div>
        </el-form-item>
        <el-form-item :label="t('taskDefs.dialog.form.targetRef')" :required="form.executor === 'service'">
          <el-select 
            v-model="form.targetRef" 
            :placeholder="getTargetRefPlaceholder()"
            clearable
            filterable
            allow-create
            style="width: 100%"
          >
            <el-option
              v-for="option in targetRefOptions"
              :key="typeof option === 'string' ? option : option.name"
              :label="typeof option === 'string' ? option : option.name"
              :value="typeof option === 'string' ? option : option.name"
            >
              <template v-if="typeof option === 'object' && option.name">
                <span :style="{ color: option.online ? '#67C23A' : '#909399' }">
                  {{ option.online ? '●' : '○' }}
                </span>
                {{ option.name }}
                <span v-if="option.online" style="font-size: 12px; color: #67C23A; margin-left: 8px;">(在线)</span>
                <span v-else style="font-size: 12px; color: #909399; margin-left: 8px;">(离线)</span>
              </template>
            </el-option>
          </el-select>
        </el-form-item>
        <template v-if="form.executor==='service'">
          <el-form-item :label="t('taskDefs.dialog.form.serviceVersion')"><el-input v-model="(form as any).serviceVersion" placeholder="如 1.0.0（可选）" /></el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.serviceProtocol')">
            <el-select v-model="(form as any).serviceProtocol" placeholder="选择协议（可选）" clearable style="width: 100%">
              <el-option label="http" value="http" />
              <el-option label="https" value="https" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.servicePort')">
            <el-select v-model="(form as any).servicePort" placeholder="选择端口（可选）" clearable filterable allow-create style="width: 100%">
              <el-option label="80" value="80" />
              <el-option label="443" value="443" />
              <el-option label="8080" value="8080" />
              <el-option label="8443" value="8443" />
              <el-option label="3000" value="3000" />
              <el-option label="5000" value="5000" />
              <el-option label="8000" value="8000" />
              <el-option label="9000" value="9000" />
            </el-select>
          </el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.servicePath')"><el-input v-model="(form as any).servicePath" placeholder="如 /task 或 /tasks/execute（可选）" /></el-form-item>
        </template>
        <template v-if="form.executor==='os_process'">
          <el-form-item :label="t('taskDefs.dialog.form.command')" required>
            <el-input v-model="command" placeholder="如 ls -la（必填）" />
          </el-form-item>
          <el-form-item label="环境变量（可选）">
            <div style="width:100%;">
              <div v-for="(envKey, index) in Object.keys(envVars)" :key="index" style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
                <el-input :model-value="envKey" @update:model-value="(newKey: string) => updateEnvKey(envKey, newKey)" placeholder="变量名" style="flex:1" />
                <span>=</span>
                <el-input v-model="envVars[envKey]" placeholder="变量值" style="flex:1" />
                <el-button size="small" type="danger" @click="delete envVars[envKey]">删除</el-button>
              </div>
              <el-button size="small" type="primary" @click="envVars[`VAR${Object.keys(envVars).length + 1}`] = ''">添加环境变量</el-button>
            </div>
            <div style="font-size:12px; color:#909399; margin-top:8px;">
              提示：GUI 程序需要设置 DISPLAY=:0 环境变量
            </div>
          </el-form-item>
        </template>
        <el-form-item label="默认Payload(JSON)">
          <el-input type="textarea" v-model="defaultPayloadText" :rows="6" placeholder="{}" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">{{ t('taskDefs.dialog.footer.cancel') }}</el-button>
        <el-button type="primary" :disabled="!form.name || !String(form.name).trim().length || (form.executor === 'os_process' && !command.trim()) || (form.executor === 'service' && (!form.targetRef || !String(form.targetRef).trim()))" @click="submit">{{ t('taskDefs.dialog.footer.submit') }}</el-button>
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
