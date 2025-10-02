<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

type NodeDTO = { nodeId: string; ip: string; labels?: Record<string,string>; lastSeen: number }
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const nodes = ref<NodeDTO[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function refresh() {
  loading.value = true
  error.value = null
  try {
    const res = await fetch(`${API_BASE}/v1/nodes`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    nodes.value = await res.json() as NodeDTO[]
  } catch (e:any) {
    error.value = e?.message || '请求失败'
  } finally {
    loading.value = false
  }
}

async function remove(id: string) {
  if (!confirm(`删除节点 ${id} ?`)) return
  const res = await fetch(`${API_BASE}/v1/nodes/${encodeURIComponent(id)}`, { method: 'DELETE' })
  if (res.ok) refresh()
}

onMounted(refresh)
const { t } = useI18n()
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="refresh">{{ t('common.refresh') }}</el-button>
      <el-alert v-if="error" type="error" :closable="false" :title="`错误：${error}`" />
    </div>
    <el-table v-loading="loading" :data="nodes" style="width:100%; margin-top:12px;">
      <el-table-column prop="nodeId" :label="t('nodes.columns.nodeId')" width="260" />
      <el-table-column prop="ip" :label="t('nodes.columns.ip')" width="180" />
      <el-table-column :label="t('nodes.columns.lastSeen')" width="220">
        <template #default="{ row }">{{ new Date(row.lastSeen*1000).toLocaleString() }}</template>
      </el-table-column>
      <el-table-column :label="t('nodes.columns.action')" width="140">
        <template #default="{ row }">
          <el-popconfirm :title="t('nodes.confirmDelete')" @confirm="remove(row.nodeId)">
            <template #reference>
              <el-button type="danger" size="small">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>


