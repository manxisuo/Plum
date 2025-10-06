<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

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
		items.value = await res.json() as Artifact[]
	} catch (e:any) {
		ElMessage.error(e?.message || '加载失败')
	} finally {
		loading.value = false
	}
}

function onSuccess() {
	ElMessage.success('上传成功')
	load()
}

function onError(err: any) {
	ElMessage.error('上传失败')
}

async function del(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/apps/${encodeURIComponent(id)}`, { method: 'DELETE' })
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
</script>

<template>
  <div>
    <el-card>
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('apps.uploadZip') }}</span>
          <small>{{ t('apps.zipTip') }}</small>
        </div>
      </template>
      <el-upload
        :action="uploadUrl"
        name="file"
        :multiple="false"
        :show-file-list="false"
        :on-success="onSuccess"
        :on-error="onError"
        accept=".zip"
      >
        <el-button type="primary">{{ t('apps.buttons.selectUpload') }}</el-button>
      </el-upload>
    </el-card>

    <el-table v-loading="loading" :data="paginatedItems" style="width:100%; margin-top:12px;">
      <el-table-column prop="name" :label="t('apps.columns.app')" width="200" />
      <el-table-column prop="version" :label="t('apps.columns.version')" width="100" />
      <el-table-column :label="t('apps.columns.artifact')">
        <template #default="{ row }">
          <a :href="artifactHref(row)" target="_blank">{{ artifactHref(row) }}</a>
        </template>
      </el-table-column>
      <el-table-column prop="sizeBytes" :label="t('apps.columns.sizeBytes')" width="140" />
      <el-table-column :label="t('apps.columns.uploadedAt')" width="180">
        <template #default="{ row }">{{ new Date(row.createdAt*1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column :label="t('common.action')" width="160">
        <template #default="{ row }">
          <el-popconfirm :title="t('apps.confirmDelete')" @confirm="del(row.artifactId)">
            <template #reference>
              <el-button type="danger" size="small">{{ t('common.delete') }}</el-button>
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
        :total="items.length"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
  
</template>


