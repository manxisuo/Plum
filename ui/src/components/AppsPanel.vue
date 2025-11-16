<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Refresh, Files, Upload, Delete } from '@element-plus/icons-vue'

type Artifact = {
	artifactId: string
	name: string
	version: string
	url: string
	sha256: string
	sizeBytes: number
	createdAt: number
	type?: string // "zip" or "image"
	imageRepository?: string
	imageTag?: string
	portMappings?: string // JSON string
}

type DeleteResult = {
	success: boolean
	messageKey?: string
	params?: Record<string, any>
	rawMessage?: string
}

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const loading = ref(false)
const deleting = ref(false)
const items = ref<Artifact[]>([])
const selectedIds = ref<string[]>([])
const uploadUrl = `${API_BASE}/v1/apps/upload`

// 镜像创建相关
const showImageDialog = ref(false)
const imageForm = ref({
	name: '',
	version: '',
	imageRepository: '',
	imageTag: '',
	portMappings: [] as Array<{ host: number; container: number }>
})
const creating = ref(false)
const dockerImages = ref<Array<{ repository: string; tag: string; imageId: string; created: string; size: string }>>([])
const loadingImages = ref(false)
const imageRepositoryOptions = ref<string[]>([])
const imageTagOptions = ref<string[]>([])

const { t } = useI18n()

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

async function load() {
	loading.value = true
	try {
		const res = await fetch(`${API_BASE}/v1/apps`)
		if (!res.ok) throw new Error(`HTTP ${res.status}`)
		const data = await res.json() as Artifact[]
		items.value = Array.isArray(data) ? data : []
		selectedIds.value = []
	} catch (e:any) {
		ElMessage.error(e?.message || '加载失败')
		// 确保在错误情况下也重置为安全值
		items.value = []
	} finally {
		loading.value = false
	}
}

function onSuccess(response: any) {
	// 检查是否是因为重复而失败
	if (response && response.error) {
		ElMessage.error(response.error)
		return
	}
	ElMessage.success('上传成功')
	load()
}

function onError(err: any) {
	// 尝试解析错误信息
	let errorMsg = '上传失败'
	if (err && err.response) {
		try {
			const responseText = err.response.text || err.response.statusText
			if (responseText && responseText.includes('已存在')) {
				errorMsg = responseText
			} else if (err.response.status === 409) {
				errorMsg = '应用包已存在，请检查应用名称和版本'
			}
		} catch (e) {
			// 忽略解析错误，使用默认消息
		}
	}
	ElMessage.error(errorMsg)
}

// 检查应用包是否已存在
function checkAppExists(file: File): Promise<boolean> {
	return new Promise((resolve) => {
		const reader = new FileReader()
		reader.onload = async (e) => {
			try {
				// 这里需要解析ZIP文件来获取meta.ini中的name和version
				// 由于浏览器环境限制，我们简化处理，先上传到后端进行校验
				resolve(false) // 先返回false，让后端处理校验
			} catch (error) {
				console.error('Error reading file:', error)
				resolve(false)
			}
		}
		reader.readAsArrayBuffer(file)
	})
}

// 上传前的校验
function beforeUpload(file: File) {
	// 检查文件类型
	if (!file.name.toLowerCase().endsWith('.zip')) {
		ElMessage.error('只能上传ZIP文件')
		return false
	}
	
	// 检查文件大小（限制为100MB）
	const maxSize = 100 * 1024 * 1024
	if (file.size > maxSize) {
		ElMessage.error('文件大小不能超过100MB')
		return false
	}
	
	return true
}

async function deleteArtifact(id: string): Promise<DeleteResult> {
	try {
		const res = await fetch(`${API_BASE}/v1/apps/${encodeURIComponent(id)}`, { method: 'DELETE' })
		if (res.status === 409) {
			return { success: false, messageKey: 'apps.messages.inUse' }
		}
		if (!res.ok) {
			return { success: false, messageKey: 'apps.messages.httpError', params: { status: res.status } }
		}
		return { success: true }
	} catch (e: any) {
		return { success: false, rawMessage: e?.message }
	}
}

