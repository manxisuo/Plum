import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    common: {
      refresh: 'Refresh',
      create: 'Create',
      details: 'Details',
      config: 'Config',
      delete: 'Delete',
      action: 'Action',
      start: 'Start',
      stop: 'Stop',
      submit: 'Submit',
      cancel: 'Cancel',
      yes: 'Yes',
      no: 'No',
      selectNode: 'Select node',
      confirmDelete: 'Confirm delete?',
    },
    nav: {
      home: 'Home',
      assignments: 'Assignments',
      nodes: 'Nodes',
      apps: 'Apps',
      services: 'Services',
      workflows: 'Workflows',
      tasks: 'Tasks',
      deployments: 'Deployments',
      language: 'Language'
    },
    home: {
      overview: 'Plum Overview',
      cards: {
        nodes: 'Nodes',
        deployments: 'Deployments',
        services: 'Services',
        artifacts: 'Artifacts',
        healthy: 'Healthy',
        unhealthy: 'Unhealthy',
        instances: 'Instances',
        endpoints: 'Endpoints'
      },
      charts: {
        nodeHealth: 'Node Health',
        endpointsTop: 'Endpoints per Service (Top 12)'
      },
      table: {
        node: 'Node',
        health: 'Health'
      },
      footerApiBase: 'API_BASE'
    },
    deployments: {
      columns: { deploymentId: 'DeploymentID', name: 'Name', instances: 'Instances' },
      buttons: { create: 'Create Deployment' },
      confirmDelete: 'Confirm delete this deployment? (will not cascade to instances)'
    },
    deploymentDetail: {
      title: 'Deployment Detail',
      buttons: { stopAll: 'Stop All', stopByNode: 'Stop By Node' },
      desc: { deploymentId: 'DeploymentID', name: 'Name', labels: 'Labels' },
      columns: { instanceId: 'InstanceID', nodeId: 'NodeID', artifact: 'Artifact', startCmd: 'StartCmd', desired: 'Desired', action: 'Action' }
    },
    deploymentConfig: {
      title: 'Deployment Config',
      entriesTitle: 'Entries (derived from assignments)',
      columns: { artifact: 'Artifact', startCmd: 'StartCmd', replicas: 'Replicas' },
      labelsTitle: 'Labels'
    },
    assignments: {
      form: { nodeId: 'Node ID' },
      columns: { deployment: 'Deployment', instance: 'Instance', desired: 'Desired', phase: 'Phase', healthy: 'Healthy', lastReportAt: 'LastReportAt', startCmd: 'StartCmd', artifact: 'Artifact', action: 'Action' }
    },
    nodes: {
      columns: { nodeId: 'NodeID', ip: 'IP', lastSeen: 'LastSeen', action: 'Action' },
      confirmDelete: 'Confirm delete this node?'
    },
    apps: {
      uploadZip: 'Upload App ZIP',
      zipTip: 'Package must include start.sh and meta.ini(name/version)',
      buttons: { selectUpload: 'Select and upload ZIP' },
      columns: { app: 'App', version: 'Version', artifact: 'Artifact', sizeBytes: 'Size(Bytes)', uploadedAt: 'UploadedAt', action: 'Action' },
      confirmDelete: 'Confirm delete this artifact?'
    },
    services: {
      title: 'Services',
      endpointsTitle: 'Endpoints - {name}',
      columns: { instance: 'Instance', node: 'Node', address: 'Address', healthy: 'Healthy', lastSeen: 'LastSeen' }
    },
    workflows: {
      buttons: { refresh: 'Refresh', create: 'Create Workflow', run: 'Run', viewLatest: 'View Latest Run' },
      columns: { workflowId: 'WorkflowID', name: 'Name', steps: 'Steps', action: 'Action' },
      dialog: {
        title: 'Create Workflow',
        form: { name: 'Name', steps: 'Steps', executor: 'Executor', timeoutSec: 'timeoutSec', maxRetries: 'maxRetries', addStep: 'Add Step', delete: 'Delete' },
        footer: { cancel: 'Cancel', submit: 'Submit' }
      }
    },
    workflowRun: {
      title: 'Workflow Run Detail',
      desc: { runId: 'RunID', workflowId: 'WorkflowID', state: 'State', created: 'Created' },
      columns: { ord: '#', step: 'Step', taskId: 'TaskID', state: 'State' }
    },
    taskDefs: {
      buttons: { refresh: 'Refresh', create: 'Create Definition', run: 'Run', details: 'Details' },
      columns: { defId: 'DefID', name: 'Name', executor: 'Executor', target: 'Target', latestState: 'Latest State', latestTime: 'Latest Time', action: 'Action' },
      dialog: { title: 'Create TaskDefinition', form: { name: 'Name', executor: 'Executor', targetKind: 'TargetKind', targetRef: 'TargetRef', serviceVersion: 'Service Version', serviceProtocol: 'Service Protocol', servicePort: 'Service Port', servicePath: 'Service Path' }, footer: { cancel: 'Cancel', submit: 'Submit' } }
    },
    taskDefDetail: {
      title: 'TaskDefinition Detail',
      desc: { defId: 'DefID', name: 'Name', executor: 'Executor' },
      runsTitle: 'Run History',
      columns: { taskId: 'TaskID', state: 'State', created: 'Created', result: 'Result', action: 'Action' },
      buttons: { start: 'Start', cancel: 'Cancel', delete: 'Delete' },
      confirmDelete: 'Confirm delete this run?'
    }
  },
  zh: {
    common: {
      refresh: '刷新',
      create: '创建',
      details: '详情',
      config: '配置',
      delete: '删除',
      action: '操作',
      start: '开始',
      stop: '停止',
      submit: '提交',
      cancel: '取消',
      yes: '是',
      no: '否',
      selectNode: '选择节点',
      confirmDelete: '确认删除？',
    },
    nav: {
      home: '首页',
      assignments: '分配',
      nodes: '节点',
      apps: '应用',
      services: '服务',
      workflows: '工作流',
      tasks: '任务',
      deployments: '部署',
      language: '语言'
    },
    home: {
      overview: 'Plum 概览',
      cards: {
        nodes: '节点',
        deployments: '部署',
        services: '服务',
        artifacts: '制品',
        healthy: '健康',
        unhealthy: '不健康',
        instances: '实例',
        endpoints: '端点'
      },
      charts: {
        nodeHealth: '节点健康',
        endpointsTop: '各服务可用端点数（Top 12）'
      },
      table: {
        node: '节点',
        health: '健康状态'
      },
      footerApiBase: 'API 基础地址'
    },
    deployments: {
      columns: { deploymentId: '部署ID', name: '名称', instances: '实例数' },
      buttons: { create: '创建部署' },
      confirmDelete: '确认删除该部署？（不会级联删除实例分配）'
    },
    deploymentDetail: {
      title: '部署详情',
      buttons: { stopAll: '全部停止', stopByNode: '按节点停止' },
      desc: { deploymentId: '部署ID', name: '名称', labels: '标签' },
      columns: { instanceId: '实例ID', nodeId: '节点ID', artifact: '制品', startCmd: '启动命令', desired: '期望状态', action: '操作' }
    },
    deploymentConfig: {
      title: '部署配置',
      entriesTitle: '条目（根据当前 assignments 推导）',
      columns: { artifact: '制品', startCmd: '启动命令', replicas: '副本数' },
      labelsTitle: '标签'
    },
    assignments: {
      form: { nodeId: '节点 ID' },
      columns: { deployment: '部署', instance: '实例', desired: '期望', phase: '阶段', healthy: '健康', lastReportAt: '最近上报', startCmd: '启动命令', artifact: '制品', action: '操作' }
    },
    nodes: {
      columns: { nodeId: '节点ID', ip: 'IP', lastSeen: '最近活跃', action: '操作' },
      confirmDelete: '确认删除该节点？'
    },
    apps: {
      uploadZip: '上传应用包（ZIP）',
      zipTip: '包内需包含 start.sh 与 meta.ini(name/version)',
      buttons: { selectUpload: '选择并上传 ZIP' },
      columns: { app: '应用', version: '版本', artifact: '制品', sizeBytes: '大小(字节)', uploadedAt: '上传时间', action: '操作' },
      confirmDelete: '确认删除该包？'
    },
    services: {
      title: '服务',
      endpointsTitle: '端点 - {name}',
      columns: { instance: '实例', node: '节点', address: '地址', healthy: '健康', lastSeen: '最近活跃' }
    },
    workflows: {
      buttons: { refresh: '刷新', create: '创建工作流', run: '运行', viewLatest: '查看最新运行' },
      columns: { workflowId: '工作流ID', name: '名称', steps: '步骤', action: '操作' },
      dialog: {
        title: '创建工作流',
        form: { name: '名称', steps: '步骤', executor: '执行器', timeoutSec: '超时秒', maxRetries: '最大重试', addStep: '添加步骤', delete: '删除' },
        footer: { cancel: '取消', submit: '提交' }
      }
    },
    workflowRun: {
      title: '工作流运行详情',
      desc: { runId: '运行ID', workflowId: '工作流ID', state: '状态', created: '创建时间' },
      columns: { ord: '#', step: '步骤', taskId: '任务ID', state: '状态' }
    },
    taskDefs: {
      buttons: { refresh: '刷新', create: '创建定义', run: '运行', details: '详情' },
      columns: { defId: '定义ID', name: '名称', executor: '执行器', target: '目标', latestState: '最新状态', latestTime: '最新时间', action: '操作' },
      dialog: { title: '创建 TaskDefinition', form: { name: '名称', executor: '执行器', targetKind: '目标类型', targetRef: '目标引用', serviceVersion: '服务版本', serviceProtocol: '服务协议', servicePort: '服务端口', servicePath: '调用路径' }, footer: { cancel: '取消', submit: '提交' } }
    },
    taskDefDetail: {
      title: 'TaskDefinition 详情',
      desc: { defId: '定义ID', name: '名称', executor: '执行器' },
      runsTitle: '运行历史',
      columns: { taskId: '任务ID', state: '状态', created: '创建时间', result: '结果', action: '操作' },
      buttons: { start: '开始', cancel: '取消', delete: '删除' },
      confirmDelete: '确认删除该任务？'
    }
  }
}

export const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  fallbackLocale: 'en',
  messages
})

export type MessageSchema = typeof messages


