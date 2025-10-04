<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''

type Resource = {
  ResourceID: string
  NodeID: string
  Type: string
  URL: string
  LastSeen: number
  CreatedAt: number
  StateDesc: Array<{
    Type: string
    Name: string
    Value: string
    Unit: string
  }>
  OpDesc: Array<{
    Type: string
    Name: string
    Value: string
    Unit: string
    Min: string
    Max: string
  }>
}

const resources = ref<Resource[]>([])
const loading = ref(false)
const selectedResource = ref<Resource | null>(null)
const resourceStates = ref<any[]>([])
const showDescriptionsView = ref(false)
const descriptionsResource = ref<Resource | null>(null)
const showOperationDialog = ref(false)
const operationResource = ref<Resource | null>(null)
const operationForm = ref<Record<string, string>>({})

let es: EventSource | null = null

// 计算属性：获取当前选中资源的状态描述列表
const currentStateDesc = computed(() => {
  if (!selectedResource.value) return []
  return selectedResource.value.StateDesc || []
})

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/resources`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json() as Resource[]
    console.log('Loaded resources:', data) // 调试日志
    resources.value = data
  } catch (e: any) {
    console.error('Load error:', e) // 调试日志
    ElMessage.error(e?.message || t('resources.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadResourceStates(ResourceID: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/resources/states?resourceId=${encodeURIComponent(ResourceID)}&limit=20`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const data = await res.json() as any[]
    resourceStates.value = data
  } catch (e: any) {
    ElMessage.error(e?.message || t('resources.messages.stateLoadFailed'))
  }
}

async function deleteResource(resourceId: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/resources/${encodeURIComponent(resourceId)}`, {
      method: 'DELETE'
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(t('resources.messages.deleteSuccess'))
    load()
  } catch (e: any) {
    ElMessage.error(e?.message || t('resources.messages.deleteFailed'))
  }
}

async function sendOperation(resource: Resource, operation: any) {
  try {
    const res = await fetch(`${API_BASE}/v1/resources/operation?resourceId=${encodeURIComponent(resource.ResourceID)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        operations: [operation]
      })
    })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(t('resources.messages.operationSent'))
  } catch (e: any) {
    ElMessage.error(e?.message || t('resources.messages.operationFailed'))
  }
}

function connectSSE() {
  try {
    es?.close()
    // 暂时禁用SSE以避免连接问题
    console.log('SSE disabled for debugging')
  } catch (e) {
    console.warn('SSE connection failed:', e)
  }
}

function selectResource(resource: Resource) {
  selectedResource.value = resource
  showDescriptionsView.value = false
  loadResourceStates(resource.ResourceID)
}

function showDescriptions(resource: Resource) {
  descriptionsResource.value = resource
  showDescriptionsView.value = true
  selectedResource.value = null
}

function showOperations(resource: Resource) {
  operationResource.value = resource
  showOperationDialog.value = true
  
  // 初始化操作表单，使用默认值
  operationForm.value = {}
  if (resource.OpDesc) {
    resource.OpDesc.forEach(op => {
      operationForm.value[op.Name] = op.Value
    })
  }
}

function closeOperationDialog() {
  showOperationDialog.value = false
  operationResource.value = null
  operationForm.value = {}
}

