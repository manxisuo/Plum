<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, RouterView, useRouter } from 'vue-router'

type Assignment = {
	instanceId: string
	desired: string
	artifactUrl: string
	startCmd: string
}

type Assignments = { items: Assignment[] }

const nodeId = ref('nodeA')
const loading = ref(false)
const error = ref<string | null>(null)
const data = ref<Assignments>({ items: [] })

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const tab = ref<'assignments' | 'nodes' | 'apps' | 'deployments'>('assignments')
const url = computed(() => `${API_BASE}/v1/assignments?nodeId=${encodeURIComponent(nodeId.value)}`)

async function fetchAssignments() {
	loading.value = true
	error.value = null
	try {
		const res = await fetch(url.value)
		if (!res.ok) throw new Error(`HTTP ${res.status}`)
		data.value = await res.json() as Assignments
	} catch (e: any) {
		error.value = e?.message || '请求失败'
	} finally {
		loading.value = false
	}
}

fetchAssignments()

const { t, locale } = useI18n()
const router = useRouter()
const lang = ref(locale.value)
function switchLang(l: string){ locale.value = l; lang.value = l }
function handleMenuSelect(index: string) {
  router.push(index)
}
</script>

<template>
  <el-container style="height:100vh;">
    <el-header>
      <div style="display:flex; align-items:center; gap:16px;">
        <img src="/plum.png" alt="Plum" style="height:48px;" />
        <el-menu mode="horizontal" :default-active="$route.path" :ellipsis="false" style="flex:1;" @select="handleMenuSelect">
          <el-menu-item index="/">{{ t('nav.home') }}</el-menu-item>
          <el-menu-item index="/nodes">{{ t('nav.nodes') }}</el-menu-item>
          <el-menu-item index="/apps">{{ t('nav.apps') }}</el-menu-item>
          <el-menu-item index="/deployments">{{ t('nav.deployments') }}</el-menu-item>
          <el-menu-item index="/assignments">{{ t('nav.assignments') }}</el-menu-item>
          <el-menu-item index="/services">{{ t('nav.services') }}</el-menu-item>
          <el-menu-item index="/tasks">{{ t('nav.tasks') }}</el-menu-item>
          <el-menu-item index="/workers">{{ t('nav.workers') }}</el-menu-item>
          <el-menu-item index="/workflows">{{ t('nav.workflows') }}</el-menu-item>
          <el-menu-item index="/dag-workflows">{{ t('nav.dagWorkflows') }}</el-menu-item>
          <el-menu-item index="/kv-store">{{ t('nav.kvStore') }}</el-menu-item>
          <el-menu-item index="/resources">{{ t('nav.resources') }}</el-menu-item>
        </el-menu>
        <div style="display:flex; align-items:center; gap:8px;">
          <el-select v-model="lang" size="small" style="width:120px;" @change="switchLang">
            <el-option label="中文" value="zh" />
            <el-option label="English" value="en" />
          </el-select>
        </div>
      </div>
    </el-header>
    <el-main>
      <RouterView />
      <!-- <div style="padding:12px 16px 0 16px; color:#888; font-size:12px;">API_BASE: {{ API_BASE || '[proxy /v1 → :8080]' }}</div> -->
    </el-main>
  </el-container>
</template>

<style scoped>
</style>

<style>
/* 完全隐藏Element Plus的loading图标和动画，保留静态图标 */
.el-button.is-loading::before {
  display: none !important;
}
.el-button.is-loading .el-icon--loading {
  display: none !important;
}
.el-button.is-loading .el-icon:not(.el-icon--loading) {
  display: inline-flex !important;
}

/* 调整所有卡片头部的高度 */
.el-card__header {
  padding: 12px 12px 12px 12px !important;
  min-height: 48px !important;
  /* display: flex !important; */
  align-items: center !important;
}

/* 调整卡片主体的内边距 */
.el-card__body {
  padding: 16px 20px !important;
}
</style>
