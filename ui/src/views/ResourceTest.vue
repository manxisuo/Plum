<template>
  <div>
    <h2>Resource Test Page</h2>
    <el-button @click="loadData">Load Resources</el-button>
    <div v-if="loading">Loading...</div>
    <div v-else>
      <p>Resources count: {{ resources.length }}</p>
      <pre>{{ JSON.stringify(resources, null, 2) }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const API_BASE = (import.meta as any).env?.VITE_API_BASE || ''
const resources = ref([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try {
    console.log('Loading from:', `${API_BASE}/v1/resources`)
    const res = await fetch(`${API_BASE}/v1/resources`)
    console.log('Response status:', res.status)
    const data = await res.json()
    console.log('Loaded data:', data)
    resources.value = data
  } catch (e: any) {
    console.error('Error:', e)
    alert('Error: ' + e.message)
  } finally {
    loading.value = false
  }
}

// 自动加载
loadData()
</script>
