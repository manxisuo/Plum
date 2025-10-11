<script setup lang="ts">
import { ref, onMounted, computed, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, FolderOpened, Key, Plus, View, Edit, Delete, Clock } from '@element-plus/icons-vue'
import IdDisplay from '../components/IdDisplay.vue'

const { t } = useI18n()
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''

interface KVItem {
  namespace: string
  key: string
  value: string
  type: string
  updatedAt: number
}

const namespaces = ref<string[]>([])
const selectedNamespace = ref<string>('')
const kvItems = ref<KVItem[]>([])
const loading = ref(false)
const loadingKeys = ref(false)

// 对话框状态
const showDetail = ref(false)
const showEdit = ref(false)
const showCreate = ref(false)
const currentItem = ref<KVItem | null>(null)

// 表单
const form = reactive({
  namespace: '',
  key: '',
  value: '',
  type: 'string'
})

// 加载所有namespace列表
async function loadNamespaces() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/kv`)
    if (!res.ok) throw new Error('Failed to fetch namespaces')
    const data = await res.json()
    namespaces.value = (data.namespaces || []) as string[]
    
    // 默认选中第一个namespace
    if (namespaces.value.length > 0 && !selectedNamespace.value) {
      selectedNamespace.value = namespaces.value[0]
      loadKeys()
    }
  } catch (err: any) {
    ElMessage.error(t('kvStore.errors.loadNamespacesFailed'))
    console.error('Load namespaces error:', err)
    namespaces.value = []
  } finally {
    loading.value = false
  }
}

// 加载指定namespace的所有keys（包括value）
async function loadKeys() {
  if (!selectedNamespace.value) return
  
  loadingKeys.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/kv/${encodeURIComponent(selectedNamespace.value)}`)
    if (!res.ok) throw new Error('Failed to fetch keys')
    const data = await res.json()
    kvItems.value = data as KVItem[]
  } catch (err: any) {
    ElMessage.error(t('kvStore.errors.loadKeysFailed'))
    console.error('Load keys error:', err)
    kvItems.value = []
  } finally {
    loadingKeys.value = false
  }
}

// 切换namespace
function onNamespaceChange() {
  loadKeys()
}

// 查看详情
function viewDetail(item: KVItem) {
  currentItem.value = item
  showDetail.value = true
}

// 打开编辑
async function openEdit(item: KVItem) {
  currentItem.value = item
  form.namespace = item.namespace
  form.key = item.key
  form.type = item.type
  
  // 对于bytes类型，需要重新获取完整数据（列表中只返回了长度）
  if (item.type === 'bytes') {
    try {
      const res = await fetch(
        `${API_BASE}/v1/kv/${encodeURIComponent(item.namespace)}/${encodeURIComponent(item.key)}`
      )
      if (res.ok) {
        const data = await res.json()
        form.value = data.value
      } else {
        form.value = ''
      }
    } catch (err) {
      console.error('Failed to fetch full value:', err)
      form.value = ''
    }
  } else {
    form.value = item.value
  }
  
  showEdit.value = true
}

// 打开创建
function openCreate() {
  form.namespace = selectedNamespace.value
  form.key = ''
  form.value = ''
  form.type = 'string'
  showCreate.value = true
}

// 验证值的类型
function validateValue(value: string, type: string): { valid: boolean; error?: string } {
  if (!value && type !== 'string') {
    return { valid: false, error: '值不能为空' }
  }
  
  switch (type) {
    case 'int':
      if (!/^-?\d+$/.test(value)) {
        return { valid: false, error: '请输入有效的整数（如：123、-456）' }
      }
      break
    case 'double':
      if (!/^-?\d+(\.\d+)?$/.test(value) && !/^-?\d+\.?\d*[eE][+-]?\d+$/.test(value)) {
        return { valid: false, error: '请输入有效的浮点数（如：3.14、-2.5、1e10）' }
      }
      break
    case 'bool':
      if (value !== 'true' && value !== 'false') {
        return { valid: false, error: '请输入 true 或 false' }
      }
      break
    case 'bytes':
      // 验证Base64格式
      const base64Regex = /^[A-Za-z0-9+/]*={0,2}$/
      if (!base64Regex.test(value)) {
        return { valid: false, error: '请输入有效的Base64编码' }
      }
      // Base64长度必须是4的倍数
      if (value.length % 4 !== 0) {
        return { valid: false, error: 'Base64编码长度必须是4的倍数' }
      }
      break
    case 'string':
      // 字符串类型不需要特殊验证
      break
    default:
      return { valid: false, error: '未知的类型' }
  }
  
  return { valid: true }
}

