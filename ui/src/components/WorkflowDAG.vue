<template>
  <div class="workflow-dag">
    <div ref="chartContainer" style="width: 100%; height: 400px; border: 1px solid #eee; border-radius: 4px;"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'
import dagre from 'dagre'

interface StepRun {
  stepId: string
  name: string
  state: string
  ord: number
  startedAt?: number
  finishedAt?: number
}

interface Props {
  steps: StepRun[]
  workflowState?: string
}

const props = defineProps<Props>()
const chartContainer = ref<HTMLElement>()
let chart: echarts.ECharts | null = null

// 获取节点状态对应的颜色
function getNodeColor(state: string): string {
  switch (state?.toLowerCase()) {
    case 'succeeded':
    case 'completed':
      return '#67c23a' // 绿色 - 成功
    case 'failed':
    case 'error':
      return '#f56c6c' // 红色 - 失败
    case 'running':
      return '#e6a23c' // 橙色 - 运行中
    case 'pending':
      return '#409EFF' // 蓝色 - 等待中/未开始
    case 'canceled':
    case 'cancelled':
      return '#909399' // 灰色 - 取消
    default:
      return '#c0c4cc' // 默认灰色
  }
}

// 生成DAG数据
function generateDAGData() {
  if (!props.steps || props.steps.length === 0) {
    return { nodes: [], links: [] }
  }

  // 按ord排序步骤
  const sortedSteps = [...props.steps].sort((a, b) => a.ord - b.ord)
  
  
  const nodes = sortedSteps.map(step => ({
    id: step.stepId,
    name: step.name || step.stepId,
    state: step.state,
    ord: step.ord,
    category: 0
  }))

  // 创建边（目前是顺序执行，将来可以支持复杂的依赖关系）
  const links = []
  for (let i = 0; i < sortedSteps.length - 1; i++) {
    links.push({
      source: sortedSteps[i].stepId,
      target: sortedSteps[i + 1].stepId
    })
  }

  return { nodes, links }
}

// 使用dagre布局
function layoutDAG(nodes: any[], links: any[]) {
  const g = new dagre.graphlib.Graph()
  g.setGraph({
    rankdir: 'LR', // 从左到右
    nodesep: 60,
    ranksep: 120,
    marginx: 20,
    marginy: 20
  })
  g.setDefaultEdgeLabel(() => ({}))
  
  // 添加节点
  nodes.forEach(node => {
    g.setNode(node.id, {
      width: Math.max(node.name.length * 10 + 40, 140),
      height: 50
    })
  })
  
  // 添加边
  links.forEach(link => {
    g.setEdge(link.source, link.target)
  })
  
  dagre.layout(g)
  
  return g
}

// 渲染图表
function renderChart() {
  if (!chartContainer.value) return
  
  const { nodes, links } = generateDAGData()
  
  if (nodes.length === 0) {
    if (chart) {
      chart.dispose()
      chart = null
    }
    return
  }

  // 使用dagre布局
  const g = layoutDAG(nodes, links)
  
  // 转换为echarts格式
  const echartsNodes = nodes.map(node => {
    const dagreNode = g.node(node.id)
    return {
      id: node.id,
      name: node.name,
      x: dagreNode.x,
      y: dagreNode.y,
      symbolSize: [dagreNode.width, dagreNode.height],
      itemStyle: {
        color: getNodeColor(node.state),
        borderColor: '#333',
        borderWidth: 2
      },
      label: {
        show: true,
        position: 'inside',
        formatter: (params: any) => {
          // 如果任务很快完成（StartedAt为0或很小），显示特殊标识
          const step = props.steps.find(s => s.stepId === params.data.id)
          if (step && step.state === 'Succeeded' && (!step.startedAt || step.startedAt === 0)) {
            return `${params.data.name} ✓`
          }
          return params.data.name
        },
        fontSize: 12,
        color: '#fff',
        fontWeight: 'bold'
      }
    }
  })

  const echartsLinks = links.map(link => ({
    source: link.source,
    target: link.target
  }))

  const option = {
    tooltip: {
      trigger: 'item',
      formatter: (params: any) => {
        if (params.dataType === 'node') {
          const step = props.steps.find(s => s.stepId === params.data.id)
          return `
            <div>
              <strong>${params.data.name}</strong><br/>
              状态: ${step?.state || 'Unknown'}<br/>
              步骤ID: ${params.data.id}
              ${step?.startedAt ? `<br/>开始时间: ${new Date(step.startedAt * 1000).toLocaleString()}` : ''}
              ${step?.finishedAt ? `<br/>结束时间: ${new Date(step.finishedAt * 1000).toLocaleString()}` : ''}
            </div>
          `
        }
        return ''
      }
    },
    animation: false,
    series: [{
      type: 'graph',
      layout: 'none', // 使用预计算的位置
      coordinateSystem: null,
      data: echartsNodes,
      links: echartsLinks,
      roam: true,
      zoom: 1,
      focusNodeAdjacency: true,
      lineStyle: {
        color: '#333',
        width: 3,
        curveness: 0.2
      },
      edgeSymbol: ['none', 'arrow'],
      edgeSymbolSize: [0, 8],
      emphasis: {
        focus: 'adjacency',
        lineStyle: {
          width: 4,
          color: '#409EFF'
        }
      }
    }]
  }

  if (!chart) {
    chart = echarts.init(chartContainer.value)
  }
  
  chart.setOption(option, true)
}

// 监听props变化
watch(() => [props.steps, props.workflowState], () => {
  nextTick(() => {
    renderChart()
  })
}, { deep: true })

onMounted(() => {
  nextTick(() => {
    renderChart()
  })
})
</script>

<style scoped>
.workflow-dag {
  margin: 16px 0;
}

.workflow-dag .echarts-container {
  width: 100%;
  height: 400px;
}
</style>
