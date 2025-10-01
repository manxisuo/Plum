<script setup lang="ts">
import { ref, computed } from 'vue'
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
const tab = ref<'assignments' | 'nodes' | 'apps' | 'tasks'>('assignments')
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
</script>

<template>
  <el-container style="height:100vh;">
    <el-header>
      <div style="display:flex; align-items:center; gap:16px;">
        <strong>Plum</strong>
        <el-menu mode="horizontal" :default-active="$route.path">
          <el-menu-item index="/"><RouterLink to="/">Home</RouterLink></el-menu-item>
          <el-menu-item index="/assignments"><RouterLink to="/assignments">Assignments</RouterLink></el-menu-item>
          <el-menu-item index="/nodes"><RouterLink to="/nodes">Nodes</RouterLink></el-menu-item>
          <el-menu-item index="/apps"><RouterLink to="/apps">Apps</RouterLink></el-menu-item>
          <el-menu-item index="/services"><RouterLink to="/services">Services</RouterLink></el-menu-item>
          <el-menu-item index="/tasks"><RouterLink to="/tasks">Tasks</RouterLink></el-menu-item>
        </el-menu>
      </div>
    </el-header>
    <el-main>
      <RouterView />
      <div style="margin-top:12px; color:#888; font-size:12px;">API_BASE: {{ API_BASE || '[proxy /v1 → :8080]' }}</div>
    </el-main>
  </el-container>
</template>

<style scoped>
</style>


