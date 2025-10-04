<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'

type Artifact = { artifactId: string; name: string; version: string; url: string }
type NodeDTO = { nodeId: string; ip: string }

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const router = useRouter()
const { t } = useI18n()

const loading = ref(false)
const form = ref<{ name: string }>({ name: '' })
const artifacts = ref<Artifact[]>([])
const nodes = ref<NodeDTO[]>([])
const entriesRows = ref<Array<{ artifactId: string | null; startCmd: string; replicas: Array<{ nodeId: string | null; count: number }> }>>([])
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
function addLabelRow() { labelRows.value.push({ key:'', value:'' }) }
function delLabelRow(i: number) { labelRows.value.splice(i, 1) }

async function doCreate() {
  try {
    if (!form.value.name) throw new Error(t('deployments.create.validation.nameRequired'))
    if (!entriesRows.value.length) throw new Error(t('deployments.create.validation.entriesRequired'))
    const entries: Array<{ artifactUrl: string; startCmd: string; replicas: Record<string, number> }> = []
    for (const e of entriesRows.value) {
      if (!e.artifactId) throw new Error(t('deployments.create.validation.artifactRequired'))
      const art = artifacts.value.find(a => a.artifactId === e.artifactId)
      if (!art) throw new Error(t('deployments.create.validation.artifactNotFound'))
      const replicas: Record<string, number> = {}
      for (const r of e.replicas) { if (r.nodeId && r.count > 0) replicas[r.nodeId] = r.count }
      if (!Object.keys(replicas).length) throw new Error(t('deployments.create.validation.replicasRequired'))
      entries.push({ artifactUrl: art.url, startCmd: e.startCmd, replicas })
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

onMounted(() => { loadRefs(); if (!entriesRows.value.length) addEntry() })
</script>

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
        <el-form-item :label="t('deployments.create.form.entries')">
          <div style="width:100%">
            <el-card v-for="(e,i) in entriesRows" :key="i" style="margin-bottom:8px;">
              <div style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
                <el-select v-model="e.artifactId" :placeholder="t('deployments.create.form.selectArtifact')" filterable style="flex: 0 0 260px; min-width: 260px;">
                  <el-option v-for="a in artifacts" :key="a.artifactId" :label="`${a.name}@${a.version}`" :value="a.artifactId" />
                </el-select>
                <el-input v-model="e.startCmd" :placeholder="t('deployments.create.form.startCmdPlaceholder')" />
                <el-button size="small" type="danger" @click="delEntry(i)">{{ t('deployments.create.buttons.deleteEntry') }}</el-button>
              </div>
              <div>
                <div v-for="(r,j) in e.replicas" :key="j" style="display:flex; gap:8px; align-items:center; margin-bottom:8px;">
                  <el-select v-model="r.nodeId" :placeholder="t('deployments.create.form.selectNode')" style="flex: 0 0 260px; min-width: 260px;">
                    <el-option v-for="n in nodes" :key="n.nodeId" :label="`${n.nodeId} (${n.ip})`" :value="n.nodeId" />
                  </el-select>
                  <el-input-number v-model="r.count" :min="0" :max="100" />
                  <el-button size="small" @click="delReplicaRow(i,j)">{{ t('common.delete') }}</el-button>
                </div>
                <el-button size="small" type="primary" @click="addReplicaRow(i)">{{ t('deployments.create.buttons.addReplica') }}</el-button>
              </div>
            </el-card>
            <el-button size="small" type="primary" @click="addEntry">{{ t('deployments.create.buttons.addEntry') }}</el-button>
          </div>
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


