<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, Files, VideoPlay, View } from '@element-plus/icons-vue'
import IdDisplay from '../components/IdDisplay.vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()

type WorkflowStep = { stepId?: string; name: string; executor: string; targetKind: string; targetRef: string; payloadJSON?: string; timeoutSec: number; maxRetries: number }
type Workflow = { workflowId: string; name: string; labels?: Record<string,string>; steps: WorkflowStep[] }

const items = ref<Workflow[]>([])
const loading = ref(false)

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/workflows`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json() as Workflow[]
    items.value = Array.isArray(data) ? data : []
  } catch (e:any) { 
    ElMessage.error(e?.message || '加载失败')
    // 确保在错误情况下也重置为安全值
    items.value = []
  }
  finally { loading.value = false }
}

async function run(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/workflows/${encodeURIComponent(id)}?action=run`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const j = await res.json()
    ElMessage.success('已启动运行')
    // 不自动跳转到运行详情，保持在当前列表页
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function viewLatest(workflowId: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/workflow-runs?workflowId=${encodeURIComponent(workflowId)}`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const runs = await res.json() as any[]
    if (runs.length === 0) { ElMessage.info('暂无运行记录'); return }
    const rid = runs[0].runId || runs[0].RunID
    router.push(`/workflow-runs/${rid}`)
  } catch (e:any) { ElMessage.error(e?.message || '查询失败') }
}

async function deleteWorkflow(workflowId: string) {
  try {
    await ElMessageBox.confirm('确定要删除这个工作流吗？删除后无法恢复，包括所有运行历史。', '确认删除', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    const res = await fetch(`${API_BASE}/v1/workflows/${encodeURIComponent(workflowId)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e: any) { 
    if (e !== 'cancel') {
      ElMessage.error(e?.message || '删除失败') 
    }
  }
}

// create dialog
const showCreate = ref(false)
const form = reactive<{ name: string; steps: WorkflowStep[] }>({ name: '', steps: [{ name:'builtin.echo', executor:'embedded', targetKind: '', targetRef: '', timeoutSec: 300, maxRetries: 0 }] })

// 可用任务列表（从任务定义加载，包括内置任务）
const availableTasks = ref<Array<{ name: string; isBuiltin: boolean }>>([])

// 加载任务定义列表（包括内置任务）
async function loadTaskDefinitions() {
  try {
    const res = await fetch(`${API_BASE}/v1/task-defs`)
    if (res.ok) {
      const taskDefs = await res.json() as any[]
      availableTasks.value = taskDefs
        .map(td => ({
          name: td.Name || td.name || '',
          isBuiltin: td.Labels?.builtin === 'true' || td.labels?.builtin === 'true'
        }))
        .filter(t => t.name)
        .sort((a, b) => a.name.localeCompare(b.name))
    }
  } catch (e) {
    console.warn('Failed to load task definitions:', e)
    availableTasks.value = []
  }
}

function openCreate() {
  form.name = ''
  form.steps = [{ name:'builtin.echo', executor:'embedded', targetKind: '', targetRef: '', timeoutSec: 300, maxRetries: 0 }]
  showCreate.value = true
  loadTaskDefinitions()
}

function addStep() { form.steps.push({ name:'builtin.echo', executor:'embedded', targetKind: '', targetRef: '', timeoutSec: 300, maxRetries: 0 }) }
function removeStep(i:number) { form.steps.splice(i,1) }
function onExecutorChange(step: WorkflowStep) {
  if (step.executor === 'service') {
    step.targetKind = 'service'
    step.targetRef = ''
    ;(step as any).serviceVersion = ''
    ;(step as any).serviceProtocol = ''
    ;(step as any).servicePort = ''
    ;(step as any).servicePath = ''
    
    // 如果任务名称不为空，自动尝试填充配置
    if (step.name.trim()) {
      autoFillServiceConfig(step)
    }
  } else {
    step.targetKind = ''
    step.targetRef = ''
    ;(step as any).serviceVersion = ''
    ;(step as any).serviceProtocol = ''
    ;(step as any).servicePort = ''
    ;(step as any).servicePath = ''
  }
}

function onTaskNameChange(step: WorkflowStep) {
  // 如果执行器是 service 且任务名称不为空，自动尝试填充配置
  if (step.executor === 'service' && step.name.trim()) {
    autoFillServiceConfig(step)
  }
}

