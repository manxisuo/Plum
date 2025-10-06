<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, RouterView } from 'vue-router'

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
const lang = ref(locale.value)
function switchLang(l: string){ locale.value = l; lang.value = l }
</script>

<template>
  <el-container style="height:100vh;">
    <el-header>
      <div style="display:flex; align-items:center; gap:16px;">
        <img src="/plum.png" alt="Plum" style="height:48px;" />
        <el-menu mode="horizontal" :default-active="$route.path" :ellipsis="false" style="flex:1;">
          <el-menu-item index="/"><RouterLink to="/">{{ t('nav.home') }}</RouterLink></el-menu-item>
          <el-menu-item index="/nodes"><RouterLink to="/nodes">{{ t('nav.nodes') }}</RouterLink></el-menu-item>
          <el-menu-item index="/apps"><RouterLink to="/apps">{{ t('nav.apps') }}</RouterLink></el-menu-item>
          <el-menu-item index="/deployments"><RouterLink to="/deployments">{{ t('nav.deployments') }}</RouterLink></el-menu-item>
          <el-menu-item index="/assignments"><RouterLink to="/assignments">{{ t('nav.assignments') }}</RouterLink></el-menu-item>
          <el-menu-item index="/services"><RouterLink to="/services">{{ t('nav.services') }}</RouterLink></el-menu-item>
          <el-menu-item index="/tasks"><RouterLink to="/tasks">{{ t('nav.tasks') }}</RouterLink></el-menu-item>
          <el-menu-item index="/workflows"><RouterLink to="/workflows">{{ t('nav.workflows') }}</RouterLink></el-menu-item>
          <el-menu-item index="/resources"><RouterLink to="/resources">{{ t('nav.resources') }}</RouterLink></el-menu-item>
          <el-menu-item index="/workers"><RouterLink to="/workers">{{ t('nav.workers') }}</RouterLink></el-menu-item>
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
