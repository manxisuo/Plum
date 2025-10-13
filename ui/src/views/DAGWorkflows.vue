<template>
  <div class="dag-workflows">
    <el-card class="header-card">
      <div class="header-content">
        <div class="title-section">
          <h2>{{ t('dag.title') }}</h2>
          <p class="subtitle">{{ t('dag.subtitle') }}</p>
        </div>
        <div class="actions">
          <el-button type="primary" @click="showCreateDialog = true">
            {{ t('dag.buttons.create') }}
          </el-button>
        </div>
      </div>
    </el-card>

    <el-card class="content-card">
      <el-table :data="workflows" v-loading="loading" style="width: 100%">
        <el-table-column prop="Name" :label="t('dag.table.name')" width="200" />
        <el-table-column :label="t('dag.table.nodes')" width="100">
          <template #default="{ row }">
            {{ Object.keys(row.Nodes || {}).length }}
          </template>
        </el-table-column>
        <el-table-column :label="t('dag.table.edges')" width="100">
          <template #default="{ row }">
            {{ (row.Edges || []).length }}
          </template>
        </el-table-column>
        <el-table-column :label="t('dag.table.createdAt')" width="180">
          <template #default="{ row }">
            {{ new Date(row.CreatedAt * 1000).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column :label="t('dag.table.actions')" width="300">
          <template #default="{ row }">
            <el-button size="small" @click="viewDAG(row)">{{ t('dag.buttons.view') }}</el-button>
            <el-button size="small" type="success" @click="runDAG(row.WorkflowID)">{{ t('dag.buttons.run') }}</el-button>
            <el-button size="small" @click="viewRuns(row.WorkflowID)">{{ t('dag.buttons.runs') }}</el-button>
            <el-button size="small" type="danger" @click="deleteDAG(row.WorkflowID)">{{ t('dag.buttons.delete') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- DAG详情对话框 -->
    <el-dialog v-model="showDetailDialog" :title="currentDAG?.Name" width="80%">
      <div v-if="currentDAG" class="dag-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="t('dag.detail.workflowId')">{{ currentDAG.WorkflowID }}</el-descriptions-item>
          <el-descriptions-item :label="t('dag.detail.version')">v{{ currentDAG.Version }}</el-descriptions-item>
          <el-descriptions-item :label="t('dag.detail.nodes')">{{ Object.keys(currentDAG.Nodes || {}).length }}</el-descriptions-item>
          <el-descriptions-item :label="t('dag.detail.edges')">{{ (currentDAG.Edges || []).length }}</el-descriptions-item>
        </el-descriptions>

        <h3 style="margin-top: 20px;">{{ t('dag.detail.visualization') }}</h3>
        <div class="dag-graph">
          <div ref="mermaidContainer" class="mermaid-render"></div>
        </div>

        <h3 style="margin-top: 20px;">{{ t('dag.detail.nodes') }}</h3>
        <el-table :data="Object.values(currentDAG.Nodes || {})" size="small">
          <el-table-column prop="NodeID" :label="t('dag.detail.nodeId')" width="150" />
          <el-table-column prop="Name" :label="t('dag.detail.nodeName')" width="150" />
          <el-table-column prop="Type" :label="t('dag.detail.nodeType')" width="100">
            <template #default="{ row }">
              <el-tag :type="getNodeTypeColor(row.Type)" size="small">{{ row.Type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="TriggerRule" :label="t('dag.detail.trigger')" width="120" />
          <el-table-column :label="t('dag.detail.config')" show-overflow-tooltip>
            <template #default="{ row }">
              <span v-if="row.Type === 'task'">
                TaskDef: {{ taskDefs[row.TaskDefID]?.Name || row.TaskDefID }}
                <span v-if="taskDefs[row.TaskDefID]" style="color: #999; font-size: 12px;">
                  ({{ taskDefs[row.TaskDefID].Executor }})
                </span>
              </span>
              <span v-else-if="row.Type === 'branch'">
                Condition: {{ row.Condition?.field }} {{ row.Condition?.operator }} {{ row.Condition?.value }}
              </span>
              <span v-else-if="row.Type === 'parallel'">WaitPolicy: {{ row.WaitPolicy || 'all' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 创建DAG对话框 -->
    <el-dialog v-model="showCreateDialog" :title="t('dag.create.title')" width="70%">
      <el-tabs v-model="createMode" type="border-card">
        <!-- JSON模式 -->
        <el-tab-pane label="JSON编辑" name="json">
          <el-form :model="createForm" label-width="120px">
            <el-form-item :label="t('dag.create.name')">
              <el-input v-model="createForm.name" :placeholder="t('dag.create.namePlaceholder')" />
            </el-form-item>
            <el-form-item :label="t('dag.create.json')">
              <el-input
                v-model="createForm.json"
                type="textarea"
                :rows="15"
                :placeholder="t('dag.create.jsonPlaceholder')"
              />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 可视化模式 -->
        <el-tab-pane label="可视化编辑" name="visual">
          <el-form :model="visualForm" label-width="120px" style="max-height: 500px; overflow-y: auto;">
            <el-form-item label="工作流名称">
              <el-input v-model="visualForm.name" placeholder="请输入工作流名称" />
            </el-form-item>
            
            <el-divider>节点配置</el-divider>
            <div class="visual-editor">
              <el-card v-for="(node, idx) in visualForm.nodes" :key="idx" class="node-card">
                <template #header>
                  <div class="node-header">
                    <span><strong>节点 {{ idx + 1 }}</strong>: {{ node.name || '未命名' }}</span>
                    <el-button type="danger" size="small" @click="removeNode(idx)">删除</el-button>
                  </div>
                </template>
                <el-form label-width="80px" size="small">
                  <el-row :gutter="10">
                    <el-col :span="8">
                      <el-form-item label="节点ID">
                        <el-input v-model="node.nodeId" placeholder="如: task1" />
                      </el-form-item>
                    </el-col>
                    <el-col :span="8">
                      <el-form-item label="名称">
                        <el-input v-model="node.name" placeholder="节点名称" />
                      </el-form-item>
                    </el-col>
                    <el-col :span="8">
                      <el-form-item label="类型">
                        <el-select v-model="node.type" style="width: 100%">
                          <el-option label="任务" value="task" />
                          <el-option label="并行" value="parallel" />
                          <el-option label="分支" value="branch" />
                        </el-select>
                      </el-form-item>
                    </el-col>
                  </el-row>
                  
                  <div v-if="node.type === 'task'">
                    <el-form-item label="任务定义">
                      <el-select v-model="node.taskDefId" style="width: 100%">
                        <el-option 
                          v-for="def in Object.values(taskDefs)" 
                          :key="def.TaskDefID" 
                          :label="`${def.Name} (${def.Executor})`" 
                          :value="def.TaskDefID" 
                        />
                      </el-select>
                    </el-form-item>
                    <el-form-item label="Payload">
                      <el-input v-model="node.payloadJson" type="textarea" :rows="2" placeholder='{"key": "value"}' />
                    </el-form-item>
                  </div>
                  
                  <div v-if="node.type === 'branch'">
                    <el-row :gutter="10">
                      <el-col :span="8">
                        <el-form-item label="字段">
                          <el-input v-model="node.conditionField" placeholder="score" />
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="操作符">
                          <el-select v-model="node.conditionOp" style="width: 100%">
                            <el-option label=">" value=">" />
                            <el-option label=">=" value=">=" />
                            <el-option label="<" value="<" />
                            <el-option label="<=" value="<=" />
                            <el-option label="==" value="==" />
                            <el-option label="!=" value="!=" />
                          </el-select>
                        </el-form-item>
                      </el-col>
                      <el-col :span="8">
                        <el-form-item label="值">
                          <el-input v-model="node.conditionValue" placeholder="60" />
                        </el-form-item>
                      </el-col>
                    </el-row>
                  </div>
                </el-form>
              </el-card>
              <el-button @click="addNode" type="primary" icon="Plus" style="width: 100%; margin-top: 10px;">添加节点</el-button>
            </div>

            <el-divider>连接配置</el-divider>
            <div class="edge-list">
              <el-card v-for="(edge, idx) in visualForm.edges" :key="idx" shadow="hover" style="margin-bottom: 10px;">
                <el-row :gutter="10" align="middle">
                  <el-col :span="10">
                    <el-select v-model="edge.from" placeholder="从" size="small" style="width: 100%">
                      <el-option v-for="n in visualForm.nodes" :key="n.nodeId" :label="n.name || n.nodeId" :value="n.nodeId" />
                    </el-select>
                  </el-col>
                  <el-col :span="2" style="text-align: center; font-size: 18px;">→</el-col>
                  <el-col :span="10">
                    <el-select v-model="edge.to" placeholder="到" size="small" style="width: 100%">
                      <el-option v-for="n in visualForm.nodes" :key="n.nodeId" :label="n.name || n.nodeId" :value="n.nodeId" />
                    </el-select>
                  </el-col>
                  <el-col :span="2">
                    <el-button type="danger" size="small" circle icon="Close" @click="removeEdge(idx)" />
                  </el-col>
                </el-row>
                <div v-if="getNodeById(edge.from)?.type === 'branch'" style="margin-top: 8px;">
                  <el-select v-model="edge.edgeType" placeholder="分支类型" size="small" style="width: 100%">
                    <el-option label="True (条件满足)" value="true" />
                    <el-option label="False (条件不满足)" value="false" />
                  </el-select>
                </div>
              </el-card>
              <el-button @click="addEdge" type="primary" icon="Plus" style="width: 100%">添加连接</el-button>
            </div>

            <el-divider>起始节点</el-divider>
            <el-select v-model="visualForm.startNodes" multiple placeholder="选择起始节点（可多选）" style="width: 100%">
              <el-option v-for="n in visualForm.nodes" :key="n.nodeId" :label="n.name || n.nodeId" :value="n.nodeId" />
            </el-select>
          </el-form>
        </el-tab-pane>
      </el-tabs>

      <template #footer>
        <el-button @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="createDAG">{{ t('common.submit') }}</el-button>
      </template>
    </el-dialog>

    <!-- 运行历史对话框 -->
    <el-dialog v-model="showRunsDialog" :title="t('dag.runs.title')" width="70%">
      <el-table :data="runs" v-loading="loadingRuns">
        <el-table-column prop="RunID" :label="t('dag.runs.runId')" width="250" />
        <el-table-column prop="State" :label="t('dag.runs.state')" width="120">
          <template #default="{ row }">
            <el-tag :type="getStateColor(row.State)">{{ row.State }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('dag.runs.createdAt')" width="180">
          <template #default="{ row }">
            {{ new Date(row.CreatedAt * 1000).toLocaleString() }}
          </template>
        </el-table-column>
        <el-table-column :label="t('dag.runs.duration')" width="120">
          <template #default="{ row }">
            <span v-if="row.FinishedAt">{{ row.FinishedAt - row.StartedAt }}s</span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('dag.runs.actions')" width="100">
          <template #default="{ row }">
            <el-button size="small" @click="viewRunDetail(row)">{{ t('dag.buttons.detail') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 运行详情对话框 -->
    <el-dialog v-model="showRunDetailDialog" :title="t('dag.runDetail.title')" width="80%">
      <div v-if="currentRun">
        <el-descriptions :column="2" border style="margin-bottom: 20px;">
          <el-descriptions-item :label="t('dag.runs.runId')">{{ currentRun.RunID }}</el-descriptions-item>
          <el-descriptions-item :label="t('dag.runs.state')">
            <el-tag :type="getStateColor(currentRun.State)">{{ currentRun.State }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('dag.runs.createdAt')">
            {{ new Date(currentRun.CreatedAt * 1000).toLocaleString() }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('dag.runs.duration')">
            <span v-if="currentRun.FinishedAt">{{ currentRun.FinishedAt - currentRun.StartedAt }}s</span>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>

        <h3>{{ t('dag.runDetail.visualization') }}</h3>
        <div class="dag-graph">
          <div ref="runMermaidContainer" class="mermaid-render"></div>
        </div>

        <h3>{{ t('dag.runDetail.nodeTasks') }}</h3>
        <el-table :data="runTasks" v-loading="loadingRunTasks" size="small">
          <el-table-column prop="NodeID" :label="t('dag.runDetail.nodeId')" width="150" />
          <el-table-column prop="TaskID" :label="t('dag.runDetail.taskId')" width="200" />
          <el-table-column prop="Name" :label="t('dag.runDetail.taskName')" width="150" />
          <el-table-column prop="State" :label="t('dag.runDetail.state')" width="100">
            <template #default="{ row }">
              <el-tag :type="getStateColor(row.State)" size="small">{{ row.State }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="t('dag.runDetail.duration')" width="100">
            <template #default="{ row }">
              <span v-if="row.FinishedAt">{{ row.FinishedAt - row.StartedAt }}s</span>
              <span v-else>-</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'
import mermaid from 'mermaid'

const { t } = useI18n()
const API_BASE = import.meta.env.VITE_API_BASE || ''

// 初始化Mermaid
mermaid.initialize({ 
  startOnLoad: false,
  theme: 'default',
  flowchart: {
    useMaxWidth: true,
    htmlLabels: true,
    curve: 'basis'
  }
})

const workflows = ref<any[]>([])
const mermaidContainer = ref<HTMLElement | null>(null)
const runMermaidContainer = ref<HTMLElement | null>(null)
const loading = ref(false)
const showDetailDialog = ref(false)
const showCreateDialog = ref(false)
const showRunsDialog = ref(false)
const showRunDetailDialog = ref(false)
const currentDAG = ref<any>(null)
const currentRun = ref<any>(null)
const currentRunDAG = ref<any>(null)
const nodeStates = ref<Record<string, string>>({})
const runs = ref<any[]>([])
const runTasks = ref<any[]>([])
const loadingRuns = ref(false)
const loadingRunTasks = ref(false)
const taskDefs = ref<Record<string, any>>({}) // taskDefId -> taskDef映射
const createMode = ref('visual') // 'json' or 'visual'
const createForm = ref({
  name: '',
  json: ''
})
const visualForm = ref({
  name: '',
  nodes: [] as any[],
  edges: [] as any[],
  startNodes: [] as string[]
})

async function loadWorkflows() {
  loading.value = true
  try {
    const [wfRes, defRes] = await Promise.all([
      fetch(`${API_BASE}/v1/dag/workflows`),
      fetch(`${API_BASE}/v1/task-defs`)
    ])
    workflows.value = await wfRes.json()
    
    // 建立taskDefId -> taskDef映射
    const defs = await defRes.json()
    taskDefs.value = {}
    for (const def of defs) {
      taskDefs.value[def.DefID] = def
    }
  } catch (e: any) {
    ElMessage.error(t('dag.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function viewDAG(workflow: any) {
  currentDAG.value = workflow
  showDetailDialog.value = true
  
  // 渲染Mermaid图
  await nextTick()
  if (mermaidContainer.value) {
    const mermaidCode = generateMermaid(workflow)
    mermaidContainer.value.innerHTML = ''
    const { svg } = await mermaid.render('mermaid-graph-' + Date.now(), mermaidCode)
    mermaidContainer.value.innerHTML = svg
  }
}

async function runDAG(workflowId: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/dag/workflows/${workflowId}/run`, { method: 'POST' })
    const data = await res.json()
    ElMessage.success(`${t('dag.messages.runSuccess')}: ${data.runId}`)
  } catch (e: any) {
    ElMessage.error(t('dag.messages.runFailed'))
  }
}

async function deleteDAG(workflowId: string) {
  try {
    await ElMessageBox.confirm(t('dag.messages.deleteConfirm'), t('common.warning'), {
      type: 'warning'
    })
    await fetch(`${API_BASE}/v1/dag/workflows/${workflowId}`, { method: 'DELETE' })
    ElMessage.success(t('dag.messages.deleteSuccess'))
    loadWorkflows()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(t('dag.messages.deleteFailed'))
    }
  }
}

// 可视化编辑器辅助函数
function addNode() {
  visualForm.value.nodes.push({
    nodeId: `node_${visualForm.value.nodes.length + 1}`,
    name: '',
    type: 'task',
    taskDefId: '',
    payloadJson: '',
    conditionField: '',
    conditionOp: '>',
    conditionValue: ''
  })
}

function removeNode(idx: number) {
  const nodeId = visualForm.value.nodes[idx].nodeId
  visualForm.value.nodes.splice(idx, 1)
  // 删除相关边
  visualForm.value.edges = visualForm.value.edges.filter((e: any) => e.from !== nodeId && e.to !== nodeId)
  // 从起始节点中删除
  visualForm.value.startNodes = visualForm.value.startNodes.filter(n => n !== nodeId)
}

function addEdge() {
  visualForm.value.edges.push({
    from: '',
    to: '',
    edgeType: ''
  })
}

function removeEdge(idx: number) {
  visualForm.value.edges.splice(idx, 1)
}

function getNodeById(nodeId: string) {
  return visualForm.value.nodes.find(n => n.nodeId === nodeId)
}

function visualFormToDAG() {
  const nodes: Record<string, any> = {}
  for (const node of visualForm.value.nodes) {
    const n: any = {
      nodeId: node.nodeId,
      type: node.type,
      name: node.name,
      triggerRule: 'all_success',
      timeoutSec: 60
    }
    
    if (node.type === 'task') {
      n.taskDefId = node.taskDefId
      if (node.payloadJson) {
        n.payloadJson = node.payloadJson
      }
    } else if (node.type === 'branch' && node.conditionField) {
      n.condition = {
        field: node.conditionField,
        operator: node.conditionOp,
        value: node.conditionValue
      }
    }
    
    nodes[node.nodeId] = n
  }
  
  const edges = visualForm.value.edges.map((e: any) => ({
    from: e.from,
    to: e.to,
    edgeType: e.edgeType || 'normal'
  }))
  
  return {
    name: visualForm.value.name,
    nodes,
    edges,
    startNodes: visualForm.value.startNodes
  }
}

async function createDAG() {
  try {
    let dagData: any
    
    if (createMode.value === 'json') {
      dagData = JSON.parse(createForm.value.json)
      dagData.name = createForm.value.name || dagData.name
    } else {
      // 可视化模式
      if (!visualForm.value.name) {
        ElMessage.error('请输入工作流名称')
        return
      }
      if (visualForm.value.nodes.length === 0) {
        ElMessage.error('请至少添加一个节点')
        return
      }
      if (visualForm.value.startNodes.length === 0) {
        ElMessage.error('请选择起始节点')
        return
      }
      
      dagData = visualFormToDAG()
    }
    
    const res = await fetch(`${API_BASE}/v1/dag/workflows`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dagData)
    })
    
    if (!res.ok) throw new Error(await res.text())
    
    ElMessage.success(t('dag.messages.createSuccess'))
    showCreateDialog.value = false
    createForm.value = { name: '', json: '' }
    visualForm.value = { name: '', nodes: [], edges: [], startNodes: [] }
    loadWorkflows()
  } catch (e: any) {
    ElMessage.error(`${t('dag.messages.createFailed')}: ${e.message}`)
  }
}

function generateMermaid(dag: any) {
  const lines = ['graph TD']
  
  // 添加节点
  for (const [nodeId, node] of Object.entries(dag.Nodes || {})) {
    const n = node as any
    const shape = n.Type === 'branch' ? '{' : n.Type === 'parallel' ? '[[' : '['
    const endShape = n.Type === 'branch' ? '}' : n.Type === 'parallel' ? ']]' : ']'
    lines.push(`  ${nodeId}${shape}"${n.Name}<br/>${n.Type}"${endShape}`)
  }
  
  // 添加边
  for (const edge of dag.Edges || []) {
    const label = edge.EdgeType && edge.EdgeType !== 'normal' ? `|${edge.EdgeType}|` : ''
    lines.push(`  ${edge.From} --${label}--> ${edge.To}`)
  }
  
  return lines.join('\n')
}

function generateMermaidWithStates(dag: any, states: Record<string, string>) {
  const lines = ['graph TD']
  
  // 添加节点（带状态样式）
  for (const [nodeId, node] of Object.entries(dag.Nodes || {})) {
    const n = node as any
    const state = states[nodeId] || 'Pending'
    const shape = n.Type === 'branch' ? '{' : n.Type === 'parallel' ? '[[' : '['
    const endShape = n.Type === 'branch' ? '}' : n.Type === 'parallel' ? ']]' : ']'
    
    // 节点文本包含状态
    lines.push(`  ${nodeId}${shape}"${n.Name}<br/>${state}"${endShape}`)
    
    // 根据状态添加样式类
    const stateClass = state.toLowerCase()
    lines.push(`  class ${nodeId} ${stateClass}`)
  }
  
  // 添加边
  for (const edge of dag.Edges || []) {
    const label = edge.EdgeType && edge.EdgeType !== 'normal' ? `|${edge.EdgeType}|` : ''
    lines.push(`  ${edge.From} --${label}--> ${edge.To}`)
  }
  
  // 添加样式定义
  lines.push(`  classDef succeeded fill:#67C23A,stroke:#67C23A,color:#fff`)
  lines.push(`  classDef running fill:#409EFF,stroke:#409EFF,color:#fff`)
  lines.push(`  classDef failed fill:#F56C6C,stroke:#F56C6C,color:#fff`)
  lines.push(`  classDef pending fill:#909399,stroke:#909399,color:#fff`)
  lines.push(`  classDef skipped fill:#E6A23C,stroke:#E6A23C,color:#fff`)
  
  return lines.join('\n')
}

function getNodeTypeColor(type: string) {
  const colors: Record<string, string> = {
    task: 'primary',
    branch: 'warning',
    parallel: 'success'
  }
  return colors[type] || 'info'
}

function getStateColor(state: string) {
  const colors: Record<string, string> = {
    Succeeded: 'success',
    Running: 'primary',
    Failed: 'danger',
    Pending: 'info'
  }
  return colors[state] || 'info'
}

async function viewRuns(workflowId: string) {
  loadingRuns.value = true
  showRunsDialog.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/workflow-runs?workflowId=${workflowId}`)
    runs.value = await res.json()
  } catch (e: any) {
    ElMessage.error(t('dag.messages.loadRunsFailed'))
  } finally {
    loadingRuns.value = false
  }
}

async function viewRunDetail(run: any) {
  currentRun.value = run
  showRunDetailDialog.value = true
  loadingRunTasks.value = true
  runTasks.value = []
  nodeStates.value = {}
  
  try {
    // 获取DAG定义和运行状态
    const [dagRes, statusRes, tasksRes] = await Promise.all([
      fetch(`${API_BASE}/v1/dag/workflows/${run.WorkflowID}`),
      fetch(`${API_BASE}/v1/dag/runs/${run.RunID}/status`),
      fetch(`${API_BASE}/v1/tasks`)
    ])
    
    currentRunDAG.value = await dagRes.json()
    const statusData = await statusRes.json()
    nodeStates.value = statusData.nodes || {}
    
    const allTasks = await tasksRes.json()
    
    // 通过Labels中的dagRunId筛选
    const tasks = allTasks.filter((t: any) => {
      return t.Labels && t.Labels.dagRunId === run.RunID
    })
    
    runTasks.value = tasks.map((t: any) => ({
      TaskID: t.TaskID,
      Name: t.Name,
      State: t.State,
      StartedAt: t.StartedAt,
      FinishedAt: t.FinishedAt,
      NodeID: t.Labels?.dagNodeId || '-'
    }))
    
    // 渲染带状态的Mermaid图
    await nextTick()
    if (runMermaidContainer.value && currentRunDAG.value) {
      const mermaidCode = generateMermaidWithStates(currentRunDAG.value, nodeStates.value)
      runMermaidContainer.value.innerHTML = ''
      const { svg } = await mermaid.render('run-mermaid-' + Date.now(), mermaidCode)
      runMermaidContainer.value.innerHTML = svg
    }
  } catch (e: any) {
    ElMessage.error(t('dag.messages.loadRunsFailed'))
  } finally {
    loadingRunTasks.value = false
  }
}

onMounted(() => {
  loadWorkflows()
})
</script>

<style scoped>
.dag-workflows {
  padding: 20px;
}

.header-card {
  margin-bottom: 20px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-section h2 {
  margin: 0 0 5px 0;
  font-size: 24px;
}

.subtitle {
  margin: 0;
  color: #666;
  font-size: 14px;
}

.content-card {
  min-height: 400px;
}

.dag-detail {
  padding: 10px 0;
}

.dag-graph {
  background: #f5f7fa;
  padding: 20px;
  border-radius: 4px;
  margin: 10px 0;
  display: flex;
  justify-content: center;
  overflow-x: auto;
}

.mermaid-render {
  min-width: 100%;
}

.mermaid-render svg {
  max-width: 100%;
  height: auto;
}

.visual-editor {
  max-height: 400px;
  overflow-y: auto;
}

.node-card {
  margin-bottom: 10px;
}

.node-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.edge-list {
  max-height: 300px;
  overflow-y: auto;
}
</style>