async function autoFillServiceConfig(step: WorkflowStep) {
  if (!step.name.trim()) {
    ElMessage.warning('请先填写任务名称')
    return
  }
  
  try {
    // 1. 首先尝试从 TaskDefinitions 中查找匹配的定义
    const taskDefsRes = await fetch(`${API_BASE}/v1/task-defs`)
    if (taskDefsRes.ok) {
      const taskDefs = await taskDefsRes.json() as any[]
      const matchedDef = taskDefs.find(td => (td.Name || td.name) === step.name.trim())
      if (matchedDef && matchedDef.Executor === 'service') {
        // 从 TaskDefinition 中获取配置
        step.targetKind = matchedDef.TargetKind || 'service'
        step.targetRef = matchedDef.TargetRef || ''
        if (matchedDef.Labels) {
          ;(step as any).serviceVersion = matchedDef.Labels.serviceVersion || ''
          ;(step as any).serviceProtocol = matchedDef.Labels.serviceProtocol || ''
          ;(step as any).servicePort = matchedDef.Labels.servicePort || ''
          ;(step as any).servicePath = matchedDef.Labels.servicePath || ''
        }
        ElMessage.success('已从任务定义中自动填充配置')
        return
      }
    }
    
    // 2. 如果没有找到 TaskDefinition，尝试从已注册的服务中推断
    const servicesRes = await fetch(`${API_BASE}/v1/services/list`)
    if (servicesRes.ok) {
      const services = await servicesRes.json() as string[]
      const serviceName = step.name.trim().toLowerCase()
      
      // 查找匹配的服务
      const matchedService = services.find(s => s.toLowerCase().includes(serviceName) || serviceName.includes(s.toLowerCase()))
      if (matchedService) {
        // 获取服务端点信息
        const endpointsRes = await fetch(`${API_BASE}/v1/discovery?service=${encodeURIComponent(matchedService)}`)
        if (endpointsRes.ok) {
          const endpoints = await endpointsRes.json() as any[]
          if (endpoints.length > 0) {
            const ep = endpoints[0]
            step.targetKind = 'service'
            step.targetRef = matchedService
            ;(step as any).serviceVersion = ep.version || ''
            ;(step as any).serviceProtocol = ep.protocol || 'http'
            ;(step as any).servicePort = ep.port?.toString() || ''
            ;(step as any).servicePath = `/${matchedService}` // 默认路径
            ElMessage.success(`已从服务 ${matchedService} 中自动填充配置`)
            return
          }
        }
      }
    }
    
    ElMessage.warning('未找到匹配的任务定义或服务，请手动配置')
  } catch (e: any) {
    ElMessage.error('自动填充失败：' + (e?.message || '未知错误'))
  }
}

async function submit() {
  // Validate service executor steps and prepare labels
  for (let i = 0; i < form.steps.length; i++) {
    const step = form.steps[i]
    if (step.executor === 'service' && (!step.targetRef || !step.targetRef.trim())) {
      ElMessage.warning(`步骤 ${i + 1}：service 执行器需要填写服务名称`)
      return
    }
    // Validate payloadJSON is valid JSON if provided
    if (step.payloadJSON && step.payloadJSON.trim()) {
      try {
        JSON.parse(step.payloadJSON.trim())
      } catch {
        ElMessage.error(`步骤 ${i + 1}：Payload 不是合法的 JSON`)
        return
      }
    }
    // Prepare service labels for service executor steps
    if (step.executor === 'service') {
      const labels: Record<string, string> = {}
      const sv = (step as any).serviceVersion
      const sp = (step as any).serviceProtocol
      const port = (step as any).servicePort
      const path = (step as any).servicePath
      if (sv && sv.trim()) labels.serviceVersion = sv.trim()
      if (sp && sp.trim()) labels.serviceProtocol = sp.trim()
      if (port && port.trim()) labels.servicePort = port.trim()
      if (path && path.trim()) labels.servicePath = path.trim()
      ;(step as any).labels = labels
    }
  }
  try {
    const res = await fetch(`${API_BASE}/v1/workflows`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name: form.name, steps: form.steps }) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建')
    showCreate.value = false
    form.name = ''
    form.steps = [{ name:'builtin.echo', executor:'embedded', targetKind: '', targetRef: '', timeoutSec: 300, maxRetries: 0 }]
    load()
  } catch (e:any) { ElMessage.error(e?.message || '创建失败') }
}

// 分页事件处理
function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1 // 重置到第一页
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

onMounted(load)
const { t } = useI18n()

// 统计计算
const totalWorkflows = computed(() => (items.value || []).length)
const totalSteps = computed(() => (items.value || []).reduce((sum, wf) => sum + (wf.steps?.length || 0), 0))

