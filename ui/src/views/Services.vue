<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, Edit, Delete, View, Search, Filter } from '@element-plus/icons-vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
type Endpoint = { 
  serviceName: string
  instanceId: string
  nodeId: string
  ip: string
  port: number
  protocol: string
  version?: string
  healthy: boolean
  lastSeen: number
  labels?: Record<string, string>
}

const services = ref<string[]>([])
const active = ref<string>('')
const loading = ref(false)
const eps = ref<Endpoint[]>([])

// 节点和实例数据（用于下拉框）
const nodes = ref<Array<{ nodeId: string; ip: string }>>([])
const instances = ref<Array<{ instanceId: string; nodeId: string; appName: string }>>([])

// 筛选和搜索
const searchText = ref('')
const filterNode = ref('')
const filterHealthy = ref('')
const filterProtocol = ref('')

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

// 弹窗状态
const showRegisterDialog = ref(false)
const showEditDialog = ref(false)
const showDetailsDialog = ref(false)
const selectedEndpoint = ref<Endpoint | null>(null)
const registerForm = ref({
  serviceName: '',
  instanceId: '',
  nodeId: '',
  ip: '',
  port: 8080,
  protocol: 'http',
  version: '',
  labels: {} as Record<string, string>
})
const editForm = ref<Endpoint | null>(null)

// 加载服务列表
async function loadServices(){
  try {
    const res = await fetch(`${API_BASE}/v1/services/list`)
    if (res.ok) {
      const newServices = await res.json() as string[]
      console.log('Loaded services:', newServices)
      services.value = newServices
      if (!active.value && services.value.length) {
        active.value = services.value[0]
        loadEndpoints()
      }
    } else {
      console.error('Failed to load services, status:', res.status)
    }
  } catch (err) {
    console.error('Failed to load services:', err)
  }
}

// 加载端点列表
async function loadEndpoints(){
  if (!active.value) return
  loading.value = true
  try {
    // 使用 all=true 参数获取所有端点（包括不健康的）
    const url = `${API_BASE}/v1/discovery?service=${encodeURIComponent(active.value)}&all=true`
    console.log('Loading endpoints for service:', active.value, 'URL:', url)
    const res = await fetch(url)
    if (res.ok) {
      const data = await res.json() as Endpoint[]
      console.log('Loaded endpoints:', data.length, data)
      eps.value = data
    } else {
      const text = await res.text()
      console.error('Failed to load endpoints, status:', res.status, 'response:', text)
    }
  } catch (err) {
    console.error('Failed to load endpoints:', err)
  } finally {
    loading.value = false
  }
}

// 加载节点列表
async function loadNodes(){
  try {
    const res = await fetch(`${API_BASE}/v1/nodes`)
    if (res.ok) {
      const data = await res.json() as Array<{ nodeId: string; ip: string }>
      nodes.value = data
    }
  } catch (err) {
    console.error('Failed to load nodes:', err)
  }
}

// 加载实例列表（从assignments）
async function loadInstances(){
  try {
    // 收集所有节点的assignments
    const allInstances: Array<{ instanceId: string; nodeId: string; appName: string }> = []
    for (const node of nodes.value) {
      const res = await fetch(`${API_BASE}/v1/assignments?nodeId=${encodeURIComponent(node.nodeId)}`)
      if (res.ok) {
        const data = await res.json() as { items: Array<{ instanceId: string; appName: string; appVersion?: string }> }
        // API返回的是 { items: [...] }，并且不包含nodeId，需要手动添加
        if (data.items) {
          for (const item of data.items) {
            allInstances.push({
              instanceId: item.instanceId,
              nodeId: node.nodeId,
              appName: item.appName || ''
            })
          }
        }
      }
    }
    instances.value = allInstances
    console.log('Loaded instances:', instances.value)
  } catch (err) {
    console.error('Failed to load instances:', err)
  }
}

onMounted(async () => {
  loadServices()
  await loadNodes()
  await loadInstances()
})
const { t } = useI18n()

