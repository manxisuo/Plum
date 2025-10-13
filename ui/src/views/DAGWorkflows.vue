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
        <el-table-column :label="t('dag.table.actions')" width="250">
          <template #default="{ row }">
            <el-button size="small" @click="viewDAG(row)">{{ t('dag.buttons.view') }}</el-button>
            <el-button size="small" type="success" @click="runDAG(row.WorkflowID)">{{ t('dag.buttons.run') }}</el-button>
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
          <pre class="mermaid-code">{{ generateMermaid(currentDAG) }}</pre>
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
              <span v-if="row.Type === 'task'">TaskDef: {{ row.TaskDefID }}</span>
              <span v-else-if="row.Type === 'branch'">Condition: {{ row.Condition?.field }} {{ row.Condition?.operator }} {{ row.Condition?.value }}</span>
              <span v-else-if="row.Type === 'parallel'">WaitPolicy: {{ row.WaitPolicy || 'all' }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-dialog>

    <!-- 创建DAG对话框 -->
    <el-dialog v-model="showCreateDialog" :title="t('dag.create.title')" width="60%">
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
      <template #footer>
        <el-button @click="showCreateDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="createDAG">{{ t('common.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const API_BASE = import.meta.env.VITE_API_BASE || ''

const workflows = ref<any[]>([])
const loading = ref(false)
const showDetailDialog = ref(false)
const showCreateDialog = ref(false)
const currentDAG = ref<any>(null)
const createForm = ref({
  name: '',
  json: ''
})

async function loadWorkflows() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/dag/workflows`)
    workflows.value = await res.json()
  } catch (e: any) {
    ElMessage.error(t('dag.messages.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function viewDAG(workflow: any) {
  currentDAG.value = workflow
  showDetailDialog.value = true
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

async function createDAG() {
  try {
    const dagData = JSON.parse(createForm.value.json)
    dagData.name = createForm.value.name || dagData.name
    
    const res = await fetch(`${API_BASE}/v1/dag/workflows`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dagData)
    })
    
    if (!res.ok) throw new Error(await res.text())
    
    ElMessage.success(t('dag.messages.createSuccess'))
    showCreateDialog.value = false
    createForm.value = { name: '', json: '' }
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

function getNodeTypeColor(type: string) {
  const colors: Record<string, string> = {
    task: 'primary',
    branch: 'warning',
    parallel: 'success'
  }
  return colors[type] || 'info'
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
}

.mermaid-code {
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.6;
  margin: 0;
  white-space: pre-wrap;
}
</style>

