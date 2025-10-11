<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Refresh, FolderOpened, Key, Coin, Document } from '@element-plus/icons-vue'

const { t } = useI18n()
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''

interface Namespace {
  name: string
  keyCount?: number
  keys?: string[]
  loading?: boolean
}

const namespaces = ref<Namespace[]>([])
const loading = ref(false)
const expandedNamespaces = ref<string[]>([])

// 加载所有namespace列表
async function loadNamespaces() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/kv`)
    if (!res.ok) throw new Error('Failed to fetch namespaces')
    const data = await res.json()
    
    const nsList = (data.namespaces || []) as string[]
    
    // 加载每个namespace的keys（并发加载）
    const nsPromises = nsList.map(async (name) => {
      try {
        const keysRes = await fetch(`${API_BASE}/v1/kv/${encodeURIComponent(name)}/keys`)
        if (keysRes.ok) {
          const keysData = await keysRes.json()
          return {
            name,
            keys: (keysData.keys || []) as string[],
            keyCount: (keysData.keys || []).length,
            loading: false
          }
        }
      } catch (err) {
        console.error(`Failed to load keys for ${name}:`, err)
      }
      return { name, keys: [], keyCount: 0, loading: false }
    })
    
    namespaces.value = await Promise.all(nsPromises)
  } catch (err: any) {
    ElMessage.error(t('kvStore.errors.loadNamespacesFailed') || 'Failed to load namespaces')
    console.error('Load namespaces error:', err)
    namespaces.value = []
  } finally {
    loading.value = false
  }
}

// 刷新当前列表
function refresh() {
  expandedNamespaces.value = []
  loadNamespaces()
}

// 统计信息
const totalNamespaces = computed(() => namespaces.value.length)
const totalKeys = computed(() => {
  return namespaces.value.reduce((sum, ns) => sum + (ns.keyCount || 0), 0)
})

onMounted(() => {
  loadNamespaces()
})
</script>

<template>
  <div>
    <!-- 操作按钮和统计信息 -->
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
      <!-- 操作按钮 -->
      <div style="display:flex; gap:8px; flex-shrink:0;">
        <el-button type="primary" :loading="loading" @click="refresh">
          <el-icon><Refresh /></el-icon>
          {{ t('common.refresh') }}
        </el-button>
      </div>
      
      <!-- 统计信息 -->
      <div style="display:flex; gap:20px; align-items:center; flex:1; justify-content:center;">
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #409EFF, #67C23A); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><FolderOpened /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalNamespaces }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('kvStore.stats.namespaces') }}</span>
        </div>
        
        <div style="display:flex; align-items:center; gap:6px;">
          <div style="width:20px; height:20px; background:linear-gradient(135deg, #E6A23C, #F56C6C); border-radius:4px; display:flex; align-items:center; justify-content:center;">
            <el-icon size="12" color="white"><Key /></el-icon>
          </div>
          <span style="font-weight:bold;">{{ totalKeys }}</span>
          <span style="font-size:12px; color:#909399;">{{ t('kvStore.stats.totalKeys') }}</span>
        </div>
      </div>
      
      <!-- 占位空间保持居中 -->
      <div style="flex-shrink:0; width:120px;"></div>
    </div>

    <!-- KV存储列表 -->
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('kvStore.title') }}</span>
          <span style="font-size:14px; color:#909399;">{{ namespaces.length }} {{ t('kvStore.stats.namespaces') }}</span>
        </div>
      </template>
      
      <!-- 空状态 -->
      <div v-if="namespaces.length === 0 && !loading" style="padding:40px 0;">
        <el-empty :description="t('kvStore.empty')" />
      </div>

      <!-- Namespace列表 -->
      <el-collapse v-else v-model="expandedNamespaces" v-loading="loading">
        <el-collapse-item 
          v-for="ns in namespaces" 
          :key="ns.name" 
          :name="ns.name"
        >
          <template #title>
            <div style="display:flex; align-items:center; gap:12px; width:100%;">
              <el-icon color="#409EFF" size="18"><FolderOpened /></el-icon>
              <span style="font-weight:600; color:#303133;">{{ ns.name }}</span>
              <el-tag type="primary" size="small" effect="plain">
                {{ ns.keyCount }} {{ t('kvStore.keys') }}
              </el-tag>
            </div>
          </template>

          <!-- Keys列表 -->
          <div v-if="ns.keys && ns.keys.length > 0" style="padding:12px 16px; background:#fafafa; border-radius:4px;">
            <div style="display:flex; flex-wrap:wrap; gap:8px;">
              <el-tag 
                v-for="key in ns.keys" 
                :key="key"
                type="info"
                effect="plain"
                size="small"
              >
                <el-icon size="12" style="margin-right:4px;"><Key /></el-icon>
                {{ key }}
              </el-tag>
            </div>
          </div>
          
          <!-- 空状态 -->
          <div v-else style="padding:20px 0; text-align:center;">
            <el-empty :description="t('kvStore.noKeys')" :image-size="60" />
          </div>
        </el-collapse-item>
      </el-collapse>
    </el-card>
  </div>
</template>

