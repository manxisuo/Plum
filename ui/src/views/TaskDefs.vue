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

// 下拉框数据源
const availableNodes = ref<string[]>([])
const availableApps = ref<Array<{ name: string; online: boolean }>>([])
const availableServices = ref<string[]>([])

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

function openCreate() {
  form.defId=''
  form.name=''
  form.executor='embedded'
  form.targetKind=''
  form.targetRef=''
  form.labels={}
  showCreate.value = true
  loadDropdownData()
}

// Executor ↔ TargetKind linkage
const ALL_KINDS: string[] = ['service','deployment','node','app']
const allowedKinds = computed<string[]>(() => {
  if (form.executor === 'service') return ['service']
  if (form.executor === 'os_process') return ['node']
  if (form.executor === 'embedded') return ['node', 'app']
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
  
  // 禁止使用 builtin.* 前缀
  if (form.name.trim().startsWith('builtin.')) {
    ElMessage.warning('任务名称不能以 "builtin." 开头（保留给系统内置任务）')
    return
  }
  
  // 检查任务名称是否已存在
  const existingDef = items.value.find(d => d.name === form.name.trim())
  if (existingDef) {
    ElMessage.warning('任务名称已存在，请使用其他名称')
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
    // optional service labels helpers
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
    if (!res.ok) {
      if (res.status === 409) {
        ElMessage.error('任务名称已存在')
      } else {
        throw new Error(`HTTP ${res.status}`)
      }
      return
    }
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
          <el-popconfirm 
            v-if="!((row as any).labels?.builtin === 'true' || (row as any).Labels?.builtin === 'true')"
            :title="'确认删除该定义？'" 
            @confirm="onDel(((row as any).defId||(row as any).DefID))"
          >
            <template #reference>
              <el-button size="small" type="danger">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
          <el-tooltip v-else content="内置任务不能删除" placement="top">
            <el-button size="small" type="danger" disabled>{{ t('common.delete') }}</el-button>
          </el-tooltip>
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
        <el-form-item :label="t('taskDefs.dialog.form.targetRef')" required>
          <el-select 
            v-model="form.targetRef" 
            placeholder="选择或输入目标引用"
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
          <el-form-item :label="t('taskDefs.dialog.form.serviceProtocol')"><el-input v-model="(form as any).serviceProtocol" placeholder="http 或 https（可选）" /></el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.servicePort')"><el-input v-model="(form as any).servicePort" placeholder="如 8080（可选）" /></el-form-item>
          <el-form-item :label="t('taskDefs.dialog.form.servicePath')"><el-input v-model="(form as any).servicePath" placeholder="如 /task 或 /tasks/execute（可选）" /></el-form-item>
        </template>
        <el-form-item label="默认Payload(JSON，可选)">
          <el-input type="textarea" v-model="defaultPayloadText" :rows="6" placeholder="{}" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">取消</el-button>
        <el-button type="primary" :disabled="!form.name || !String(form.name).trim().length" @click="submit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>
