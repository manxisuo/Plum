<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
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
}

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const loading = ref(false)
const items = ref<Artifact[]>([])
const uploadUrl = `${API_BASE}/v1/apps/upload`

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

async function del(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/apps/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (res.status === 409) {
      ElMessage.error('该应用包正在被部署使用，无法删除')
      return
    }
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || '删除失败')
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
const { t } = useI18n()

function artifactHref(row: Artifact): string {
  const u = row.url || ''
  if (u.startsWith('http://') || u.startsWith('https://')) return u
  const base = API_BASE || 'http://127.0.0.1:8080'
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
           :multiple="false"
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
      
      <el-table v-loading="loading" :data="paginatedItems" style="width:100%;" stripe>
        <el-table-column prop="name" :label="t('apps.columns.app')" width="200" />
        <el-table-column prop="version" :label="t('apps.columns.version')" width="100" />
        <el-table-column :label="t('apps.columns.artifact')">
          <template #default="{ row }">
            <a :href="artifactHref(row)" target="_blank" style="color:#409EFF; text-decoration:none;">
              {{ artifactHref(row) }}
            </a>
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
  </div>
</template>