// 提交编辑
async function submitEdit() {
  if (!form.key || !form.namespace) {
    ElMessage.warning('Namespace和Key不能为空')
    return
  }
  
  // 验证值的类型
  const validation = validateValue(form.value, form.type)
  if (!validation.valid) {
    ElMessage.warning(validation.error || '值格式不正确')
    return
  }
  
  try {
    const res = await fetch(
      `${API_BASE}/v1/kv/${encodeURIComponent(form.namespace)}/${encodeURIComponent(form.key)}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value: form.value, type: form.type })
      }
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    
    ElMessage.success('更新成功')
    showEdit.value = false
    loadKeys()
  } catch (err: any) {
    ElMessage.error('更新失败: ' + err.message)
  }
}

// 提交创建
async function submitCreate() {
  if (!form.key || !form.namespace) {
    ElMessage.warning('Namespace和Key不能为空')
    return
  }
  
  // 验证值的类型
  const validation = validateValue(form.value, form.type)
  if (!validation.valid) {
    ElMessage.warning(validation.error || '值格式不正确')
    return
  }
  
  try {
    const res = await fetch(
      `${API_BASE}/v1/kv/${encodeURIComponent(form.namespace)}/${encodeURIComponent(form.key)}`,
      {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ value: form.value, type: form.type })
      }
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    
    ElMessage.success('创建成功')
    showCreate.value = false
    
    // 如果是新namespace，重新加载namespace列表
    if (!namespaces.value.includes(form.namespace)) {
      await loadNamespaces()
      selectedNamespace.value = form.namespace
    }
    
    loadKeys()
  } catch (err: any) {
    ElMessage.error('创建失败: ' + err.message)
  }
}

// 删除key
async function deleteKey(item: KVItem) {
  try {
    await ElMessageBox.confirm(
      `确认删除 ${item.namespace}/${item.key}?`,
      '删除确认',
      { type: 'warning' }
    )
    
    const res = await fetch(
      `${API_BASE}/v1/kv/${encodeURIComponent(item.namespace)}/${encodeURIComponent(item.key)}`,
      { method: 'DELETE' }
    )
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    
    ElMessage.success('删除成功')
    loadKeys()
    
    // 如果删除后namespace为空，重新加载namespace列表
    if (kvItems.value.length <= 1) {
      loadNamespaces()
    }
  } catch (err: any) {
    if (err !== 'cancel') {
      ElMessage.error('删除失败: ' + err.message)
    }
  }
}

// 格式化时间
function formatTime(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleString('zh-CN')
}

// 格式化值显示
function formatValue(item: KVItem): string {
  if (item.type === 'bytes') {
    // 后端已返回长度（优化后不再返回完整内容）
    return `(${item.value} bytes)`
  }
  
  // 其他类型，如果太长就截断
  if (item.value.length > 100) {
    return item.value.substring(0, 100) + '...'
  }
  return item.value
}

// 获取值的占位符
function getValuePlaceholder(type: string): string {
  switch (type) {
    case 'int': return '输入整数（如：123、-456）'
    case 'double': return '输入浮点数（如：3.14、-2.5）'
    case 'bool': return '输入 true 或 false'
    case 'bytes': return '输入Base64编码的数据'
    case 'string': return '输入字符串值'
    default: return '输入值'
  }
}

// 获取值的提示信息
function getValueHint(type: string): string {
  switch (type) {
    case 'int': return '提示: 只能输入整数，支持负数'
    case 'double': return '提示: 支持小数和科学计数法（如：1e10）'
    case 'bool': return '提示: 只能是 true 或 false'
    case 'bytes': return '提示: Base64编码，长度必须是4的倍数'
    case 'string': return '提示: 任意文本内容'
    default: return ''
  }
}

// 刷新
function refresh() {
  const currentNs = selectedNamespace.value
  loadNamespaces().then(() => {
    // 如果之前有选中的namespace，重新选中并加载keys
    if (currentNs && namespaces.value.includes(currentNs)) {
      selectedNamespace.value = currentNs
      loadKeys()
    }
  })
}

// 统计信息
const totalNamespaces = computed(() => namespaces.value.length)
const currentKeys = computed(() => kvItems.value.length)

onMounted(() => {
  loadNamespaces()
})
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="refresh">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
        <el-button type="success" @click="openCreate" :disabled="namespaces.length === 0">
          <el-icon><Plus /></el-icon>
          {{ t('kvStore.buttons.create') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><FolderOpened /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalNamespaces }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('kvStore.stats.namespaces') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Key /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ currentKeys }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('kvStore.stats.currentKeys') }}</span>
        </div>
      </div>
      
      <!-- Namespace选择器 -->
      <div style="display:flex; align-items:center; gap:8px; flex-shrink:0;">
        <span style="font-size:14px; color:#606266;">Namespace:</span>
        <el-select 
          v-model="selectedNamespace" 
          @change="onNamespaceChange"
          style="width:200px;"
          :loading="loading"
          :disabled="namespaces.length === 0"
        >
          <el-option 
            v-for="ns in namespaces" 
            :key="ns" 
            :label="ns" 
            :value="ns"
          />
        </el-select>
      </div>
    </div>

    <!-- Keys表格 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('kvStore.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ kvItems.length }} {{ t('kvStore.keys') }}</span>
        </div>
      </template>
      
      <!-- 空状态 -->
      <div v-if="namespaces.length === 0 && !loading" style="padding:40px 0;">
        <el-empty :description="t('kvStore.empty')" />
      </div>

      <!-- 表格 -->
      <el-table 
        v-else
        v-loading="loadingKeys" 
        :data="kvItems" 
        style="width:100%;" 
        stripe
      >
        <el-table-column prop="key" label="Key" width="250">
          <template #default="{ row }">
            <div style="display:flex; align-items:center; gap:6px;">
              <el-icon size="16" color="#409EFF"><Key /></el-icon>
              <span style="font-family:monospace;">{{ row.key }}</span>
            </div>
          </template>
        </el-table-column>
        
        <el-table-column prop="type" label="Type" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type === 'bytes' ? 'warning' : 'info'">
              {{ row.type }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="value" label="Value" min-width="200">
          <template #default="{ row }">
            <span style="font-family:monospace; color:#606266;">{{ formatValue(row) }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="updatedAt" label="Updated At" width="180">
          <template #default="{ row }">
            <div style="display:flex; align-items:center; gap:4px; font-size:13px; color:#909399;">
              <el-icon size="14"><Clock /></el-icon>
              {{ formatTime(row.updatedAt) }}
            </div>
          </template>
        </el-table-column>
        
        <el-table-column :label="t('common.action')" width="260" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="viewDetail(row)">
              <el-icon><View /></el-icon>
              {{ t('kvStore.buttons.view') }}
            </el-button>
            <el-button size="small" type="primary" @click="openEdit(row)">
              <el-icon><Edit /></el-icon>
              {{ t('kvStore.buttons.edit') }}
            </el-button>
            <el-button size="small" type="danger" @click="deleteKey(row)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 详情对话框 -->
    <el-dialog v-model="showDetail" :title="t('kvStore.dialog.detailTitle')" width="600px">
      <div v-if="currentItem" style="line-height:1.8;">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="Namespace">
            <el-tag type="primary">{{ currentItem.namespace }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Key">
            <span style="font-family:monospace;">{{ currentItem.key }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="Type">
            <el-tag :type="currentItem.type === 'bytes' ? 'warning' : 'info'">
              {{ currentItem.type }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Value">
            <div v-if="currentItem.type === 'bytes'" style="color:#909399;">
              {{ formatValue(currentItem) }}
            </div>
            <pre v-else style="margin:0; font-family:monospace; white-space:pre-wrap; word-break:break-all;">{{ currentItem.value }}</pre>
          </el-descriptions-item>
          <el-descriptions-item label="Updated At">
            {{ formatTime(currentItem.updatedAt) }}
          </el-descriptions-item>
        </el-descriptions>
      </div>
    </el-dialog>

    <!-- 编辑对话框 -->
    <el-dialog v-model="showEdit" :title="t('kvStore.dialog.editTitle')" width="600px">
      <el-form label-width="100px">
        <el-form-item label="Namespace">
          <el-input v-model="form.namespace" disabled />
        </el-form-item>
        <el-form-item label="Key">
          <el-input v-model="form.key" disabled />
        </el-form-item>
        <el-form-item label="Type">
          <el-select v-model="form.type" style="width:100%;">
            <el-option label="string" value="string" />
            <el-option label="int" value="int" />
            <el-option label="double" value="double" />
            <el-option label="bool" value="bool" />
            <el-option label="bytes" value="bytes" />
          </el-select>
        </el-form-item>
        <el-form-item label="Value">
          <el-input 
            v-model="form.value" 
            type="textarea" 
            :rows="6"
            :placeholder="getValuePlaceholder(form.type)"
          />
          <div style="margin-top:4px; font-size:12px; color:#909399;">
            {{ getValueHint(form.type) }}
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEdit = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitEdit">{{ t('kvStore.buttons.edit') }}</el-button>
      </template>
    </el-dialog>

    <!-- 创建对话框 -->
    <el-dialog v-model="showCreate" :title="t('kvStore.dialog.createTitle')" width="600px">
      <el-form label-width="100px">
        <el-form-item label="Namespace">
          <el-input v-model="form.namespace" placeholder="输入namespace（可创建新的）" />
        </el-form-item>
        <el-form-item label="Key">
          <el-input v-model="form.key" placeholder="输入key名称" />
        </el-form-item>
        <el-form-item label="Type">
          <el-select v-model="form.type" style="width:100%;">
            <el-option label="string" value="string" />
            <el-option label="int" value="int" />
            <el-option label="double" value="double" />
            <el-option label="bool" value="bool" />
            <el-option label="bytes" value="bytes" />
          </el-select>
        </el-form-item>
        <el-form-item label="Value">
          <el-input 
            v-model="form.value" 
            type="textarea" 
            :rows="6"
            :placeholder="getValuePlaceholder(form.type)"
          />
          <div style="margin-top:4px; font-size:12px; color:#909399;">
            {{ getValueHint(form.type) }}
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitCreate">{{ t('kvStore.buttons.create') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

