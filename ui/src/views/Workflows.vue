<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()

type WorkflowStep = { stepId?: string; name: string; executor: string; timeoutSec: number; maxRetries: number }
type Workflow = { workflowId: string; name: string; labels?: Record<string,string>; steps: WorkflowStep[] }

const items = ref<Workflow[]>([])
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    const res = await fetch(`${API_BASE}/v1/workflows`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    items.value = await res.json() as Workflow[]
  } catch (e:any) { ElMessage.error(e?.message || '加载失败') }
  finally { loading.value = false }
}

async function run(id: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/workflows/${encodeURIComponent(id)}?action=run`, { method: 'POST' })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const j = await res.json()
    ElMessage.success('已启动运行')
    router.push(`/workflow-runs/${j.runId}`)
  } catch (e:any) { ElMessage.error(e?.message || '操作失败') }
}

async function viewLatest(workflowId: string) {
  try {
    const res = await fetch(`${API_BASE}/v1/workflow-runs`)
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const runs = await res.json() as any[]
    const list = (runs||[]).filter(r => (r.workflowId||r.WorkflowID) === workflowId)
    if (list.length === 0) { ElMessage.info('暂无运行记录'); return }
    list.sort((a,b)=> (b.createdAt||0)-(a.createdAt||0))
    const rid = list[0].runId || list[0].RunID
    router.push(`/workflow-runs/${rid}`)
  } catch (e:any) { ElMessage.error(e?.message || '查询失败') }
}

// create dialog
const showCreate = ref(false)
const form = reactive<{ name: string; steps: WorkflowStep[] }>({ name: '', steps: [{ name:'builtin.echo', executor:'embedded', timeoutSec: 300, maxRetries: 0 }] })

function openCreate() {
  form.name = ''
  form.steps = [{ name:'builtin.echo', executor:'embedded', timeoutSec: 300, maxRetries: 0 }]
  showCreate.value = true
}

function addStep() { form.steps.push({ name:'builtin.echo', executor:'embedded', timeoutSec: 300, maxRetries: 0 }) }
function removeStep(i:number) { form.steps.splice(i,1) }

async function submit() {
  try {
    const res = await fetch(`${API_BASE}/v1/workflows`, { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({ name: form.name, steps: form.steps }) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success('已创建')
    showCreate.value = false
    form.name = ''
    form.steps = [{ name:'builtin.echo', executor:'embedded', timeoutSec: 300, maxRetries: 0 }]
    load()
  } catch (e:any) { ElMessage.error(e?.message || '创建失败') }
}

onMounted(load)
const { t } = useI18n()
</script>

<template>
  <div>
    <div style="display:flex; gap:8px; align-items:center;">
      <el-button type="primary" :loading="loading" @click="load">{{ t('workflows.buttons.refresh') }}</el-button>
      <el-button type="success" @click="openCreate">{{ t('workflows.buttons.create') }}</el-button>
    </div>

    <el-table :data="items" v-loading="loading" style="width:100%; margin-top:12px;">
      <el-table-column :label="t('workflows.columns.workflowId')" width="320">
        <template #default="{ row }">{{ (row as any).workflowId || (row as any).WorkflowID }}</template>
      </el-table-column>
      <el-table-column :label="t('workflows.columns.name')" width="200">
        <template #default="{ row }">{{ (row as any).name || (row as any).Name }}</template>
      </el-table-column>
      <el-table-column :label="t('workflows.columns.steps')">
        <template #default="{ row }">
          <code>
            {{ (()=>{ const a = (row as any).steps || (row as any).Steps || []; return Array.isArray(a) ? a.map((s:any)=> s?.name || s?.Name || s?.definitionId || s?.DefinitionID || '').join(' -> ') : '' })() }}
          </code>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.action')" width="260">
        <template #default="{ row }">
          <el-button size="small" type="primary" @click="run(((row as any).workflowId||(row as any).WorkflowID))">{{ t('workflows.buttons.run') }}</el-button>
          <el-button size="small" @click="viewLatest(((row as any).workflowId||(row as any).WorkflowID))">{{ t('workflows.buttons.viewLatest') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="showCreate" :title="t('workflows.dialog.title')" width="700px">
      <el-form label-width="120px">
        <el-form-item :label="t('workflows.dialog.form.name')"><el-input v-model="form.name" placeholder="workflow 名称" /></el-form-item>
        <el-form-item :label="t('workflows.dialog.form.steps')">
          <div style="display:flex; flex-direction:column; gap:8px; width:100%">
            <div v-for="(s, i) in form.steps" :key="i" style="display:flex; gap:8px; align-items:center;">
              <el-input v-model="s.name" placeholder="taskName，如 builtin.echo" style="flex:2" />
              <el-select v-model="s.executor" style="flex:1">
                <el-option label="embedded" value="embedded" />
                <el-option label="service" value="service" />
                <el-option label="os_process" value="os_process" />
              </el-select>
              <el-input v-model.number="s.timeoutSec" :placeholder="t('workflows.dialog.form.timeoutSec')" style="width:120px" />
              <el-input v-model.number="s.maxRetries" :placeholder="t('workflows.dialog.form.maxRetries')" style="width:120px" />
              <el-button size="small" type="danger" @click="removeStep(i)">{{ t('workflows.dialog.form.delete') }}</el-button>
            </div>
            <el-button size="small" @click="addStep">{{ t('workflows.dialog.form.addStep') }}</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate=false">{{ t('workflows.dialog.footer.cancel') }}</el-button>
        <el-button type="primary" @click="submit">{{ t('workflows.dialog.footer.submit') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>