// 统计计算
const totalServices = computed(() => services.value.length)
const totalEndpoints = computed(() => filteredEndpoints.value.length)
const healthyEndpoints = computed(() => filteredEndpoints.value.filter(ep => ep.healthy).length)

// 过滤后的端点
const filteredEndpoints = computed(() => {
  let result = eps.value

  // 搜索过滤
  if (searchText.value) {
    const search = searchText.value.toLowerCase()
    result = result.filter(ep =>
      ep.instanceId.toLowerCase().includes(search) ||
      ep.ip.toLowerCase().includes(search) ||
      ep.nodeId.toLowerCase().includes(search) ||
      (ep.version && ep.version.toLowerCase().includes(search))
    )
  }

  // 节点过滤
  if (filterNode.value) {
    result = result.filter(ep => ep.nodeId === filterNode.value)
  }

  // 健康状态过滤
  if (filterHealthy.value !== '') {
    const isHealthy = filterHealthy.value === 'true'
    result = result.filter(ep => ep.healthy === isHealthy)
  }

  // 协议过滤
  if (filterProtocol.value) {
    result = result.filter(ep => ep.protocol === filterProtocol.value)
  }

  return result
})

// 计算属性：分页后的数据
const paginatedEndpoints = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredEndpoints.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(filteredEndpoints.value.length / pageSize.value)
})

// 分页事件处理
function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

// 打开注册弹窗
function openRegisterDialog() {
  registerForm.value = {
    serviceName: active.value || '',
    instanceId: '', // 可以留空，也可以选择已有实例
    nodeId: '',
    ip: '',
    port: 8080,
    protocol: 'http',
    version: '',
    labels: {}
  }
  showRegisterDialog.value = true
}

