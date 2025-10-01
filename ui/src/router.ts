import { createRouter, createWebHistory } from 'vue-router'
import Home from './views/Home.vue'
import TaskList from './views/TaskList.vue'
import TaskCreate from './views/TaskCreate.vue'
import TaskDetail from './views/TaskDetail.vue'
import TaskConfig from './views/TaskConfig.vue'
import Nodes from './views/Nodes.vue'
import Apps from './views/Apps.vue'
import Assignments from './views/Assignments.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/nodes', component: Nodes },
    { path: '/apps', component: Apps },
    { path: '/assignments', component: Assignments },
    { path: '/tasks', component: TaskList },
    { path: '/tasks/create', component: TaskCreate },
    { path: '/tasks/:id', component: TaskDetail, props: true },
    { path: '/tasks/:id/config', component: TaskConfig, props: true },
  ]
})


