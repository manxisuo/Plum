<script setup lang="ts">
import { ref, onMounted, watch, onBeforeUnmount, nextTick, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import * as echarts from 'echarts'
import { 
  Monitor, DataBoard, Box, Files, 
  TrendCharts, PieChart, 
  ArrowUp, ArrowDown, 
  Refresh, MoreFilled,
  CircleCheck, CircleClose,
  Connection, Cpu
} from '@element-plus/icons-vue'
const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''

type NodeRow = { nodeId: string; health?: string }
type DeploymentRow = { deploymentId: string; instances: number }
type Artifact = { sizeBytes: number }
type Endpoint = { serviceName: string }

const loading = ref(false)
const nodes = ref<NodeRow[]>([])
const deployments = ref<DeploymentRow[]>([])
const services = ref<string[]>([])
const endpointsCount = ref(0)
const artifactsTotal = ref(0)

let chartServices: echarts.ECharts | null = null
let chartNodes: echarts.ECharts | null = null
const chartServicesEl = ref<HTMLDivElement | null>(null)
const chartNodesEl = ref<HTMLDivElement | null>(null)

async function loadAll() {
  loading.value = true
  try {
    const [nRes, tRes, sRes, aRes] = await Promise.all([
      fetch(`${API_BASE}/v1/nodes`),
      fetch(`${API_BASE}/v1/deployments`),
      fetch(`${API_BASE}/v1/services/list`),
      fetch(`${API_BASE}/v1/apps`),
    ])
    if (nRes.ok) nodes.value = await nRes.json() as any
    if (tRes.ok) deployments.value = await tRes.json() as any
    if (sRes.ok) services.value = await sRes.json() as any
    if (aRes.ok) {
      const apps = await aRes.json() as Artifact[]
      artifactsTotal.value = apps.reduce((s, x) => s + (x.sizeBytes||0), 0)
    }
    // count endpoints by listing per service (轻量，MVP 可接受)
    let ep = 0
    for (const s of services.value) {
      try { const r = await fetch(`${API_BASE}/v1/discovery?service=${encodeURIComponent(s)}`); if (r.ok) { const arr = await r.json() as Endpoint[]; ep += arr.length } } catch {}
    }
    endpointsCount.value = ep
  } finally { loading.value = false }
  await nextTick()
  renderCharts()
}

// 计算属性
const healthyNodes = computed(() => nodes.value.filter(n => (n as any).health === 'Healthy').length)
const unhealthyNodes = computed(() => nodes.value.length - healthyNodes.value)
const runningInstances = computed(() => deployments.value.reduce((s, t) => s + (t.instances||0), 0))
const totalNodes = computed(() => nodes.value.length)
const totalDeployments = computed(() => deployments.value.length)
const totalServices = computed(() => services.value.length)
const totalArtifactsSize = computed(() => (artifactsTotal.value / 1024 / 1024).toFixed(1))

// 健康状态比例
const healthyPercentage = computed(() => {
  if (totalNodes.value === 0) return 0
  return Math.round((healthyNodes.value / totalNodes.value) * 100)
})

// 最近活动节点
const recentNodes = computed(() => nodes.value.slice(0, 5))

onMounted(loadAll)
onBeforeUnmount(()=>{ chartServices?.dispose(); chartNodes?.dispose(); chartServices=null; chartNodes=null })

function renderCharts(){
  // Services endpoints bar
  if (chartServicesEl.value) {
    chartServices?.dispose(); chartServices = echarts.init(chartServicesEl.value)
    const names = services.value.slice(0, 12)
    const data: number[] = []
    ;(async()=>{
      for (const s of names) {
        try { const r = await fetch(`${API_BASE}/v1/discovery?service=${encodeURIComponent(s)}`); if (r.ok) { const arr = await r.json() as any[]; data.push(arr.length) } else data.push(0) } catch { data.push(0) }
      }
      chartServices!.setOption({
        tooltip: { trigger: 'axis' },
        xAxis: { type: 'category', data: names },
        yAxis: { type: 'value' },
        series: [{ type: 'bar', data, itemStyle: { color: '#409EFF' } }]
      })
    })()
  }
  // Nodes health pie
  if (chartNodesEl.value) {
    chartNodes?.dispose(); chartNodes = echarts.init(chartNodesEl.value)
    chartNodes.setOption({
      tooltip: { 
        trigger: 'item',
        formatter: '{b}: {c} ({d}%)'
      },
      legend: {
        show: false
      },
      series: [{
        type: 'pie',
        radius: ['50%','80%'],
        center: ['40%', '50%'],
        avoidLabelOverlap: false,
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: '20',
            fontWeight: 'bold'
          }
        },
        labelLine: {
          show: false
        },
        data: [
          { 
            name: t('home.cards.healthy'), 
            value: healthyNodes.value,
            itemStyle: { color: '#67C23A' }
          },
          { 
            name: t('home.cards.unhealthy'), 
            value: unhealthyNodes.value,
            itemStyle: { color: '#F56C6C' }
          }
        ]
      }]
    })
  }
}

