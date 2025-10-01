<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'

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
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">刷新</el-button>
      <router-link to="/deployments/create"><el-button type="success">创建部署</el-button></router-link>
    </div>
    <el-table v-loading="loading" :data="items" style="width:100%; margin-top:12px;">
      <el-table-column prop="deploymentId" label="DeploymentID" width="320" />
      <el-table-column prop="name" label="Name" width="220" />
      <el-table-column prop="instances" label="Instances" width="120" />
      <el-table-column label="Action" width="260">
        <template #default="{ row }">
          <div style="display:flex; gap:8px; align-items:center;">
            <router-link :to="'/deployments/'+row.deploymentId"><el-button size="small">详情</el-button></router-link>
            <router-link :to="'/deployments/'+row.deploymentId+'/config'"><el-button size="small">配置</el-button></router-link>
            <el-popconfirm title="确认删除该部署？（不会级联删除实例分配）" @confirm="removeDeployment(row.deploymentId)">
              <template #reference>
                <el-button type="danger" size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>