async function del(id: string) {
	const result = await deleteArtifact(id)
	if (result.success) {
		ElMessage.success(t('apps.messages.deleteSuccess'))
		load()
	} else if (result.messageKey) {
		ElMessage.error(t(result.messageKey, result.params ?? {}))
	} else if (result.rawMessage) {
		ElMessage.error(result.rawMessage)
	} else {
		ElMessage.error(t('apps.messages.deleteFailed'))
	}
}

function handleSelectionChange(rows: Artifact[]) {
	selectedIds.value = rows.map(row => row.artifactId)
}

async function deleteSelected() {
	if (!selectedIds.value.length) return
	try {
		await ElMessageBox.confirm(
			t('apps.confirmBatchDelete', { count: selectedIds.value.length }),
			t('common.confirm'),
			{
				type: 'warning',
				confirmButtonText: t('common.delete'),
				cancelButtonText: t('common.cancel')
			}
		)
	} catch {
		return
	}

	deleting.value = true
	const failed: string[] = []

	for (const id of selectedIds.value) {
		const result = await deleteArtifact(id)
		if (!result.success) {
			if (result.messageKey) {
				failed.push(t(result.messageKey, result.params ?? {}))
			} else if (result.rawMessage) {
				failed.push(result.rawMessage)
			} else {
				failed.push(id)
			}
		}
	}

	deleting.value = false
	load()

	if (failed.length === 0) {
		ElMessage.success(t('apps.messages.deleteSuccess'))
	} else {
		ElMessage.error(t('apps.messages.deletePartial', { detail: failed.join(', ') }))
	}
}

// 计算属性：分页后的数据
const paginatedItems = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return items.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(items.value.length / pageSize.value)
})

// 分页事件处理
function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1 // 重置到第一页
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

onMounted(load)

