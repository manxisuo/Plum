<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh, Connection, CircleCheck, CircleClose } from '@element-plus/icons-vue'
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
type Endpoint = { serviceName: string; instanceId: string; nodeId: string; ip: string; port: number; protocol: string; version?: string; healthy: boolean; lastSeen: number; labels?: Record<string,string> }
const services = ref<string[]>([])
const active = ref<string>('')
const loading = ref(false)
const eps = ref<Endpoint[]>([])

async function loadServices(){
  try { const res = await fetch(`${API_BASE}/v1/services/list`); if (res.ok) services.value = await res.json() as string[]; if (!active.value && services.value.length) { active.value = services.value[0]; loadEndpoints() } } catch {}
}
async function loadEndpoints(){
  if (!active.value) return
  loading.value = true
  try { const res = await fetch(`${API_BASE}/v1/discovery?service=${encodeURIComponent(active.value)}`); if (res.ok) eps.value = await res.json() as Endpoint[] } finally { loading.value = false }
}
onMounted(loadServices)
const { t } = useI18n()

// 统计计算
const totalServices = computed(() => services.value.length)
const totalEndpoints = computed(() => eps.value.length)
const healthyEndpoints = computed(() => eps.value.filter(ep => ep.healthy).length)
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
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Connection /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalServices }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('services.stats.services') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><CircleCheck /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalEndpoints }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('services.stats.endpoints') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><CircleCheck /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ healthyEndpoints }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('services.stats.healthy') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <div style="display:flex; gap:12px;">
      <el-card style="width:240px;">
        <template #header>{{ t('services.title') }}</template>
        <el-menu :default-active="active" @select="(k:string)=>{active=k; loadEndpoints()}">
          <el-menu-item v-for="s in services" :key="s" :index="s">{{ s }}</el-menu-item>
        </el-menu>
      </el-card>
      <el-card style="flex:1;">
        <template #header>{{ t('services.endpointsTitle', { name: active || '-' }) }}</template>
        <el-table :data="eps" v-loading="loading" style="width:100%" stripe>
        <el-table-column prop="instanceId" :label="t('services.columns.instance')" width="320" />
        <el-table-column prop="nodeId" :label="t('services.columns.node')" width="160" />
        <el-table-column :label="t('services.columns.address')">
          <template #default="{ row }">{{ row.ip }}:{{ row.port }} ({{ row.protocol }})</template>
        </el-table-column>
        <el-table-column :label="t('services.columns.healthy')" width="100">
          <template #default="{ row }"><el-tag :type="row.healthy?'success':'danger'">{{ row.healthy?'true':'false' }}</el-tag></template>
        </el-table-column>
        <el-table-column :label="t('services.columns.lastSeen')" width="200">
          <template #default="{ row }">{{ new Date(row.lastSeen*1000).toLocaleString() }}</template>
        </el-table-column>
        </el-table>
      </el-card>
    </div>
  </div>
</template>


