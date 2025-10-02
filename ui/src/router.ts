import { createRouter, createWebHistory } from 'vue-router'
import Home from './views/Home.vue'
import DeploymentsList from './views/DeploymentsList.vue'
import TaskCreate from './views/TaskCreate.vue'
import DeploymentDetail from './views/DeploymentDetail.vue'
import DeploymentConfig from './views/DeploymentConfig.vue'
import Nodes from './views/Nodes.vue'
import Apps from './views/Apps.vue'
import Assignments from './views/Assignments.vue'
// @ts-ignore: vite handles .vue type
import Services from './views/Services.vue'
// @ts-ignore
import Tasks from './views/Tasks.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/nodes', component: Nodes },
    { path: '/apps', component: Apps },
    { path: '/services', component: Services },
    { path: '/tasks', component: Tasks },
    { path: '/assignments', component: Assignments },
    // deployments
    { path: '/deployments', component: DeploymentsList },
    { path: '/deployments/create', component: TaskCreate },
    { path: '/deployments/:id', component: DeploymentDetail, props: true },
    { path: '/deployments/:id/config', component: DeploymentConfig, props: true },
  ]
})