watch([services, nodes], ()=>{ nextTick().then(renderCharts) })

const { t } = useI18n()
</script>

<template>
  <div class="home-container">
    <!-- 页面标题和欢迎信息 -->
    <!-- <el-card class="welcome-card" shadow="never">
      <div class="welcome-content">
        <div class="welcome-text">
          <h1 class="welcome-title">{{ t('home.welcome.title') }}</h1>
          <p class="welcome-subtitle">{{ t('home.welcome.subtitle') }}</p>
        </div>
        <div class="welcome-actions">
          <el-button type="primary" :loading="loading" @click="loadAll">
            <el-icon><Refresh /></el-icon>
            {{ t('home.buttons.refresh') }}
          </el-button>
          <el-button type="success" @click="$router.push('/deployments/create')">
            <el-icon><Plus /></el-icon>
            {{ t('home.buttons.createDeployment') }}
          </el-button>
        </div>
      </div>
    </el-card> -->

    <!-- 统计概览卡片 -->
    <div class="stats-grid">
      <el-card class="stat-card nodes-card" shadow="hover">
        <div class="stat-content">
          <div class="stat-icon">
            <el-icon size="32"><Monitor /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-title">{{ t('home.cards.nodes') }}</div>
            <div class="stat-value">{{ totalNodes }}</div>
            <div class="stat-detail">
              <el-tag type="success" size="small">
                <el-icon><CircleCheck /></el-icon>
                {{ healthyNodes }}
              </el-tag>
              <el-tag type="danger" size="small" v-if="unhealthyNodes > 0">
                <el-icon><CircleClose /></el-icon>
                {{ unhealthyNodes }}
              </el-tag>
            </div>
          </div>
          <div class="stat-link">
            <RouterLink to="/nodes">
              <el-icon><ArrowUp /></el-icon>
            </RouterLink>
          </div>
        </div>
      </el-card>

      <el-card class="stat-card deployments-card" shadow="hover">
        <div class="stat-content">
          <div class="stat-icon">
            <el-icon size="32"><DataBoard /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-title">{{ t('home.cards.deployments') }}</div>
            <div class="stat-value">{{ totalDeployments }}</div>
            <div class="stat-detail">
              <span class="stat-subtitle">{{ t('home.cards.instances') }}: {{ runningInstances }}</span>
            </div>
          </div>
          <div class="stat-link">
            <RouterLink to="/deployments">
              <el-icon><ArrowUp /></el-icon>
            </RouterLink>
          </div>
        </div>
      </el-card>

      <el-card class="stat-card services-card" shadow="hover">
        <div class="stat-content">
          <div class="stat-icon">
            <el-icon size="32"><Connection /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-title">{{ t('home.cards.services') }}</div>
            <div class="stat-value">{{ totalServices }}</div>
            <div class="stat-detail">
              <span class="stat-subtitle">{{ t('home.cards.endpoints') }}: {{ endpointsCount }}</span>
            </div>
          </div>
          <div class="stat-link">
            <RouterLink to="/services">
              <el-icon><ArrowUp /></el-icon>
            </RouterLink>
          </div>
        </div>
      </el-card>

      <el-card class="stat-card artifacts-card" shadow="hover">
        <div class="stat-content">
          <div class="stat-icon">
            <el-icon size="32"><Box /></el-icon>
          </div>
          <div class="stat-info">
            <div class="stat-title">{{ t('home.cards.artifacts') }}</div>
            <div class="stat-value">{{ totalArtifactsSize }} MB</div>
            <div class="stat-detail">
              <span class="stat-subtitle">{{ t('home.cards.totalSize') }}</span>
            </div>
          </div>
          <div class="stat-link">
            <RouterLink to="/apps">
              <el-icon><ArrowUp /></el-icon>
            </RouterLink>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 图表和分析区域 -->
    <div class="charts-grid">
      <!-- 节点健康状态 -->
      <el-card class="chart-card" shadow="hover">
        <template #header>
          <div class="chart-header">
            <div class="chart-title">
              <el-icon><PieChart /></el-icon>
              {{ t('home.charts.nodeHealth') }}
            </div>
            <div class="health-indicator">
              <span class="health-percentage">{{ healthyPercentage }}%</span>
              <span class="health-label">{{ t('home.health.healthy') }}</span>
            </div>
          </div>
        </template>
        <div class="chart-content">
          <div ref="chartNodesEl" class="pie-chart"></div>
          <div class="chart-summary">
            <div class="summary-item">
              <div class="summary-label">{{ t('home.cards.healthy') }}</div>
              <div class="summary-value success">{{ healthyNodes }}</div>
            </div>
            <div class="summary-item">
              <div class="summary-label">{{ t('home.cards.unhealthy') }}</div>
              <div class="summary-value danger">{{ unhealthyNodes }}</div>
            </div>
          </div>
        </div>
        <div class="chart-footer">
          <el-table :data="recentNodes" size="small" max-height="120">
            <el-table-column prop="nodeId" :label="t('home.table.node')" />
            <el-table-column :label="t('home.table.health')" width="100">
              <template #default="{ row }">
                <el-tag :type="row.health==='Healthy'?'success':'danger'" size="small">
                  {{ row.health || '-' }}
                </el-tag>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>

      <!-- 服务端点分布 -->
      <el-card class="chart-card" shadow="hover">
        <template #header>
          <div class="chart-header">
            <div class="chart-title">
              <el-icon><TrendCharts /></el-icon>
              {{ t('home.charts.endpointsTop') }}
            </div>
            <div class="chart-info">
              <el-tag type="info" size="small">{{ services.length }} {{ t('home.cards.services') }}</el-tag>
            </div>
          </div>
        </template>
        <div class="chart-content">
          <div ref="chartServicesEl" class="bar-chart"></div>
        </div>
        <div class="chart-footer">
          <div class="service-tags">
            <el-tag 
              v-for="s in services.slice(0,12)" 
              :key="s" 
              effect="plain" 
              size="small"
              class="service-tag">
              {{ s }}
            </el-tag>
            <el-tag v-if="services.length > 12" type="info" size="small">
              +{{ services.length - 12 }} {{ t('home.more') }}
            </el-tag>
          </div>
        </div>
      </el-card>
    </div>

    <!-- 快速操作区域 -->
    <el-card class="quick-actions-card" shadow="hover">
      <template #header>
        <div class="chart-title">
          <el-icon><MoreFilled /></el-icon>
          {{ t('home.quickActions.title') }}
        </div>
      </template>
      <div class="quick-actions">
        <el-button type="primary" @click="$router.push('/deployments/create')">
          <el-icon><DataBoard /></el-icon>
          {{ t('home.quickActions.createDeployment') }}
        </el-button>
        <el-button type="success" @click="$router.push('/tasks')">
          <el-icon><Cpu /></el-icon>
          {{ t('home.quickActions.runTask') }}
        </el-button>
        <el-button type="warning" @click="$router.push('/resources')">
          <el-icon><Monitor /></el-icon>
          {{ t('home.quickActions.manageResources') }}
        </el-button>
        <el-button type="info" @click="$router.push('/workflows')">
          <el-icon><Files /></el-icon>
          {{ t('home.quickActions.viewWorkflows') }}
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.home-container {
  padding: 0;
}

