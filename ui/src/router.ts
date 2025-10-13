import { createRouter, createWebHistory } from 'vue-router'
import Home from './views/Home.vue'
import DeploymentsList from './views/DeploymentsList.vue'
import DeploymentCreate from './views/DeploymentCreate.vue'
import DeploymentDetail from './views/DeploymentDetail.vue'
import DeploymentConfig from './views/DeploymentConfig.vue'
import Nodes from './views/Nodes.vue'
import Apps from './views/Apps.vue'
import Assignments from './views/Assignments.vue'
// @ts-ignore: vite handles .vue type
import Services from './views/Services.vue'
// @ts-ignore
import Workflows from './views/Workflows.vue'
// @ts-ignore
import WorkflowRunDetail from './views/WorkflowRunDetail.vue'
// @ts-ignore
import WorkflowRuns from './views/WorkflowRuns.vue'
// @ts-ignore
import Tasks from './views/Tasks.vue'
// @ts-ignore
import TaskDefs from './views/TaskDefs.vue'
// @ts-ignore
import TaskDefDetail from './views/TaskDefDetail.vue'
// @ts-ignore
import Resources from './views/Resources.vue'
import Workers from './views/Workers.vue'
// @ts-ignore
import KVStore from './views/KVStore.vue'
// @ts-ignore
import DAGWorkflows from './views/DAGWorkflows.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: Home },
    { path: '/nodes', component: Nodes },
    { path: '/apps', component: Apps },
    { path: '/services', component: Services },
    { path: '/workflows', component: Workflows },
    { path: '/dag-workflows', component: DAGWorkflows },
    { path: '/workflows/:workflowId/runs', component: WorkflowRuns, props: true },
    { path: '/workflow-runs/:id', component: WorkflowRunDetail, props: true },
    { path: '/task-defs', component: TaskDefs },
    { path: '/task-defs/:id', component: TaskDefDetail, props: true },
    { path: '/tasks/defs/:id', component: TaskDefDetail, props: true },
    { path: '/tasks', component: Tasks },
    { path: '/resources', component: Resources },
    { path: '/workers', component: Workers },
    { path: '/assignments', component: Assignments },
    { path: '/kv-store', component: KVStore },
    // deployments
    { path: '/deployments', component: DeploymentsList },
    { path: '/deployments/create', component: DeploymentCreate },
    { path: '/deployments/:id', component: DeploymentDetail, props: true },
    { path: '/deployments/:id/config', component: DeploymentConfig, props: true },
  ]
})