async function submitSingleOperation(operationName: string) {
  if (!operationResource.value) return
  
  try {
    const operation = {
      name: operationName,
      value: String(operationForm.value[operationName])
    }
    
    // 验证范围
    const opDesc = operationResource.value.OpDesc.find(op => op.Name === operationName)
    if (opDesc && !validateRange(operation.value, opDesc.Type, opDesc.Min, opDesc.Max)) {
      ElMessage.error(t('resources.validation.rangeError'))
      return
    }
    
    const res = await fetch(`${API_BASE}/v1/resources/operation?resourceId=${encodeURIComponent(operationResource.value.ResourceID)}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        resourceId: operationResource.value.ResourceID,
        timestamp: Math.floor(Date.now() / 1000),
        operations: [operation]
      })
    })
    
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(`${operationName}: ${t('resources.messages.operationSent')}`)
  } catch (e: any) {
    ElMessage.error(e?.message || t('resources.messages.operationFailed'))
  }
}

function formatTimestamp(timestamp: number) {
  if (!timestamp || isNaN(timestamp)) {
    return 'Invalid Date'
  }
  // 如果时间戳看起来像毫秒（大于某个阈值），直接使用
  // 否则乘以1000转换为毫秒
  const ms = timestamp > 1000000000000 ? timestamp : timestamp * 1000
  return new Date(ms).toLocaleString()
}

// 获取操作名称（不包含范围）
function getOperationDisplayName(op: any) {
  return op.Name
}

// 验证输入值是否在范围内
function validateRange(value: string, type: string, min?: string, max?: string): boolean {
  if (!min || !max) return true
  
  try {
    if (type === 'INT') {
      const num = parseInt(value)
      return num >= parseInt(min) && num <= parseInt(max)
    } else if (type === 'DOUBLE') {
      const num = parseFloat(value)
      return num >= parseFloat(min) && num <= parseFloat(max)
    }
  } catch (e) {
    return false
  }
  return true
}

// 获取布尔类型的选择项
function getBooleanOptions() {
  return [
    { label: 'true', value: 'true' },
    { label: 'false', value: 'false' }
  ]
}

// 获取枚举类型的选择项
function getEnumOptions(min?: string, max?: string) {
  if (!min || !max) return []
  
  // 简单的枚举处理，可以根据实际需求扩展
  const options = []
  try {
    const minVal = parseInt(min)
    const maxVal = parseInt(max)
    for (let i = minVal; i <= maxVal; i++) {
      options.push({ label: i.toString(), value: i.toString() })
    }
  } catch (e) {
    // 如果不是数字范围，返回空数组
  }
  return options
}

function getHealthStatus(lastSeen: number) {
  const now = Date.now() / 1000
  const diff = now - lastSeen
  if (diff < 30) return { status: 'healthy', type: 'success', text: t('resources.status.healthy') }
  if (diff < 120) return { status: 'warning', type: 'warning', text: t('resources.status.warning') }
  return { status: 'error', type: 'danger', text: t('resources.status.offline') }
}

onMounted(() => {
  load()
  connectSSE()
})

onBeforeUnmount(() => {
  if (es) {
    es.close()
    es = null
  }
})

const { t } = useI18n()
</script>

<template>
  <div>
    <div style="display:flex; gap:16px; height:640px;">
      <!-- 资源列表 -->
      <el-card style="flex:1.1;">
        <template #header>
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span>{{ t('resources.sections.resourceList') }}</span>
            <el-button type="primary" :loading="loading" @click="load">{{ t('resources.buttons.refresh') }}</el-button>
          </div>
        </template>
        <el-table :data="resources" v-loading="loading" style="width:100%;">
          <el-table-column prop="ResourceID" :label="t('resources.columns.resourceId')" width="120" />
          <el-table-column prop="Type" :label="t('resources.columns.type')" width="90" />
          <el-table-column prop="NodeID" :label="t('resources.columns.nodeId')" width="90" />
          <el-table-column :label="t('resources.columns.status')" width="70">
            <template #default="{ row }">
              <el-tag :type="getHealthStatus(row.LastSeen).type" size="small">
                {{ getHealthStatus(row.LastSeen).text }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('resources.columns.action')" width="280">
            <template #default="{ row }">
              <el-button size="small" @click="selectResource(row)">{{ t('common.details') }}</el-button>
              <el-button size="small" type="info" @click="showDescriptions(row)">{{ t('common.descriptions') }}</el-button>
              <el-button size="small" type="warning" @click="showOperations(row)">{{ t('resources.buttons.operation') }}</el-button>
              <el-button size="small" type="danger" @click="deleteResource(row.ResourceID)">{{ t('resources.buttons.delete') }}</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>

      <!-- 右侧面板 -->
      <el-card style="flex:1;" v-if="selectedResource && !showDescriptionsView">
        <template #header>
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span>{{ t('resources.sections.resourceDetail') }} - {{ selectedResource.ResourceID }}</span>
          </div>
        </template>
        <el-descriptions :column="2" border style="margin-bottom:16px;">
          <el-descriptions-item :label="t('resources.desc.resourceId')">{{ selectedResource.ResourceID }}</el-descriptions-item>
          <el-descriptions-item :label="t('resources.desc.type')">{{ selectedResource.Type }}</el-descriptions-item>
          <el-descriptions-item :label="t('resources.desc.nodeId')">{{ selectedResource.NodeID }}</el-descriptions-item>
          <el-descriptions-item :label="t('resources.desc.status')">
            <el-tag :type="getHealthStatus(selectedResource.LastSeen).type">
              {{ getHealthStatus(selectedResource.LastSeen).text }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('resources.desc.createdAt')">{{ formatTimestamp(selectedResource.CreatedAt) }}</el-descriptions-item>
          <el-descriptions-item :label="t('resources.desc.lastHeartbeat')">{{ formatTimestamp(selectedResource.LastSeen) }}</el-descriptions-item>
        </el-descriptions>

        <!-- 历史状态 -->
        <el-card class="box-card" style="margin-top: 16px;">
          <template #header>
            <div style="display:flex; justify-content:space-between; align-items:center;">
              <span>{{ t('resources.sections.historyStates') }}</span>
            </div>
          </template>
          <div style="height: 290px; overflow-y: auto;">
            <el-table :data="resourceStates" size="small" style="width:100%;" :max-height="280">
              <!-- 时间列 -->
              <el-table-column :label="t('resources.columns.time')" width="130" fixed="left">
                <template #default="{ row }">
                  {{ formatTimestamp(row.Timestamp) }}
                </template>
              </el-table-column>
              
              <!-- 动态状态列 -->
              <el-table-column 
                v-for="stateDesc in currentStateDesc" 
                :key="stateDesc.Name"
                :label="stateDesc.Unit ? `${stateDesc.Name} (${stateDesc.Unit})` : stateDesc.Name" 
                :width="120"
                :show-overflow-tooltip="true">
                <template #default="{ row }">
                  {{ row.States[stateDesc.Name] || '-' }}
                </template>
              </el-table-column>
              
              <!-- 如果没有状态描述，显示原始JSON -->
              <el-table-column 
                v-if="currentStateDesc.length === 0"
                :label="t('resources.columns.stateData')">
                <template #default="{ row }">
                  <pre style="font-size:12px; margin:0;">{{ JSON.stringify(row.States, null, 2) }}</pre>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-card>
      </el-card>

      <!-- 资源描述视图 -->
      <el-card style="flex:1;" v-else-if="showDescriptionsView && descriptionsResource">
        <template #header>
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span>{{ t('resources.sections.resourceDescription') }} - {{ descriptionsResource.ResourceID }}</span>
          </div>
        </template>
          <!-- 状态描述 -->
          <el-card class="box-card" style="height: 260px; margin-bottom: 16px;">
            <template #header>
              <div style="display:flex; justify-content:space-between; align-items:center;">
                <span>{{ t('resources.sections.stateDescription') }}</span>
              </div>
            </template>
            <div style="height: 200px; overflow-y: auto;">
              <el-table :data="descriptionsResource.StateDesc" size="small" style="width:100%;">
                <el-table-column prop="Name" :label="t('resources.columns.name')" />
                <el-table-column prop="Type" :label="t('resources.columns.dataType')" />
                <el-table-column prop="Value" :label="t('resources.columns.defaultValue')" />
                <el-table-column prop="Unit" :label="t('resources.columns.unit')" />
              </el-table>
            </div>
          </el-card>

          <!-- 操作描述 -->
          <el-card class="box-card" style="height: 260px;">
            <template #header>
              <div style="display:flex; justify-content:space-between; align-items:center;">
                <span>{{ t('resources.sections.operationDescription') }}</span>
              </div>
            </template>
            <div style="height: 200px; overflow-y: auto;">
              <el-table :data="descriptionsResource.OpDesc" size="small" style="width:100%;">
                <el-table-column prop="Name" :label="t('resources.columns.name')" />
                <el-table-column prop="Type" :label="t('resources.columns.dataType')" />
                <el-table-column prop="Value" :label="t('resources.columns.defaultValue')" />
                <el-table-column prop="Unit" :label="t('resources.columns.unit')" />
                <el-table-column :label="t('resources.columns.range')" width="120">
                  <template #default="{ row }">
                    {{ row.Min }} ~ {{ row.Max }}
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </el-card>
      </el-card>

      <!-- 默认提示 -->
      <el-card style="flex:1;" v-else>
        <div style="text-align:center; color:#999; margin-top:100px;">
          {{ t('resources.messages.selectResource') }}
        </div>
      </el-card>
    </div>

    <!-- 资源操作弹窗 -->
    <el-dialog 
      v-model="showOperationDialog" 
      :title="`${t('resources.dialogs.operationTitle')} - ${operationResource?.ResourceID || ''}`"
      width="600px"
      :before-close="closeOperationDialog">
      
      <el-form v-if="operationResource" label-width="120px">
        <el-form-item 
          v-for="op in operationResource.OpDesc" 
          :key="op.Name"
          :label="getOperationDisplayName(op)">
          
          <div style="display: flex; align-items: center; gap: 8px;">
            <!-- 布尔类型下拉框 -->
            <el-select 
              v-if="op.Type === 'BOOL'" 
              v-model="operationForm[op.Name]" 
              style="width: 200px;">
              <el-option 
                v-for="option in getBooleanOptions()" 
                :key="option.value"
                :label="option.label" 
                :value="option.value">
              </el-option>
            </el-select>
            
            <!-- 枚举类型下拉框 -->
            <el-select 
              v-else-if="op.Type === 'ENUM'" 
              v-model="operationForm[op.Name]" 
              style="width: 200px;">
              <el-option 
                v-for="option in getEnumOptions(op.Min, op.Max)" 
                :key="option.value"
                :label="option.label" 
                :value="option.value">
              </el-option>
            </el-select>
            
            <!-- 整数输入框 -->
            <el-input-number 
              v-else-if="op.Type === 'INT'" 
              v-model="operationForm[op.Name]" 
              :min="op.Min ? parseInt(op.Min) : undefined"
              :max="op.Max ? parseInt(op.Max) : undefined"
              style="width: 200px;">
            </el-input-number>
            
            <!-- 浮点数输入框 -->
            <el-input-number 
              v-else-if="op.Type === 'DOUBLE'" 
              v-model="operationForm[op.Name]" 
              :min="op.Min ? parseFloat(op.Min) : undefined"
              :max="op.Max ? parseFloat(op.Max) : undefined"
              :precision="2"
              style="width: 200px;">
            </el-input-number>
            
            <!-- 字符串输入框 -->
            <el-input 
              v-else 
              v-model="operationForm[op.Name]" 
              style="width: 200px;">
            </el-input>
            
            <!-- 单位显示或占位符 -->
            <span style="color: #666; min-width: 60px;">{{ op.Unit || '&nbsp;' }}</span>
            
            <!-- 下发按钮 -->
            <el-button 
              type="primary" 
              size="small" 
              @click="submitSingleOperation(op.Name)">
              {{ t('resources.buttons.send') }}
            </el-button>
          </div>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="closeOperationDialog">{{ t('common.cancel') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>