/* 欢迎卡片样式 */
.welcome-card {
  margin-bottom: 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  color: white;
}

.welcome-card :deep(.el-card__body) {
  padding: 32px;
}

.welcome-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.welcome-title {
  font-size: 32px;
  font-weight: 700;
  margin: 0 0 8px 0;
  color: white;
}

.welcome-subtitle {
  font-size: 16px;
  margin: 0;
  opacity: 0.9;
}

.welcome-actions {
  display: flex;
  gap: 12px;
}

/* 统计卡片网格 */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 20px;
  margin-bottom: 24px;
}

.stat-card {
  transition: all 0.3s ease;
  border: 1px solid #EBEEF5;
  position: relative;
  overflow: hidden;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: var(--gradient);
}

.nodes-card::before { background: linear-gradient(90deg, #409EFF, #67C23A); }
.deployments-card::before { background: linear-gradient(90deg, #E6A23C, #F56C6C); }
.services-card::before { background: linear-gradient(90deg, #67C23A, #85CE61); }
.artifacts-card::before { background: linear-gradient(90deg, #F56C6C, #F78989); }

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 0;
}

.stat-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  color: #606266;
}

.nodes-card .stat-icon { background: linear-gradient(135deg, #409EFF, #67C23A); color: white; }
.deployments-card .stat-icon { background: linear-gradient(135deg, #E6A23C, #F56C6C); color: white; }
.services-card .stat-icon { background: linear-gradient(135deg, #67C23A, #85CE61); color: white; }
.artifacts-card .stat-icon { background: linear-gradient(135deg, #F56C6C, #F78989); color: white; }

.stat-info {
  flex: 1;
}

.stat-title {
  font-size: 14px;
  color: #909399;
  margin-bottom: 4px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #303133;
  margin-bottom: 8px;
}

.stat-detail {
  display: flex;
  gap: 8px;
  align-items: center;
}

.stat-subtitle {
  font-size: 12px;
  color: #909399;
}

.stat-link {
  color: #409EFF;
  font-size: 18px;
}

.stat-link:hover {
  color: #66b1ff;
}

/* 图表网格 */
.charts-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin-bottom: 24px;
}

.chart-card {
  border: 1px solid #EBEEF5;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.chart-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #303133;
}

.health-indicator {
  text-align: right;
}

.health-percentage {
  display: block;
  font-size: 24px;
  font-weight: 700;
  color: #67C23A;
}

.health-label {
  font-size: 12px;
  color: #909399;
}

.chart-content {
  position: relative;
  min-height: 200px;
}

.pie-chart {
  width: 100%;
  height: 200px;
}

.bar-chart {
  width: 100%;
  height: 260px;
}

.chart-summary {
  position: absolute;
  right: 20px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  flex-direction: column;
  gap: 16px;
  z-index: 10;
}

.summary-item {
  text-align: right;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.summary-label {
  font-size: 12px;
  color: #909399;
}

.summary-value {
  font-size: 18px;
  font-weight: 600;
  min-width: 24px;
}

.summary-value.success { color: #67C23A; }
.summary-value.danger { color: #F56C6C; }

.chart-footer {
  margin-top: 16px;
}

.service-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.service-tag {
  transition: all 0.2s ease;
}

.service-tag:hover {
  transform: scale(1.05);
}

/* 快速操作卡片 */
.quick-actions-card {
  border: 1px solid #EBEEF5;
}

.quick-actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .charts-grid {
    grid-template-columns: 1fr;
  }
  
  .chart-summary {
    position: static;
    transform: none;
    flex-direction: row;
    justify-content: center;
    margin-top: 16px;
  }
}

@media (max-width: 768px) {
  .welcome-content {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .quick-actions {
    grid-template-columns: 1fr;
  }
}

/* 动画效果 */
.stat-card {
  animation: fadeInUp 0.6s ease-out;
}

.stat-card:nth-child(1) { animation-delay: 0.1s; }
.stat-card:nth-child(2) { animation-delay: 0.2s; }
.stat-card:nth-child(3) { animation-delay: 0.3s; }
.stat-card:nth-child(4) { animation-delay: 0.4s; }

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.chart-card {
  animation: fadeInUp 0.6s ease-out;
}

.chart-card:nth-child(1) { animation-delay: 0.5s; }
.chart-card:nth-child(2) { animation-delay: 0.6s; }

/* 链接样式 */
.card-link {
  color: var(--el-color-primary);
  text-decoration: none;
  cursor: pointer;
  transition: all 0.2s ease;
}

.card-link:hover {
  text-decoration: underline;
  color: var(--el-color-primary-light-3);
}
</style>