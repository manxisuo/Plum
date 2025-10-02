<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

type Deployment = { deploymentId: string; name: string; labels?: Record<string,string>; instances: number }

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const items = ref<Deployment[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/deployments`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    items.value = await res.json() as Deployment[]
  } catch (e:any) {
    ElMessage.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function removeDeployment(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}`, { method: 'DELETE' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已删除')
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || '删除失败')
  }
}

onMounted(load)
const { t } = useI18n()
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">{{ t('common.refresh') }}</el-button>
      <router-link to="/deployments/create"><el-button type="success">{{ t('deployments.buttons.create') }}</el-button></router-link>
    </div>
    <el-table v-loading="loading" :data="items" style="width:100%; margin-top:12px;">
      <el-table-column prop="deploymentId" :label="t('deployments.columns.deploymentId')" width="320" />
      <el-table-column prop="name" :label="t('deployments.columns.name')" width="220" />
      <el-table-column prop="instances" :label="t('deployments.columns.instances')" width="120" />
      <el-table-column :label="t('common.action')" width="260">
        <template #default="{ row }">
          <div style="display:flex; gap:8px; align-items:center;">
            <router-link :to="'/deployments/'+row.deploymentId"><el-button size="small">{{ t('common.details') }}</el-button></router-link>
            <router-link :to="'/deployments/'+row.deploymentId+'/config'"><el-button size="small">{{ t('common.config') }}</el-button></router-link>
            <el-popconfirm :title="t('deployments.confirmDelete')" @confirm="removeDeployment(row.deploymentId)">
              <template #reference>
                <el-button type="danger" size="small">{{ t('common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>


