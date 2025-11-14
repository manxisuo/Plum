<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

type Artifact = { artifactId: string; name: string; version: string; url: string; type?: string }
type NodeDTO = { nodeId: string; ip: string }

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const activeTab = ref<'artifact' | 'node'>('artifact')
const form = ref<{ name: string }>({ name: '' })
const artifacts = ref<Artifact[]>([])
const nodes = ref<NodeDTO[]>([])
const entriesRows = ref<Array<{ artifactId: string | null; startCmd: string; replicas: Array<{ nodeId: string | null; count: number }> }>>([])
const nodeEntriesRows = ref<Array<{ nodeId: string | null; apps: Array<{ artifactId: string | null; startCmd: string; count: number }> }>>([])
const labelRows = ref<Array<{ key: string; value: string }>>([])

async function loadRefs() {
  try {
    const [aRes, nRes] = await Promise.all([
      fetch(`${API_BASE}/v1/apps`),
      fetch(`${API_BASE}/v1/nodes`)
    ])
    if (aRes.ok) artifacts.value = await aRes.json() as Artifact[]
    if (nRes.ok) nodes.value = await nRes.json() as NodeDTO[]
  } catch {}
}

function addEntry() {
  entriesRows.value.push({ artifactId: null, startCmd: '', replicas: [{ nodeId: nodes.value[0]?.nodeId || null, count: 1 }] })
}
function delEntry(i: number) { entriesRows.value.splice(i, 1) }
function addReplicaRow(i: number) { entriesRows.value[i].replicas.push({ nodeId: null, count: 1 }) }
function delReplicaRow(i: number, j: number) { entriesRows.value[i].replicas.splice(j, 1) }

function addNodeEntry() {
  nodeEntriesRows.value.push({
    nodeId: nodes.value[0]?.nodeId || null,
    apps: [{ artifactId: null, startCmd: '', count: 1 }]
  })
}
function delNodeEntry(i: number) { nodeEntriesRows.value.splice(i, 1) }
function addNodeApp(i: number) { nodeEntriesRows.value[i].apps.push({ artifactId: null, startCmd: '', count: 1 }) }
function delNodeApp(i: number, j: number) { nodeEntriesRows.value[i].apps.splice(j, 1) }

function addLabelRow() { labelRows.value.push({ key:'', value:'' }) }
function delLabelRow(i: number) { labelRows.value.splice(i, 1) }

async function doCreate() {
  try {
    if (!form.value.name) throw new Error(t('deployments.create.validation.nameRequired'))
    const entries: Array<{ artifactUrl: string; startCmd: string; replicas: Record<string, number> }> = []

    if (activeTab.value === 'artifact') {
      if (!entriesRows.value.length) throw new Error(t('deployments.create.validation.entriesRequired'))
      for (const e of entriesRows.value) {
        if (!e.artifactId) throw new Error(t('deployments.create.validation.artifactRequired'))
        const art = artifacts.value.find(a => a.artifactId === e.artifactId)
        if (!art) throw new Error(t('deployments.create.validation.artifactNotFound'))
        const replicas: Record<string, number> = {}
        for (const r of e.replicas) {
          if (r.nodeId && r.count > 0) {
            replicas[r.nodeId] = (replicas[r.nodeId] || 0) + r.count
          }
        }
        if (!Object.keys(replicas).length) throw new Error(t('deployments.create.validation.replicasRequired'))
        // 对于镜像应用，使用 image://{artifactId} 作为标识符
        let artifactUrl = art.url
        if (!artifactUrl && art.type === 'image') {
          artifactUrl = `image://${art.artifactId}`
        }
        entries.push({ artifactUrl: artifactUrl, startCmd: e.startCmd, replicas })
      }
    } else {
      if (!nodeEntriesRows.value.length) throw new Error(t('deployments.create.validation.nodeEntriesRequired'))
      const agg = new Map<string, { artifact: Artifact; startCmd: string; replicas: Record<string, number> }>()
      for (const entry of nodeEntriesRows.value) {
        if (!entry.nodeId) throw new Error(t('deployments.create.validation.nodeRequired'))
        for (const app of entry.apps) {
          if (!app.artifactId) throw new Error(t('deployments.create.validation.artifactRequired'))
          if (app.count <= 0) continue
          const art = artifacts.value.find(a => a.artifactId === app.artifactId)
          if (!art) throw new Error(t('deployments.create.validation.artifactNotFound'))
          const key = `${app.artifactId}::${app.startCmd || ''}`
          if (!agg.has(key)) {
            agg.set(key, { artifact: art, startCmd: app.startCmd, replicas: {} })
          }
          const item = agg.get(key)!
          item.replicas[entry.nodeId!] = (item.replicas[entry.nodeId!] || 0) + app.count
        }
      }
      if (!agg.size) throw new Error(t('deployments.create.validation.replicasRequired'))
      agg.forEach((value) => {
        // 对于镜像应用，使用 image://{artifactId} 作为标识符
        let artifactUrl = value.artifact.url
        if (!artifactUrl && value.artifact.type === 'image') {
          artifactUrl = `image://${value.artifact.artifactId}`
        }
        entries.push({
          artifactUrl: artifactUrl,
          startCmd: value.startCmd,
          replicas: value.replicas
        })
      })
    }

    const labels: Record<string,string> = {}
    for (const kv of labelRows.value) { if (kv.key) labels[kv.key] = kv.value }
    const body = { name: form.value.name, entries, labels: Object.keys(labels).length ? labels : undefined }
    loading.value = true
    const res = await fetch(`${API_BASE}/v1/deployments`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    ElMessage.success(t('deployments.create.messages.created'))
    router.push('/deployments')
  } catch (e:any) {
    ElMessage.error(e?.message || t('deployments.create.messages.createFailed'))
  } finally { loading.value = false }
}

onMounted(() => {
  loadRefs()
  if (!entriesRows.value.length) addEntry()
  if (!nodeEntriesRows.value.length) addNodeEntry()
})
</script>

<style scoped>
.entries-tabs {
  width: 100%;
}

.entries-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.entry-card {
  width: 100%;
}

.entries-actions {
  display: flex;
  justify-content: flex-start;
}

.replica-row,
.node-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}