function artifactHref(row: Artifact): string {
  const u = row.url || ''
  if (u.startsWith('http://') || u.startsWith('https://')) return u
  const base = API_BASE || 'http://plum-controller:8080'
  return u.startsWith('/') ? `${base}${u}` : `${base}/${u}`
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatTimestamp(timestamp: number): string {
  if (!timestamp) return ''
  return new Date(timestamp * 1000).toLocaleString()
}

function resetImageForm() {
  imageForm.value = {
    name: '',
    version: '',
    imageRepository: '',
    imageTag: '',
    portMappings: []
  }
}

// 加载 Docker 镜像列表
async function loadDockerImages() {
  loadingImages.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/apps/docker-images`)
    if (!res.ok) {
      // 如果 API 失败，静默处理（可能是 Docker 不可用）
      dockerImages.value = []
      return
    }
    const data = await res.json()
    dockerImages.value = Array.isArray(data) ? data : []
    
    // 提取唯一的仓库名称
    const repos = new Set<string>()
    dockerImages.value.forEach(img => {
      if (img.repository && img.repository !== '<none>') {
        repos.add(img.repository)
      }
    })
    imageRepositoryOptions.value = Array.from(repos).sort()
    
    // 更新标签选项（基于选中的仓库）
    updateTagOptions()
  } catch (e: any) {
    // 静默处理错误（Docker 可能不可用）
    dockerImages.value = []
    imageRepositoryOptions.value = []
    imageTagOptions.value = []
  } finally {
    loadingImages.value = false
  }
}

// 根据选中的仓库更新标签选项
function updateTagOptions() {
  if (!imageForm.value.imageRepository) {
    imageTagOptions.value = []
    return
  }
  const tags = dockerImages.value
    .filter(img => img.repository === imageForm.value.imageRepository && img.tag !== '<none>')
    .map(img => img.tag)
  imageTagOptions.value = Array.from(new Set(tags)).sort()
}

// 为 autocomplete 提供仓库建议
function queryRepositorySuggestions(queryString: string, cb: (suggestions: Array<{ value: string }>) => void) {
  const suggestions = imageRepositoryOptions.value
    .filter(repo => repo.toLowerCase().includes(queryString.toLowerCase()))
    .map(repo => ({ value: repo }))
  // 如果用户输入的内容不在建议列表中，也添加进去（允许自定义输入）
  if (queryString && !suggestions.some(s => s.value === queryString)) {
    suggestions.unshift({ value: queryString })
  }
  cb(suggestions)
}

// 当仓库改变时，更新标签选项并清空当前标签
function onRepositoryChange() {
  updateTagOptions()
  // 如果当前标签不在新选项中，清空它
  if (imageForm.value.imageTag && !imageTagOptions.value.includes(imageForm.value.imageTag)) {
    imageForm.value.imageTag = ''
  }
}

// 打开镜像创建对话框时加载镜像列表
function openImageDialog() {
  showImageDialog.value = true
  loadDockerImages()
}

async function createImageApp() {
  // 只验证必填字段：镜像仓库和镜像标签
  if (!imageForm.value.imageRepository || !imageForm.value.imageTag) {
    ElMessage.error('请填写镜像仓库和镜像标签')
    return
  }
  
  // 如果应用名称为空，自动使用镜像仓库的值
  const appName = imageForm.value.name || imageForm.value.imageRepository
  
  // 如果版本为空，自动使用镜像标签的值
  const appVersion = imageForm.value.version || imageForm.value.imageTag
  
  creating.value = true
  try {
    const portMappings = imageForm.value.portMappings.filter(m => m.host > 0 && m.container > 0)
    const res = await fetch(`${API_BASE}/v1/apps/create-image`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: appName,
        version: appVersion,
        imageRepository: imageForm.value.imageRepository,
        imageTag: imageForm.value.imageTag,
        portMappings: portMappings
      })
    })
    if (!res.ok) {
      const text = await res.text()
      throw new Error(text || `HTTP ${res.status}`)
    }
    ElMessage.success('创建成功')
    showImageDialog.value = false
    load()
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    creating.value = false
  }
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
          {{ t('apps.buttons.refresh') }}
        </el-button>
         <el-upload
           :action="uploadUrl"
           name="file"
           :multiple="true"
           :show-file-list="false"
           :before-upload="beforeUpload"
           :on-success="onSuccess"
           :on-error="onError"
           accept=".zip"
         >
          <el-button type="primary">
            <el-icon><Upload /></el-icon>
            {{ t('apps.buttons.selectUpload') }}
          </el-button>
        </el-upload>
                <el-button type="primary" @click="openImageDialog">
                  {{ t('apps.buttons.createFromImage') }}
                </el-button>
        <el-button
          type="danger"
          :disabled="!selectedIds.length"
          :loading="deleting"
          @click="deleteSelected"
        >
          <el-icon><Delete /></el-icon>
          {{ t('apps.buttons.batchDelete') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <!-- <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Files /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ (items || []).length }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('apps.stats.total') }}</span>
        </div>
      </div> -->
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 应用包列表表格 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('apps.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ (items || []).length }} {{ t('apps.table.items') }}</span>
        </div>
      </template>
      
      <el-table
        v-loading="loading"
        :data="paginatedItems"
        style="width:100%;"
        stripe
        row-key="artifactId"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="48" />
        <el-table-column prop="name" :label="t('apps.columns.app')" width="200" />
        <el-table-column prop="version" :label="t('apps.columns.version')" width="100" />
        <el-table-column label="类型" width="80">
          <template #default="{ row }">
            <el-tag :type="row.type === 'image' ? 'success' : 'info'" size="small">
              {{ row.type === 'image' ? '镜像' : 'ZIP' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('apps.columns.artifact')">
          <template #default="{ row }">
            <template v-if="row.type === 'image'">
              <span style="color:#67C23A; font-family: monospace;">
                {{ row.imageRepository }}:{{ row.imageTag }}
              </span>
            </template>
            <template v-else>
              <a :href="artifactHref(row)" target="_blank" style="color:#409EFF; text-decoration:none;">
                {{ artifactHref(row) }}
              </a>
            </template>
          </template>
        </el-table-column>
        <el-table-column prop="sizeBytes" :label="t('apps.columns.sizeBytes')" width="140">
          <template #default="{ row }">
            {{ formatFileSize(row.sizeBytes) }}
          </template>
        </el-table-column>
        <el-table-column :label="t('apps.columns.uploadedAt')" width="180">
          <template #default="{ row }">{{ formatTimestamp(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.action')" width="160">
          <template #default="{ row }">
            <el-popconfirm :title="t('apps.confirmDelete')" @confirm="del(row.artifactId)">
              <template #reference>
                <el-button type="danger" size="small">
                  <el-icon><Delete /></el-icon>
                  {{ t('common.delete') }}
                </el-button>
              </template>
            </el-popconfirm>
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

    <!-- 使用镜像创建应用弹窗 -->
    <el-dialog
      v-model="showImageDialog"
      :title="t('apps.createImage.title')"
      width="600px"
      @close="resetImageForm"
    >
      <el-form :model="imageForm" label-width="120px">
        <el-form-item :label="t('apps.createImage.imageRepository')" required>
          <el-autocomplete
            v-model="imageForm.imageRepository"
            :placeholder="t('apps.createImage.example.repository') || '例如: nginx 或 registry.example.com/namespace/image'"
            :fetch-suggestions="queryRepositorySuggestions"
            style="width: 100%"
            clearable
            @select="onRepositoryChange"
            @change="onRepositoryChange"
          >
            <template #default="{ item }">
              <div>{{ item.value }}</div>
            </template>
          </el-autocomplete>
          <div style="font-size: 12px; color: #909399; margin-top: 4px;">
            可以输入本地镜像或完整的镜像仓库路径（如: registry.example.com/namespace/image）
          </div>
        </el-form-item>
        <el-form-item :label="t('apps.createImage.imageTag')" required>
          <el-select
            v-model="imageForm.imageTag"
            :placeholder="t('apps.createImage.example.tag')"
            filterable
            allow-create
            default-first-option
            style="width: 100%"
            :disabled="!imageForm.imageRepository"
          >
            <el-option
              v-for="tag in imageTagOptions"
              :key="tag"
              :label="tag"
              :value="tag"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('apps.createImage.name')">
          <el-input v-model="imageForm.name" :placeholder="t('apps.createImage.name')" />
          <div style="font-size: 12px; color: #909399; margin-top: 4px;">
            留空时自动使用镜像仓库的值
          </div>
        </el-form-item>
        <el-form-item :label="t('apps.createImage.version')">
          <el-input v-model="imageForm.version" :placeholder="t('apps.createImage.version')" />
          <div style="font-size: 12px; color: #909399; margin-top: 4px;">
            留空时自动使用镜像标签的值
          </div>
        </el-form-item>
        <el-form-item :label="t('apps.createImage.portMappings')">
          <div v-for="(mapping, index) in imageForm.portMappings" :key="index" style="display: flex; gap: 8px; margin-bottom: 8px;">
            <el-input-number
              v-model="mapping.host"
              :placeholder="t('apps.createImage.hostPort')"
              :min="1"
              :max="65535"
              style="flex: 1"
            />
            <span style="line-height: 32px;">:</span>
            <el-input-number
              v-model="mapping.container"
              :placeholder="t('apps.createImage.containerPort')"
              :min="1"
              :max="65535"
              style="flex: 1"
            />
            <el-button type="danger" size="small" @click="imageForm.portMappings.splice(index, 1)">
              {{ t('apps.createImage.remove') }}
            </el-button>
          </div>
          <el-button type="primary" size="small" @click="imageForm.portMappings.push({ host: 0, container: 0 })">
            {{ t('apps.createImage.addPortMapping') }}
          </el-button>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showImageDialog = false">{{ t('apps.createImage.cancel') }}</el-button>
        <el-button type="primary" :loading="creating" @click="createImageApp">
          {{ t('apps.createImage.submit') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>