// 计算属性：分页后的数据
const paginatedItems = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return (items.value || []).slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil((items.value || []).length / pageSize.value)
})
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          {{ t('workflows.buttons.refresh') }}
        </el-button>
        <el-button type="success" @click="openCreate">
          <el-icon><Plus /></el-icon>
          {{ t('workflows.buttons.create') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Files /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalWorkflows }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workflows.stats.workflows') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPlay /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalSteps }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('workflows.stats.steps') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 工作流列表表格 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('workflows.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ (items || []).length }} {{ t('workflows.table.items') }}</span>
        </div>
      </template>
      
      <el-table :data="paginatedItems" v-loading="loading" style="width:100%;" stripe>
      <el-table-column :label="t('workflows.columns.workflowId')" width="120">
        <template #default="{ row }">
          <IdDisplay :id="(row as any).workflowId || (row as any).WorkflowID" :length="8" />
        </template>
      </el-table-column>
      <el-table-column :label="t('workflows.columns.name')" width="180">
        <template #default="{ row }">{{ (row as any).name || (row as any).Name }}</template>
      </el-table-column>
      <el-table-column :label="t('workflows.columns.steps')">
        <template #default="{ row }">
            {{ (()=>{ const a = (row as any).steps || (row as any).Steps || []; return Array.isArray(a) ? a.map((s:any)=> s?.name || s?.Name || s?.definitionId || s?.DefinitionID || '').join(' -> ') : '' })() }}
        </template>
      </el-table-column>
      <el-table-column :label="t('common.action')" width="400">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="run(((row as any).workflowId||(row as any).WorkflowID))">{{ t('workflows.buttons.run') }}</el-button>
          <el-button size="small" @click="viewLatest(((row as any).workflowId||(row as any).WorkflowID))">{{ t('workflows.buttons.viewLatest') }}</el-button>
          <el-button size="small" @click="router.push(`/workflows/${(row as any).workflowId||(row as any).WorkflowID}/runs`)">查看所有运行</el-button>
          <el-button size="small" type="danger" @click="deleteWorkflow(((row as any).workflowId||(row as any).WorkflowID))">删除</el-button>
        </template>
      </el-table-column>
      </el-table>
      
      <!-- 分页组件 -->
      <div style="margin-top: 16px; display: flex; justify-content: center;">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="pageSizes"
          :total="(items || []).length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <el-dialog v-model="showCreate" :title="t('workflows.dialog.title')" width="700px">
      <el-form label-width="60px">
        <el-form-item :label="t('workflows.dialog.form.name')"><el-input v-model="form.name" placeholder="workflow 名称" /></el-form-item>
        <el-form-item :label="t('workflows.dialog.form.steps')">
          <div style="display:flex; flex-direction:column; gap:8px; width:100%">
            <div v-for="(s, i) in form.steps" :key="i" style="display:flex; flex-direction:column; gap:8px; padding:12px; border:1px solid #eee; border-radius:4px;">
              <div style="display:flex; gap:8px; align-items:center;">
                <el-select 
                  v-model="s.name" 
                  placeholder="选择或输入任务名称"
                  filterable
                  allow-create
                  style="flex:1"
                  @blur="onTaskNameChange(s)"
                >
                  <el-option-group label="内置任务" v-if="availableTasks.filter(t => t.isBuiltin).length > 0">
                    <el-option
                      v-for="task in availableTasks.filter(t => t.isBuiltin)"
                      :key="task.name"
                      :label="task.name"
                      :value="task.name"
                    >
                      <span style="color: #409EFF">⚡</span> {{ task.name }}
                    </el-option>
                  </el-option-group>
                  <el-option-group label="任务定义" v-if="availableTasks.filter(t => !t.isBuiltin).length > 0">
                    <el-option
                      v-for="task in availableTasks.filter(t => !t.isBuiltin)"
                      :key="task.name"
                      :label="task.name"
                      :value="task.name"
                    />
                  </el-option-group>
                </el-select>
                <el-select v-model="s.executor" style="flex:1" @change="onExecutorChange(s)">
                  <el-option label="embedded" value="embedded" />
                  <el-option label="service" value="service" />
                  <el-option label="os_process" value="os_process" />
                </el-select>
                <el-input v-model.number="s.timeoutSec" :placeholder="t('workflows.dialog.form.timeoutSec')" style="width:90px" />
                <el-input v-model.number="s.maxRetries" :placeholder="t('workflows.dialog.form.maxRetries')" style="width:90px" />
                <el-button size="small" type="danger" @click="removeStep(i)">{{ t('workflows.dialog.form.delete') }}</el-button>
              </div>
              <div v-if="s.executor === 'service'" style="display:flex; flex-direction:column; gap:8px;">
                <div style="display:flex; gap:8px; align-items:center;">
                  <el-input v-model="s.targetKind" placeholder="目标类型，如 service" style="width:150px" />
                  <el-input v-model="s.targetRef" placeholder="服务名称（必填）" style="flex:1" />
                  <el-button size="small" @click="autoFillServiceConfig(s)" :disabled="!s.name.trim()">自动填充</el-button>
                </div>
                <div style="display:flex; gap:8px; align-items:center;">
                  <el-input v-model="(s as any).serviceVersion" placeholder="服务版本（可选）" style="width:120px" />
                  <el-input v-model="(s as any).serviceProtocol" placeholder="协议（可选）" style="width:100px" />
                  <el-input v-model="(s as any).servicePort" placeholder="端口（可选）" style="width:100px" />
                  <el-input v-model="(s as any).servicePath" placeholder="路径，如 /task001（可选）" style="flex:1" />
                </div>
              </div>
              <div style="margin-top: 8px;">
                <div style="font-size:12px; color:#909399; margin-bottom:4px;">
                  Payload（JSON，可选）- 留空则使用任务定义的默认值
                </div>
                <el-input 
                  v-model="s.payloadJSON" 
                  type="textarea" 
                  :rows="3" 
                  placeholder='如: {"seconds": 5} 或留空使用默认值'
                  style="width:100%"
                />
              </div>
            </div>
            <el-button size="small" @click="addStep">{{ t('workflows.dialog.form.addStep') }}</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">{{ t('workflows.dialog.footer.cancel') }}</el-button>
        <el-button type="primary" @click="submit">{{ t('workflows.dialog.footer.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>