.node-apps {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.add-app-btn {
  align-self: flex-start;
}
</style>

<template>
  <div>
    <el-card class="box-card">
      <template #header>
        <div style="display:flex; justify-content:space-between; align-items:center;">
          <span>{{ t('deployments.create.title') }}</span>
        </div>
      </template>
      
      <el-form label-width="120px" :disabled="loading">
        <el-form-item :label="t('deployments.create.form.name')"><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="activeTab === 'artifact' ? t('deployments.create.form.entriesByApp') : t('deployments.create.form.entriesByNode')">
          <el-tabs v-model="activeTab" type="border-card" class="entries-tabs">
            <el-tab-pane :label="t('deployments.create.tabs.byApp')" name="artifact">
              <div class="entries-panel">
                <el-card v-for="(e,i) in entriesRows" :key="i" class="entry-card">
                  <div style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
                    <el-select v-model="e.artifactId" :placeholder="t('deployments.create.form.selectArtifact')" filterable style="flex: 0 0 260px; min-width: 260px;">
                      <el-option v-for="a in artifacts" :key="a.artifactId" :label="`${a.name}@${a.version}`" :value="a.artifactId" />
                    </el-select>
                    <el-input v-model="e.startCmd" :placeholder="t('deployments.create.form.startCmdPlaceholder')" />
                    <el-button size="small" type="danger" @click="delEntry(i)">{{ t('deployments.create.buttons.deleteEntry') }}</el-button>
                  </div>
                  <div>
                    <div v-for="(r,j) in e.replicas" :key="j" class="replica-row">
                      <el-select v-model="r.nodeId" :placeholder="t('deployments.create.form.selectNode')" style="flex: 0 0 260px; min-width: 260px;">
                        <el-option v-for="n in nodes" :key="n.nodeId" :label="`${n.nodeId} (${n.ip})`" :value="n.nodeId" />
                      </el-select>
                      <el-input-number v-model="r.count" :min="0" :max="100" />
                      <el-button size="small" @click="delReplicaRow(i,j)">{{ t('common.delete') }}</el-button>
                    </div>
                  </div>
                </el-card>
                <div class="entries-actions">
                  <el-button size="small" type="primary" @click="addEntry">{{ t('deployments.create.buttons.addEntry') }}</el-button>
                </div>
              </div>
            </el-tab-pane>
            <el-tab-pane :label="t('deployments.create.tabs.byNode')" name="node">
              <div class="entries-panel">
                <el-card v-for="(entry,i) in nodeEntriesRows" :key="i" class="entry-card">
                  <div class="node-row">
                    <el-select v-model="entry.nodeId" :placeholder="t('deployments.create.form.selectNode')" filterable style="flex: 0 0 260px; min-width: 260px;">
                      <el-option v-for="n in nodes" :key="n.nodeId" :label="`${n.nodeId} (${n.ip})`" :value="n.nodeId" />
                    </el-select>
                    <el-button size="small" type="danger" @click="delNodeEntry(i)">{{ t('deployments.create.buttons.deleteNodeEntry') }}</el-button>
                  </div>
                  <div class="node-apps">
                    <div v-for="(app,j) in entry.apps" :key="j" class="replica-row">
                      <el-select v-model="app.artifactId" :placeholder="t('deployments.create.form.selectArtifact')" filterable style="flex: 0 0 260px; min-width: 260px;">
                        <el-option v-for="a in artifacts" :key="a.artifactId" :label="`${a.name}@${a.version}`" :value="a.artifactId" />
                      </el-select>
                      <el-input v-model="app.startCmd" :placeholder="t('deployments.create.form.startCmdPlaceholder')" />
                      <el-input-number v-model="app.count" :min="0" :max="100" />
                      <el-button size="small" @click="delNodeApp(i,j)">{{ t('common.delete') }}</el-button>
                    </div>
                    <el-button size="small" type="primary" class="add-app-btn" @click="addNodeApp(i)">{{ t('deployments.create.buttons.addApp') }}</el-button>
                  </div>
                </el-card>
                <div class="entries-actions">
                  <el-button size="small" type="primary" @click="addNodeEntry">{{ t('deployments.create.buttons.addNodeEntry') }}</el-button>
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-form-item>
        <el-form-item :label="t('deployments.create.form.labels')">
          <div style="width:100%">
            <div v-for="(kv,i) in labelRows" :key="i" style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
              <el-input v-model="kv.key" :placeholder="t('deployments.create.form.keyPlaceholder')" style="flex:1" />
              <el-input v-model="kv.value" :placeholder="t('deployments.create.form.valuePlaceholder')" style="flex:1" />
              <el-button size="small" @click="delLabelRow(i)">{{ t('common.delete') }}</el-button>
            </div>
            <el-button size="small" type="primary" @click="addLabelRow">{{ t('deployments.create.buttons.addLabel') }}</el-button>
          </div>
        </el-form-item>
        <el-form-item>
          <el-button @click="$router.back()">{{ t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="loading" @click="doCreate">{{ t('deployments.create.buttons.create') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>


