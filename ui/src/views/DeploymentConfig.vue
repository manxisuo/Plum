<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Refresh, DataBoard, List, Setting } from '@element-plus/icons-vue'

type Artifact = { artifactId: string; name: string; version: string; url: string }
type NodeDTO = { nodeId: string; ip: string }

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const route = useRoute()
const id = route.params.id as string
const loading = ref(false)
const deployment = ref<any>(null)
const artifacts = ref<Artifact[]>([])
const nodes = ref<NodeDTO[]>([])
// entries read-only for now (future: edit)
const entries = ref<any[]>([])
const labels = ref<Record<string,string>>({})

// 分页相关
const currentPage = ref(1)
const pageSize = ref(10)
const pageSizes = [10, 20, 50, 100]

// 计算属性：统计信息
const totalEntries = computed(() => entries.value.length)
const totalLabels = computed(() => Object.keys(labels.value).length)
const totalReplicas = computed(() => {
  return entries.value.reduce((sum, entry) => {
    return sum + Object.values(entry.replicas || {}).reduce((entrySum: number, count: any) => entrySum + (count || 0), 0)
  }, 0)
})

// 计算属性：分页后的数据
const paginatedEntries = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return entries.value.slice(start, end)
})

// 计算属性：总页数
const totalPages = computed(() => {
  return Math.ceil(entries.value.length / pageSize.value)
})

// 分页事件处理
function handleSizeChange(val: number) {
  pageSize.value = val
  currentPage.value = 1 // 重置到第一页
}

function handleCurrentChange(val: number) {
  currentPage.value = val
}

async function load() {
  loading.value = true
  try {
    const [tRes, aRes, nRes] = await Promise.all([
      fetch(`${API_BASE}/v1/deployments/${encodeURIComponent(id)}`),
      fetch(`${API_BASE}/v1/apps`),
      fetch(`${API_BASE}/v1/nodes`)
    ])
    if (!tRes.ok) throw new Error('HTTP '+tRes.status)
    const tj = await tRes.json()
    deployment.value = tj.deployment
    labels.value = (tj.deployment?.labels || tj.deployment?.Labels || {})
    // reconstruct entries roughly from assignments (group by artifactUrl+startCmd)
    const asgs = (tj.assignments || []) as any[]
    const keyMap = new Map<string, any>()
    for (const a of asgs) {
      const art = a.artifactUrl || a.ArtifactURL
      const cmd = a.startCmd || a.StartCmd
      const node = a.nodeId || a.NodeID
      const key = `${art}|${cmd}`
      if (!keyMap.has(key)) keyMap.set(key, { artifactUrl: art, startCmd: cmd, replicas: {} as Record<string, number> })
      const entry = keyMap.get(key)
      entry.replicas[node] = (entry.replicas[node] || 0) + 1
    }
    entries.value = Array.from(keyMap.values())
    if (aRes.ok) artifacts.value = await aRes.json() as Artifact[]
    if (nRes.ok) nodes.value = await nRes.json() as NodeDTO[]
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}


onMounted(load)
const { t } = useI18n()
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
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><List /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalEntries }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentConfig.stats.entries') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><DataBoard /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalReplicas }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentConfig.stats.replicas') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #67C23A, #85CE61); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Setting /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalLabels }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('deploymentConfig.stats.labels') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- 部署配置详情 -->
    <el-card class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deploymentConfig.title') }}</span>
        </div>
      </template>
      
      <el-descriptions v-if="deployment" :column="2" border>
        <el-descriptions-item :label="t('deploymentDetail.desc.deploymentId')">{{ deployment.deploymentId }}</el-descriptions-item>
        <el-descriptions-item :label="t('deploymentDetail.desc.name')">{{ deployment.name || deployment.Name }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 条目配置 -->
    <el-card class="box-card" style="margin-bottom: 16px;">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deploymentConfig.entriesTitle') }}</span>
          <span style="font-size:14px; color:#909399;">{{ entries.length }} {{ t('deploymentConfig.table.items') }}</span>
        </div>
      </template>
      
      <el-table :data="paginatedEntries" v-loading="loading" style="width:100%;" stripe>
        <el-table-column prop="artifactUrl" :label="t('deploymentConfig.columns.artifact')" />
        <el-table-column prop="startCmd" :label="t('deploymentConfig.columns.startCmd')" />
        <el-table-column :label="t('deploymentConfig.columns.replicas')">
          <template #default="{ row }">
            <code>{{ JSON.stringify(row.replicas) }}</code>
          </template>
        </el-table-column>
      </el-table>
      
      <!-- 分页组件 -->
      <div style="margin-top: 16px; display: flex; justify-content: center;">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="pageSizes"
          :total="entries.length"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 标签配置 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deploymentConfig.labelsTitle') }}</span>
          <span style="font-size:14px; color:#909399;">{{ totalLabels }} {{ t('deploymentConfig.table.labels') }}</span>
        </div>
      </template>
      
      <div v-if="totalLabels > 0">
        <div v-for="(v,k) in labels" :key="k" style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
          <el-input :model-value="k" disabled style="flex:1" />
          <el-input v-model="labels[k]" style="flex:2" />
        </div>
      </div>
      <div v-else style="text-align:center; color:#909399; padding:20px;">
        {{ t('deploymentConfig.noLabels') }}
      </div>
    </el-card>
  </div>
</template>


