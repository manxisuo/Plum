import { createI18n } from 'vue-i18n'

const messages = {
  en: {
    common: {
      refresh: 'Refresh',
      create: 'Create',
      details: 'Details',
      descriptions: 'Descriptions',
      config: 'Config',
      delete: 'Delete',
      action: 'Action',
      start: 'Start',
      stop: 'Stop',
      submit: 'Submit',
      reset: 'Reset',
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
      resources: 'Resources',
      workers: 'Workers',
      language: 'Language'
    },
    home: {
      welcome: {
        title: 'Welcome to Plum',
        subtitle: 'Distributed Task Orchestration Platform'
      },
      buttons: {
        refresh: 'Refresh Data',
        createDeployment: 'Create Deployment'
      },
      cards: {
        nodes: 'Nodes',
        deployments: 'Deployments',
        services: 'Services',
        artifacts: 'Artifacts',
        healthy: 'Healthy',
        unhealthy: 'Unhealthy',
        instances: 'Instances',
        endpoints: 'Endpoints',
        totalSize: 'Total Size'
      },
      charts: {
        nodeHealth: 'Node Health Status',
        endpointsTop: 'Service Endpoints Distribution (Top 12)'
      },
      health: {
        healthy: 'Healthy Rate'
      },
      quickActions: {
        title: 'Quick Actions',
        createDeployment: 'Create Deployment',
        runTask: 'Run Task',
        manageResources: 'Manage Resources',
        viewWorkflows: 'View Workflows'
      },
      more: 'more',
      table: {
        node: 'Node',
        health: 'Health'
      }
    },
    deployments: {
      columns: { deploymentId: 'DeploymentID', name: 'Name', instances: 'Instances' },
      buttons: { create: 'Create Deployment' },
      stats: { deployments: 'Deployments', instances: 'Instances' },
      table: { title: 'Deployment List', items: 'items' },
      confirmDelete: 'Confirm delete this deployment? (will not cascade to instances)',
      create: {
        title: 'Create Deployment',
        form: {
          name: 'Name',
          entries: 'Entries',
          labels: 'Labels',
          selectArtifact: 'Select Artifact',
          selectNode: 'Select Node',
          startCmdPlaceholder: 'Start command (optional, override default ./start.sh)',
          keyPlaceholder: 'Key',
          valuePlaceholder: 'Value'
        },
        buttons: {
          addEntry: 'Add Entry',
          deleteEntry: 'Delete Entry',
          addReplica: 'Add Replica',
          addLabel: 'Add Label',
          create: 'Create'
        },
        validation: {
          nameRequired: 'Please enter Name',
          entriesRequired: 'Please add at least one entry',
          artifactRequired: 'Please select Artifact',
          artifactNotFound: 'Artifact not found',
          replicasRequired: 'Please configure replicas for entry'
        },
        messages: {
          created: 'Created successfully',
          createFailed: 'Create failed'
        }
      }
    },
    deploymentDetail: {
      title: 'Deployment Detail',
      buttons: { stopAll: 'Stop All', stopByNode: 'Stop By Node' },
      desc: { deploymentId: 'DeploymentID', name: 'Name', labels: 'Labels' },
      columns: { instanceId: 'InstanceID', nodeId: 'NodeID', artifact: 'Artifact', startCmd: 'StartCmd', desired: 'Desired', action: 'Action' },
      stats: { instances: 'Instances', running: 'Running', stopped: 'Stopped', healthy: 'Healthy' },
      table: { title: 'Instance List', items: 'items' }
    },
    deploymentConfig: {
      title: 'Deployment Config',
      entriesTitle: 'Entries (derived from assignments)',
      columns: { artifact: 'Artifact', startCmd: 'StartCmd', replicas: 'Replicas' },
      labelsTitle: 'Labels',
      stats: { entries: 'Entries', replicas: 'Replicas', labels: 'Labels' },
      table: { items: 'items', labels: 'labels' },
      noLabels: 'No labels configured'
    },
    assignments: {
      form: { nodeId: 'Node ID' },
      columns: { deployment: 'Deployment', instance: 'Instance', desired: 'Desired', phase: 'Phase', healthy: 'Healthy', lastReportAt: 'LastReportAt', startCmd: 'StartCmd', artifact: 'Artifact', action: 'Action' },
      stats: { total: 'Total', running: 'Running', stopped: 'Stopped', healthy: 'Healthy' },
      table: { title: 'Instance Assignments', items: 'items' },
      error: { title: 'Error' }
    },
    nodes: {
      columns: { nodeId: 'NodeID', ip: 'IP', health: 'Health', lastSeen: 'LastSeen', action: 'Action' },
      stats: { total: 'Total', healthy: 'Healthy', unhealthy: 'Unhealthy', unknown: 'Unknown' },
      table: { title: 'Node List', items: 'items' },
      error: { title: 'Error' },
      confirmDelete: 'Confirm delete this node?'
    },
    apps: {
      uploadZip: 'Upload App ZIP',
      zipTip: 'Package must include start.sh and meta.ini(name/version)',
      uploadDescription: 'Upload a ZIP file containing your application package with start.sh and meta.ini',
      buttons: { refresh: 'Refresh', selectUpload: 'Select and upload ZIP' },
      stats: { total: 'Total' },
      table: { title: 'Application Packages', items: 'items' },
      columns: { app: 'App', version: 'Version', artifact: 'Artifact', sizeBytes: 'Size(Bytes)', uploadedAt: 'UploadedAt', action: 'Action' },
      confirmDelete: 'Confirm delete this artifact?'
    },
    workers: {
      title: 'Worker Management',
      subtitle: 'Manage embedded workers and HTTP workers',
      stats: {
        totalWorkers: 'Total Workers',
        activeApps: 'Active Apps',
        supportedServices: 'Supported Services',
        healthRate: 'Health Rate'
      },
      tabs: {
        embedded: 'Embedded Workers',
        http: 'HTTP Workers'
      },
      filters: {
        appName: 'App Name',
        node: 'Node',
        status: 'Status',
        search: 'Search...'
      },
      status: {
        healthy: 'Healthy',
        warning: 'Warning',
        offline: 'Offline'
      },
      columns: {
        appInfo: 'App Info',
        node: 'Node',
        supportedTasks: 'Supported Tasks',
        status: 'Status',
        lastSeen: 'Last Seen',
        actions: 'Actions'
      },
      buttons: {
        refresh: 'Refresh',
        details: 'Details',
        delete: 'Delete'
      },
      details: {
        title: 'Worker Details',
        basicInfo: 'Basic Info',
        supportedTasks: 'Supported Tasks',
        labels: 'Labels',
        workerId: 'Worker ID',
        appName: 'App Name',
        version: 'Version',
        instanceId: 'Instance ID',
        node: 'Node',
        grpcAddress: 'gRPC Address',
        httpUrl: 'HTTP URL',
        lastHeartbeat: 'Last Heartbeat',
        capacity: 'Capacity'
      },
      messages: {
        loadFailed: 'Load failed',
        deleteSuccess: 'Delete successful',
        deleteFailed: 'Delete failed'
      },
      confirmDelete: 'Confirm delete this worker?'
    },
    services: {
      title: 'Services',
      endpointsTitle: 'Endpoints - {name}',
      stats: { services: 'Services', endpoints: 'Endpoints', healthy: 'Healthy' },
      columns: { instance: 'Instance', node: 'Node', address: 'Address', healthy: 'Healthy', lastSeen: 'LastSeen' }
    },
    workflows: {
      buttons: { refresh: 'Refresh', create: 'Create Workflow', run: 'Run', viewLatest: 'View Latest Run' },
      stats: { workflows: 'Workflows', steps: 'Steps' },
      table: { title: 'Workflow List', items: 'items' },
      columns: { workflowId: 'WorkflowID', name: 'Name', steps: 'Steps', action: 'Action' },
      dialog: {
        title: 'Create Workflow',
        form: { name: 'Name', steps: 'Steps', executor: 'Executor', timeoutSec: 'timeoutSec', maxRetries: 'maxRetries', addStep: 'Add Step', delete: 'Delete' },
        footer: { cancel: 'Cancel', submit: 'Submit' }
      }
    },
    workflowRuns: {
      title: 'Workflow Run History ({workflowId})',
      buttons: { back: '← Back to Workflows', view: 'View Details' },
      stats: { total: 'Total', succeeded: 'Succeeded', running: 'Running', failed: 'Failed' },
      table: { items: 'items' },
      columns: { runId: 'Run ID', state: 'State', createdAt: 'Created', startedAt: 'Started', finishedAt: 'Finished' }
    },
    workflowRun: {
      title: 'Workflow Run Detail',
      desc: { runId: 'RunID', workflowId: 'WorkflowID', state: 'State', created: 'Created' },
      columns: { ord: '#', step: 'Step', taskId: 'TaskID', state: 'State' }
    },
    taskDefs: {
      title: 'Task Definitions',
      buttons: { refresh: 'Refresh', create: 'Create Definition', run: 'Run', details: 'Details' },
      columns: { defId: 'DefID', name: 'Name', executor: 'Executor', target: 'Target', latestState: 'Latest State', latestTime: 'Latest Time', action: 'Action' },
      dialog: { 
        title: 'Create Task Definition', 
        form: { 
          name: 'Name', 
          executor: 'Executor', 
          targetKind: 'Target Type', 
          targetRef: 'Target Reference', 
          serviceVersion: 'Service Version', 
          serviceProtocol: 'Service Protocol', 
          servicePort: 'Service Port', 
          servicePath: 'Service Path', 
          command: 'Command' 
        }, 
        footer: { cancel: 'Cancel', submit: 'Submit' },
        help: {
          embeddedNode: 'Execute on a specific node using embedded worker',
          embeddedApp: 'Execute on workers belonging to a specific application',
          service: 'Execute via HTTP call to service endpoint',
          osProcessNode: 'Execute OS command on a specific node'
        }
      },
      stats: { total: 'Total', running: 'Running', succeeded: 'Succeeded', failed: 'Failed' },
      search: { placeholder: 'Search by name, ID, or executor...' },
      filter: { executor: 'Executor', state: 'State', all: 'All' },
      table: { title: 'Task Definitions', items: 'items' },
      status: { neverRun: 'Never Run', running: 'Running', completed: 'Completed', succeeded: 'Succeeded', failed: 'Failed', cancelled: 'Cancelled', pending: 'Pending' },
      confirm: { delete: 'Confirm delete this definition?' }
    },
    taskDefDetail: {
      title: 'TaskDefinition Detail',
      desc: { defId: 'DefID', name: 'Name', executor: 'Executor' },
      runsTitle: 'Run History',
      columns: { taskId: 'TaskID', state: 'State', created: 'Created', result: 'Result', action: 'Action' },
      buttons: { start: 'Start', cancel: 'Cancel', delete: 'Delete' },
      confirmDelete: 'Confirm delete this run?'
    },
    resources: {
      title: 'Resource Management',
      buttons: { refresh: 'Refresh', delete: 'Delete', send: 'Send', operation: 'Operation', submit: 'Submit' },
      columns: { 
        resourceId: 'Resource ID', 
        type: 'Type', 
        nodeId: 'Node', 
        status: 'Status',
        action: 'Action',
        name: 'Name',
        dataType: 'Data Type',
        defaultValue: 'Default Value',
        unit: 'Unit',
        range: 'Range',
        time: 'Time',
        stateData: 'State Data'
      },
      status: {
        healthy: 'Healthy',
        warning: 'Warning',
        offline: 'Offline'
      },
      sections: {
        resourceList: 'Resource List',
        resourceDetail: 'Resource Detail',
        resourceDescription: 'Resource Description',
        stateDescription: 'State Description',
        operationDescription: 'Operation Description',
        historyStates: 'History States (Latest 20)'
      },
      desc: {
        resourceId: 'Resource ID',
        type: 'Type',
        nodeId: 'Node',
        status: 'Status',
        createdAt: 'Created At',
        lastHeartbeat: 'Last Heartbeat'
      },
      messages: {
        selectResource: 'Please select a resource to view details',
        deleteSuccess: 'Deleted successfully',
        deleteFailed: 'Delete failed',
        operationSent: 'Operation sent successfully',
        operationFailed: 'Operation sending failed',
        loadFailed: 'Load failed',
        stateLoadFailed: 'Failed to load states'
      },
      dialogs: {
        operationTitle: 'Resource Operation'
      },
      validation: {
        rangeError: 'Value is out of range'
      },
      confirmDelete: 'Confirm delete this resource?'
    }
  },
  zh: {
    common: {
      refresh: '刷新',
      create: '创建',
      details: '详情',
      descriptions: '描述',
      config: '配置',
      delete: '删除',
      action: '操作',
      start: '开始',
      stop: '停止',
      submit: '提交',
      reset: '重置',
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
      resources: '资源',
      workers: '工作器',
      language: '语言'
    },
    home: {
      welcome: {
        title: '欢迎使用 Plum',
        subtitle: '分布式任务编排平台'
      },
      buttons: {
        refresh: '刷新数据',
        createDeployment: '创建部署'
      },
      cards: {
        nodes: '节点',
        deployments: '部署',
        services: '服务',
        artifacts: '制品',
        healthy: '健康',
        unhealthy: '不健康',
        instances: '实例',
        endpoints: '端点',
        totalSize: '总大小'
      },
      charts: {
        nodeHealth: '节点健康状态',
        endpointsTop: '服务端点分布（Top 12）'
      },
      health: {
        healthy: '健康率'
      },
      quickActions: {
        title: '快速操作',
        createDeployment: '创建部署',
        runTask: '运行任务',
        manageResources: '管理资源',
        viewWorkflows: '查看工作流'
      },
      more: '更多',
      table: {
        node: '节点',
        health: '健康状态'
      }
    },
    deployments: {
      columns: { deploymentId: '部署ID', name: '名称', instances: '实例数' },
      buttons: { create: '创建部署' },
      stats: { deployments: '部署', instances: '实例' },
      table: { title: '部署列表', items: '项' },
      confirmDelete: '确认删除该部署？（不会级联删除实例分配）',
      create: {
        title: '创建部署',
        form: {
          name: '名称',
          entries: '条目',
          labels: '标签',
          selectArtifact: '选择制品',
          selectNode: '选择节点',
          startCmdPlaceholder: '启动命令（可选，覆盖包默认 ./start.sh）',
          keyPlaceholder: '键',
          valuePlaceholder: '值'
        },
        buttons: {
          addEntry: '新增条目',
          deleteEntry: '删除条目',
          addReplica: '新增副本行',
          addLabel: '新增标签',
          create: '创建'
        },
        validation: {
          nameRequired: '请输入名称',
          entriesRequired: '请添加至少一条条目',
          artifactRequired: '请选择制品',
          artifactNotFound: '制品不存在',
          replicasRequired: '请为条目配置副本'
        },
        messages: {
          created: '创建成功',
          createFailed: '创建失败'
        }
      }
    },
    deploymentDetail: {
      title: '部署详情',
      buttons: { stopAll: '全部停止', stopByNode: '按节点停止' },
      desc: { deploymentId: '部署ID', name: '名称', labels: '标签' },
      columns: { instanceId: '实例ID', nodeId: '节点ID', artifact: '制品', startCmd: '启动命令', desired: '期望状态', action: '操作' },
      stats: { instances: '实例', running: '运行中', stopped: '已停止', healthy: '健康' },
      table: { title: '实例列表', items: '项' }
    },
    deploymentConfig: {
      title: '部署配置',
      entriesTitle: '条目（根据当前 assignments 推导）',
      columns: { artifact: '制品', startCmd: '启动命令', replicas: '副本数' },
      labelsTitle: '标签',
      stats: { entries: '条目', replicas: '副本', labels: '标签' },
      table: { items: '项', labels: '个标签' },
      noLabels: '暂无标签配置'
    },
    assignments: {
      form: { nodeId: '节点 ID' },
      columns: { deployment: '部署', instance: '实例', desired: '期望', phase: '阶段', healthy: '健康', lastReportAt: '最近上报', startCmd: '启动命令', artifact: '制品', action: '操作' },
      stats: { total: '总计', running: '运行中', stopped: '已停止', healthy: '健康' },
      table: { title: '实例分配', items: '项' },
      error: { title: '错误' }
    },
    nodes: {
      columns: { nodeId: '节点ID', ip: 'IP', health: '健康状态', lastSeen: '最近活跃', action: '操作' },
      stats: { total: '总计', healthy: '健康', unhealthy: '不健康', unknown: '未知' },
      table: { title: '节点列表', items: '项' },
      error: { title: '错误' },
      confirmDelete: '确认删除该节点？'
    },
    apps: {
      uploadZip: '上传应用包（ZIP）',
      zipTip: '包内需包含 start.sh 与 meta.ini(name/version)',
      uploadDescription: '上传包含应用程序包的ZIP文件，需要包含start.sh和meta.ini文件',
      buttons: { refresh: '刷新', selectUpload: '选择并上传 ZIP' },
      stats: { total: '总数' },
      table: { title: '应用包列表', items: '项' },
      columns: { app: '应用', version: '版本', artifact: '制品', sizeBytes: '大小(字节)', uploadedAt: '上传时间', action: '操作' },
      confirmDelete: '确认删除该包？'
    },
    workers: {
      title: '工作器管理',
      subtitle: '管理嵌入式工作器和HTTP工作器',
      stats: {
        totalWorkers: '总工作器',
        activeApps: '活跃应用',
        supportedServices: '支持服务',
        healthRate: '健康率'
      },
      tabs: {
        embedded: '嵌入式工作器',
        http: 'HTTP工作器'
      },
      filters: {
        appName: '应用名称',
        node: '节点',
        status: '状态',
        search: '搜索...'
      },
      status: {
        healthy: '健康',
        warning: '警告',
        offline: '离线'
      },
      columns: {
        appInfo: '应用信息',
        node: '节点',
        supportedTasks: '支持的任务',
        status: '状态',
        lastSeen: '最后心跳',
        actions: '操作'
      },
      buttons: {
        refresh: '刷新',
        details: '详情',
        delete: '删除'
      },
      details: {
        title: '工作器详情',
        basicInfo: '基本信息',
        supportedTasks: '支持的任务',
        labels: '标签信息',
        workerId: '工作器ID',
        appName: '应用名称',
        version: '版本',
        instanceId: '实例ID',
        node: '节点',
        grpcAddress: 'gRPC地址',
        httpUrl: 'HTTP地址',
        lastHeartbeat: '最后心跳',
        capacity: '容量'
      },
      messages: {
        loadFailed: '加载失败',
        deleteSuccess: '删除成功',
        deleteFailed: '删除失败'
      },
      confirmDelete: '确认删除该工作器？'
    },
    services: {
      title: '服务',
      endpointsTitle: '端点 - {name}',
      stats: { services: '服务', endpoints: '端点', healthy: '健康' },
      columns: { instance: '实例', node: '节点', address: '地址', healthy: '健康', lastSeen: '最近活跃' }
    },
    workflows: {
      buttons: { refresh: '刷新', create: '创建工作流', run: '运行', viewLatest: '查看最新运行' },
      stats: { workflows: '工作流', steps: '步骤' },
      table: { title: '工作流列表', items: '项' },
      columns: { workflowId: '工作流ID', name: '名称', steps: '步骤', action: '操作' },
      dialog: {
        title: '创建工作流',
        form: { name: '名称', steps: '步骤', executor: '执行器', timeoutSec: '超时秒', maxRetries: '最大重试', addStep: '添加步骤', delete: '删除' },
        footer: { cancel: '取消', submit: '提交' }
      }
    },
    workflowRuns: {
      title: '工作流运行历史 ({workflowId})',
      buttons: { back: '← 返回工作流列表', view: '查看详情' },
      stats: { total: '总计', succeeded: '成功', running: '运行中', failed: '失败' },
      table: { items: '项' },
      columns: { runId: '运行ID', state: '状态', createdAt: '创建时间', startedAt: '开始时间', finishedAt: '结束时间' }
    },
    workflowRun: {
      title: '工作流运行详情',
      desc: { runId: '运行ID', workflowId: '工作流ID', state: '状态', created: '创建时间' },
      columns: { ord: '#', step: '步骤', taskId: '任务ID', state: '状态' }
    },
    taskDefs: {
      title: '任务定义',
      buttons: { refresh: '刷新', create: '创建定义', run: '运行', details: '详情' },
      columns: { defId: '定义ID', name: '名称', executor: '执行器', target: '目标', latestState: '最新状态', latestTime: '最新时间', action: '操作' },
      dialog: { 
        title: '创建任务定义', 
        form: { 
          name: '名称', 
          executor: '执行器', 
          targetKind: '目标类型', 
          targetRef: '目标引用', 
          serviceVersion: '服务版本', 
          serviceProtocol: '服务协议', 
          servicePort: '服务端口', 
          servicePath: '调用路径', 
          command: '命令' 
        }, 
        footer: { cancel: '取消', submit: '提交' },
        help: {
          embeddedNode: '在指定节点上使用嵌入式工作器执行',
          embeddedApp: '在属于特定应用的嵌入式工作器上执行',
          service: '通过HTTP调用服务端点执行',
          osProcessNode: '在指定节点上执行操作系统命令'
        }
      },
      stats: { total: '总数', running: '运行中', succeeded: '成功', failed: '失败' },
      search: { placeholder: '按名称、ID或执行器搜索...' },
      filter: { executor: '执行器', state: '状态', all: '全部' },
      table: { title: '任务定义列表', items: '项' },
      status: { neverRun: '从未运行', running: '运行中', completed: '已完成', succeeded: '成功', failed: '失败', cancelled: '已取消', pending: '等待中' },
      confirm: { delete: '确认删除该定义？' }
    },
    taskDefDetail: {
      title: 'TaskDefinition 详情',
      desc: { defId: '定义ID', name: '名称', executor: '执行器' },
      runsTitle: '运行历史',
      columns: { taskId: '任务ID', state: '状态', created: '创建时间', result: '结果', action: '操作' },
      buttons: { start: '开始', cancel: '取消', delete: '删除' },
      confirmDelete: '确认删除该任务？'
    },
    resources: {
      title: '资源管理',
      buttons: { refresh: '刷新', delete: '删除', send: '发送', operation: '操作', submit: '提交' },
      columns: { 
        resourceId: '资源ID', 
        type: '类型', 
        nodeId: '节点', 
        status: '状态',
        action: '操作',
        name: '名称',
        dataType: '数据类型',
        defaultValue: '默认值',
        unit: '单位',
        range: '范围',
        time: '时间',
        stateData: '状态数据'
      },
      status: {
        healthy: '正常',
        warning: '警告',
        offline: '离线'
      },
      sections: {
        resourceList: '资源列表',
        resourceDetail: '资源详情',
        resourceDescription: '资源描述',
        stateDescription: '状态描述',
        operationDescription: '操作描述',
        historyStates: '历史状态（最近20条）'
      },
      desc: {
        resourceId: '资源ID',
        type: '类型',
        nodeId: '节点',
        status: '状态',
        createdAt: '创建时间',
        lastHeartbeat: '最后心跳'
      },
      messages: {
        selectResource: '请选择一个资源查看详情',
        deleteSuccess: '删除成功',
        deleteFailed: '删除失败',
        operationSent: '操作发送成功',
        operationFailed: '操作发送失败',
        loadFailed: '加载失败',
        stateLoadFailed: '加载状态失败'
      },
      dialogs: {
        operationTitle: '资源操作'
      },
      validation: {
        rangeError: '数值超出范围'
      },
      confirmDelete: '确认删除该资源？'
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


