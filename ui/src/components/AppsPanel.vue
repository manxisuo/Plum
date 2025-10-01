<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

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

onMounted(load)

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
          <span>上传应用包（zip）</span>
          <small>包内需包含 start.sh 与 meta.ini(name/version)</small>
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
        <el-button type="primary">选择并上传 ZIP</el-button>
      </el-upload>
    </el-card>

    <el-table v-loading="loading" :data="items" style="width:100%; margin-top:12px;">
      <el-table-column prop="name" label="App" width="240" />
      <el-table-column prop="version" label="Version" width="140" />
      <el-table-column label="Artifact">
        <template #default="{ row }">
          <a :href="artifactHref(row)" target="_blank">{{ artifactHref(row) }}</a>
        </template>
      </el-table-column>
      <el-table-column prop="sizeBytes" label="Size(Bytes)" width="140" />
      <el-table-column label="UploadedAt" width="200">
        <template #default="{ row }">{{ new Date(row.createdAt*1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column label="Action" width="140">
        <template #default="{ row }">
          <el-popconfirm title="确认删除该包？" @confirm="del(row.artifactId)">
            <template #reference>
              <el-button type="danger" size="small">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </div>
  
</template>


