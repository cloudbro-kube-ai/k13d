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
        th_actions: 'ACTIONS',

        // Login Page
        login_token_hint: 'Sign in with your Kubernetes ServiceAccount token',
        login_token_help_toggle: 'How to create a token',
        login_token_step1: 'Create ServiceAccount (optional):',
        login_token_step2: 'Grant permissions:',
        login_token_step3: 'Create token:',
        login_token_tip: 'Paste the token below and press Enter!',
        login_local_hint: 'Sign in with your local account',
        login_local_sub_hint: 'Ask your administrator for credentials'
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
        th_actions: '작업',

        // Login Page
        login_token_hint: 'Kubernetes ServiceAccount 토큰으로 로그인하세요',
        login_token_help_toggle: '토큰 생성 방법 보기',
        login_token_step1: 'ServiceAccount 생성 (선택사항):',
        login_token_step2: '권한 부여:',
        login_token_step3: '토큰 생성:',
        login_token_tip: '위 토큰을 아래에 붙여넣고 Enter를 누르세요!',
        login_local_hint: '로컬 계정으로 로그인하세요',
        login_local_sub_hint: '관리자에게 계정 정보를 문의하세요'
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

