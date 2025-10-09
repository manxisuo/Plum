<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Refresh, Plus, DataBoard, List, VideoPlay, VideoPause } from '@element-plus/icons-vue'
import IdDisplay from './IdDisplay.vue'

type Deployment = { deploymentId: string; name: string; labels?: Record<string,string>; status: string; instances: number }

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
    ElMessage.success(t('deployments.deleteSuccess'))
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || t('deployments.deleteFailed'))
  }
}

async function startDeployment(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}?action=start`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(t('deployments.startSuccess'))
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || t('deployments.startFailed'))
  }
}

async function stopDeployment(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}?action=stop`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(t('deployments.stopSuccess'))
    load()
  } catch (e:any) {
    ElMessage.error(e?.message || t('deployments.stopFailed'))
  }
}

onMounted(load)
const { t } = useI18n()

// 统计计算
const totalDeployments = computed(() => items.value.length)
const runningDeployments = computed(() => items.value.filter(d => d.status === 'Running').length)
const stoppedDeployments = computed(() => items.value.filter(d => d.status === 'Stopped').length)
const totalInstances = computed(() => items.value.reduce((sum, item) => sum + item.instances, 0))
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="load">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
        <router-link to="/deployments/create">
          <el-button type="success">
            <el-icon><Plus /></el-icon>
            {{ t('deployments.buttons.create') }}
          </el-button>
        </router-link>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><DataBoard /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalDeployments }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deployments.stats.total') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPlay /></el-icon>
          </div>
          <span style="font-weight:bold; color:#67C23A;">{{ runningDeployments }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deployments.stats.running') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #909399, #C0C4CC); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><VideoPause /></el-icon>
          </div>
          <span style="font-weight:bold; color:#909399;">{{ stoppedDeployments }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deployments.stats.stopped') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><List /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalInstances }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deployments.stats.instances') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 部署列表表格 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deployments.table.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ items.length }} {{ t('deployments.table.items') }}</span>
        </div>
      </template>
      
      <el-table v-loading="loading" :data="items" style="width:100%;" stripe>
      <el-table-column prop="deploymentId" :label="t('deployments.columns.id')" width="280">
        <template #default="{ row }">
          <IdDisplay :id="row.deploymentId" :length="32" />
        </template>
      </el-table-column>
      <el-table-column prop="name" :label="t('deployments.columns.name')" width="200" />
      <el-table-column :label="t('common.status')" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'Running' ? 'success' : 'info'" size="small">
            {{ row.status === 'Running' ? t('deployments.status.running') : t('deployments.status.stopped') }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="instances" :label="t('deployments.columns.instances')" width="100" />
      <el-table-column :label="t('common.action')" width="380">
        <template #default="{ row }">
          <div style="display:flex; gap:8px; align-items:center;">
            <el-button 
              v-if="row.status === 'Stopped'"
              type="success" 
              size="small"
              @click="startDeployment(row.deploymentId)">
              <el-icon><VideoPlay /></el-icon>
              {{ t('deployments.buttons.start') }}
            </el-button>
            <el-button 
              v-else
              type="warning" 
              size="small"
              @click="stopDeployment(row.deploymentId)">
              <el-icon><VideoPause /></el-icon>
              {{ t('deployments.buttons.stop') }}
            </el-button>
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
    </el-card>
  </div>
</template>


