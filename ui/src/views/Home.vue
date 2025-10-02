<script setup lang="ts">
import { ref, onMounted, watch, onBeforeUnmount, nextTick } from 'vue'
import { RouterLink } from 'vue-router'
import * as echarts from 'echarts'
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

const healthyNodes = () => nodes.value.filter(n => (n as any).health === 'Healthy').length
const unhealthyNodes = () => nodes.value.length - healthyNodes()
const runningInstances = () => deployments.value.reduce((s, t) => s + (t.instances||0), 0)

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
    const healthy = healthyNodes()
    const unhealthy = unhealthyNodes()
    chartNodes.setOption({
      tooltip: { trigger: 'item' },
      series: [{ type: 'pie', radius: ['40%','70%'], label: { formatter: '{b}: {c}' }, data: [
        { name: 'Healthy', value: healthy },
        { name: 'Unhealthy', value: unhealthy }
      ]}]
    })
  }
}

watch([services, nodes], ()=>{ nextTick().then(renderCharts) })
</script>

<template>
  <div>
    <h3>Plum 概览</h3>
    <div style="display:grid; grid-template-columns: repeat(4, 1fr); gap:12px; margin-bottom:12px;">
      <el-card>
        <div>
          <RouterLink to="/nodes" class="card-link"><strong>Nodes</strong></RouterLink>
          <div style="font-size:24px;">{{ nodes.length }}</div>
          <small>Healthy {{ healthyNodes() }} / Unhealthy {{ unhealthyNodes() }}</small>
        </div>
      </el-card>
      <el-card>
        <div>
          <RouterLink to="/deployments" class="card-link"><strong>Deployments</strong></RouterLink>
          <div style="font-size:24px;">{{ deployments.length }}</div>
          <small>Instances ~ {{ runningInstances() }}</small>
        </div>
      </el-card>
      <el-card>
        <div>
          <RouterLink to="/services" class="card-link"><strong>Services</strong></RouterLink>
          <div style="font-size:24px;">{{ services.length }}</div>
          <small>Endpoints {{ endpointsCount }}</small>
        </div>
      </el-card>
      <el-card>
        <div>
          <RouterLink to="/apps" class="card-link"><strong>Artifacts</strong></RouterLink>
          <div style="font-size:24px;">≈ {{ (artifactsTotal/1024/1024).toFixed(1) }} MB</div>
        </div>
      </el-card>
    </div>

    <div style="display:grid; grid-template-columns: 1fr 1fr; gap:12px;">
      <el-card>
        <template #header>节点健康</template>
        <div style="display:flex; gap:8px; align-items:center;">
          <div ref="chartNodesEl" style="width: 240px; height: 180px;"></div>
          <div style="display:flex; flex-direction:column; gap:6px;">
            <el-tag type="success">Healthy {{ healthyNodes() }}</el-tag>
            <el-tag type="danger">Unhealthy {{ unhealthyNodes() }}</el-tag>
          </div>
        </div>
        <el-table :data="nodes" size="small" style="margin-top:8px;">
          <el-table-column prop="nodeId" label="Node" width="240" />
          <el-table-column label="Health" width="140">
            <template #default="{ row }"><el-tag :type="row.health==='Healthy'?'success':'danger'">{{ row.health || '-' }}</el-tag></template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card>
        <template #header>各服务可用端点数（Top 12）</template>
        <div ref="chartServicesEl" style="width:100%; height:260px;"></div>
        <div style="display:flex; gap:8px; margin-top:8px; flex-wrap:wrap;">
          <el-tag v-for="s in services.slice(0,12)" :key="s" effect="plain">{{ s }}</el-tag>
        </div>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.card-link {
  color: var(--el-color-primary);
  text-decoration: none;
  cursor: pointer;
}
.card-link:hover {
  text-decoration: underline;
}
</style>