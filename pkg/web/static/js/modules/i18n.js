// i18n Translations
const translations = {
    en: {
        // Navigation
        nav_pods: 'Pods',
        nav_deployments: 'Deployments',
        nav_daemonsets: 'DaemonSets',
        nav_statefulsets: 'StatefulSets',
        nav_replicasets: 'ReplicaSets',
        nav_jobs: 'Jobs',
        nav_cronjobs: 'CronJobs',
        nav_services: 'Services',
        nav_ingresses: 'Ingresses',
        nav_configmaps: 'ConfigMaps',
        nav_secrets: 'Secrets',
        nav_namespaces: 'Namespaces',
        nav_nodes: 'Nodes',
        nav_events: 'Events',
        nav_pvcs: 'PVCs',
        nav_pvs: 'PVs',
        nav_networkpolicies: 'NetworkPolicies',
        nav_serviceaccounts: 'ServiceAccounts',
        nav_roles: 'Roles',
        nav_rolebindings: 'RoleBindings',
        nav_clusterroles: 'ClusterRoles',
        nav_clusterrolebindings: 'ClusterRoleBindings',
        nav_overview: 'Overview',
        nav_topology: 'Topology',
        nav_applications: 'Applications',
        nav_rbac_viewer: 'RBAC Viewer',
        nav_netpol_map: 'Net Policy Map',
        nav_timeline: 'Event Timeline',
        nav_metrics: 'Metrics',
        nav_audit_logs: 'Audit Logs',
        nav_reports: 'Reports',

        // Section Headers
        header_rbac: 'RBAC',
        header_visualization: 'Visualization',
        header_monitoring: 'Monitoring',
        header_applications: 'Applications',
        header_rbac_viewer: 'RBAC Viewer',
        header_netpol_map: 'Network Policy Map',
        header_timeline: 'Event Timeline',

        // Application View
        msg_all_namespaces: 'All Namespaces',
        msg_loading_applications: 'Loading applications...',
        msg_no_apps: 'No applications found.',

        // Buttons
        btn_logs: 'Logs',
        btn_terminal: 'Terminal',
        btn_forward: 'Forward',
        btn_yaml: 'YAML',
        btn_describe: 'Describe',
        btn_analyze: 'Analyze',
        btn_delete: 'Delete',
        btn_scale: 'Scale',
        btn_restart: 'Restart',
        btn_refresh: 'Refresh',
        btn_save: 'Save',
        btn_cancel: 'Cancel',
        btn_close: 'Close',
        btn_approve: 'Approve',
        btn_reject: 'Reject',

        // Headers
        header_resources: 'Resources',
        header_workloads: 'Workloads',
        header_network: 'Network',
        header_config: 'Config',
        header_storage: 'Storage',
        header_cluster: 'Cluster',
        header_ai_assistant: 'AI Assistant',
        header_settings: 'Settings',
        header_audit_logs: 'Audit Logs',

        // Status
        status_running: 'Running',
        status_pending: 'Pending',
        status_failed: 'Failed',
        status_succeeded: 'Succeeded',
        status_unknown: 'Unknown',
        status_ready: 'Ready',
        status_not_ready: 'Not Ready',

        // Messages
        msg_loading: 'Loading...',
        msg_no_data: 'No data available',
        msg_error: 'Error',
        msg_success: 'Success',
        msg_confirm_delete: 'Are you sure you want to delete this resource?',
        msg_connection_test: 'Testing connection...',
        msg_connected: 'Connected',
        msg_disconnected: 'Disconnected',
        msg_settings_saved: 'Settings saved!',
        msg_enter_question: 'Please enter your question.',

        // AI
        ai_placeholder: 'Ask AI anything about your cluster...',
        ai_thinking: 'AI is thinking...',
        ai_approval_required: 'Approval Required',
        ai_command: 'Command',
        ai_hint: 'Enter send · Shift+Enter newline · ↑↓ history',

        // Settings
        settings_general: 'General',
        settings_llm: 'AI/LLM',
        settings_appearance: 'Appearance',
        settings_language: 'Language',
        settings_provider: 'Provider',
        settings_model: 'Model',
        settings_endpoint: 'Endpoint',
        settings_api_key: 'API Key',
        settings_test_connection: 'Test Connection',

        // Reports
        report_generate: 'Generate Report',
        report_preview: 'Preview',
        report_download: 'Download',
        report_include_ai: 'Include AI Analysis',

        // Table Headers
        th_name: 'NAME',
        th_namespace: 'NAMESPACE',
        th_status: 'STATUS',
        th_ready: 'READY',
        th_restarts: 'RESTARTS',
        th_age: 'AGE',
        th_node: 'NODE',
        th_ip: 'IP',
        th_type: 'TYPE',
        th_ports: 'PORTS',
        th_actions: 'ACTIONS'
    },
    ko: {
        // Navigation
        nav_pods: '파드',
        nav_deployments: '디플로이먼트',
        nav_daemonsets: '데몬셋',
        nav_statefulsets: '스테이트풀셋',
        nav_replicasets: '레플리카셋',
        nav_jobs: '잡',
        nav_cronjobs: '크론잡',
        nav_services: '서비스',
        nav_ingresses: '인그레스',
        nav_configmaps: '컨피그맵',
        nav_secrets: '시크릿',
        nav_namespaces: '네임스페이스',
        nav_nodes: '노드',
        nav_events: '이벤트',
        nav_pvcs: 'PVC',
        nav_pvs: 'PV',
        nav_networkpolicies: '네트워크 정책',
        nav_serviceaccounts: '서비스 어카운트',
        nav_roles: '롤',
        nav_rolebindings: '롤바인딩',
        nav_clusterroles: '클러스터롤',
        nav_clusterrolebindings: '클러스터롤바인딩',
        nav_overview: '개요',
        nav_topology: '토폴로지',
        nav_applications: '애플리케이션',
        nav_rbac_viewer: 'RBAC 뷰어',
        nav_netpol_map: '네트워크 정책 맵',
        nav_timeline: '이벤트 타임라인',
        nav_metrics: '메트릭',
        nav_audit_logs: '감사 로그',
        nav_reports: '리포트',

        // Section Headers
        header_rbac: 'RBAC',
        header_visualization: '시각화',
        header_monitoring: '모니터링',
        header_applications: '애플리케이션',
        header_rbac_viewer: 'RBAC 뷰어',
        header_netpol_map: '네트워크 정책 맵',
        header_timeline: '이벤트 타임라인',

        // Application View
        msg_all_namespaces: '전체 네임스페이스',
        msg_loading_applications: '애플리케이션 로딩 중...',
        msg_no_apps: '애플리케이션이 없습니다.',

        // Buttons
        btn_logs: '로그',
        btn_terminal: '터미널',
        btn_forward: '포워드',
        btn_yaml: 'YAML',
        btn_describe: '상세정보',
        btn_analyze: '분석',
        btn_delete: '삭제',
        btn_scale: '스케일',
        btn_restart: '재시작',
        btn_refresh: '새로고침',
        btn_save: '저장',
        btn_cancel: '취소',
        btn_close: '닫기',
        btn_approve: '승인',
        btn_reject: '거부',

        // Headers
        header_resources: '리소스',
        header_workloads: '워크로드',
        header_network: '네트워크',
        header_config: '설정',
        header_storage: '스토리지',
        header_cluster: '클러스터',
        header_ai_assistant: 'AI 어시스턴트',
        header_settings: '설정',
        header_audit_logs: '감사 로그',

        // Status
        status_running: '실행 중',
        status_pending: '대기 중',
        status_failed: '실패',
        status_succeeded: '성공',
        status_unknown: '알 수 없음',
        status_ready: '준비됨',
        status_not_ready: '준비 안됨',

        // Messages
        msg_loading: '로딩 중...',
        msg_no_data: '데이터가 없습니다',
        msg_error: '오류',
        msg_success: '성공',
        msg_confirm_delete: '이 리소스를 삭제하시겠습니까?',
        msg_connection_test: '연결 테스트 중...',
        msg_connected: '연결됨',
        msg_disconnected: '연결 끊김',
        msg_settings_saved: '설정이 저장되었습니다!',
        msg_enter_question: '질문을 입력해 주세요.',

        // AI
        ai_placeholder: '클러스터에 대해 AI에게 질문하세요...',
        ai_thinking: 'AI가 생각 중입니다...',
        ai_approval_required: '승인 필요',
        ai_command: '명령어',
        ai_hint: 'Enter 전송 · Shift+Enter 줄바꿈 · ↑↓ 히스토리',

        // Settings
        settings_general: '일반',
        settings_llm: 'AI/LLM',
        settings_appearance: '외관',
        settings_language: '언어',
        settings_provider: '제공자',
        settings_model: '모델',
        settings_endpoint: '엔드포인트',
        settings_api_key: 'API 키',
        settings_test_connection: '연결 테스트',

        // Reports
        report_generate: '리포트 생성',
        report_preview: '미리보기',
        report_download: '다운로드',
        report_include_ai: 'AI 분석 포함',

        // Table Headers
        th_name: '이름',
        th_namespace: '네임스페이스',
        th_status: '상태',
        th_ready: '준비',
        th_restarts: '재시작',
        th_age: '나이',
        th_node: '노드',
        th_ip: 'IP',
        th_type: '유형',
        th_ports: '포트',
        th_actions: '작업'
    },
    zh: {
        // Navigation
        nav_pods: 'Pods',
        nav_deployments: 'Deployments',
        nav_daemonsets: 'DaemonSets',
        nav_statefulsets: 'StatefulSets',
        nav_replicasets: 'ReplicaSets',
        nav_jobs: 'Jobs',
        nav_cronjobs: 'CronJobs',
        nav_services: '服务',
        nav_ingresses: '入口',
        nav_configmaps: '配置映射',
        nav_secrets: '密钥',
        nav_namespaces: '命名空间',
        nav_nodes: '节点',
        nav_events: '事件',
        nav_pvcs: 'PVC',
        nav_pvs: 'PV',
        nav_networkpolicies: '网络策略',
        nav_serviceaccounts: '服务账户',
        nav_roles: '角色',
        nav_rolebindings: '角色绑定',
        nav_clusterroles: '集群角色',
        nav_clusterrolebindings: '集群角色绑定',
        nav_overview: '概览',
        nav_topology: '拓扑',
        nav_applications: '应用',
        nav_rbac_viewer: 'RBAC 查看器',
        nav_netpol_map: '网络策略地图',
        nav_timeline: '事件时间线',
        nav_metrics: '指标',
        nav_audit_logs: '审计日志',
        nav_reports: '报告',
        header_rbac: 'RBAC',
        header_visualization: '可视化',
        header_monitoring: '监控',
        header_applications: '应用',
        header_rbac_viewer: 'RBAC 查看器',
        header_netpol_map: '网络策略地图',
        header_timeline: '事件时间线',
        msg_all_namespaces: '所有命名空间',
        msg_loading_applications: '加载应用中...',
        msg_no_apps: '未找到应用。',

        // Buttons
        btn_logs: '日志',
        btn_terminal: '终端',
        btn_forward: '转发',
        btn_yaml: 'YAML',
        btn_describe: '描述',
        btn_analyze: '分析',
        btn_delete: '删除',
        btn_scale: '扩缩',
        btn_restart: '重启',
        btn_refresh: '刷新',
        btn_save: '保存',
        btn_cancel: '取消',
        btn_close: '关闭',
        btn_approve: '批准',
        btn_reject: '拒绝',

        // Headers
        header_resources: '资源',
        header_workloads: '工作负载',
        header_network: '网络',
        header_config: '配置',
        header_storage: '存储',
        header_cluster: '集群',
        header_ai_assistant: 'AI 助手',
        header_settings: '设置',
        header_audit_logs: '审计日志',

        // Status
        status_running: '运行中',
        status_pending: '等待中',
        status_failed: '失败',
        status_succeeded: '成功',
        status_unknown: '未知',
        status_ready: '就绪',
        status_not_ready: '未就绪',

        // Messages
        msg_loading: '加载中...',
        msg_no_data: '暂无数据',
        msg_error: '错误',
        msg_success: '成功',
        msg_confirm_delete: '确定要删除此资源吗？',
        msg_connection_test: '测试连接中...',
        msg_connected: '已连接',
        msg_disconnected: '已断开',
        msg_settings_saved: '设置已保存！',
        msg_enter_question: '请输入您的问题。',

        // AI
        ai_placeholder: '向 AI 询问有关集群的任何问题...',
        ai_thinking: 'AI 正在思考...',
        ai_approval_required: '需要批准',
        ai_command: '命令',
        ai_hint: 'Enter 发送 · Shift+Enter 换行 · ↑↓ 历史',

        // Settings
        settings_general: '常规',
        settings_llm: 'AI/LLM',
        settings_appearance: '外观',
        settings_language: '语言',
        settings_provider: '提供商',
        settings_model: '模型',
        settings_endpoint: '端点',
        settings_api_key: 'API 密钥',
        settings_test_connection: '测试连接',

        // Reports
        report_generate: '生成报告',
        report_preview: '预览',
        report_download: '下载',
        report_include_ai: '包含 AI 分析',

        // Table Headers
        th_name: '名称',
        th_namespace: '命名空间',
        th_status: '状态',
        th_ready: '就绪',
        th_restarts: '重启',
        th_age: '时间',
        th_node: '节点',
        th_ip: 'IP',
        th_type: '类型',
        th_ports: '端口',
        th_actions: '操作'
    },
    ja: {
        // Navigation
        nav_pods: 'ポッド',
        nav_deployments: 'デプロイメント',
        nav_daemonsets: 'デーモンセット',
        nav_statefulsets: 'ステートフルセット',
        nav_replicasets: 'レプリカセット',
        nav_jobs: 'ジョブ',
        nav_cronjobs: 'クロンジョブ',
        nav_services: 'サービス',
        nav_ingresses: 'イングレス',
        nav_configmaps: 'コンフィグマップ',
        nav_secrets: 'シークレット',
        nav_namespaces: '名前空間',
        nav_nodes: 'ノード',
        nav_events: 'イベント',
        nav_pvcs: 'PVC',
        nav_pvs: 'PV',
        nav_networkpolicies: 'ネットワークポリシー',
        nav_serviceaccounts: 'サービスアカウント',
        nav_roles: 'ロール',
        nav_rolebindings: 'ロールバインディング',
        nav_clusterroles: 'クラスターロール',
        nav_clusterrolebindings: 'クラスターロールバインディング',
        nav_overview: '概要',
        nav_topology: 'トポロジー',
        nav_applications: 'アプリケーション',
        nav_rbac_viewer: 'RBAC ビューア',
        nav_netpol_map: 'ネットワークポリシーマップ',
        nav_timeline: 'イベントタイムライン',
        nav_metrics: 'メトリクス',
        nav_audit_logs: '監査ログ',
        nav_reports: 'レポート',
        header_rbac: 'RBAC',
        header_visualization: 'ビジュアライゼーション',
        header_monitoring: 'モニタリング',
        header_applications: 'アプリケーション',
        header_rbac_viewer: 'RBAC ビューア',
        header_netpol_map: 'ネットワークポリシーマップ',
        header_timeline: 'イベントタイムライン',
        msg_all_namespaces: '全ての名前空間',
        msg_loading_applications: 'アプリケーションを読み込み中...',
        msg_no_apps: 'アプリケーションが見つかりません。',

        // Buttons
        btn_logs: 'ログ',
        btn_terminal: 'ターミナル',
        btn_forward: '転送',
        btn_yaml: 'YAML',
        btn_describe: '詳細',
        btn_analyze: '分析',
        btn_delete: '削除',
        btn_scale: 'スケール',
        btn_restart: '再起動',
        btn_refresh: '更新',
        btn_save: '保存',
        btn_cancel: 'キャンセル',
        btn_close: '閉じる',
        btn_approve: '承認',
        btn_reject: '拒否',

        // Headers
        header_resources: 'リソース',
        header_workloads: 'ワークロード',
        header_network: 'ネットワーク',
        header_config: '設定',
        header_storage: 'ストレージ',
        header_cluster: 'クラスター',
        header_ai_assistant: 'AI アシスタント',
        header_settings: '設定',
        header_audit_logs: '監査ログ',

        // Status
        status_running: '実行中',
        status_pending: '保留中',
        status_failed: '失敗',
        status_succeeded: '成功',
        status_unknown: '不明',
        status_ready: '準備完了',
        status_not_ready: '準備未完',

        // Messages
        msg_loading: '読み込み中...',
        msg_no_data: 'データがありません',
        msg_error: 'エラー',
        msg_success: '成功',
        msg_confirm_delete: 'このリソースを削除しますか？',
        msg_connection_test: '接続をテスト中...',
        msg_connected: '接続済み',
        msg_disconnected: '切断',
        msg_settings_saved: '設定を保存しました！',
        msg_enter_question: '質問を入力してください。',

        // AI
        ai_placeholder: 'クラスターについてAIに質問...',
        ai_thinking: 'AIが考え中...',
        ai_approval_required: '承認が必要',
        ai_command: 'コマンド',
        ai_hint: 'Enter 送信 · Shift+Enter 改行 · ↑↓ 履歴',

        // Settings
        settings_general: '一般',
        settings_llm: 'AI/LLM',
        settings_appearance: '外観',
        settings_language: '言語',
        settings_provider: 'プロバイダー',
        settings_model: 'モデル',
        settings_endpoint: 'エンドポイント',
        settings_api_key: 'API キー',
        settings_test_connection: '接続テスト',

        // Reports
        report_generate: 'レポート生成',
        report_preview: 'プレビュー',
        report_download: 'ダウンロード',
        report_include_ai: 'AI 分析を含む',

        // Table Headers
        th_name: '名前',
        th_namespace: '名前空間',
        th_status: 'ステータス',
        th_ready: '準備',
        th_restarts: '再起動',
        th_age: '経過',
        th_node: 'ノード',
        th_ip: 'IP',
        th_type: 'タイプ',
        th_ports: 'ポート',
        th_actions: 'アクション'
    }
};

// i18n helper function
function t(key) {
    const lang = translations[currentLanguage] || translations['en'];
    return lang[key] || translations['en'][key] || key;
}

// Update UI language
function updateUILanguage() {
    // Update document lang attribute for accessibility
    document.documentElement.lang = currentLanguage;

    // Update AI placeholder
    const aiInput = document.getElementById('ai-input');
    if (aiInput) aiInput.placeholder = t('ai_placeholder');

    // Update sidebar navigation (dynamic elements need special handling)
    document.querySelectorAll('[data-i18n]').forEach(el => {
        const key = el.getAttribute('data-i18n');
        el.textContent = t(key);
    });
}

