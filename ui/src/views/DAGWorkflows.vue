<template>
  <div class="dag-workflows">
    <!-- 列表视图 -->
    <div v-if="viewMode === 'list'">
      <!-- 操作按钮和统计 -->
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px; gap:24px;">
        <div style="display:flex; gap:8px; flex-shrink:0;">
          <el-button type="primary" :loading="loading" @click="loadWorkflows">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
          <el-button type="success" @click="openCreateView">
            <el-icon><Plus /></el-icon>
            {{ t('dag.buttons.create') }}
          </el-button>
        </div>
      </div>

      <el-card>
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
    </div>

    <!-- 创建视图 -->
    <div v-if="viewMode === 'create'">
      <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
        <h4 style="margin:0;">创建DAG工作流</h4>
        <div style="display:flex; gap:8px;">
          <el-button @click="cancelCreate">取消</el-button>
          <el-button type="primary" @click="createDAG">提交创建</el-button>
        </div>
      </div>

      <el-card>
        <el-tabs v-model="createMode" type="border-card">
          <!-- 拖拽编辑 -->
          <el-tab-pane label="拖拽编辑" name="flow">
            <div style="display: flex; gap: 10px;">
              <div style="flex: 1; height: 500px; border: 1px solid #ddd; position: relative;">
                <VueFlow 
                  v-model:nodes="flowNodes"
                  v-model:edges="flowEdges"
                  @connect="onConnect"
                  @edge-click="onEdgeClick"
                >
                  <Background />
                  <Controls />
                  <template #node-custom="{ data, id }">
                    <!-- Branch节点的特殊Handle配置 -->
                    <template v-if="data.type === 'branch'">
                      <Handle id="top-t" type="target" :position="Top" />
                      <Handle id="left-t" type="target" :position="Left" />
                      <Handle id="true-src" type="source" :position="Right" />
                      <Handle id="false-src" type="source" :position="Bottom" />
                    </template>
                    <!-- 其他节点的标准Handle配置 -->
                    <template v-else>
                      <Handle id="top-s" type="source" :position="Top" />
                      <Handle id="top-t" type="target" :position="Top" />
                      <Handle id="right-s" type="source" :position="Right" />
                      <Handle id="right-t" type="target" :position="Right" />
                      <Handle id="bottom-s" type="source" :position="Bottom" />
                      <Handle id="bottom-t" type="target" :position="Bottom" />
                      <Handle id="left-s" type="source" :position="Left" />
                      <Handle id="left-t" type="target" :position="Left" />
                    </template>
                    <div :class="['custom-node', `node-${data.type}`, { 'node-selected': editingNodeId === id }]" @click="editFlowNodeProps(id, data)">
                      <div class="node-label">{{ data.label }}</div>
                      <div class="node-type">{{ data.type }}</div>
                      <div v-if="data.taskDefId" class="node-task">{{ taskDefs[data.taskDefId]?.Name }}</div>
                      <div v-if="data.type === 'branch'" class="branch-labels">
                        <div class="branch-true">True→</div>
                        <div class="branch-false">False↓</div>
                      </div>
                    </div>
                  </template>
                </VueFlow>
              </div>
              
              <!-- 右侧属性面板 -->
              <div style="width: 300px; border: 1px solid #ddd; padding: 15px; background: #f5f7fa; overflow-y: auto; height: 470px;">
                <h4 style="margin-top: 0;">属性编辑</h4>
                <el-form v-if="editingNode" label-width="70px" size="small">
                  <el-form-item label="节点ID">
                    <el-input v-model="editingNodeId" disabled />
                  </el-form-item>
                  <el-form-item label="名称">
                    <el-input v-model="editingNode.label" />
                  </el-form-item>
                  <el-form-item label="类型">
                    <el-tag>{{ editingNode.type }}</el-tag>
                  </el-form-item>
                  <div v-if="editingNode.type === 'task'">
                    <el-form-item label="任务定义">
                      <el-select v-model="editingNode.taskDefId" style="width: 100%" size="small">
                        <el-option v-for="def in Object.values(taskDefs)" :key="def.DefID" :label="def.Name" :value="def.DefID" />
                      </el-select>
                    </el-form-item>
                    <el-form-item label="Payload">
                      <el-input v-model="editingNode.payloadJson" type="textarea" :rows="4" />
                    </el-form-item>
                  </div>
                  <div v-if="editingNode.type === 'branch'">
                    <el-form-item label="字段">
                      <el-input v-model="editingNode.conditionField" placeholder="score" />
                    </el-form-item>
                    <el-form-item label="操作符">
                      <el-select v-model="editingNode.conditionOp" style="width: 100%">
                        <el-option label=">" value=">" />
                        <el-option label=">=" value=">=" />
                        <el-option label="<" value="<" />
                        <el-option label="<=" value="<=" />
                        <el-option label="==" value="==" />
                        <el-option label="!=" value="!=" />
                      </el-select>
                    </el-form-item>
                    <el-form-item label="值">
                      <el-input v-model="editingNode.conditionValue" placeholder="60" />
                    </el-form-item>
                    <el-divider />
                    <div style="font-size: 12px; color: #666; margin-bottom: 10px;">
                      <div>连线说明：</div>
                      <div>• 从 <strong style="color: #67C23A;">绿色Handle（右侧）</strong> 拖出 → True分支</div>
                      <div>• 从 <strong style="color: #F56C6C;">红色Handle（底部）</strong> 拖出 → False分支</div>
                    </div>
                  </div>
                  <el-button type="primary" size="small" @click="saveNodeEdit" style="width: 100%">保存</el-button>
                </el-form>
                <el-empty v-else description="点击节点编辑属性" :image-size="80" />
              </div>
            </div>
            <div style="margin-top: 10px;">
              <el-input v-model="flowForm.name" placeholder="工作流名称" style="width: 200px; margin-right: 10px;" />
              <el-button @click="addFlowNode('task')" size="small">+ Task</el-button>
              <el-button @click="addFlowNode('parallel')" size="small">+ Parallel</el-button>
              <el-button @click="addFlowNode('branch')" size="small">+ Branch</el-button>
              <el-button @click="showManualConnect = true" size="small" type="success">+ 手动连线</el-button>
              <el-button @click="deleteSelectedNode" size="small" type="warning" :disabled="!editingNodeId">删除节点</el-button>
              <el-button @click="deleteSelectedEdge" size="small" type="warning" :disabled="!selectedEdgeId">删除连线</el-button>
              <el-button @click="clearFlow" size="small" type="danger">清空</el-button>
            </div>
          </el-tab-pane>
          
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
        </el-tabs>
      </el-card>
    </div>

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


    <!-- 手动连线对话框 -->
    <el-dialog v-model="showManualConnect" title="手动添加连线" width="400px">
      <el-form label-width="60px">
        <el-form-item label="从">
          <el-select v-model="manualConnectForm.source" style="width: 100%">
            <el-option v-for="n in flowNodes" :key="n.id" :label="n.data.label" :value="n.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="到">
          <el-select v-model="manualConnectForm.target" style="width: 100%">
            <el-option v-for="n in flowNodes" :key="n.id" :label="n.data.label" :value="n.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showManualConnect = false">取消</el-button>
        <el-button type="primary" @click="addManualConnection">添加</el-button>
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
          <el-table-column prop="NodeID" :label="t('dag.runDetail.nodeId')" width="120" />
          <el-table-column prop="Name" :label="t('dag.runDetail.taskName')" width="120" />
          <el-table-column prop="State" :label="t('dag.runDetail.state')" width="100">
            <template #default="{ row }">
              <el-tag :type="getStateColor(row.State)" size="small">{{ row.State }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="输入" width="200">
            <template #default="{ row }">
              <el-popover trigger="hover" width="400" v-if="row.PayloadJSON">
                <pre style="font-size: 11px; max-height: 200px; overflow: auto;">{{ formatJSON(row.PayloadJSON) }}</pre>
                <template #reference>
                  <el-button size="small" link>查看</el-button>
                </template>
              </el-popover>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column label="输出" width="200">
            <template #default="{ row }">
              <el-popover trigger="hover" width="400" v-if="row.ResultJSON">
                <pre style="font-size: 11px; max-height: 200px; overflow: auto;">{{ formatJSON(row.ResultJSON) }}</pre>
                <template #reference>
                  <el-button size="small" link>查看</el-button>
                </template>
              </el-popover>
              <span v-else>-</span>
            </template>
          </el-table-column>
          <el-table-column :label="t('dag.runDetail.duration')" width="80">
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
import { Refresh, Plus, Files } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import mermaid from 'mermaid'
import { VueFlow, Handle, Position } from '@vue-flow/core'
import { Background } from '@vue-flow/background'
import { Controls } from '@vue-flow/controls'
import '@vue-flow/core/dist/style.css'

const { t } = useI18n()
const API_BASE = import.meta.env.VITE_API_BASE || ''

// 暴露Position给模板
const { Top, Right, Bottom, Left } = Position

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
const viewMode = ref('list') // 'list' or 'create'
const showDetailDialog = ref(false)
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
const flowNodes = ref([])
const flowEdges = ref([])
const flowForm = ref({ name: '' })
let flowNodeCounter = 0
const editingNode = ref<any>(null)
const editingNodeId = ref('')
const selectedEdgeId = ref('')
const showManualConnect = ref(false)
const manualConnectForm = ref({ source: '', target: '' })

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

function openCreateView() {
  createForm.value = { name: '', json: '' }
  visualForm.value = { name: '', nodes: [], edges: [], startNodes: [] }
  flowNodes.value = []
  flowEdges.value = []
  flowForm.value = { name: '' }
  editingNode.value = null
  editingNodeId.value = ''
  nodeUidCounter = 0
  edgeUidCounter = 0
  flowNodeCounter = 0
  createMode.value = 'flow'
  viewMode.value = 'create'
}

function cancelCreate() {
  viewMode.value = 'list'
}

// Vue Flow拖拽编辑
function addFlowNode(type: string) {
  flowNodeCounter++
  const id = `node_${flowNodeCounter}`
  flowNodes.value.push({
    id,
    type: 'custom',
    position: { x: 100 + flowNodeCounter * 80, y: 50 + flowNodeCounter * 100 },
    data: {
      label: `${type}_${flowNodeCounter}`,
      type,
      taskDefId: '',
      payloadJson: '',
      conditionField: '',
      conditionOp: '>',
      conditionValue: ''
    }
  })
}

function editFlowNodeProps(id: string, data: any) {
  editingNodeId.value = id
  editingNode.value = { ...data }
  
  // 取消连线选中
  selectedEdgeId.value = ''
  flowEdges.value = flowEdges.value.map((e: any) => ({ ...e, selected: false }))
}

function saveNodeEdit() {
  const node = flowNodes.value.find((n: any) => n.id === editingNodeId.value)
  if (node && editingNode.value) {
    node.data = { ...editingNode.value }
  }
  ElMessage.success('已保存')
}

function onConnect(params: any) {
  console.log('onConnect =', params)
  
  let src = params.source
  let tgt = params.target
  let srcHandle = params.sourceHandle
  let tgtHandle = params.targetHandle
  
  // Branch节点特殊处理
  const sourceNode = flowNodes.value.find((n: any) => n.id === src)
  if (sourceNode && sourceNode.data.type === 'branch') {
    console.log('Branch node connection:', srcHandle)
    if (srcHandle === 'true-src') {
      console.log('True branch connection')
    } else if (srcHandle === 'false-src') {
      console.log('False branch connection')
    }
  }
  
  // 如果从target型Handle拖出，交换方向
  if (srcHandle && srcHandle.endsWith('-t')) {
    [src, tgt] = [tgt, src]
    ;[srcHandle, tgtHandle] = [tgtHandle, srcHandle]
  }
  
  // 取消连线选中
  selectedEdgeId.value = ''
  
  flowEdges.value.push({
    id: `e${src}-${tgt}-${Date.now()}`,
    source: src,
    target: tgt,
    sourceHandle: srcHandle,
    targetHandle: tgtHandle,
    markerEnd: 'arrowclosed'
  })
}

function addManualConnection() {
  if (!manualConnectForm.value.source || !manualConnectForm.value.target) {
    ElMessage.error('请选择源节点和目标节点')
    return
  }
  flowEdges.value.push({
    id: `e${manualConnectForm.value.source}-${manualConnectForm.value.target}-${Date.now()}`,
    source: manualConnectForm.value.source,
    target: manualConnectForm.value.target,
    markerEnd: 'arrowclosed'
  })
  manualConnectForm.value = { source: '', target: '' }
  showManualConnect.value = false
}

function onEdgeClick(event: any) {
  const edgeId = event.edge.id
  
  // 更新选中状态
  flowEdges.value = flowEdges.value.map((e: any) => ({
    ...e,
    selected: e.id === edgeId
  }))
  
  selectedEdgeId.value = edgeId
  
  // 取消节点选中
  editingNode.value = null
  editingNodeId.value = ''
}

function deleteSelectedNode() {
  if (!editingNodeId.value) return
  
  flowNodes.value = flowNodes.value.filter((n: any) => n.id !== editingNodeId.value)
  flowEdges.value = flowEdges.value.filter((e: any) => 
    e.source !== editingNodeId.value && e.target !== editingNodeId.value
  )
  
  editingNode.value = null
  editingNodeId.value = ''
  ElMessage.success('已删除节点')
}

function deleteSelectedEdge() {
  if (!selectedEdgeId.value) return
  
  flowEdges.value = flowEdges.value.filter((e: any) => e.id !== selectedEdgeId.value)
  selectedEdgeId.value = ''
  ElMessage.success('已删除连线')
}

function clearFlow() {
  flowNodes.value = []
  flowEdges.value = []
  flowNodeCounter = 0
  editingNode.value = null
  editingNodeId.value = ''
  selectedEdgeId.value = ''
}

function flowToDAG() {
  const nodes: Record<string, any> = {}
  const edges: any[] = []
  const startNodeIds: string[] = []
  const hasIncoming = new Set<string>()
  
  // 收集边
  for (const edge of flowEdges.value as any[]) {
    let edgeType = 'normal'
    
    // 如果是Branch节点的输出，根据sourceHandle确定分支类型
    const sourceNode = flowNodes.value.find((n: any) => n.id === edge.source)
    if (sourceNode && sourceNode.data.type === 'branch') {
      if (edge.sourceHandle === 'true-src') {
        edgeType = 'true'
      } else if (edge.sourceHandle === 'false-src') {
        edgeType = 'false'
      }
    }
    
    edges.push({ from: edge.source, to: edge.target, edgeType })
    hasIncoming.add(edge.target)
  }
  
  // 收集节点
  for (const node of flowNodes.value as any[]) {
    const n: any = {
      nodeId: node.id,
      type: node.data.type,
      name: node.data.label,
      triggerRule: 'all_success',
      timeoutSec: 60
    }
    if (node.data.type === 'task') {
      n.taskDefId = node.data.taskDefId || ''
      if (node.data.payloadJson) {
        n.payloadJson = node.data.payloadJson
      }
    } else if (node.data.type === 'branch') {
      if (node.data.conditionField && node.data.conditionOp) {
        n.condition = {
          field: node.data.conditionField,
          operator: node.data.conditionOp,
          value: node.data.conditionValue
        }
      }
    }
    nodes[node.id] = n
    
    if (!hasIncoming.has(node.id)) {
      startNodeIds.push(node.id)
    }
  }
  
  return {
    name: flowForm.value.name,
    nodes,
    edges,
    startNodes: startNodeIds.length > 0 ? startNodeIds : [Object.keys(nodes)[0]]
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
let nodeUidCounter = 0
function addNode() {
  visualForm.value.nodes.push({
    _uid: ++nodeUidCounter, // 唯一ID，用于v-for的key
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

let edgeUidCounter = 0
function addEdge() {
  visualForm.value.edges.push({
    _uid: ++edgeUidCounter, // 唯一ID，用于v-for的key
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
  
  // 过滤掉_uid字段
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
    } else if (createMode.value === 'flow') {
      if (!flowForm.value.name) {
        ElMessage.error('请输入工作流名称')
        return
      }
      if (flowNodes.value.length === 0) {
        ElMessage.error('请至少添加一个节点')
        return
      }
      dagData = flowToDAG()
    } else {
      // 可视化模式验证
      if (!visualForm.value.name) {
        ElMessage.error('请输入工作流名称')
        return
      }
      if (visualForm.value.nodes.length === 0) {
        ElMessage.error('请至少添加一个节点')
        return
      }
      
      // 验证Task节点必须选择TaskDef
      for (const node of visualForm.value.nodes) {
        if (!node.nodeId) {
          ElMessage.error(`节点 "${node.name || '未命名'}" 缺少节点ID`)
          return
        }
        if (!node.name) {
          ElMessage.error(`节点 "${node.nodeId}" 缺少名称`)
          return
        }
        if (node.type === 'task' && !node.taskDefId) {
          ElMessage.error(`Task节点 "${node.name}" 必须选择任务定义`)
          return
        }
        if (node.type === 'branch' && (!node.conditionField || !node.conditionOp)) {
          ElMessage.error(`Branch节点 "${node.name}" 必须配置完整的条件表达式`)
          return
        }
      }
      
      // 验证边的完整性
      for (const edge of visualForm.value.edges) {
        if (!edge.from || !edge.to) {
          ElMessage.error('存在未完成的连接配置')
          return
        }
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
    viewMode.value = 'list'
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
    let label = ''
    if (edge.EdgeType === 'true') {
      label = '|True|'
    } else if (edge.EdgeType === 'false') {
      label = '|False|'
    } else if (edge.EdgeType && edge.EdgeType !== 'normal') {
      label = `|${edge.EdgeType}|`
    }
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
    let label = ''
    if (edge.EdgeType === 'true') {
      label = '|True|'
    } else if (edge.EdgeType === 'false') {
      label = '|False|'
    } else if (edge.EdgeType && edge.EdgeType !== 'normal') {
      label = `|${edge.EdgeType}|`
    }
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

function formatJSON(jsonStr: string) {
  try {
    return JSON.stringify(JSON.parse(jsonStr), null, 2)
  } catch {
    return jsonStr
  }
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
      NodeID: t.Labels?.dagNodeId || '-',
      PayloadJSON: t.PayloadJSON,
      ResultJSON: t.ResultJSON
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

.custom-node {
  padding: 8px 15px;
  border-radius: 6px;
  background: white;
  border: 2px solid #409EFF;
  min-width: 80px;
  max-width: 150px;
  text-align: center;
  cursor: pointer;
  position: relative;
}

.node-task { border-color: #409EFF; }
.node-parallel { border-color: #67C23A; }
.node-branch { border-color: #E6A23C; }

.node-selected {
  border-width: 3px !important;
  box-shadow: 0 0 12px rgba(64, 158, 255, 0.5);
  transform: scale(1.05);
}

.node-label {
  font-weight: bold;
  margin-bottom: 4px;
}

.node-type {
  font-size: 12px;
  color: #999;
}

.node-task {
  font-size: 11px;
  color: #67C23A;
  margin-top: 2px;
}

.branch-labels {
  display: flex;
  justify-content: space-between;
  margin-top: 4px;
  font-size: 10px;
}

.branch-true {
  color: #67C23A;
  font-weight: bold;
}

.branch-false {
  color: #F56C6C;
  font-weight: bold;
}

/* Vue Flow样式 */
:deep(.vue-flow__edge-path) {
  stroke-width: 2px;
}

:deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke-width: 4px;
  stroke: #409EFF;
  animation: dash 1s linear infinite;
}

@keyframes dash {
  to {
    stroke-dashoffset: -20;
  }
}

:deep(.vue-flow__handle) {
  width: 14px;
  height: 14px;
  background: #555;
  border: 2px solid white;
  box-shadow: 0 2px 4px rgba(0,0,0,0.3);
}

:deep(.vue-flow__handle:hover) {
  background: #409EFF;
  transform: scale(1.2);
}

/* Branch节点Handle特殊样式 */
/* :deep(.vue-flow__handle[id="true-src"]) {
  background: #67C23A !important;
  border: 2px solid white !important;
  width: 16px !important;
  height: 16px !important;
  z-index: 100 !important;
}

:deep(.vue-flow__handle[id="false-src"]) {
  background: #F56C6C !important;
  border: 2px solid white !important;
  width: 16px !important;
  height: 16px !important;
  z-index: 100 !important;
}

:deep(.vue-flow__handle[id="true-src"]:hover),
:deep(.vue-flow__handle[id="false-src"]:hover) {
  transform: scale(1.3) !important;
  box-shadow: 0 0 8px rgba(0,0,0,0.5) !important;
} */

/* Controls按钮样式 */
:deep(.vue-flow__controls-button) {
  width: 28px;
  height: 28px;
  background: white;
  border: 1px solid #ccc;
}

:deep(.vue-flow__controls-zoomin)::before {
  /* content: '+'; */
  font-size: 18px;
  font-weight: bold;
}

:deep(.vue-flow__controls-zoomout)::before {
  /* content: '−'; */
  font-size: 18px;
  font-weight: bold;
}

:deep(.vue-flow__controls-fitview)::before {
  /* content: '⊡'; */
  font-size: 16px;
}
</style>

