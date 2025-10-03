<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
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
</script>

<template>
  <div style="display:flex; gap:12px;">
    <el-card style="width:240px;">
      <template #header>{{ t('services.title') }}</template>
      <el-menu :default-active="active" @select="(k:string)=>{active=k; loadEndpoints()}">
        <el-menu-item v-for="s in services" :key="s" :index="s">{{ s }}</el-menu-item>
      </el-menu>
    </el-card>
    <el-card style="flex:1;">
      <template #header>{{ t('services.endpointsTitle', { name: active || '-' }) }}</template>
      <el-table :data="eps" v-loading="loading" style="width:100%">
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
  
</template>