// 提交注册
async function submitRegister() {
  if (!registerForm.value.serviceName || !registerForm.value.ip || !registerForm.value.port) {
    ElMessage.error('请填写服务名、IP地址和端口')
    return
  }
  
  // 如果没有填写instanceId，生成一个虚拟实例ID（用于完全独立的手动注册）
  if (!registerForm.value.instanceId) {
    const timestamp = Date.now()
    const random = Math.floor(Math.random() * 10000)
    registerForm.value.instanceId = `manual-${timestamp}-${random}`
  }
  
  // 如果没有填写nodeId，使用默认值
  if (!registerForm.value.nodeId) {
    registerForm.value.nodeId = 'manual'
  }

  try {
    const payload = {
      instanceId: registerForm.value.instanceId,
      nodeId: registerForm.value.nodeId,
      ip: registerForm.value.ip,
      endpoints: [{
        serviceName: registerForm.value.serviceName,
        port: registerForm.value.port,
        protocol: registerForm.value.protocol,
        version: registerForm.value.version || '',
        labels: registerForm.value.labels || {}
      }]
    }
    console.log('Registering endpoint with payload:', payload)
    const res = await fetch(`${API_BASE}/v1/services/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })

    if (res.ok || res.status === 204) {
      ElMessage.success('注册成功')
      showRegisterDialog.value = false
      
      const registeredServiceName = registerForm.value.serviceName
      console.log('Registered service:', registeredServiceName)
      
      // 先刷新服务列表（新注册的服务名可能不在列表中）
      await loadServices()
      console.log('Current services after refresh:', services.value)
      
      // 如果注册的服务名在新服务列表中，切换到该服务
      // 如果注册的服务名是新服务或与当前选中的服务不同，切换到新服务
      if (registeredServiceName) {
        // 检查服务名是否在列表中
        if (services.value.includes(registeredServiceName)) {
          active.value = registeredServiceName
          console.log('Switched to service:', registeredServiceName)
        } else {
          console.warn('Service not found in list:', registeredServiceName, 'Available:', services.value)
        }
      }
      
      // 刷新端点列表
      if (active.value) {
        await loadEndpoints()
      }
    } else {
      const text = await res.text()
      ElMessage.error(`注册失败: ${text}`)
    }
  } catch (err) {
    ElMessage.error(`注册失败: ${err}`)
  }
}

// 打开编辑弹窗
function openEditDialog(endpoint: Endpoint) {
  selectedEndpoint.value = endpoint
  editForm.value = { ...endpoint }
  showEditDialog.value = true
}

// 提交编辑
async function submitEdit() {
  if (!editForm.value) return

  const ep = editForm.value
  if (!ep.serviceName || !ep.instanceId || !ep.ip || !ep.port) {
    ElMessage.error('请填写必填字段')
    return
  }

  try {
    // 使用查询参数传递旧端点信息，body传递新信息
    const oldEp = selectedEndpoint.value!
    const url = `${API_BASE}/v1/services/endpoint?serviceName=${encodeURIComponent(oldEp.serviceName)}&instanceId=${encodeURIComponent(oldEp.instanceId)}&ip=${encodeURIComponent(oldEp.ip)}&port=${oldEp.port}&protocol=${encodeURIComponent(oldEp.protocol)}`
    
    const res = await fetch(url, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(ep)
    })

    if (res.ok || res.status === 204) {
      ElMessage.success('更新成功')
      showEditDialog.value = false
      selectedEndpoint.value = null
      loadEndpoints()
    } else {
      const text = await res.text()
      ElMessage.error(`更新失败: ${text}`)
    }
  } catch (err) {
    ElMessage.error(`更新失败: ${err}`)
  }
}

// 删除端点
async function deleteEndpoint(endpoint: Endpoint) {
  try {
    await ElMessageBox.confirm(
      `确认删除端点 ${endpoint.serviceName} @ ${endpoint.ip}:${endpoint.port}?`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    const url = `${API_BASE}/v1/services/endpoint?serviceName=${encodeURIComponent(endpoint.serviceName)}&instanceId=${encodeURIComponent(endpoint.instanceId)}&ip=${encodeURIComponent(endpoint.ip)}&port=${endpoint.port}&protocol=${encodeURIComponent(endpoint.protocol)}`
    const res = await fetch(url, { method: 'DELETE' })

    if (res.ok || res.status === 204) {
      ElMessage.success('删除成功')
      loadEndpoints()
    } else {
      const text = await res.text()
      ElMessage.error(`删除失败: ${text}`)
    }
  } catch (err) {
    if (err !== 'cancel') {
      ElMessage.error(`删除失败: ${err}`)
    }
  }
}

// 查看详情
function viewDetails(endpoint: Endpoint) {
  selectedEndpoint.value = endpoint
  showDetailsDialog.value = true
}

// 节点选择变化时更新实例列表
function onNodeChange(nodeId: string) {
  registerForm.value.instanceId = ''
  // 实例列表已经包含所有节点，这里不需要重新加载
}
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="loadServices">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
        <el-button type="success" @click="openRegisterDialog">
          <el-icon><Plus /></el-icon>
          手动注册端点
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Refresh /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalServices }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('services.stats.services') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><View /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalEndpoints }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('services.stats.endpoints') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><View /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ healthyEndpoints }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('services.stats.healthy') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <div style="display:flex; gap:12px;">
      <!-- 服务列表 -->
      <el-card style="width:240px;">
        <template #header>{{ t('services.title') }}</template>
        <el-menu :default-active="active" @select="(k:string)=>{active=k; loadEndpoints()}">
          <el-menu-item v-for="s in services" :key="s" :index="s">{{ s }}</el-menu-item>
        </el-menu>
      </el-card>

      <!-- 端点表格 -->
      <el-card style="flex:1;">
        <template #header>
          <div style="display:flex; justify-content:space-between; align-items:center;">
            <span>{{ t('services.endpointsTitle', { name: active || '-' }) }}</span>
            <!-- 筛选和搜索 -->
            <div style="display:flex; gap:8px; align-items:center;">
              <el-input
                v-model="searchText"
                placeholder="搜索实例ID、IP、节点..."
                clearable
                style="width:200px;"
                @clear="searchText = ''"
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
              </el-input>
              <el-select v-model="filterNode" placeholder="节点" clearable style="width:120px;">
                <el-option
                  v-for="node in nodes"
                  :key="node.nodeId"
                  :label="node.nodeId"
                  :value="node.nodeId"
                />
              </el-select>
              <el-select v-model="filterHealthy" placeholder="健康状态" clearable style="width:120px;">
                <el-option label="健康" value="true" />
                <el-option label="不健康" value="false" />
              </el-select>
              <el-select v-model="filterProtocol" placeholder="协议" clearable style="width:100px;">
                <el-option label="http" value="http" />
                <el-option label="https" value="https" />
                <el-option label="grpc" value="grpc" />
                <el-option label="tcp" value="tcp" />
                <el-option label="udp" value="udp" />
              </el-select>
            </div>
          </div>
        </template>

        <el-table :data="paginatedEndpoints" v-loading="loading" style="width:100%" stripe>
          <el-table-column prop="instanceId" :label="t('services.columns.instance')" width="180" show-overflow-tooltip />
          <el-table-column prop="nodeId" :label="t('services.columns.node')" width="120" />
          <el-table-column :label="t('services.columns.address')" min-width="180">
            <template #default="{ row }">{{ row.ip }}:{{ row.port }} ({{ row.protocol }})</template>
          </el-table-column>
          <el-table-column prop="version" label="版本" width="100" />
          <el-table-column :label="t('services.columns.healthy')" width="100">
            <template #default="{ row }">
              <el-tag :type="row.healthy ? 'success' : 'danger'">
                {{ row.healthy ? '健康' : '不健康' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('services.columns.lastSeen')" width="180">
            <template #default="{ row }">{{ new Date(row.lastSeen*1000).toLocaleString() }}</template>
          </el-table-column>
          <el-table-column label="操作" width="200" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" size="small" @click="viewDetails(row)">
                <el-icon><View /></el-icon>
                详情
              </el-button>
              <el-button link type="primary" size="small" @click="openEditDialog(row)">
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button link type="danger" size="small" @click="deleteEndpoint(row)">
                <el-icon><Delete /></el-icon>
                删除
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        
        <!-- 分页组件 -->
        <div style="margin-top: 16px; display: flex; justify-content: center;">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="pageSizes"
            :total="filteredEndpoints.length"
            layout="total, sizes, prev, pager, next, jumper"
            @size-change="handleSizeChange"
            @current-change="handleCurrentChange"
          />
        </div>
      </el-card>
    </div>

    <!-- 注册端点弹窗 -->
    <el-dialog v-model="showRegisterDialog" title="手动注册端点" width="600px">
      <el-form :model="registerForm" label-width="120px">
        <el-form-item label="服务名" required>
          <el-input v-model="registerForm.serviceName" placeholder="输入服务名" />
        </el-form-item>
        <el-form-item label="节点" required>
          <el-select v-model="registerForm.nodeId" placeholder="选择节点" @change="onNodeChange" style="width:100%;">
            <el-option
              v-for="node in nodes"
              :key="node.nodeId"
              :label="`${node.nodeId} (${node.ip})`"
              :value="node.nodeId"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="实例ID">
          <el-select 
            v-model="registerForm.instanceId" 
            placeholder="选择已有实例（或留空生成虚拟ID）" 
            filterable 
            allow-create
            style="width:100%;"
          >
            <el-option
              v-for="inst in instances.filter((i: { instanceId: string; nodeId: string; appName: string }) => !registerForm.nodeId || i.nodeId === registerForm.nodeId)"
              :key="inst.instanceId"
              :label="`${inst.instanceId} (${inst.appName || ''})`"
              :value="inst.instanceId"
            />
          </el-select>
          <div style="font-size:12px; color:#909399; margin-top:4px;">
            提示：选择已有实例可在真实应用上注册服务；留空则生成虚拟实例ID（用于独立服务）
          </div>
        </el-form-item>
        <el-form-item label="IP地址" required>
          <el-input v-model="registerForm.ip" placeholder="输入IP地址" />
        </el-form-item>
        <el-form-item label="端口" required>
          <el-input-number v-model="registerForm.port" :min="1" :max="65535" style="width:100%;" />
        </el-form-item>
        <el-form-item label="协议" required>
          <el-select v-model="registerForm.protocol" style="width:100%;">
            <el-option label="http" value="http" />
            <el-option label="https" value="https" />
            <el-option label="grpc" value="grpc" />
            <el-option label="tcp" value="tcp" />
            <el-option label="udp" value="udp" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="registerForm.version" placeholder="可选，如 v1.0.0" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRegisterDialog = false">取消</el-button>
        <el-button type="primary" @click="submitRegister">注册</el-button>
      </template>
    </el-dialog>

    <!-- 编辑端点弹窗 -->
    <el-dialog v-model="showEditDialog" title="编辑端点" width="600px" v-if="editForm">
      <el-form :model="editForm" label-width="120px">
        <el-form-item label="服务名">
          <el-input v-model="editForm.serviceName" disabled />
        </el-form-item>
        <el-form-item label="实例ID">
          <el-input v-model="editForm.instanceId" disabled />
        </el-form-item>
        <el-form-item label="节点">
          <el-select v-model="editForm.nodeId" style="width:100%;">
            <el-option
              v-for="node in nodes"
              :key="node.nodeId"
              :label="`${node.nodeId} (${node.ip})`"
              :value="node.nodeId"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="IP地址" required>
          <el-input v-model="editForm.ip" />
        </el-form-item>
        <el-form-item label="端口" required>
          <el-input-number v-model="editForm.port" :min="1" :max="65535" style="width:100%;" />
        </el-form-item>
        <el-form-item label="协议" required>
          <el-select v-model="editForm.protocol" style="width:100%;">
            <el-option label="http" value="http" />
            <el-option label="https" value="https" />
            <el-option label="grpc" value="grpc" />
            <el-option label="tcp" value="tcp" />
            <el-option label="udp" value="udp" />
          </el-select>
        </el-form-item>
        <el-form-item label="版本">
          <el-input v-model="editForm.version" />
        </el-form-item>
        <el-form-item label="健康状态">
          <el-switch v-model="editForm.healthy" active-text="健康" inactive-text="不健康" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showEditDialog = false">取消</el-button>
        <el-button type="primary" @click="submitEdit">保存</el-button>
      </template>
    </el-dialog>

    <!-- 端点详情弹窗 -->
    <el-dialog v-model="showDetailsDialog" title="端点详情" width="700px" v-if="selectedEndpoint">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="服务名">{{ selectedEndpoint.serviceName }}</el-descriptions-item>
        <el-descriptions-item label="实例ID">{{ selectedEndpoint.instanceId }}</el-descriptions-item>
        <el-descriptions-item label="节点ID">{{ selectedEndpoint.nodeId }}</el-descriptions-item>
        <el-descriptions-item label="IP地址">{{ selectedEndpoint.ip }}</el-descriptions-item>
        <el-descriptions-item label="端口">{{ selectedEndpoint.port }}</el-descriptions-item>
        <el-descriptions-item label="协议">{{ selectedEndpoint.protocol }}</el-descriptions-item>
        <el-descriptions-item label="版本">{{ selectedEndpoint.version || '-' }}</el-descriptions-item>
        <el-descriptions-item label="健康状态">
          <el-tag :type="selectedEndpoint.healthy ? 'success' : 'danger'">
            {{ selectedEndpoint.healthy ? '健康' : '不健康' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="最后心跳">
          {{ new Date(selectedEndpoint.lastSeen * 1000).toLocaleString() }}
        </el-descriptions-item>
        <el-descriptions-item label="标签" :span="2">
          <div v-if="selectedEndpoint.labels && Object.keys(selectedEndpoint.labels).length > 0">
            <el-tag
              v-for="(value, key) in selectedEndpoint.labels"
              :key="key"
              style="margin-right: 8px; margin-bottom: 4px;"
            >
              {{ key }}: {{ value }}
            </el-tag>
          </div>
          <span v-else style="color: #909399;">无</span>
        </el-descriptions-item>
      </el-descriptions>
      <template #footer>
        <el-button @click="showDetailsDialog = false">关闭</el-button>
        <el-button type="primary" @click="openEditDialog(selectedEndpoint); showDetailsDialog = false">编辑</el-button>
      </template>
    </el-dialog>
  </div>
</template>