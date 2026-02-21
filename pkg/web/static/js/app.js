/**
 * k13d Web UI Application
 * Main application JavaScript
 * 
 * Modules:
 *   - State & Config (global state, table headers, resources)
 *   - i18n (translations)
 *   - Core (init, auth, API, refresh, sorting, pagination)
 *   - Dashboard (table rendering, resource views, detail panels)
 *   - AI Chat (messaging, streaming, tool approval, guardrails)
 *   - Settings (settings modal, LLM config, Ollama, security, admin)
 *   - Topology (graph visualization)
 *   - Terminal (WebSocket terminal)
 *   - Log Viewer (pod logs)
 *   - Metrics (cluster metrics, charts)
 *   - YAML Editor
 *   - Search & Command Bar
 *   - Chat History
 *   - Reports
 *   - Port Forwarding
 */

        // State
        let currentResource = 'pods';
        let currentNamespace = '';
        let isLoading = false;
        let authToken = localStorage.getItem('k13d_token');
        let currentUser = null;
        let sidebarCollapsed = localStorage.getItem('k13d_sidebar_collapsed') === 'true';
        let debugMode = localStorage.getItem('k13d_debug_mode') === 'true';
        let aiContextItems = []; // Resources added as context for AI
        let currentLanguage = 'ko'; // Default language (Korean)
        let currentLLMModel = ''; // Current LLM model name
        let llmConnected = false; // LLM connection status
        let currentSessionId = sessionStorage.getItem('k13d_session_id') || ''; // AI conversation session ID
        let appTimezone = localStorage.getItem('k13d_timezone') || 'auto'; // Timezone setting

        // Timezone formatting helpers
        function getTimezoneOptions() {
            if (appTimezone === 'auto' || !appTimezone) return {};
            return { timeZone: appTimezone };
        }

        function formatTime(isoString) {
            const date = new Date(isoString);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', ...getTimezoneOptions() });
        }

        function formatDateTime(isoString) {
            const date = new Date(isoString);
            return date.toLocaleString([], getTimezoneOptions());
        }

        function formatTimeShort(date) {
            if (typeof date === 'string') date = new Date(date);
            return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', ...getTimezoneOptions() });
        }

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

                // AI
                ai_placeholder: 'Ask AI anything about your cluster...',
                ai_thinking: 'AI is thinking...',
                ai_approval_required: 'Approval Required',
                ai_command: 'Command',

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

                // AI
                ai_placeholder: '클러스터에 대해 AI에게 질문하세요...',
                ai_thinking: 'AI가 생각 중입니다...',
                ai_approval_required: '승인 필요',
                ai_command: '명령어',

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

                // AI
                ai_placeholder: '向 AI 询问有关集群的任何问题...',
                ai_thinking: 'AI 正在思考...',
                ai_approval_required: '需要批准',
                ai_command: '命令',

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

                // AI
                ai_placeholder: 'クラスターについてAIに質問...',
                ai_thinking: 'AIが考え中...',
                ai_approval_required: '承認が必要',
                ai_command: 'コマンド',

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
            // Update AI placeholder
            const aiInput = document.getElementById('ai-input');
            if (aiInput) aiInput.placeholder = t('ai_placeholder');

            // Update sidebar navigation (dynamic elements need special handling)
            document.querySelectorAll('[data-i18n]').forEach(el => {
                const key = el.getAttribute('data-i18n');
                el.textContent = t(key);
            });
        }

        // Auto-refresh settings (default to enabled with 30s interval)
        let autoRefreshEnabled = localStorage.getItem('k13d_auto_refresh') !== 'false'; // default true
        let autoRefreshInterval = parseInt(localStorage.getItem('k13d_refresh_interval')) || 30; // seconds
        let autoRefreshTimer = null;

        // SSE streaming settings
        let useStreaming = localStorage.getItem('k13d_use_streaming') !== 'false'; // default true
        let currentEventSource = null;

        // Reasoning effort setting (for Solar Pro2)
        let reasoningEffort = localStorage.getItem('k13d_reasoning_effort') || 'minimal'; // default minimal

        // Table headers for all resource types
        const tableHeaders = {
            pods: ['NAME', 'NAMESPACE', 'READY', 'STATUS', 'RESTARTS', 'AGE', 'IP'],
            deployments: ['NAME', 'NAMESPACE', 'READY', 'UP-TO-DATE', 'AVAILABLE', 'AGE'],
            daemonsets: ['NAME', 'NAMESPACE', 'DESIRED', 'CURRENT', 'READY', 'AGE'],
            statefulsets: ['NAME', 'NAMESPACE', 'READY', 'AGE'],
            replicasets: ['NAME', 'NAMESPACE', 'DESIRED', 'CURRENT', 'READY', 'AGE'],
            jobs: ['NAME', 'NAMESPACE', 'COMPLETIONS', 'DURATION', 'AGE'],
            cronjobs: ['NAME', 'NAMESPACE', 'SCHEDULE', 'SUSPEND', 'ACTIVE', 'LAST SCHEDULE'],
            services: ['NAME', 'NAMESPACE', 'TYPE', 'CLUSTER-IP', 'PORTS', 'AGE'],
            ingresses: ['NAME', 'NAMESPACE', 'CLASS', 'HOSTS', 'ADDRESS', 'AGE'],
            networkpolicies: ['NAME', 'NAMESPACE', 'POD-SELECTOR', 'AGE'],
            configmaps: ['NAME', 'NAMESPACE', 'DATA', 'AGE'],
            secrets: ['NAME', 'NAMESPACE', 'TYPE', 'DATA', 'AGE'],
            serviceaccounts: ['NAME', 'NAMESPACE', 'SECRETS', 'AGE'],
            persistentvolumes: ['NAME', 'CAPACITY', 'ACCESS MODES', 'RECLAIM POLICY', 'STATUS', 'CLAIM'],
            persistentvolumeclaims: ['NAME', 'NAMESPACE', 'STATUS', 'VOLUME', 'CAPACITY', 'ACCESS MODES'],
            nodes: ['NAME', 'STATUS', 'ROLES', 'VERSION', 'AGE'],
            namespaces: ['NAME', 'STATUS', 'AGE'],
            events: ['NAME', 'TYPE', 'REASON', 'MESSAGE', 'COUNT', 'LAST SEEN'],
            roles: ['NAME', 'NAMESPACE', 'AGE'],
            rolebindings: ['NAME', 'NAMESPACE', 'ROLE', 'AGE'],
            clusterroles: ['NAME', 'AGE'],
            clusterrolebindings: ['NAME', 'ROLE', 'AGE']
        };

        // All supported resource types
        const allResources = [
            'pods', 'deployments', 'daemonsets', 'statefulsets', 'replicasets', 'jobs', 'cronjobs',
            'services', 'ingresses', 'networkpolicies',
            'configmaps', 'secrets', 'serviceaccounts',
            'persistentvolumes', 'persistentvolumeclaims',
            'nodes', 'namespaces', 'events',
            'roles', 'rolebindings', 'clusterroles', 'clusterrolebindings'
        ];

        // Cluster-scoped resources (no namespace)
        const clusterScopedResources = ['nodes', 'namespaces', 'persistentvolumes', 'clusterroles', 'clusterrolebindings'];

        // Custom Resource state
        let loadedCRDs = []; // List of CRDs with their info
        let currentCRD = null; // Currently selected CRD (for viewing instances)

        // Sorting and Pagination State
        let sortColumn = null;
        let sortDirection = 'asc'; // 'asc' or 'desc'
        let currentPage = 1;
        let pageSize = 50;
        let allItems = []; // All items before pagination
        let filteredItems = []; // Items after filtering

        // Column filter state
        let columnFiltersVisible = false;
        let columnFilters = {}; // { 'NAME': 'nginx', 'STATUS': 'Running' }

        // Field mapping for sorting (header name -> item property)
        const fieldMapping = {
            'NAME': 'name',
            'NAMESPACE': 'namespace',
            'READY': 'ready',
            'STATUS': 'status',
            'RESTARTS': 'restarts',
            'AGE': 'age',
            'IP': 'ip',
            'UP-TO-DATE': 'upToDate',
            'AVAILABLE': 'available',
            'DESIRED': 'desired',
            'CURRENT': 'current',
            'COMPLETIONS': 'completions',
            'DURATION': 'duration',
            'SCHEDULE': 'schedule',
            'SUSPEND': 'suspend',
            'ACTIVE': 'active',
            'LAST SCHEDULE': 'lastSchedule',
            'TYPE': 'type',
            'CLUSTER-IP': 'clusterIP',
            'PORTS': 'ports',
            'CLASS': 'class',
            'HOSTS': 'hosts',
            'ADDRESS': 'address',
            'POD-SELECTOR': 'podSelector',
            'DATA': 'data',
            'SECRETS': 'secrets',
            'CAPACITY': 'capacity',
            'ACCESS MODES': 'accessModes',
            'RECLAIM POLICY': 'reclaimPolicy',
            'CLAIM': 'claim',
            'VOLUME': 'volume',
            'ROLES': 'roles',
            'VERSION': 'version',
            'REASON': 'reason',
            'MESSAGE': 'message',
            'COUNT': 'count',
            'LAST SEEN': 'lastSeen',
            'ROLE': 'role'
        };

        // Sort items by column
        function sortItems(items, column, direction) {
            const field = fieldMapping[column] || column.toLowerCase().replace(/[- ]/g, '');
            return [...items].sort((a, b) => {
                let valA = a[field];
                let valB = b[field];

                // Handle age sorting (convert to comparable values)
                if (column === 'AGE' || column === 'LAST SEEN' || column === 'DURATION') {
                    valA = parseAgeToSeconds(valA);
                    valB = parseAgeToSeconds(valB);
                }
                // Handle numeric fields
                else if (column === 'RESTARTS' || column === 'COUNT' || column === 'DESIRED' ||
                         column === 'CURRENT' || column === 'AVAILABLE' || column === 'ACTIVE' ||
                         column === 'DATA' || column === 'SECRETS') {
                    valA = parseInt(valA) || 0;
                    valB = parseInt(valB) || 0;
                }
                // Handle ready format (e.g., "1/1")
                else if (column === 'READY' || column === 'COMPLETIONS') {
                    valA = parseReadyValue(valA);
                    valB = parseReadyValue(valB);
                }
                // Handle strings (case-insensitive)
                else {
                    valA = (valA || '').toString().toLowerCase();
                    valB = (valB || '').toString().toLowerCase();
                }

                if (valA < valB) return direction === 'asc' ? -1 : 1;
                if (valA > valB) return direction === 'asc' ? 1 : -1;
                return 0;
            });
        }

        // Parse age string to seconds for sorting
        function parseAgeToSeconds(age) {
            if (!age || age === '-') return 0;
            const str = age.toString();
            const match = str.match(/(\d+)([smhd])/);
            if (!match) return 0;
            const value = parseInt(match[1]);
            const unit = match[2];
            switch (unit) {
                case 's': return value;
                case 'm': return value * 60;
                case 'h': return value * 3600;
                case 'd': return value * 86400;
                default: return value;
            }
        }

        // Parse ready value (e.g., "1/1" -> 1)
        function parseReadyValue(ready) {
            if (!ready || ready === '-') return 0;
            const parts = ready.toString().split('/');
            return parseInt(parts[0]) || 0;
        }

        // Handle column header click for sorting
        function onColumnSort(column, headerElement) {
            // Toggle direction if same column, otherwise default to asc
            if (sortColumn === column) {
                sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
            } else {
                sortColumn = column;
                sortDirection = 'asc';
            }

            // Update header styling
            document.querySelectorAll('#table-header th').forEach(th => {
                th.classList.remove('sort-asc', 'sort-desc');
            });
            headerElement.classList.add(sortDirection === 'asc' ? 'sort-asc' : 'sort-desc');

            // Re-render with sorted data
            currentPage = 1;
            applyFilterAndSort();
        }

        // Apply filter and sort to items
        function applyFilterAndSort() {
            const filterText = document.getElementById('filter-input').value.toLowerCase();

            // Filter items by global filter
            filteredItems = allItems.filter(item => {
                if (!filterText) return true;
                return Object.values(item).some(val =>
                    val && val.toString().toLowerCase().includes(filterText)
                );
            });

            // Apply column-specific filters
            const activeColumnFilters = Object.entries(columnFilters).filter(([_, v]) => v && v.trim());
            if (activeColumnFilters.length > 0) {
                filteredItems = filteredItems.filter(item => {
                    return activeColumnFilters.every(([column, filterVal]) => {
                        const field = fieldMapping[column] || column.toLowerCase().replace(/[- ]/g, '');
                        const itemValue = item[field];
                        if (itemValue === undefined || itemValue === null) return false;
                        return itemValue.toString().toLowerCase().includes(filterVal.toLowerCase());
                    });
                });
            }

            // Sort items
            if (sortColumn) {
                filteredItems = sortItems(filteredItems, sortColumn, sortDirection);
            }

            // Render current page
            renderCurrentPage();

            // Update active column filters display
            updateActiveColumnFiltersDisplay();
        }

        // Toggle column filters visibility
        function toggleColumnFilters() {
            columnFiltersVisible = !columnFiltersVisible;
            const filterRow = document.getElementById('column-filter-row');
            const toggleBtn = document.getElementById('column-filter-toggle');

            if (filterRow) {
                filterRow.classList.toggle('active', columnFiltersVisible);
            }
            if (toggleBtn) {
                toggleBtn.classList.toggle('active', columnFiltersVisible);
            }

            // Focus first filter input when showing
            if (columnFiltersVisible && filterRow) {
                const firstInput = filterRow.querySelector('.column-filter-input');
                if (firstInput) {
                    setTimeout(() => firstInput.focus(), 50);
                }
            }
        }

        // Handle column filter input change
        function onColumnFilterChange(event, column) {
            const value = event.target.value;
            columnFilters[column] = value;

            // Debounce the filter application
            clearTimeout(window.columnFilterTimeout);
            window.columnFilterTimeout = setTimeout(() => {
                currentPage = 1;
                applyFilterAndSort();
            }, 200);
        }

        // Update the active column filters chips display
        function updateActiveColumnFiltersDisplay() {
            const container = document.getElementById('active-column-filters');
            if (!container) return;

            const activeFilters = Object.entries(columnFilters).filter(([_, v]) => v && v.trim());

            if (activeFilters.length === 0) {
                container.innerHTML = '';
                return;
            }

            container.innerHTML = activeFilters.map(([col, val]) =>
                `<span class="column-filter-chip">
                    <span class="col-name">${col}:</span>
                    <span>${val}</span>
                    <span class="remove-col-filter" onclick="clearColumnFilter('${col}')">&times;</span>
                </span>`
            ).join('');
        }

        // Clear a specific column filter
        function clearColumnFilter(column) {
            delete columnFilters[column];

            // Update the input field if visible
            const input = document.querySelector(`.column-filter-input[data-column="${column}"]`);
            if (input) {
                input.value = '';
            }

            currentPage = 1;
            applyFilterAndSort();
        }

        // Clear all column filters
        function clearAllColumnFilters() {
            columnFilters = {};

            // Clear all input fields
            document.querySelectorAll('.column-filter-input').forEach(input => {
                input.value = '';
            });

            currentPage = 1;
            applyFilterAndSort();
        }

        // Render current page of items
        function renderCurrentPage() {
            const totalItems = filteredItems.length;
            const totalPages = pageSize === -1 ? 1 : Math.ceil(totalItems / pageSize);
            currentPage = Math.min(currentPage, Math.max(1, totalPages));

            // Get items for current page
            let pageItems;
            if (pageSize === -1) {
                pageItems = filteredItems;
            } else {
                const startIdx = (currentPage - 1) * pageSize;
                const endIdx = startIdx + pageSize;
                pageItems = filteredItems.slice(startIdx, endIdx);
            }

            // Render table body
            renderTableBody(currentResource, pageItems);

            // Update pagination info
            updatePaginationUI(totalItems, totalPages);
        }

        // Update pagination UI
        function updatePaginationUI(totalItems, totalPages) {
            const startItem = totalItems === 0 ? 0 : (currentPage - 1) * (pageSize === -1 ? totalItems : pageSize) + 1;
            const endItem = pageSize === -1 ? totalItems : Math.min(currentPage * pageSize, totalItems);

            document.getElementById('pagination-info').textContent =
                `Showing ${startItem}-${endItem} of ${totalItems} items`;
            document.getElementById('page-indicator').textContent = `${currentPage} / ${totalPages || 1}`;

            document.getElementById('prev-page-btn').disabled = currentPage <= 1;
            document.getElementById('next-page-btn').disabled = currentPage >= totalPages;
        }

        // Pagination controls
        function goToNextPage() {
            currentPage++;
            renderCurrentPage();
        }

        function goToPrevPage() {
            currentPage--;
            renderCurrentPage();
        }

        function onPageSizeChange() {
            pageSize = parseInt(document.getElementById('page-size-select').value);
            currentPage = 1;
            renderCurrentPage();
        }

        // Theme toggle (dark/light)
        function initTheme() {
            const saved = localStorage.getItem('k13d_theme') || 'tokyo-night';
            if (saved === 'light') {
                document.documentElement.removeAttribute('data-theme');
            } else {
                document.documentElement.setAttribute('data-theme', saved);
            }
            updateThemeIcon();
        }

        function toggleTheme() {
            const current = document.documentElement.getAttribute('data-theme');
            if (!current || current === 'light') {
                // Switch to Tokyo Night
                applyTheme('tokyo-night');
            } else {
                // Switch to Light
                applyTheme('light');
            }
        }

        function updateThemeIcon() {
            const btn = document.getElementById('theme-toggle');
            if (!btn) return;
            const theme = document.documentElement.getAttribute('data-theme');
            const isLight = !theme || theme === 'light';
            btn.textContent = isLight ? '☀️' : '🌙';
            btn.title = isLight ? 'Switch to dark theme' : 'Switch to light theme';
        }

        // Apply theme immediately (before DOM ready)
        initTheme();

        // Initialize
        async function init() {
            if (authToken) {
                try {
                    const health = await fetch('/api/health').then(r => r.json());
                    if (health.auth_enabled) {
                        const user = await fetchWithAuth('/api/auth/me').then(r => r.json());
                        currentUser = user;
                        showApp();
                    } else {
                        showApp();
                    }
                } catch (e) {
                    showLogin();
                }
            } else {
                // Check if auth is enabled
                const health = await fetch('/api/health').then(r => r.json());
                if (!health.auth_enabled) {
                    authToken = 'anonymous';
                    showApp();
                } else {
                    showLogin();
                }
            }
        }

        async function showLogin() {
            document.getElementById('login-page').style.display = 'flex';
            document.getElementById('app').classList.remove('active');

            // Fetch auth status to determine environment and auth mode
            try {
                const status = await fetch('/api/auth/status').then(r => r.json());
                updateLoginPageForAuthMode(status);
            } catch (e) {
                console.error('Failed to fetch auth status:', e);
                // Default to showing token form
                document.getElementById('token-login-form').style.display = 'block';
                document.getElementById('password-login-form').style.display = 'none';
            }
        }

        // Update login page UI based on auth mode (token vs local)
        function updateLoginPageForAuthMode(status) {
            const authModeEl = document.getElementById('auth-mode-indicator');
            const tokenForm = document.getElementById('token-login-form');
            const passwordForm = document.getElementById('password-login-form');

            const authMode = status.auth_mode || status.mode || 'token';

            if (authMode === 'token') {
                // Token authentication mode - show token form only
                authModeEl.className = 'auth-mode-indicator token-mode';
                authModeEl.innerHTML = '🔐 Kubernetes Token 인증 모드';
                tokenForm.style.display = 'block';
                passwordForm.style.display = 'none';

                // Focus on token input
                setTimeout(() => {
                    document.getElementById('login-token').focus();
                }, 100);
            } else if (authMode === 'local') {
                // Local authentication mode - show password form only
                authModeEl.className = 'auth-mode-indicator local-mode';
                authModeEl.innerHTML = '👤 로컬 계정 인증 모드';
                tokenForm.style.display = 'none';
                passwordForm.style.display = 'block';

                // Focus on username input
                setTimeout(() => {
                    document.getElementById('login-username').focus();
                }, 100);
            } else {
                // Default or mixed mode - show token form
                authModeEl.style.display = 'none';
                tokenForm.style.display = 'block';
                passwordForm.style.display = 'none';
            }
        }

        // Handle Enter key in token textarea
        function handleTokenKeydown(event) {
            if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault();
                loginWithToken();
            }
        }

        // Handle Enter key in password form
        function handlePasswordKeydown(event) {
            if (event.key === 'Enter') {
                event.preventDefault();
                login();
            }
        }

        // Toggle token help dropdown
        function toggleTokenHelp() {
            const box = document.getElementById('token-help-box');
            if (box) {
                box.classList.toggle('expanded');
            }
        }

        // Login with kubeconfig credentials (local mode only)
        async function loginWithKubeconfig() {
            const errorEl = document.getElementById('login-error');
            errorEl.textContent = '';

            try {
                const resp = await fetch('/api/auth/kubeconfig', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' }
                });

                const data = await resp.json();
                if (resp.ok) {
                    authToken = data.token;
                    localStorage.setItem('k13d_token', authToken);
                    currentUser = { username: data.username, role: data.role };
                    showApp();
                } else {
                    errorEl.textContent = data.error || 'Kubeconfig login failed';
                }
            } catch (e) {
                errorEl.textContent = 'Login failed: ' + e.message;
            }
        }

        function showApp() {
            document.getElementById('login-page').style.display = 'none';
            document.getElementById('app').classList.add('active');
            if (currentUser) {
                document.getElementById('user-badge').textContent = currentUser.username;
            } else if (authToken === 'anonymous') {
                document.getElementById('user-badge').textContent = 'anonymous';
                // Hide logout button when auth is disabled
                document.getElementById('logout-btn').style.display = 'none';
            }
            // Restore sidebar state
            if (sidebarCollapsed) {
                document.getElementById('sidebar').classList.add('collapsed');
                document.getElementById('hamburger-btn').classList.add('active');
            }
            // Restore debug mode
            if (debugMode) {
                document.getElementById('debug-panel').classList.add('active');
                document.getElementById('debug-toggle').style.background = 'var(--accent-purple)';
            }
            loadNamespaces();
            switchResource('pods');
            setupResizeHandle();
            setupHealthCheck();
            // Initialize auto-refresh
            updateAutoRefreshUI();
            updateLastRefreshTime();
            if (autoRefreshEnabled) {
                startAutoRefresh();
            }
            // Initialize AI status (model name and connection status)
            updateAIStatus();
            // Load user permissions for feature gating
            loadUserPermissions();
        }

        // Login tab switching
        function switchLoginTab(tab) {
            document.querySelectorAll('.login-tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.login-form').forEach(f => f.classList.remove('active'));

            if (tab === 'token') {
                document.querySelector('.login-tab:first-child').classList.add('active');
                document.getElementById('token-login-form').classList.add('active');
            } else {
                document.querySelector('.login-tab:last-child').classList.add('active');
                document.getElementById('password-login-form').classList.add('active');
            }
        }

        // Token-based login (K8s RBAC)
        async function loginWithToken() {
            const token = document.getElementById('login-token').value.trim();
            if (!token) {
                document.getElementById('login-error').textContent = 'Please enter a token';
                return;
            }

            try {
                const resp = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ token })
                });

                const data = await resp.json();
                if (resp.ok) {
                    authToken = data.token;
                    localStorage.setItem('k13d_token', authToken);
                    currentUser = { username: data.username, role: data.role };
                    showApp();
                } else {
                    document.getElementById('login-error').textContent = data.error || 'Invalid token';
                }
            } catch (e) {
                document.getElementById('login-error').textContent = 'Login failed: ' + e.message;
            }
        }

        // Username/password login
        async function login() {
            const username = document.getElementById('login-username').value;
            const password = document.getElementById('login-password').value;

            try {
                const resp = await fetch('/api/auth/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, password })
                });

                if (resp.ok) {
                    const data = await resp.json();
                    authToken = data.token;
                    localStorage.setItem('k13d_token', authToken);
                    currentUser = { username: data.username, role: data.role };
                    showApp();
                } else {
                    document.getElementById('login-error').textContent = 'Invalid credentials';
                }
            } catch (e) {
                document.getElementById('login-error').textContent = 'Login failed';
            }
        }

        async function logout() {
            try {
                // Send logout request with credentials (cookies) and auth header
                await fetch('/api/auth/logout', {
                    method: 'POST',
                    credentials: 'include',
                    headers: authToken ? { 'Authorization': `Bearer ${authToken}` } : {}
                });
            } catch (e) {
                console.error('Logout request failed:', e);
            }
            // Clear local storage and state regardless of server response
            localStorage.removeItem('k13d_token');
            localStorage.removeItem('k13d_auto_refresh');
            localStorage.removeItem('k13d_refresh_interval');
            authToken = null;
            currentUser = null;
            // Stop auto-refresh timer
            if (autoRefreshTimer) {
                clearInterval(autoRefreshTimer);
                autoRefreshTimer = null;
            }
            location.reload();
        }

        async function fetchWithAuth(url, options = {}) {
            const headers = { ...options.headers };
            if (authToken && authToken !== 'anonymous') {
                headers['Authorization'] = `Bearer ${authToken}`;
            }
            return fetch(url, { ...options, headers });
        }

        async function loadNamespaces() {
            try {
                const resp = await fetchWithAuth('/api/k8s/namespaces');
                const data = await resp.json();
                const select = document.getElementById('namespace-select');
                select.innerHTML = '<option value="">All Namespaces</option>';
                if (data.items) {
                    data.items.forEach(ns => {
                        const option = document.createElement('option');
                        option.value = ns.name;
                        option.textContent = ns.name;
                        select.appendChild(option);
                    });
                }
            } catch (e) {
                console.error('Failed to load namespaces:', e);
            }
        }

        async function loadData() {
            for (const resource of allResources) {
                try {
                    const isClusterScoped = clusterScopedResources.includes(resource);
                    const ns = isClusterScoped ? '' : currentNamespace;
                    const url = ns ? `/api/k8s/${resource}?namespace=${ns}` : `/api/k8s/${resource}`;
                    const resp = await fetchWithAuth(url);

                    if (!resp.ok) {
                        console.error(`API error for ${resource}: ${resp.status}`);
                        continue;
                    }

                    const data = await resp.json();

                    // Check for API error in response
                    if (data.error) {
                        console.error(`API returned error for ${resource}:`, data.error);
                        continue;
                    }

                    const countEl = document.getElementById(`${resource}-count`);
                    if (countEl) {
                        countEl.textContent = data.items ? data.items.length : 0;
                    }
                    if (resource === currentResource) {
                        renderTable(resource, data.items || []);
                    }
                } catch (e) {
                    console.error(`Failed to load ${resource}:`, e);
                }
            }

            // Also load CRDs
            loadCRDs();
        }

        // Load Custom Resource Definitions
        async function loadCRDs() {
            try {
                const resp = await fetchWithAuth('/api/crd/');
                const data = await resp.json();

                // Check for server error response
                if (data.error) {
                    console.error('CRD API error:', data.error);
                    document.getElementById('crd-count').textContent = '-';
                    // Show user-friendly message for common permission error
                    const errorMsg = data.error.includes('forbidden') || data.error.includes('Forbidden')
                        ? 'No permission'
                        : 'Error loading';
                    document.getElementById('crd-nav-items').innerHTML = `<div style="font-size: 11px; color: var(--accent-yellow); padding: 4px 8px;" title="${escapeHtml(data.error)}">${errorMsg}</div>`;
                    return;
                }

                if (data.items && data.items.length > 0) {
                    loadedCRDs = data.items;
                    document.getElementById('crd-count').textContent = data.items.length;

                    // Group CRDs by group for better organization
                    const grouped = {};
                    data.items.forEach(crd => {
                        const group = crd.group || 'core';
                        if (!grouped[group]) grouped[group] = [];
                        grouped[group].push(crd);
                    });

                    // Render CRD nav items (limited to first 10 for performance)
                    const container = document.getElementById('crd-nav-items');
                    const sortedGroups = Object.keys(grouped).sort();
                    let html = '';
                    let count = 0;

                    for (const group of sortedGroups) {
                        for (const crd of grouped[group]) {
                            if (count >= 15) break; // Limit to 15 items
                            const shortGroup = group.split('.')[0];
                            html += `<div class="nav-item" data-crd="${crd.name}" onclick="switchToCRD('${crd.name}')" title="${crd.name}">
                                <span style="font-size: 11px;">${crd.kind}</span>
                                <span class="count" style="font-size: 9px; opacity: 0.7;">${shortGroup}</span>
                            </div>`;
                            count++;
                        }
                        if (count >= 15) break;
                    }

                    if (data.items.length > 15) {
                        html += `<div class="nav-item" onclick="showAllCRDs()" style="font-style: italic; opacity: 0.8;">
                            <span>View all ${data.items.length} CRDs...</span>
                        </div>`;
                    }

                    container.innerHTML = html;
                } else {
                    document.getElementById('crd-count').textContent = '0';
                    document.getElementById('crd-nav-items').innerHTML = '<div style="font-size: 11px; color: var(--text-secondary); padding: 4px 8px;">No CRDs found</div>';
                }
            } catch (e) {
                console.error('Failed to load CRDs:', e);
                document.getElementById('crd-nav-items').innerHTML = '<div style="font-size: 11px; color: var(--accent-red); padding: 4px 8px;">Failed to load</div>';
            }
        }

        // Switch to viewing a Custom Resource's instances
        async function switchToCRD(crdName) {
            closeMobileSidebar();
            currentCRD = loadedCRDs.find(c => c.name === crdName);
            if (!currentCRD) return;

            currentResource = `crd:${crdName}`;

            // Update active nav item
            document.querySelectorAll('.nav-item').forEach(item => {
                item.classList.remove('active');
            });
            document.querySelector(`[data-crd="${crdName}"]`)?.classList.add('active');

            // Update panel title
            document.getElementById('panel-title').textContent = `${currentCRD.kind} (${currentCRD.group})`;
            document.getElementById('resource-summary').innerHTML = '';

            // Clear filters
            columnFilters = {};
            sortColumn = null;
            sortDirection = 'asc';
            updateActiveColumnFiltersDisplay();

            // Load instances
            await loadCRDInstances(currentCRD);
        }

        // Load instances of a Custom Resource
        async function loadCRDInstances(crdInfo) {
            try {
                // For namespaced resources, use current namespace (empty = all namespaces)
                const ns = crdInfo.namespaced ? currentNamespace : '';
                const url = ns ? `/api/crd/${crdInfo.name}/instances?namespace=${encodeURIComponent(ns)}` : `/api/crd/${crdInfo.name}/instances`;

                console.log(`Loading CR instances: ${url} (namespaced: ${crdInfo.namespaced}, ns: "${ns}")`);

                const resp = await fetchWithAuth(url);

                if (!resp.ok) {
                    const errorText = await resp.text();
                    throw new Error(`HTTP ${resp.status}: ${errorText}`);
                }

                const data = await resp.json();
                console.log(`CR instances response:`, data);

                // Check for API error in response
                if (data.error) {
                    throw new Error(data.error);
                }

                // Build dynamic headers from printerColumns
                const printerCols = data.printerColumns || crdInfo.printerColumns || [];
                const extraColNames = printerCols
                    .filter(c => {
                        const key = c.name.toLowerCase();
                        return key !== 'age' && key !== 'name' && key !== 'namespace' && (c.priority || 0) === 0;
                    })
                    .map(c => c.name.toUpperCase());

                let headers;
                if (crdInfo.namespaced) {
                    headers = ['NAME', 'NAMESPACE', ...extraColNames, 'STATUS', 'AGE'];
                } else {
                    headers = ['NAME', ...extraColNames, 'STATUS', 'AGE'];
                }

                // Store printer column info for renderTableBody
                crdInfo._extraColumns = extraColNames;

                // Store headers dynamically
                tableHeaders[`crd:${crdInfo.name}`] = headers;

                // Render table
                allItems = data.items || [];
                filteredItems = [...allItems];
                currentPage = 1;

                // Update summary for CRD instances
                const summaryEl = document.getElementById('resource-summary');
                if (summaryEl) {
                    summaryEl.innerHTML = `<span class="summary-item"><span class="summary-count">${allItems.length}</span> instances</span>`;
                }

                // Render headers with filter row
                const headerRow = `<tr>${headers.map(h => {
                    const sortClass = sortColumn === h ? (sortDirection === 'asc' ? 'sort-asc' : 'sort-desc') : '';
                    return `<th class="${sortClass}" onclick="onColumnSort('${h}', this)">${h}<span class="sort-icon"></span></th>`;
                }).join('')}</tr>`;

                const filterRow = `<tr class="column-filter-row ${columnFiltersVisible ? 'active' : ''}" id="column-filter-row">
                    ${headers.map(h => {
                        const filterValue = columnFilters[h] || '';
                        return `<th><input type="text" class="column-filter-input" placeholder="Filter ${h.toLowerCase()}..."
                            value="${filterValue}"
                            data-column="${h}"
                            onkeyup="onColumnFilterChange(event, '${h}')"
                            onclick="event.stopPropagation()"></th>`;
                    }).join('')}
                </tr>`;

                document.getElementById('table-header').innerHTML = headerRow + filterRow;

                if (!data.items || data.items.length === 0) {
                    const nsInfo = crdInfo.namespaced ? (ns ? ` in namespace "${ns}"` : ' (all namespaces)') : '';
                    document.getElementById('table-body').innerHTML =
                        `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">
                            <div style="color:var(--text-secondary);">No ${crdInfo.kind} instances found${nsInfo}</div>
                            <div style="font-size:11px;color:var(--text-secondary);margin-top:8px;">
                                CRD: ${crdInfo.group}/${crdInfo.version}
                            </div>
                        </td></tr>`;
                    updatePaginationUI(0, 0);
                    return;
                }

                applyFilterAndSort();
            } catch (e) {
                console.error('Failed to load CR instances:', e);
                const headers = crdInfo.namespaced ? ['NAME', 'NAMESPACE', 'STATUS', 'AGE'] : ['NAME', 'STATUS', 'AGE'];
                document.getElementById('table-body').innerHTML =
                    `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">
                        <div style="color:var(--accent-red);">Failed to load ${crdInfo.kind} instances</div>
                        <div style="font-size:11px;color:var(--text-secondary);margin-top:8px;">${escapeHtml(e.message)}</div>
                    </td></tr>`;
            }
        }

        // Show all CRDs in a modal
        function showAllCRDs() {
            let html = `
                <div class="modal-overlay" onclick="closeModal(event)">
                    <div class="modal detail-modal" style="max-width: 800px;" onclick="event.stopPropagation()">
                        <div class="modal-header">
                            <h3>All Custom Resource Definitions (${loadedCRDs.length})</h3>
                            <button class="modal-close" onclick="closeAllModals()">&times;</button>
                        </div>
                        <div class="modal-body" style="max-height: 70vh; overflow-y: auto;">
                            <input type="text" id="crd-search" placeholder="Search CRDs..." style="width: 100%; padding: 8px; margin-bottom: 12px; background: var(--bg-primary); border: 1px solid var(--border-color); border-radius: 4px; color: var(--text-primary);" oninput="filterCRDList(this.value)">
                            <div id="crd-list-container">
                                ${renderCRDList(loadedCRDs)}
                            </div>
                        </div>
                    </div>
                </div>
            `;

            const modalContainer = document.createElement('div');
            modalContainer.id = 'crd-modal';
            modalContainer.innerHTML = html;
            document.body.appendChild(modalContainer);
        }

        // Render CRD list for modal
        function renderCRDList(crds) {
            if (!crds || crds.length === 0) {
                return '<p style="color: var(--text-secondary);">No CRDs found</p>';
            }

            // Group by group
            const grouped = {};
            crds.forEach(crd => {
                const group = crd.group || 'core';
                if (!grouped[group]) grouped[group] = [];
                grouped[group].push(crd);
            });

            let html = '';
            const sortedGroups = Object.keys(grouped).sort();

            for (const group of sortedGroups) {
                html += `<div style="margin-bottom: 16px;">
                    <div style="font-size: 12px; color: var(--text-secondary); margin-bottom: 8px; border-bottom: 1px solid var(--border-color); padding-bottom: 4px;">${group}</div>`;

                for (const crd of grouped[group]) {
                    const shortNames = crd.shortNames?.length ? ` (${crd.shortNames.join(', ')})` : '';
                    const scope = crd.namespaced ? 'Namespaced' : 'Cluster';
                    html += `<div class="nav-item" style="margin: 4px 0; padding: 8px; cursor: pointer;" onclick="closeAllModals(); switchToCRD('${crd.name}')">
                        <div style="display: flex; justify-content: space-between; width: 100%;">
                            <span><strong>${crd.kind}</strong>${shortNames}</span>
                            <span style="font-size: 11px; color: var(--text-secondary);">${scope} • ${crd.version}</span>
                        </div>
                    </div>`;
                }

                html += '</div>';
            }

            return html;
        }

        // Filter CRD list in modal
        function filterCRDList(query) {
            const filtered = loadedCRDs.filter(crd => {
                const q = query.toLowerCase();
                return crd.name.toLowerCase().includes(q) ||
                       crd.kind.toLowerCase().includes(q) ||
                       crd.group.toLowerCase().includes(q) ||
                       (crd.shortNames || []).some(s => s.toLowerCase().includes(q));
            });
            document.getElementById('crd-list-container').innerHTML = renderCRDList(filtered);
        }

        function switchResource(resource) {
            closeMobileSidebar();
            currentResource = resource;

            // Clear column filters when switching resources
            columnFilters = {};

            // Reset sort when switching resources
            sortColumn = null;
            sortDirection = 'asc';

            document.querySelectorAll('.nav-item').forEach(item => {
                item.classList.toggle('active', item.dataset.resource === resource);
            });
            document.getElementById('panel-title').textContent = resource.charAt(0).toUpperCase() + resource.slice(1);

            // Hide topology view, custom views and overview panel, show main panel
            hideTopologyView();
            hideAllCustomViews();
            hideOverviewPanel();

            // Update active column filters display
            updateActiveColumnFiltersDisplay();

            loadData();
        }

        function onNamespaceChange() {
            currentNamespace = document.getElementById('namespace-select').value;
            trackNamespaceUsage(currentNamespace);
            loadData();
        }

        function refreshData() {
            loadData();
            updateLastRefreshTime();
        }

        // Auto-refresh functions
        function startAutoRefresh() {
            if (autoRefreshTimer) {
                clearInterval(autoRefreshTimer);
            }
            if (autoRefreshEnabled && autoRefreshInterval > 0) {
                autoRefreshTimer = setInterval(() => {
                    loadData();
                    updateLastRefreshTime();
                }, autoRefreshInterval * 1000);
                updateAutoRefreshUI();
            }
        }

        function stopAutoRefresh() {
            if (autoRefreshTimer) {
                clearInterval(autoRefreshTimer);
                autoRefreshTimer = null;
            }
            updateAutoRefreshUI();
        }

        function toggleAutoRefresh() {
            autoRefreshEnabled = !autoRefreshEnabled;
            localStorage.setItem('k13d_auto_refresh', autoRefreshEnabled);
            if (autoRefreshEnabled) {
                startAutoRefresh();
            } else {
                stopAutoRefresh();
            }
        }

        function setAutoRefreshInterval(seconds) {
            autoRefreshInterval = Math.max(5, Math.min(300, seconds)); // 5s to 5min
            localStorage.setItem('k13d_refresh_interval', autoRefreshInterval);
            if (autoRefreshEnabled) {
                startAutoRefresh();
            }
        }

        function updateAutoRefreshUI() {
            const toggle = document.getElementById('auto-refresh-toggle');
            const intervalSelect = document.getElementById('refresh-interval');
            if (toggle) {
                toggle.classList.toggle('active', autoRefreshEnabled);
                toggle.title = autoRefreshEnabled
                    ? `Auto-refresh: ON (every ${autoRefreshInterval}s)`
                    : 'Auto-refresh: OFF';
            }
            if (intervalSelect) {
                intervalSelect.value = autoRefreshInterval;
            }
        }

        async function manualRefresh() {
            const btn = document.querySelector('.refresh-btn');
            if (btn) {
                btn.classList.add('spinning');
            }
            try {
                await loadData();
                updateLastRefreshTime();
            } finally {
                if (btn) {
                    setTimeout(() => btn.classList.remove('spinning'), 500);
                }
            }
        }

        function updateLastRefreshTime() {
            const el = document.getElementById('last-refresh-time');
            if (el) {
                el.textContent = new Date().toLocaleTimeString();
            }
        }

        function updateResourceSummary(resource, items) {
            const summaryEl = document.getElementById('resource-summary');
            if (!summaryEl) return;

            if (!items || items.length === 0) {
                summaryEl.innerHTML = '<span class="summary-item"><span class="summary-count">0</span> total</span>';
                return;
            }

            const total = items.length;
            let html = `<span class="summary-item"><span class="summary-count">${total}</span> total</span>`;

            // Resource-specific status breakdown
            if (resource === 'pods') {
                const statusCounts = {};
                items.forEach(item => {
                    const status = (item.status || 'Unknown').toLowerCase();
                    statusCounts[status] = (statusCounts[status] || 0) + 1;
                });
                const statusOrder = ['running', 'pending', 'succeeded', 'failed', 'unknown'];
                statusOrder.forEach(status => {
                    if (statusCounts[status]) {
                        html += `<span class="summary-item"><span class="summary-count status-${status}">${statusCounts[status]}</span> ${status}</span>`;
                    }
                });
                // Handle other statuses
                Object.keys(statusCounts).forEach(status => {
                    if (!statusOrder.includes(status) && statusCounts[status]) {
                        html += `<span class="summary-item"><span class="summary-count">${statusCounts[status]}</span> ${status}</span>`;
                    }
                });
            } else if (resource === 'deployments' || resource === 'statefulsets' || resource === 'replicasets') {
                let ready = 0, notReady = 0;
                items.forEach(item => {
                    const readyStr = String(item.ready || '0/0');
                    const parts = readyStr.includes('/') ? readyStr.split('/') : [readyStr, readyStr];
                    if (parts.length === 2 && parts[0] === parts[1] && parts[0] !== '0') {
                        ready++;
                    } else {
                        notReady++;
                    }
                });
                if (ready > 0) html += `<span class="summary-item"><span class="summary-count status-running">${ready}</span> ready</span>`;
                if (notReady > 0) html += `<span class="summary-item"><span class="summary-count status-pending">${notReady}</span> not ready</span>`;
            } else if (resource === 'nodes') {
                let ready = 0, notReady = 0;
                items.forEach(item => {
                    if (item.status === 'Ready') ready++;
                    else notReady++;
                });
                if (ready > 0) html += `<span class="summary-item"><span class="summary-count status-running">${ready}</span> ready</span>`;
                if (notReady > 0) html += `<span class="summary-item"><span class="summary-count status-failed">${notReady}</span> not ready</span>`;
            } else if (resource === 'jobs') {
                let complete = 0, running = 0, failed = 0;
                items.forEach(item => {
                    const status = (item.status || '').toLowerCase();
                    if (status.includes('complete') || status === 'succeeded') complete++;
                    else if (status.includes('fail')) failed++;
                    else running++;
                });
                if (complete > 0) html += `<span class="summary-item"><span class="summary-count status-succeeded">${complete}</span> complete</span>`;
                if (running > 0) html += `<span class="summary-item"><span class="summary-count status-pending">${running}</span> running</span>`;
                if (failed > 0) html += `<span class="summary-item"><span class="summary-count status-failed">${failed}</span> failed</span>`;
            } else if (resource === 'events') {
                const typeCounts = {};
                items.forEach(item => {
                    const type = item.type || 'Unknown';
                    typeCounts[type] = (typeCounts[type] || 0) + 1;
                });
                if (typeCounts['Normal']) html += `<span class="summary-item"><span class="summary-count status-running">${typeCounts['Normal']}</span> normal</span>`;
                if (typeCounts['Warning']) html += `<span class="summary-item"><span class="summary-count status-pending">${typeCounts['Warning']}</span> warning</span>`;
            } else if (resource === 'services') {
                const typeCounts = {};
                items.forEach(item => {
                    const type = item.type || 'ClusterIP';
                    typeCounts[type] = (typeCounts[type] || 0) + 1;
                });
                Object.keys(typeCounts).forEach(type => {
                    html += `<span class="summary-item"><span class="summary-count">${typeCounts[type]}</span> ${type}</span>`;
                });
            }

            summaryEl.innerHTML = html;
        }

        function renderTable(resource, items) {
            const headers = tableHeaders[resource];

            // Update resource summary
            updateResourceSummary(resource, items);

            // Store all items for sorting/filtering
            allItems = items || [];
            filteredItems = [...allItems];

            // Reset pagination on new data load
            currentPage = 1;

            // Render sortable headers with column filter row
            const headerRow = `<tr>${headers.map(h => {
                const sortClass = sortColumn === h ? (sortDirection === 'asc' ? 'sort-asc' : 'sort-desc') : '';
                return `<th class="${sortClass}" onclick="onColumnSort('${h}', this)">${h}<span class="sort-icon"></span></th>`;
            }).join('')}</tr>`;

            const filterRow = `<tr class="column-filter-row ${columnFiltersVisible ? 'active' : ''}" id="column-filter-row">
                ${headers.map(h => {
                    const filterValue = columnFilters[h] || '';
                    return `<th><input type="text" class="column-filter-input" placeholder="Filter ${h.toLowerCase()}..."
                        value="${filterValue}"
                        data-column="${h}"
                        onkeyup="onColumnFilterChange(event, '${h}')"
                        onclick="event.stopPropagation()"></th>`;
                }).join('')}
            </tr>`;

            document.getElementById('table-header').innerHTML = headerRow + filterRow;

            if (!items || items.length === 0) {
                document.getElementById('table-body').innerHTML =
                    `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">No ${resource} found</td></tr>`;
                updatePaginationUI(0, 0);
                return;
            }

            // Apply current filter and sort, then render
            applyFilterAndSort();
        }

        // Render table body only (used by pagination)
        function renderTableBody(resource, items) {
            const headers = tableHeaders[resource];
            if (!items || items.length === 0) {
                document.getElementById('table-body').innerHTML =
                    `<tr><td colspan="${headers.length}" style="text-align:center;padding:40px;">No ${resource} found</td></tr>`;
                return;
            }

            document.getElementById('table-body').innerHTML = items.map((item, index) => {
                switch (resource) {
                    case 'pods':
                        const containers = item.containers || ['default'];
                        const containersJson = JSON.stringify(containers).replace(/'/g, "\\'");
                        return `<tr data-index="${index}" data-containers='${containersJson}'>
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.ready}</td>
                            <td class="status-${item.status.toLowerCase()}">${item.status}</td>
                            <td>${item.restarts}</td>
                            <td>${item.age}</td>
                            <td>${item.ip || '-'}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn terminal" onclick="event.stopPropagation(); openTerminal('${item.name}', '${item.namespace}')">Terminal</button>
                                <button class="resource-action-btn logs" onclick="event.stopPropagation(); openLogViewerFromRow(this, '${item.name}', '${item.namespace}')">Logs</button>
                                <button class="resource-action-btn portforward" onclick="event.stopPropagation(); openPortForward('${item.name}', '${item.namespace}')">Forward</button>
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Pod', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'deployments':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.ready}</td>
                            <td>${item.upToDate || item.up_to_date || '-'}</td>
                            <td>${item.available || '-'}</td>
                            <td>${item.age}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button>
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Deployment', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'daemonsets':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.desired || '-'}</td>
                            <td>${item.current || '-'}</td>
                            <td>${item.ready || '-'}</td>
                            <td>${item.age}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button>
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('DaemonSet', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'statefulsets':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.ready || '-'}</td>
                            <td>${item.age}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button>
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('StatefulSet', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'replicasets':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.desired || '-'}</td>
                            <td>${item.current || '-'}</td>
                            <td>${item.ready || '-'}</td>
                            <td>${item.age}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button>
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('ReplicaSet', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'jobs':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.completions || '-'}</td>
                            <td>${item.duration || '-'}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'cronjobs':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.schedule || '-'}</td>
                            <td>${item.suspend ? 'Yes' : 'No'}</td>
                            <td>${item.active || 0}</td>
                            <td>${item.lastSchedule || '-'}</td>
                        </tr>`;
                    case 'services':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.type}</td>
                            <td>${item.clusterIP}</td>
                            <td>${item.ports}</td>
                            <td>${item.age}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Service', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'ingresses':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.class || item.ingressClass || '-'}</td>
                            <td>${item.hosts || '-'}</td>
                            <td>${item.address || '-'}</td>
                            <td>${item.age}</td>
                            <td class="resource-actions">
                                <button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Ingress', '${item.name}', '${item.namespace}')">Topo</button>
                            </td>
                        </tr>`;
                    case 'networkpolicies':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.podSelector || '-'}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'configmaps':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.data || item.dataCount || 0}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'secrets':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.type || '-'}</td>
                            <td>${item.data || item.dataCount || 0}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'serviceaccounts':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.secrets || 0}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'persistentvolumes':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.capacity || '-'}</td>
                            <td>${item.accessModes || '-'}</td>
                            <td>${item.reclaimPolicy || '-'}</td>
                            <td class="status-${(item.status || '').toLowerCase()}">${item.status || '-'}</td>
                            <td>${item.claim || '-'}</td>
                        </tr>`;
                    case 'persistentvolumeclaims':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td class="status-${(item.status || '').toLowerCase()}">${item.status || '-'}</td>
                            <td>${item.volume || '-'}</td>
                            <td>${item.capacity || '-'}</td>
                            <td>${item.accessModes || '-'}</td>
                        </tr>`;
                    case 'nodes':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td class="status-${(item.status || '').toLowerCase()}">${item.status}</td>
                            <td>${item.roles}</td>
                            <td>${item.version}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'namespaces':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td class="status-active">${item.status}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'events':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.type}</td>
                            <td>${item.reason}</td>
                            <td>${item.message?.substring(0, 50) || '-'}${item.message?.length > 50 ? '...' : ''}</td>
                            <td>${item.count}</td>
                            <td>${item.lastSeen}</td>
                        </tr>`;
                    case 'roles':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'rolebindings':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.namespace}</td>
                            <td>${item.role || item.roleRef || '-'}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'clusterroles':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    case 'clusterrolebindings':
                        return `<tr data-index="${index}">
                            <td>${item.name}</td>
                            <td>${item.role || item.roleRef || '-'}</td>
                            <td>${item.age}</td>
                        </tr>`;
                    default:
                        // Handle Custom Resources (crd:xxx format) and unknown types
                        if (resource.startsWith('crd:')) {
                            const crdInfo = currentCRD;
                            const extra = item.extra || {};
                            const extraCols = crdInfo?._extraColumns || [];
                            const extraCells = extraCols.map(col => {
                                const key = col.toLowerCase().replace(/[- ]/g, '_');
                                return `<td>${escapeHtml(extra[key] || '-')}</td>`;
                            }).join('');
                            const statusVal = item.status || '-';
                            const statusClass = statusVal.toLowerCase().includes('ready') || statusVal.toLowerCase() === 'true' ? 'status-running' :
                                               statusVal.toLowerCase().includes('failed') || statusVal.toLowerCase() === 'false' ? 'status-failed' : '';
                            if (crdInfo && crdInfo.namespaced) {
                                return `<tr data-index="${index}" onclick="showCRDetail('${crdInfo.name}', '${item.namespace || ''}', '${item.name}')">
                                    <td>${item.name}</td>
                                    <td>${item.namespace || '-'}</td>
                                    ${extraCells}
                                    <td class="${statusClass}">${escapeHtml(statusVal)}</td>
                                    <td>${item.age || '-'}</td>
                                </tr>`;
                            } else {
                                return `<tr data-index="${index}" onclick="showCRDetail('${crdInfo?.name || ''}', '', '${item.name}')">
                                    <td>${item.name}</td>
                                    ${extraCells}
                                    <td class="${statusClass}">${escapeHtml(statusVal)}</td>
                                    <td>${item.age || '-'}</td>
                                </tr>`;
                            }
                        }
                        // Generic fallback for unknown resource types
                        const defaultHeaders = tableHeaders[resource] || ['NAME'];
                        return `<tr data-index="${index}">${defaultHeaders.map(h => `<td>${item[h.toLowerCase().replace(/[- ]/g, '')] || item.name || '-'}</td>`).join('')}</tr>`;
                }
            }).join('');
        }

        // Show Custom Resource detail using the shared detail-modal
        async function showCRDetail(crdName, namespace, name) {
            const crdInfo = loadedCRDs.find(c => c.name === crdName);
            if (!crdInfo) return;

            try {
                // Fetch full CR as JSON for overview
                const ns = namespace ? `&namespace=${namespace}` : '';
                const resp = await fetchWithAuth(`/api/crd/${crdName}/instances/${name}?${ns}`);
                const crData = await resp.json();

                // Store as selectedResource for YAML/Events tabs
                selectedResource = {
                    name: name,
                    namespace: namespace,
                    _isCR: true,
                    _crdName: crdName,
                    _crdInfo: crdInfo,
                    _crData: crData,
                };

                document.getElementById('detail-title').textContent = `${crdInfo.kind}: ${name}`;

                // Overview tab
                document.getElementById('detail-overview').innerHTML = generateCROverview(crdInfo, crData);

                // YAML tab - load on demand
                document.getElementById('detail-yaml').innerHTML = '<div class="yaml-viewer" style="color: var(--text-secondary);">Click the YAML tab to load...</div>';
                document.getElementById('detail-yaml').dataset.loaded = 'false';

                // Events tab - load on demand
                document.getElementById('detail-events').innerHTML = '<p style="color: var(--text-secondary);">Click the Events tab to load...</p>';
                document.getElementById('detail-events').dataset.loaded = 'false';

                // Hide Related Pods tab
                document.getElementById('detail-pods-tab').style.display = 'none';

                document.getElementById('detail-modal').classList.add('active');
                switchDetailTab('overview');
            } catch (e) {
                console.error('Failed to load CR detail:', e);
            }
        }

        // Generate rich overview for Custom Resources
        function generateCROverview(crdInfo, crData) {
            const metadata = crData.metadata || {};
            const spec = crData.spec || {};
            const status = crData.status || {};
            const labels = metadata.labels || {};
            const annotations = metadata.annotations || {};

            // Determine status from common patterns
            let statusText = '-';
            let statusColor = 'var(--text-secondary)';
            const conditions = status.conditions || [];

            if (status.phase) {
                statusText = status.phase;
            } else if (status.state) {
                statusText = status.state;
            } else if (typeof status.ready === 'boolean') {
                statusText = status.ready ? 'Ready' : 'NotReady';
            } else if (conditions.length > 0) {
                const readyCond = conditions.find(c => c.type === 'Ready' || c.type === 'Available' || c.type === 'Synced');
                if (readyCond) {
                    statusText = readyCond.status === 'True' ? readyCond.type : `Not${readyCond.type}`;
                }
            }

            // Also check printer columns for status
            if (statusText === '-' && crdInfo.printerColumns) {
                for (const col of crdInfo.printerColumns) {
                    const key = col.name.toLowerCase();
                    if (key === 'status' || key === 'phase' || key === 'state' || key === 'ready') {
                        const val = resolveJSONPathClient(crData, col.jsonPath || col.JSONPath);
                        if (val) { statusText = String(val); break; }
                    }
                }
            }

            const readyStates = ['ready', 'running', 'active', 'healthy', 'synced', 'true', 'available', 'bound', 'succeeded', 'complete'];
            const failedStates = ['failed', 'error', 'notready', 'unavailable', 'false', 'degraded', 'crashloopbackoff'];
            const statusLower = statusText.toLowerCase();
            if (readyStates.some(s => statusLower.includes(s))) {
                statusColor = 'var(--accent-green)';
            } else if (failedStates.some(s => statusLower.includes(s))) {
                statusColor = 'var(--accent-red)';
            } else if (statusText !== '-') {
                statusColor = 'var(--accent-yellow)';
            }

            // Build labels HTML
            const labelHtml = Object.keys(labels).length > 0
                ? Object.entries(labels).map(([k, v]) =>
                    `<span style="display:inline-block;padding:2px 8px;margin:2px;border-radius:4px;background:var(--accent-blue)15;color:var(--accent-blue);font-size:11px;border:1px solid var(--accent-blue)30;font-family:monospace;">${escapeHtml(k)}=${escapeHtml(v)}</span>`
                ).join('')
                : '<span style="color:var(--text-secondary);font-size:12px;">None</span>';

            // Build spec fields (top-level only, skip large nested objects)
            const specEntries = Object.entries(spec).filter(([k, v]) => {
                if (v === null || v === undefined) return false;
                if (typeof v === 'object' && !Array.isArray(v) && Object.keys(v).length > 5) return false;
                return true;
            }).slice(0, 12);

            const specHtml = specEntries.length > 0
                ? specEntries.map(([k, v]) => {
                    let display;
                    if (typeof v === 'object') {
                        display = Array.isArray(v) ? `[${v.length} items]` : JSON.stringify(v);
                        if (display.length > 80) display = display.substring(0, 77) + '...';
                    } else {
                        display = String(v);
                    }
                    return `<div class="overview-stat">
                        <span class="stat-label">${escapeHtml(k)}</span>
                        <span class="stat-value" style="font-family:monospace;font-size:12px;word-break:break-all;">${escapeHtml(display)}</span>
                    </div>`;
                }).join('')
                : '<div style="color:var(--text-secondary);font-size:12px;padding:8px;">No spec fields</div>';

            // Build conditions table
            let conditionsHtml = '';
            if (conditions.length > 0) {
                conditionsHtml = `
                    <div class="overview-card" style="grid-column: 1 / -1;">
                        <div class="overview-card-title">Conditions</div>
                        <div style="overflow-x:auto;">
                            <table style="width:100%;border-collapse:collapse;font-size:12px;">
                                <thead>
                                    <tr style="border-bottom:1px solid var(--border-color);">
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">TYPE</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">STATUS</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">REASON</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">MESSAGE</th>
                                        <th style="text-align:left;padding:6px 8px;color:var(--text-secondary);">LAST TRANSITION</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${conditions.map(c => {
                                        const condColor = c.status === 'True' ? 'var(--accent-green)' : c.status === 'False' ? 'var(--accent-red)' : 'var(--accent-yellow)';
                                        const age = c.lastTransitionTime ? formatTimeShort(c.lastTransitionTime) : '-';
                                        return `<tr style="border-bottom:1px solid var(--border-color)20;">
                                            <td style="padding:6px 8px;font-weight:500;">${escapeHtml(c.type || '-')}</td>
                                            <td style="padding:6px 8px;color:${condColor};font-weight:600;">${escapeHtml(c.status || '-')}</td>
                                            <td style="padding:6px 8px;color:var(--text-secondary);">${escapeHtml(c.reason || '-')}</td>
                                            <td style="padding:6px 8px;color:var(--text-secondary);max-width:300px;overflow:hidden;text-overflow:ellipsis;" title="${escapeHtml(c.message || '')}">${escapeHtml(c.message || '-')}</td>
                                            <td style="padding:6px 8px;color:var(--text-secondary);">${age}</td>
                                        </tr>`;
                                    }).join('')}
                                </tbody>
                            </table>
                        </div>
                    </div>`;
            }

            // Build status fields (excluding conditions)
            const statusEntries = Object.entries(status).filter(([k]) => k !== 'conditions').slice(0, 8);
            const statusFieldsHtml = statusEntries.length > 0
                ? statusEntries.map(([k, v]) => {
                    let display;
                    if (typeof v === 'object') {
                        display = JSON.stringify(v);
                        if (display.length > 80) display = display.substring(0, 77) + '...';
                    } else {
                        display = String(v);
                    }
                    return `<div class="overview-stat">
                        <span class="stat-label">${escapeHtml(k)}</span>
                        <span class="stat-value" style="font-family:monospace;font-size:12px;">${escapeHtml(display)}</span>
                    </div>`;
                }).join('')
                : '';

            // Build printer columns card
            let printerColsHtml = '';
            const printerCols = crdInfo.printerColumns || [];
            const displayCols = printerCols.filter(c => {
                const key = c.name.toLowerCase();
                return key !== 'age' && key !== 'name' && key !== 'namespace';
            });
            if (displayCols.length > 0) {
                const colValues = displayCols.map(c => {
                    const val = resolveJSONPathClient(crData, c.jsonPath || c.JSONPath) || '-';
                    return `<div class="overview-stat">
                        <span class="stat-label">${escapeHtml(c.name)}</span>
                        <span class="stat-value" style="font-family:monospace;font-size:12px;">${escapeHtml(String(val))}</span>
                    </div>`;
                }).join('');
                printerColsHtml = `
                    <div class="overview-card">
                        <div class="overview-card-title">Key Fields</div>
                        <div class="overview-card-content">${colValues}</div>
                    </div>`;
            }

            // Annotations (show first 5, truncated)
            const annotationEntries = Object.entries(annotations).slice(0, 5);
            const annotationsHtml = annotationEntries.length > 0
                ? annotationEntries.map(([k, v]) => {
                    const shortVal = v.length > 60 ? v.substring(0, 57) + '...' : v;
                    return `<div class="overview-stat">
                        <span class="stat-label" style="font-size:11px;" title="${escapeHtml(k)}">${escapeHtml(k.split('/').pop())}</span>
                        <span class="stat-value" style="font-size:11px;font-family:monospace;" title="${escapeHtml(v)}">${escapeHtml(shortVal)}</span>
                    </div>`;
                }).join('')
                : '';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(statusText)}
                    </div>
                    <span style="color:var(--text-secondary);font-size:12px;margin-left:12px;">${escapeHtml(crdInfo.group)}/${escapeHtml(crdInfo.version)}</span>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Name</span>
                                <span class="stat-value" style="font-family:monospace;">${escapeHtml(metadata.name || '-')}</span>
                            </div>
                            ${metadata.namespace ? `<div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(metadata.namespace)}</span>
                            </div>` : ''}
                            <div class="overview-stat">
                                <span class="stat-label">Created</span>
                                <span class="stat-value">${metadata.creationTimestamp ? formatTimeShort(metadata.creationTimestamp) : '-'}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Generation</span>
                                <span class="stat-value">${metadata.generation || '-'}</span>
                            </div>
                        </div>
                    </div>
                    ${printerColsHtml}
                    <div class="overview-card">
                        <div class="overview-card-title">Spec</div>
                        <div class="overview-card-content">${specHtml}</div>
                    </div>
                    ${statusFieldsHtml ? `<div class="overview-card">
                        <div class="overview-card-title">Status</div>
                        <div class="overview-card-content">${statusFieldsHtml}</div>
                    </div>` : ''}
                </div>
                ${conditionsHtml}
                <div class="overview-card" style="margin-top:12px;">
                    <div class="overview-card-title">Labels</div>
                    <div style="padding:8px;">${labelHtml}</div>
                </div>
                ${annotationsHtml ? `<div class="overview-card" style="margin-top:12px;">
                    <div class="overview-card-title">Annotations</div>
                    <div class="overview-card-content">${annotationsHtml}</div>
                </div>` : ''}
            `;
        }

        // Client-side JSONPath resolver (mirrors Go's ResolveJSONPath)
        function resolveJSONPathClient(obj, path) {
            if (!path || !obj) return null;
            path = path.replace(/^\./, '');
            return _resolvePathRec(obj, path);
        }

        function _resolvePathRec(current, path) {
            if (!path || current === null || current === undefined) return current;
            if (typeof current !== 'object' || Array.isArray(current)) return null;

            const dotIdx = path.indexOf('.');
            const bracketIdx = path.indexOf('[');

            // Simple field (no dot, no bracket)
            if (dotIdx < 0 && bracketIdx < 0) return current[path];

            // Array bracket comes before dot (or no dot)
            if (bracketIdx >= 0 && (dotIdx < 0 || bracketIdx < dotIdx)) {
                const fieldName = path.substring(0, bracketIdx);
                const rest = path.substring(bracketIdx);
                const arr = current[fieldName];
                if (!Array.isArray(arr)) return null;

                const bracketEnd = rest.indexOf(']');
                if (bracketEnd < 0) return null;
                const bracketContent = rest.substring(1, bracketEnd);
                let remaining = rest.substring(bracketEnd + 1);
                if (remaining.startsWith('.')) remaining = remaining.substring(1);

                // Array filter: ?(@.key=="value")
                if (bracketContent.startsWith('?(@.')) {
                    const expr = bracketContent.substring(4, bracketContent.length - 1); // strip ?(@. and )
                    const eqParts = expr.split('==');
                    if (eqParts.length === 2) {
                        const key = eqParts[0];
                        const value = eqParts[1].replace(/['"]/g, '');
                        const found = arr.find(item => item && String(item[key]) === value);
                        return remaining ? _resolvePathRec(found, remaining) : found;
                    }
                    return null;
                }

                // Numeric index
                const idx = parseInt(bracketContent);
                if (!isNaN(idx) && idx >= 0 && idx < arr.length) {
                    return remaining ? _resolvePathRec(arr[idx], remaining) : arr[idx];
                }
                return null;
            }

            // Dot-separated path
            const fieldName = path.substring(0, dotIdx);
            const rest = path.substring(dotIdx + 1);
            return _resolvePathRec(current[fieldName], rest);
        }

        // Escape HTML for safe display
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Agentic Mode State - DEFAULT ON for tool execution
        let pendingApproval = null;

        // Resource name mappings for AI command parsing
        const resourceAliases = {
            // Korean aliases
            '파드': 'pods', '팟': 'pods', '포드': 'pods',
            '디플로이먼트': 'deployments', '배포': 'deployments',
            '서비스': 'services', '서비스들': 'services',
            '노드': 'nodes', '노드들': 'nodes',
            '네임스페이스': 'namespaces', '네임스페이스들': 'namespaces',
            '컨피그맵': 'configmaps', '설정': 'configmaps',
            '시크릿': 'secrets', '비밀': 'secrets',
            '인그레스': 'ingresses',
            '이벤트': 'events', '이벤트들': 'events',
            '스테이트풀셋': 'statefulsets',
            '데몬셋': 'daemonsets',
            '레플리카셋': 'replicasets',
            '잡': 'jobs', '작업': 'jobs',
            '크론잡': 'cronjobs', '스케줄잡': 'cronjobs',
            '볼륨': 'persistentvolumeclaims', 'pvc': 'persistentvolumeclaims',
            '롤': 'roles', '역할': 'roles',
            '서비스계정': 'serviceaccounts',
            // English aliases
            'pod': 'pods', 'deployment': 'deployments', 'deploy': 'deployments',
            'service': 'services', 'svc': 'services',
            'node': 'nodes', 'namespace': 'namespaces', 'ns': 'namespaces',
            'configmap': 'configmaps', 'cm': 'configmaps',
            'secret': 'secrets', 'ingress': 'ingresses', 'ing': 'ingresses',
            'event': 'events', 'ev': 'events',
            'statefulset': 'statefulsets', 'sts': 'statefulsets',
            'daemonset': 'daemonsets', 'ds': 'daemonsets',
            'replicaset': 'replicasets', 'rs': 'replicasets',
            'job': 'jobs', 'cronjob': 'cronjobs', 'cj': 'cronjobs',
            'pv': 'persistentvolumes', 'persistentvolume': 'persistentvolumes',
            'role': 'roles', 'rolebinding': 'rolebindings', 'rb': 'rolebindings',
            'clusterrole': 'clusterroles', 'cr': 'clusterroles',
            'clusterrolebinding': 'clusterrolebindings', 'crb': 'clusterrolebindings',
            'serviceaccount': 'serviceaccounts', 'sa': 'serviceaccounts',
            'networkpolicy': 'networkpolicies', 'netpol': 'networkpolicies'
        };

        // Parse user message and AI response for dashboard commands
        async function handleAIDashboardCommands(aiResponse, userMessage) {
            const msg = userMessage.toLowerCase();
            const resp = aiResponse.toLowerCase();

            // Detect show/list resource commands from user message
            const showPatterns = [
                /(?:show|display|list|get|보여|조회|확인|봐|봐줘|보기|리스트).*?(pods?|deployments?|services?|nodes?|namespaces?|configmaps?|secrets?|ingress(?:es)?|events?|statefulsets?|daemonsets?|replicasets?|jobs?|cronjobs?|persistentvolume(?:claim)?s?|roles?|rolebindings?|clusterroles?|clusterrolebindings?|serviceaccounts?|networkpolic(?:y|ies)|파드|팟|포드|디플로이먼트|배포|서비스|노드|네임스페이스|컨피그맵|설정|시크릿|비밀|인그레스|이벤트|스테이트풀셋|데몬셋|레플리카셋|잡|작업|크론잡|스케줄잡|볼륨|pvc|pv|롤|역할|서비스계정|svc|ns|cm|ing|ev|sts|ds|rs|cj|rb|cr|crb|sa|netpol)/i,
                /(?:pods?|deployments?|services?|nodes?|namespaces?|configmaps?|secrets?|ingress(?:es)?|events?|statefulsets?|daemonsets?|replicasets?|jobs?|cronjobs?|persistentvolume(?:claim)?s?|roles?|rolebindings?|clusterroles?|clusterrolebindings?|serviceaccounts?|networkpolic(?:y|ies)|파드|팟|포드|디플로이먼트|배포|서비스|노드|네임스페이스|컨피그맵|설정|시크릿|비밀|인그레스|이벤트|스테이트풀셋|데몬셋|레플리카셋|잡|작업|크론잡|스케줄잡|볼륨|pvc|pv|롤|역할|서비스계정|svc|ns|cm|ing|ev|sts|ds|rs|cj|rb|cr|crb|sa|netpol).*?(?:show|display|list|보여|조회|확인|봐|봐줘|보기|리스트)/i
            ];

            let detectedResource = null;
            let detectedNamespace = null;

            // Check user message for resource commands
            for (const pattern of showPatterns) {
                const match = msg.match(pattern);
                if (match) {
                    const resourceWord = match[1] || match[0];
                    detectedResource = resourceAliases[resourceWord.toLowerCase()] || resourceWord.toLowerCase();
                    // Ensure it's a valid resource
                    if (allResources.includes(detectedResource)) {
                        break;
                    }
                    detectedResource = null;
                }
            }

            // Check for namespace specification
            const nsPatterns = [
                /(?:namespace|ns|네임스페이스)[:\s=]+([a-z0-9-]+)/i,
                /(?:in|from|에서|의)\s+([a-z0-9-]+)\s+(?:namespace|ns|네임스페이스)/i,
                /-n\s+([a-z0-9-]+)/i
            ];

            for (const pattern of nsPatterns) {
                const match = msg.match(pattern);
                if (match) {
                    detectedNamespace = match[1];
                    break;
                }
            }

            // Also check AI response for explicit dashboard commands
            // AI can include special markers like [[SHOW:pods]] or [[NAMESPACE:default]]
            const aiShowMatch = aiResponse.match(/\[\[SHOW:([a-z]+)\]\]/i);
            const aiNsMatch = aiResponse.match(/\[\[NAMESPACE:([a-z0-9-]*)\]\]/i);

            if (aiShowMatch) {
                detectedResource = aiShowMatch[1].toLowerCase();
            }
            if (aiNsMatch) {
                detectedNamespace = aiNsMatch[1] || ''; // empty string means all namespaces
            }

            // Execute dashboard navigation if resource detected
            if (detectedResource && allResources.includes(detectedResource)) {
                // Show notification
                showDashboardActionNotification(`Switching to ${detectedResource}...`);

                // Switch namespace first if specified
                if (detectedNamespace !== null) {
                    const nsSelect = document.getElementById('namespace-select');
                    if (nsSelect) {
                        // Check if namespace exists in dropdown
                        const nsExists = Array.from(nsSelect.options).some(opt => opt.value === detectedNamespace);
                        if (nsExists || detectedNamespace === '') {
                            nsSelect.value = detectedNamespace;
                            currentNamespace = detectedNamespace;
                        }
                    }
                }

                // Switch to the resource view
                switchResource(detectedResource);

                // Scroll dashboard into view on mobile
                const dashboardPanel = document.querySelector('.dashboard-panel');
                if (dashboardPanel && window.innerWidth < 768) {
                    dashboardPanel.scrollIntoView({ behavior: 'smooth' });
                }
            }

            // Check for filter commands
            const filterPatterns = [
                /(?:filter|find|search|필터|검색|찾아)[:\s]+["']?([^"'\n]+)["']?/i,
                /["']([^"']+)["'].*?(?:filter|find|search|필터|검색|찾아)/i
            ];

            for (const pattern of filterPatterns) {
                const match = msg.match(pattern);
                if (match && match[1]) {
                    const filterText = match[1].trim();
                    if (filterText && filterText.length > 1) {
                        const filterInput = document.getElementById('filter-input');
                        if (filterInput) {
                            filterInput.value = filterText;
                            filterTable(filterText.toLowerCase());
                            showDashboardActionNotification(`Filtering by "${filterText}"...`);
                        }
                    }
                    break;
                }
            }
        }

        // Show a brief notification for dashboard actions
        function showDashboardActionNotification(message) {
            const notification = document.createElement('div');
            notification.className = 'dashboard-action-notification';
            notification.textContent = message;
            notification.style.cssText = `
                position: fixed;
                top: 60px;
                left: 50%;
                transform: translateX(-50%);
                background: var(--accent-blue);
                color: white;
                padding: 8px 16px;
                border-radius: 4px;
                z-index: 10000;
                font-size: 13px;
                animation: fadeInOut 2s ease-in-out;
            `;
            document.body.appendChild(notification);

            setTimeout(() => {
                notification.remove();
            }, 2000);
        }

        // AI Chat
        async function sendMessage() {
            const input = document.getElementById('ai-input');
            const message = input.value.trim();
            if (!message || isLoading) return;

            // Check guardrails (K8s safety analysis)
            const guardrailCheck = checkGuardrails(message);

            if (!guardrailCheck.allowed) {
                showToast(guardrailCheck.reason, 'error');
                return;
            }

            // Show safety confirmation dialog for risky operations
            if (guardrailCheck.requireConfirmation) {
                const analysis = {
                    riskLevel: guardrailCheck.riskLevel || 'warning',
                    explanation: guardrailCheck.reason,
                    warnings: [guardrailCheck.reason],
                    recommendations: guardrailCheck.riskLevel === 'critical' ?
                        ['Consider using --dry-run=client first', 'Verify the correct cluster context'] :
                        ['Review the operation before proceeding']
                };

                return new Promise((resolve) => {
                    showSafetyConfirmation(analysis,
                        () => {
                            // User confirmed - proceed
                            proceedWithMessage(message);
                            resolve();
                        },
                        () => {
                            // User cancelled
                            showToast('Operation cancelled', 'info');
                            resolve();
                        }
                    );
                });
            }

            await proceedWithMessage(message);
        }

        async function proceedWithMessage(message) {
            isLoading = true;
            document.getElementById('send-btn').disabled = true;
            document.getElementById('ai-input').value = '';
            saveQueryToHistory(message);
            aiHistoryIndex = -1;
            aiCurrentDraft = '';

            // Save user message to chat history
            saveCurrentChatMessage(message, true);
            addMessage(message, true);

            // Always use agentic mode
            console.log('[DEBUG] sendMessage - using agentic mode');
            await sendMessageAgentic(message);

            isLoading = false;
            document.getElementById('send-btn').disabled = false;
        }

        // Format resource links in AI responses to make them clickable
        function formatResourceLinks(text) {
            // Common Kubernetes resource patterns
            // Match patterns like: pod/nginx-xxx, deployment/my-app, service/my-svc
            // or just: nginx-pod, my-deployment (when context is clear)

            // Pattern 1: explicit resource/name format (e.g., pod/nginx-xxx, deployment/my-app)
            const explicitPattern = /\b(pod|deployment|service|statefulset|daemonset|configmap|secret|ingress|node|namespace|replicaset|job|cronjob)s?\/([a-z0-9][-a-z0-9]*[a-z0-9])\b/gi;

            text = text.replace(explicitPattern, (match, kind, name) => {
                const resourceMap = {
                    'pod': 'pods', 'pods': 'pods',
                    'deployment': 'deployments', 'deployments': 'deployments',
                    'service': 'services', 'services': 'services',
                    'statefulset': 'statefulsets', 'statefulsets': 'statefulsets',
                    'daemonset': 'daemonsets', 'daemonsets': 'daemonsets',
                    'configmap': 'configmaps', 'configmaps': 'configmaps',
                    'secret': 'secrets', 'secrets': 'secrets',
                    'ingress': 'ingresses', 'ingresses': 'ingresses',
                    'node': 'nodes', 'nodes': 'nodes',
                    'namespace': 'namespaces', 'namespaces': 'namespaces',
                    'replicaset': 'replicasets', 'replicasets': 'replicasets',
                    'job': 'jobs', 'jobs': 'jobs',
                    'cronjob': 'cronjobs', 'cronjobs': 'cronjobs'
                };
                const resourceType = resourceMap[kind.toLowerCase()] || 'pods';
                return `<a href="#" class="resource-link" onclick="navigateToResource('${resourceType}', '${name}'); return false;">${match}</a>`;
            });

            // Pattern 2: backtick-quoted names that look like k8s resources
            // e.g., `nginx-deployment`, `my-service`, `coredns-xxxxx`
            const backtickPattern = /`([a-z][a-z0-9]*(?:-[a-z0-9]+)+)`/gi;
            text = text.replace(backtickPattern, (match, name) => {
                // Only convert if it looks like a k8s resource name (has hyphens)
                if (name.includes('-')) {
                    return `<a href="#" class="resource-link" onclick="searchAndNavigateToResource('${name}'); return false;">\`${name}\`</a>`;
                }
                return match;
            });

            return text;
        }

        // Navigate directly to a known resource type
        function navigateToResource(resourceType, name) {
            switchResource(resourceType);
            setTimeout(() => {
                document.getElementById('filter-input').value = name;
                currentFilter = name.toLowerCase();
                applyFilterAndSort();
            }, 500);
        }

        // Search for resource and navigate (when type is unknown)
        async function searchAndNavigateToResource(name) {
            try {
                const response = await fetch(`/api/search?q=${encodeURIComponent(name)}&namespace=${currentNamespace || ''}`, {
                    headers: { 'Authorization': `Bearer ${authToken}` }
                });
                if (response.ok) {
                    const data = await response.json();
                    if (data.results && data.results.length > 0) {
                        navigateToSearchResult(data.results[0]);
                        return;
                    }
                }
            } catch (e) {
                console.error('Search error:', e);
            }
            // Fallback: just filter current view
            document.getElementById('filter-input').value = name;
            currentFilter = name.toLowerCase();
            applyFilterAndSort();
        }

        // Agentic chat with tool calling and Decision Required flow
        async function sendMessageAgentic(message) {
            const container = document.getElementById('ai-messages');
            const div = document.createElement('div');
            div.className = 'message assistant streaming';
            div.id = 'streaming-message';
            div.innerHTML = `<div class="message-content"><span class="cursor">▊</span></div>`;
            container.appendChild(div);
            container.scrollTop = container.scrollHeight;

            const contentEl = div.querySelector('.message-content');
            let fullContent = '';

            try {
                const response = await fetch('/api/chat/agentic', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({ message, language: currentLanguage, session_id: currentSessionId })
                });

                if (!response.ok) {
                    const errorText = await response.text();
                    throw new Error(errorText || `HTTP ${response.status}`);
                }

                const reader = response.body.getReader();
                const decoder = new TextDecoder();

                let currentEventType = null;

                while (true) {
                    const { done, value } = await reader.read();
                    if (done) break;

                    const chunk = decoder.decode(value, { stream: true });
                    const lines = chunk.split('\n');

                    for (const line of lines) {
                        // Handle event type lines
                        if (line.startsWith('event: ')) {
                            currentEventType = line.slice(7).trim();
                            continue;
                        }

                        if (line.startsWith('data: ')) {
                            const data = line.slice(6);

                            if (data === '[DONE]') {
                                break;
                            }

                            // Handle session events - save session_id for conversation continuity
                            if (currentEventType === 'session') {
                                try {
                                    const sessionInfo = JSON.parse(data);
                                    if (sessionInfo.session_id) {
                                        currentSessionId = sessionInfo.session_id;
                                        sessionStorage.setItem('k13d_session_id', currentSessionId);
                                    }
                                } catch (e) {
                                    console.error('Failed to parse session:', e);
                                }
                                currentEventType = null;
                                continue;
                            }

                            // Handle tool_execution events - insert before the AI response text
                            if (currentEventType === 'tool_execution') {
                                try {
                                    const execInfo = JSON.parse(data);
                                    showToolExecution(execInfo, div, contentEl);
                                } catch (e) {
                                    console.error('Failed to parse tool_execution:', e);
                                }
                                currentEventType = null;
                                continue;
                            }

                            // Check if this is an approval request
                            if (currentEventType === 'approval') {
                                try {
                                    const parsed = JSON.parse(data);
                                    if (parsed.type === 'approval_required') {
                                        showApprovalModal(parsed);
                                    }
                                } catch (e) {
                                    console.error('Failed to parse approval:', e);
                                }
                                currentEventType = null;
                                continue;
                            }

                            // Try parsing as JSON for other event types
                            try {
                                const parsed = JSON.parse(data);
                                if (parsed.type === 'approval_required') {
                                    showApprovalModal(parsed);
                                    continue;
                                }
                                if (parsed.type === 'tool_execution') {
                                    showToolExecution(parsed, div, contentEl);
                                    continue;
                                }
                            } catch (e) {
                                // Not JSON, treat as regular text
                            }

                            // Regular text streaming
                            const text = data.replace(/\\n/g, '\n');
                            fullContent += text;

                            let formatted = fullContent;
                            formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
                            formatted = formatted.replace(/\n/g, '<br>');
                            contentEl.innerHTML = formatted + '<span class="cursor">▊</span>';
                            container.scrollTop = container.scrollHeight;

                            currentEventType = null;
                        }
                    }
                }

                // Finalize
                div.classList.remove('streaming');
                div.id = '';
                let formatted = fullContent;
                formatted = formatted.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
                formatted = formatResourceLinks(formatted);
                formatted = formatted.replace(/\n/g, '<br>');
                contentEl.innerHTML = formatted;

                // Save AI response to chat history
                if (fullContent.trim()) {
                    saveCurrentChatMessage(fullContent, false);
                }

                // Parse AI response for dashboard commands and execute them
                await handleAIDashboardCommands(fullContent, message);

                // Refresh resource list after potential changes
                await loadData();

            } catch (e) {
                div.classList.remove('streaming');
                div.id = '';

                // Provide user-friendly error messages
                let errorMsg = e.message;
                if (e.message.includes('AI client not configured') || e.message.includes('503')) {
                    errorMsg = `<strong>AI Assistant Not Configured</strong><br><br>
                        The AI assistant requires an LLM provider to be configured. Please go to
                        <strong>Settings → AI/LLM Settings</strong> to configure your preferred provider
                        (OpenAI, Anthropic, Ollama, etc.).<br><br>
                        <em>Note: You need an API key from your chosen provider.</em>`;
                } else if (e.message.includes('does not support tool calling')) {
                    errorMsg = `<strong>Tool Calling Not Supported</strong><br><br>
                        The current AI provider does not support tool calling (agentic mode).
                        Please configure a provider that supports tool calling, such as:<br>
                        • OpenAI (GPT-4, GPT-3.5-turbo)<br>
                        • Anthropic Claude<br>
                        • Ollama with compatible models`;
                }

                contentEl.innerHTML = `<span style="color: var(--accent-red)">${errorMsg}</span>`;
            }
        }

        // Show tool execution info with expandable result
        // messageDiv: the AI message div, contentEl: the text content element inside it
        function showToolExecution(execInfo, messageDiv, contentEl) {
            const execDiv = document.createElement('div');
            execDiv.className = 'tool-execution';

            const isError = execInfo.is_error;
            const statusIcon = isError ? '❌' : '✅';
            const statusColor = isError ? 'var(--accent-red)' : 'var(--accent-green)';

            const uniqueId = 'tool-result-' + Date.now();
            const resultLength = execInfo.result ? execInfo.result.length : 0;

            execDiv.innerHTML = `
                <div class="tool-header" style="display: flex; align-items: center; gap: 8px; margin-bottom: 6px;">
                    <span style="color: ${statusColor};">${statusIcon}</span>
                    <span class="tool-name">${execInfo.tool}</span>
                </div>
                <div class="tool-command" style="background: var(--bg-primary); padding: 8px; border-radius: 4px; font-family: monospace; font-size: 12px; margin-bottom: 8px; word-break: break-all;">
                    $ ${escapeHtml(execInfo.command || 'N/A')}
                </div>
                ${execInfo.result ? `
                    <div class="tool-result-container">
                        <div class="tool-result-full" id="${uniqueId}-full" style="display: none; background: var(--bg-primary); padding: 8px; border-radius: 4px; font-family: monospace; font-size: 11px; max-height: 400px; overflow: auto; white-space: pre-wrap; word-break: break-all; color: ${isError ? 'var(--accent-red)' : 'var(--text-secondary)'};">
${escapeHtml(execInfo.result)}</div>
                        <button onclick="toggleToolResult('${uniqueId}')" id="${uniqueId}-btn" style="margin-top: 6px; padding: 4px 8px; font-size: 11px; background: var(--bg-tertiary); border: none; border-radius: 4px; color: var(--text-primary); cursor: pointer;">
                            ▼ Show Result (${resultLength} chars)
                        </button>
                    </div>
                ` : ''}
            `;

            // Insert tool execution before the content element (AI response text)
            messageDiv.insertBefore(execDiv, contentEl);
            const container = document.getElementById('ai-messages');
            container.scrollTop = container.scrollHeight;

            // Log to debug panel
            addDebugLog('tool', 'Tool Executed', {
                tool: execInfo.tool,
                command: execInfo.command,
                result_length: resultLength,
                is_error: isError
            });
        }

        // Toggle tool result expansion
        function toggleToolResult(uniqueId) {
            const full = document.getElementById(uniqueId + '-full');
            const btn = document.getElementById(uniqueId + '-btn');

            if (full.style.display === 'none') {
                full.style.display = 'block';
                btn.textContent = '▲ Hide Result';
            } else {
                full.style.display = 'none';
                btn.textContent = btn.textContent.replace('▲ Hide Result', '▼ Show Result');
            }
        }

        // Show Decision Required approval modal
        function showApprovalModal(approval) {
            pendingApproval = approval;

            const isDangerous = approval.category === 'dangerous';
            const icon = isDangerous ? '⚠️' : '🔧';
            const title = isDangerous ? 'Dangerous Operation' : 'Decision Required';

            const modal = document.createElement('div');
            modal.className = 'approval-modal';
            modal.id = 'approval-modal';
            modal.innerHTML = `
                <div class="approval-box ${isDangerous ? 'dangerous' : ''}">
                    <div class="approval-header">
                        <span class="approval-icon">${icon}</span>
                        <span class="approval-title">${title}</span>
                    </div>
                    <div class="approval-category ${approval.category}">${approval.category}</div>
                    <p>The AI wants to execute the following command:</p>
                    <div class="approval-command">${escapeHtml(approval.command)}</div>
                    <p style="font-size: 12px; color: var(--text-secondary);">
                        Tool: <strong>${approval.tool_name}</strong>
                    </p>
                    <div class="approval-buttons">
                        <button class="btn-reject" onclick="respondToApproval(false)">
                            ✕ Reject
                        </button>
                        <button class="btn-approve" onclick="respondToApproval(true)">
                            ✓ Approve
                        </button>
                    </div>
                </div>
            `;

            document.body.appendChild(modal);

            // Add keyboard handlers
            document.addEventListener('keydown', handleApprovalKeypress);
        }

        function handleApprovalKeypress(e) {
            if (!pendingApproval) return;

            if (e.key === 'Enter' || e.key === 'y' || e.key === 'Y') {
                respondToApproval(true);
            } else if (e.key === 'Escape' || e.key === 'n' || e.key === 'N') {
                respondToApproval(false);
            }
        }

        async function respondToApproval(approved) {
            if (!pendingApproval) return;

            const approvalId = pendingApproval.id;
            pendingApproval = null;

            // Remove modal
            const modal = document.getElementById('approval-modal');
            if (modal) {
                modal.remove();
            }

            // Remove keyboard handler
            document.removeEventListener('keydown', handleApprovalKeypress);

            // Send response to server
            try {
                await fetch('/api/tool/approve', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({ id: approvalId, approved })
                });

                // Add temporary status message to chat (auto-removes after 5 seconds)
                const container = document.getElementById('ai-messages');
                const statusDiv = document.createElement('div');
                statusDiv.className = 'tool-execution';
                statusDiv.style.transition = 'opacity 0.3s ease-out';
                statusDiv.innerHTML = approved
                    ? `<span class="tool-name">✓ Approved:</span> Command execution proceeding...`
                    : `<span class="tool-name" style="color: var(--accent-red)">✕ Rejected:</span> Command was cancelled by user.`;
                container.appendChild(statusDiv);
                container.scrollTop = container.scrollHeight;

                // Auto-remove the status message after 5 seconds
                setTimeout(() => {
                    statusDiv.style.opacity = '0';
                    setTimeout(() => statusDiv.remove(), 300);
                }, 5000);

            } catch (e) {
                console.error('Failed to send approval response:', e);
            }
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function addMessage(content, isUser = false) {
            const container = document.getElementById('ai-messages');
            const div = document.createElement('div');
            div.className = `message ${isUser ? 'user' : 'assistant'}`;

            let formatted = content;
            if (!isUser) {
                formatted = content.replace(/```(\w*)\n?([\s\S]*?)```/g, '<pre><code>$2</code></pre>');
                formatted = formatted.replace(/\n/g, '<br>');
            }

            div.innerHTML = `<div class="message-content">${formatted}</div>`;
            container.appendChild(div);
            container.scrollTop = container.scrollHeight;
        }

        function addLoadingMessage() {
            const container = document.getElementById('ai-messages');
            const div = document.createElement('div');
            div.className = 'message assistant';
            div.id = 'loading-message';
            div.innerHTML = `<div class="message-content"><div class="loading-dots"><span></span><span></span><span></span></div></div>`;
            container.appendChild(div);
            container.scrollTop = container.scrollHeight;
        }

        function removeLoadingMessage() {
            const loading = document.getElementById('loading-message');
            if (loading) loading.remove();
        }

        // AI input query history
        let aiQueryHistory = JSON.parse(localStorage.getItem('k13d_query_history') || '[]');
        let aiHistoryIndex = -1;
        let aiCurrentDraft = '';

        function saveQueryToHistory(query) {
            if (!query.trim()) return;
            // Avoid duplicates at the end
            if (aiQueryHistory.length > 0 && aiQueryHistory[aiQueryHistory.length - 1] === query) return;
            aiQueryHistory.push(query);
            // Keep last 50 entries
            if (aiQueryHistory.length > 50) aiQueryHistory = aiQueryHistory.slice(-50);
            localStorage.setItem('k13d_query_history', JSON.stringify(aiQueryHistory));
        }

        function clearAiInput() {
            const input = document.getElementById('ai-input');
            input.value = '';
            aiHistoryIndex = -1;
            aiCurrentDraft = '';
            input.focus();
        }

        function toggleAiExpand() {
            const container = document.getElementById('ai-input-container');
            const btn = document.getElementById('ai-expand-btn');
            const input = document.getElementById('ai-input');
            container.classList.toggle('expanded');
            if (container.classList.contains('expanded')) {
                btn.innerHTML = '&#x2716;'; // X to close
                btn.title = 'Exit fullscreen';
                input.rows = 20;
            } else {
                btn.innerHTML = '&#x26F6;'; // expand icon
                btn.title = 'Expand input area';
                input.rows = 2;
            }
            input.focus();
        }

        document.getElementById('ai-input').addEventListener('keydown', (e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
            } else if (e.key === 'Escape') {
                const container = document.getElementById('ai-input-container');
                if (container.classList.contains('expanded')) {
                    toggleAiExpand();
                }
            } else if (e.key === 'ArrowUp' && !e.shiftKey) {
                const input = e.target;
                // Only navigate history if cursor is at the start or input is single-line
                if (input.selectionStart === 0 && aiQueryHistory.length > 0) {
                    e.preventDefault();
                    if (aiHistoryIndex === -1) {
                        aiCurrentDraft = input.value;
                        aiHistoryIndex = aiQueryHistory.length - 1;
                    } else if (aiHistoryIndex > 0) {
                        aiHistoryIndex--;
                    }
                    input.value = aiQueryHistory[aiHistoryIndex];
                }
            } else if (e.key === 'ArrowDown' && !e.shiftKey) {
                const input = e.target;
                if (aiHistoryIndex !== -1) {
                    e.preventDefault();
                    if (aiHistoryIndex < aiQueryHistory.length - 1) {
                        aiHistoryIndex++;
                        input.value = aiQueryHistory[aiHistoryIndex];
                    } else {
                        aiHistoryIndex = -1;
                        input.value = aiCurrentDraft;
                    }
                }
            }
        });

        // Resizable panel
        function setupResizeHandle() {
            const handle = document.getElementById('resize-handle');
            const aiPanel = document.getElementById('ai-panel');
            let isResizing = false;

            handle.addEventListener('mousedown', (e) => {
                isResizing = true;
                document.body.style.cursor = 'col-resize';
                document.body.style.userSelect = 'none';
            });

            document.addEventListener('mousemove', (e) => {
                if (!isResizing) return;
                const containerWidth = document.querySelector('.content-wrapper').offsetWidth;
                const newWidth = containerWidth - e.clientX + document.querySelector('.sidebar').offsetWidth;
                if (newWidth >= 280 && newWidth <= 600) {
                    aiPanel.style.width = newWidth + 'px';
                }
            });

            document.addEventListener('mouseup', () => {
                isResizing = false;
                document.body.style.cursor = '';
                document.body.style.userSelect = '';
            });
        }

        // Health check
        function setupHealthCheck() {
            setInterval(async () => {
                try {
                    const resp = await fetch('/api/health');
                    const data = await resp.json();
                    const dot = document.getElementById('health-dot');
                    const status = document.getElementById('health-status');

                    if (data.status === 'ok' && data.k8s_ready) {
                        dot.className = 'health-dot ok';
                        status.textContent = 'Connected';
                    } else {
                        dot.className = 'health-dot warning';
                        status.textContent = 'Degraded';
                    }
                } catch (e) {
                    document.getElementById('health-dot').className = 'health-dot error';
                    document.getElementById('health-status').textContent = 'Disconnected';
                }
            }, 10000);
        }

        // Settings
        function showSettings() {
            document.getElementById('settings-modal').classList.add('active');
            loadSettings();
            loadVersionInfo();
            // Show Admin tab only for admin users
            const adminTab = document.getElementById('admin-tab');
            if (adminTab) {
                adminTab.style.display = (currentUser && currentUser.role === 'admin') ? 'block' : 'none';
            }
        }

        function closeSettings() {
            document.getElementById('settings-modal').classList.remove('active');
        }

        function switchSettingsTab(tab) {
            document.querySelectorAll('.tabs .tab').forEach((t, i) => {
                t.classList.toggle('active', t.textContent.toLowerCase().includes(tab));
            });
            document.querySelectorAll('.settings-content').forEach(c => c.style.display = 'none');
            document.getElementById(`settings-${tab}`).style.display = 'block';

            // Load data for specific tabs
            if (tab === 'ai') {
                loadModelProfiles();
                updateEndpointPlaceholder();
                loadLLMStatus();
                onLLMTabOpened();
                loadToolApprovalSettings();
                loadAgentSettings();
            } else if (tab === 'mcp') {
                loadMCPServers();
                loadMCPTools();
            } else if (tab === 'admin') {
                loadAdminUsers();
                loadAuthStatus();
                loadRoles();
            } else if (tab === 'security') {
                checkTrivyStatus();
                loadTrivyInstructions();
            } else if (tab === 'metrics') {
                loadPrometheusSettings();
            } else if (tab === 'general') {
                // Load saved theme
                const saved = localStorage.getItem('k13d_theme') || 'tokyo-night';
                const sel = document.getElementById('setting-theme');
                if (sel) sel.value = saved;
            }
        }

        // Theme / Skin support
        function applyTheme(theme) {
            const html = document.documentElement;
            if (theme === 'light') {
                html.removeAttribute('data-theme');
            } else {
                html.setAttribute('data-theme', theme);
            }
            localStorage.setItem('k13d_theme', theme);
            updateThemeIcon();
            // Sync settings dropdown
            const sel = document.getElementById('setting-theme');
            if (sel) sel.value = theme;
        }

        // Apply saved theme on load
        (function initSettingsTheme() {
            const saved = localStorage.getItem('k13d_theme') || 'tokyo-night';
            applyTheme(saved);
        })();

        // ==========================================
        // Trivy/Security Functions
        // ==========================================
        async function checkTrivyStatus() {
            const indicator = document.getElementById('trivy-status-indicator');
            const statusText = document.getElementById('trivy-status-text');
            const versionEl = document.getElementById('trivy-version');
            const pathEl = document.getElementById('trivy-path');
            const installBtn = document.getElementById('trivy-install-btn');
            const instructionsDiv = document.getElementById('trivy-instructions');

            try {
                const resp = await fetchWithAuth('/api/security/trivy/status');
                const status = await resp.json();

                if (status.installed) {
                    indicator.style.background = 'var(--accent-green)';
                    indicator.style.boxShadow = '0 0 8px var(--accent-green)';
                    statusText.textContent = 'Installed';
                    versionEl.textContent = status.version ? `Version: ${status.version}` : '';
                    pathEl.textContent = status.path || '';
                    installBtn.style.display = 'none';
                    instructionsDiv.style.display = 'none';

                    if (status.update_available) {
                        versionEl.innerHTML += ` <span style="color:var(--accent-yellow);">(Update available: ${status.latest_version})</span>`;
                    }
                } else {
                    indicator.style.background = 'var(--accent-red)';
                    indicator.style.boxShadow = '0 0 8px var(--accent-red)';
                    statusText.textContent = 'Not Installed';
                    versionEl.textContent = status.latest_version ? `Latest: ${status.latest_version}` : '';
                    pathEl.textContent = '';
                    installBtn.style.display = 'inline-block';
                    instructionsDiv.style.display = 'block';
                }
            } catch (e) {
                indicator.style.background = 'var(--text-secondary)';
                statusText.textContent = 'Unknown';
                versionEl.textContent = '';
                pathEl.textContent = '';
                console.error('Failed to check Trivy status:', e);
            }
        }

        async function loadTrivyInstructions() {
            try {
                const resp = await fetchWithAuth('/api/security/trivy/instructions');
                const data = await resp.json();
                document.getElementById('trivy-install-commands').textContent = data.instructions || '';
            } catch (e) {
                console.error('Failed to load Trivy instructions:', e);
            }
        }

        async function installTrivy() {
            const btn = document.getElementById('trivy-install-btn');
            const progressDiv = document.getElementById('trivy-install-progress');
            const progressBar = document.getElementById('trivy-progress-bar');
            const progressText = document.getElementById('trivy-progress-text');

            btn.disabled = true;
            btn.textContent = 'Installing...';
            progressDiv.style.display = 'block';
            progressBar.style.width = '10%';
            progressText.textContent = 'Starting download...';

            try {
                const resp = await fetchWithAuth('/api/security/trivy/install', { method: 'POST' });
                const result = await resp.json();

                if (result.success) {
                    progressBar.style.width = '100%';
                    progressText.textContent = result.message;
                    showToast('Trivy installed successfully', 'success');
                    setTimeout(() => {
                        checkTrivyStatus();
                        progressDiv.style.display = 'none';
                    }, 1500);
                } else {
                    progressBar.style.background = 'var(--accent-red)';
                    progressText.textContent = 'Error: ' + result.error;
                    showToast('Failed to install Trivy: ' + result.error, 'error');
                }
            } catch (e) {
                progressBar.style.background = 'var(--accent-red)';
                progressText.textContent = 'Installation failed';
                showToast('Failed to install Trivy', 'error');
            } finally {
                btn.disabled = false;
                btn.textContent = 'Install Trivy';
            }
        }

        async function runSecurityScan() {
            const resultDiv = document.getElementById('security-scan-result');
            resultDiv.style.display = 'block';
            resultDiv.innerHTML = '<div style="color:var(--text-secondary);"><span class="loading-spinner"></span> Running full security scan...</div>';

            try {
                const resp = await fetchWithAuth('/api/security/scan', { method: 'POST' });
                const result = await resp.json();

                if (result.error) {
                    resultDiv.innerHTML = `<div style="color:var(--accent-red);">Error: ${escapeHtml(result.error)}</div>`;
                    return;
                }

                // Display summary
                const critical = result.image_vulns?.severity_counts?.CRITICAL || 0;
                const high = result.image_vulns?.severity_counts?.HIGH || 0;
                const podIssues = result.pod_security_issues?.length || 0;
                const rbacIssues = result.rbac_issues?.length || 0;

                resultDiv.innerHTML = `
                    <div style="padding:12px;background:var(--bg-primary);border-radius:8px;border:1px solid var(--border-color);">
                        <div style="font-weight:600;margin-bottom:8px;">Scan Complete</div>
                        <div style="display:grid;grid-template-columns:repeat(4,1fr);gap:8px;">
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-red);">${critical}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Critical CVEs</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-yellow);">${high}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">High CVEs</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-purple);">${podIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Pod Issues</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-cyan);">${rbacIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">RBAC Issues</div>
                            </div>
                        </div>
                        <div style="margin-top:8px;font-size:11px;color:var(--text-secondary);">
                            Duration: ${result.duration || 'N/A'} | Score: ${(result.overall_score || 0).toFixed(1)}/100
                        </div>
                    </div>
                `;
                showToast('Security scan completed', 'success');
            } catch (e) {
                resultDiv.innerHTML = `<div style="color:var(--accent-red);">Failed to run security scan</div>`;
                showToast('Security scan failed', 'error');
            }
        }

        async function runQuickSecurityScan() {
            const resultDiv = document.getElementById('security-scan-result');
            resultDiv.style.display = 'block';
            resultDiv.innerHTML = '<div style="color:var(--text-secondary);"><span class="loading-spinner"></span> Running quick scan...</div>';

            try {
                const resp = await fetchWithAuth('/api/security/scan/quick', { method: 'POST' });
                const result = await resp.json();

                if (result.error) {
                    resultDiv.innerHTML = `<div style="color:var(--accent-red);">Error: ${escapeHtml(result.error)}</div>`;
                    return;
                }

                const podIssues = result.pod_security_issues?.length || 0;
                const rbacIssues = result.rbac_issues?.length || 0;
                const networkIssues = result.network_issues?.length || 0;

                resultDiv.innerHTML = `
                    <div style="padding:12px;background:var(--bg-primary);border-radius:8px;border:1px solid var(--border-color);">
                        <div style="font-weight:600;margin-bottom:8px;">Quick Scan Complete</div>
                        <div style="display:grid;grid-template-columns:repeat(3,1fr);gap:8px;">
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-purple);">${podIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Pod Issues</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-cyan);">${rbacIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">RBAC Issues</div>
                            </div>
                            <div style="text-align:center;padding:8px;background:var(--bg-tertiary);border-radius:4px;">
                                <div style="font-size:18px;font-weight:600;color:var(--accent-yellow);">${networkIssues}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">Network Issues</div>
                            </div>
                        </div>
                        <div style="margin-top:8px;font-size:11px;color:var(--text-secondary);">
                            Score: ${(result.overall_score || 0).toFixed(1)}/100
                        </div>
                    </div>
                `;
                showToast('Quick scan completed', 'success');
            } catch (e) {
                resultDiv.innerHTML = `<div style="color:var(--accent-red);">Failed to run quick scan</div>`;
                showToast('Quick scan failed', 'error');
            }
        }

        // LLM Connection Test Functions
        async function testLLMConnection() {
            const btn = document.getElementById('llm-test-btn');
            const btnText = document.getElementById('llm-test-btn-text');
            const indicator = document.getElementById('llm-status-indicator');
            const statusText = document.getElementById('llm-status-text');
            const statusDetail = document.getElementById('llm-status-detail');

            // Show testing state
            btn.disabled = true;
            btnText.textContent = 'Testing...';
            indicator.style.background = '#888';
            indicator.style.boxShadow = '0 0 8px rgba(136,136,136,0.5)';
            indicator.style.animation = 'pulse 1s infinite';
            statusText.textContent = 'Testing Connection...';
            statusDetail.textContent = 'Please wait...';

            // Get current form values to test
            const testConfig = {
                provider: document.getElementById('setting-llm-provider').value,
                model: document.getElementById('setting-llm-model').value,
                endpoint: document.getElementById('setting-llm-endpoint').value,
                api_key: document.getElementById('setting-llm-apikey').value
            };

            try {
                const resp = await fetchWithAuth('/api/llm/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(testConfig)
                });
                const status = await resp.json();

                if (status.connected) {
                    // Success - green light
                    indicator.style.background = '#10b981';
                    indicator.style.boxShadow = '0 0 12px rgba(16,185,129,0.8)';
                    indicator.style.animation = '';
                    statusText.textContent = 'Connection Successful';
                    statusText.style.color = 'var(--accent-green)';
                    statusDetail.textContent = `${status.provider} / ${status.model} - Response time: ${status.response_time_ms}ms`;
                } else {
                    // Failure - red light
                    indicator.style.background = '#ef4444';
                    indicator.style.boxShadow = '0 0 12px rgba(239,68,68,0.8)';
                    indicator.style.animation = '';
                    statusText.textContent = 'Connection Failed';
                    statusText.style.color = 'var(--accent-red)';
                    statusDetail.textContent = status.error || 'Unknown error';
                    if (status.message) {
                        statusDetail.textContent += ' - ' + status.message;
                    }
                }
            } catch (e) {
                // Error - red light
                indicator.style.background = '#ef4444';
                indicator.style.boxShadow = '0 0 12px rgba(239,68,68,0.8)';
                indicator.style.animation = '';
                statusText.textContent = 'Connection Error';
                statusText.style.color = 'var(--accent-red)';
                statusDetail.textContent = e.message || 'Failed to test connection';
            } finally {
                btn.disabled = false;
                btnText.textContent = 'Test Connection';
            }
        }

        async function loadLLMStatus() {
            try {
                const resp = await fetchWithAuth('/api/llm/status');
                const status = await resp.json();

                const indicator = document.getElementById('llm-status-indicator');
                const statusText = document.getElementById('llm-status-text');
                const statusDetail = document.getElementById('llm-status-detail');

                // Check if using embedded LLM - disable settings if so
                if (status.embedded_llm) {
                    indicator.style.background = 'var(--accent-green)';
                    indicator.style.boxShadow = '0 0 8px rgba(158,206,106,0.5)';
                    statusText.textContent = 'Embedded LLM Active';
                    statusText.style.color = 'var(--accent-green)';
                    statusDetail.textContent = `${status.provider} / ${status.model} (Local llama.cpp server)`;

                    // Disable all LLM settings inputs
                    disableLLMSettings(true, 'Embedded LLM is active. Settings are managed via CLI flags.');
                    return;
                }

                // Re-enable settings if not using embedded LLM
                disableLLMSettings(false);

                if (status.configured && status.ready) {
                    indicator.style.background = '#f59e0b';
                    indicator.style.boxShadow = '0 0 8px rgba(245,158,11,0.5)';
                    statusText.textContent = 'LLM Configured';
                    statusText.style.color = 'var(--accent-yellow)';
                    statusDetail.textContent = `${status.provider} / ${status.model} - Click 'Test Connection' to verify`;
                } else if (!status.configured) {
                    indicator.style.background = '#888';
                    indicator.style.boxShadow = '0 0 8px rgba(136,136,136,0.5)';
                    statusText.textContent = 'LLM Not Configured';
                    statusText.style.color = 'var(--text-secondary)';
                    statusDetail.textContent = 'Configure provider, model, and API key below';
                } else {
                    indicator.style.background = '#888';
                    indicator.style.boxShadow = '0 0 8px rgba(136,136,136,0.5)';
                    statusText.textContent = 'Configuration Incomplete';
                    statusText.style.color = 'var(--text-secondary)';
                    const missing = [];
                    if (!status.has_api_key) missing.push('API key');
                    if (!status.endpoint && !status.default_endpoint) missing.push('endpoint');
                    statusDetail.textContent = missing.length > 0 ? 'Missing: ' + missing.join(', ') : 'Check configuration';
                }
            } catch (e) {
                console.error('Failed to load LLM status:', e);
            }
        }

        function disableLLMSettings(disabled, message) {
            const settingsLLM = document.getElementById('settings-llm');
            if (!settingsLLM) return;

            const inputs = settingsLLM.querySelectorAll('input, select, button');
            inputs.forEach(input => {
                // Don't disable the test connection button
                if (input.id === 'llm-test-btn') return;
                input.disabled = disabled;
                input.style.opacity = disabled ? '0.5' : '1';
                input.style.cursor = disabled ? 'not-allowed' : '';
            });

            // Show/hide embedded LLM notice
            let notice = document.getElementById('embedded-llm-notice');
            if (disabled && message) {
                if (!notice) {
                    notice = document.createElement('div');
                    notice.id = 'embedded-llm-notice';
                    notice.style.cssText = 'margin:16px 0;padding:12px 16px;background:linear-gradient(135deg,rgba(158,206,106,0.15),rgba(122,162,247,0.15));border:1px solid rgba(158,206,106,0.3);border-radius:8px;display:flex;align-items:center;gap:12px;';
                    notice.innerHTML = `
                        <span style="font-size:24px;">🤖</span>
                        <div>
                            <div style="font-weight:600;color:var(--accent-green);margin-bottom:4px;">Embedded LLM Mode</div>
                            <div style="font-size:12px;color:var(--text-secondary);">${message}</div>
                        </div>
                    `;
                    const firstSection = settingsLLM.querySelector('.settings-section');
                    if (firstSection) {
                        firstSection.parentNode.insertBefore(notice, firstSection);
                    }
                }
            } else if (notice) {
                notice.remove();
            }

            // Hide Ollama setup section when embedded LLM is active
            const ollamaSection = document.getElementById('ollama-setup-section');
            if (ollamaSection) {
                ollamaSection.style.display = disabled ? 'none' : '';
            }
        }

        function updateEndpointPlaceholder() {
            const provider = document.getElementById('setting-llm-provider').value;
            const endpointInput = document.getElementById('setting-llm-endpoint');
            const hint = document.getElementById('endpoint-hint');

            const defaults = {
                'upstage': { placeholder: 'https://api.upstage.ai/v1', hint: '(Default: Upstage Solar API)', model: 'solar-pro2', apiKeyHint: 'up_...' },
                'openai': { placeholder: 'https://api.openai.com/v1', hint: '(Default: OpenAI API)', model: 'gpt-4', apiKeyHint: 'sk-...' },
                'ollama': { placeholder: 'http://localhost:11434', hint: '(Required for Ollama)', model: 'llama3', apiKeyHint: '' },
                'gemini': { placeholder: 'https://generativelanguage.googleapis.com/v1beta', hint: '(Default: Gemini API)', model: 'gemini-2.5-flash', apiKeyHint: 'AIza...' },
                'anthropic': { placeholder: 'https://api.anthropic.com', hint: '(Default: Anthropic API)', model: 'claude-3-opus', apiKeyHint: 'sk-ant-...' },
                'bedrock': { placeholder: '', hint: '(Uses AWS credentials)', model: '', apiKeyHint: '' },
                'azopenai': { placeholder: 'https://your-resource.openai.azure.com', hint: '(Azure resource endpoint required)', model: '', apiKeyHint: '' }
            };

            const config = defaults[provider] || { placeholder: '', hint: '', model: '', apiKeyHint: '' };
            endpointInput.placeholder = config.placeholder;
            hint.textContent = config.hint;

            // Update model value and placeholder when switching providers
            const modelInput = document.getElementById('setting-llm-model');
            if (modelInput) {
                modelInput.placeholder = config.model || '';
                // Always set model to provider default when switching
                if (config.model) {
                    modelInput.value = config.model;
                }
            }

            // Always set endpoint to provider default when switching
            if (config.placeholder) {
                endpointInput.value = config.placeholder;
            }

            // Update API key placeholder
            const apiKeyInput = document.getElementById('setting-llm-apikey');
            if (apiKeyInput && config.apiKeyHint) {
                apiKeyInput.placeholder = config.apiKeyHint;
            }

            // Update API key link based on provider
            const apiKeyLabel = apiKeyInput?.parentElement?.querySelector('label');
            if (apiKeyLabel) {
                const existingLink = apiKeyLabel.querySelector('a');
                if (existingLink) existingLink.remove();

                const links = {
                    'upstage': { url: 'https://console.upstage.ai/api-keys', text: 'Get API Key →' },
                    'openai': { url: 'https://platform.openai.com/api-keys', text: 'Get API Key →' },
                    'anthropic': { url: 'https://console.anthropic.com/settings/keys', text: 'Get API Key →' },
                    'gemini': { url: 'https://aistudio.google.com/app/apikey', text: 'Get API Key →' }
                };

                if (links[provider]) {
                    const link = document.createElement('a');
                    link.href = links[provider].url;
                    link.target = '_blank';
                    link.style.cssText = 'font-size:11px;color:var(--accent-blue);margin-left:8px;';
                    link.textContent = links[provider].text;
                    apiKeyLabel.appendChild(link);
                }
            }

            // Update reasoning effort UI visibility (only for Solar)
            updateReasoningEffortUI();

            // Show/hide "Fetch Models" button based on provider
            const fetchBtn = document.getElementById('fetch-models-btn');
            const fetchableProviders = ['gemini', 'ollama'];
            if (fetchBtn) {
                fetchBtn.style.display = fetchableProviders.includes(provider) ? 'inline' : 'none';
            }
            // Clear previous suggestions when switching providers
            const datalist = document.getElementById('model-suggestions');
            if (datalist) datalist.innerHTML = '';
        }

        async function fetchAvailableModels() {
            const provider = document.getElementById('setting-llm-provider').value;
            const apiKey = document.getElementById('setting-llm-apikey').value;
            const endpoint = document.getElementById('setting-llm-endpoint').value;
            const status = document.getElementById('fetch-models-status');
            const datalist = document.getElementById('model-suggestions');
            const btn = document.getElementById('fetch-models-btn');

            if (!apiKey && provider !== 'ollama') {
                status.textContent = 'API key required';
                status.style.color = 'var(--accent-red)';
                return;
            }

            btn.disabled = true;
            status.textContent = 'Fetching...';
            status.style.color = 'var(--text-secondary)';

            try {
                const params = new URLSearchParams({ provider, api_key: apiKey });
                if (endpoint) params.set('endpoint', endpoint);
                const resp = await fetchWithAuth('/api/llm/available-models?' + params);
                const data = await resp.json();

                if (data.error) {
                    status.textContent = data.error;
                    status.style.color = 'var(--accent-red)';
                    return;
                }

                const models = data.models || [];
                if (models.length === 0) {
                    status.textContent = 'No models found';
                    status.style.color = 'var(--accent-yellow)';
                    return;
                }

                datalist.innerHTML = models.map(m => `<option value="${escapeHtml(m)}">`).join('');
                status.textContent = `${models.length} models available`;
                status.style.color = 'var(--accent-green)';
            } catch (e) {
                status.textContent = 'Failed to fetch';
                status.style.color = 'var(--accent-red)';
            } finally {
                btn.disabled = false;
            }
        }

        // Model Management Functions
        async function loadModelProfiles() {
            try {
                const resp = await fetchWithAuth('/api/models');
                const data = await resp.json();
                const container = document.getElementById('models-list');

                if (!data.models || data.models.length === 0) {
                    container.innerHTML = '<p style="color:var(--text-secondary);">No model profiles configured.</p>';
                    return;
                }

                container.innerHTML = data.models.map(m => `
                    <div class="settings-row" style="background:var(--bg-primary);padding:12px;border-radius:8px;margin-bottom:8px;">
                        <div style="flex:1;">
                            <div style="font-weight:bold;display:flex;align-items:center;gap:8px;">
                                ${escapeHtml(m.name)}
                                ${m.is_active ? '<span style="background:var(--accent-green);color:var(--bg-primary);padding:2px 8px;border-radius:4px;font-size:10px;">ACTIVE</span>' : ''}
                            </div>
                            <div style="font-size:12px;color:var(--text-secondary);margin-top:4px;">
                                ${escapeHtml(m.provider)} / ${escapeHtml(m.model)} ${m.description ? '- ' + escapeHtml(m.description) : ''}
                            </div>
                        </div>
                        <div style="display:flex;gap:8px;">
                            ${!m.is_active ? `<button class="btn btn-secondary" onclick="switchModel('${escapeHtml(m.name)}')" style="padding:4px 12px;font-size:12px;">Use</button>` : ''}
                            <button class="btn btn-secondary" onclick="deleteModel('${escapeHtml(m.name)}')" style="padding:4px 12px;font-size:12px;color:var(--accent-red);">Delete</button>
                        </div>
                    </div>
                `).join('');
            } catch (e) {
                console.error('Failed to load models:', e);
            }
        }

        async function switchModel(name) {
            try {
                await fetchWithAuth('/api/models/active', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name })
                });
                loadModelProfiles();
                alert('Switched to model: ' + name);
            } catch (e) {
                alert('Failed to switch model: ' + e.message);
            }
        }

        async function deleteModel(name) {
            if (!confirm('Delete model profile "' + name + '"?')) return;
            try {
                await fetchWithAuth('/api/models?name=' + encodeURIComponent(name), {
                    method: 'DELETE'
                });
                loadModelProfiles();
            } catch (e) {
                alert('Failed to delete model: ' + e.message);
            }
        }

        function showAddModelForm() {
            document.getElementById('add-model-form').style.display = 'block';
        }

        function hideAddModelForm() {
            document.getElementById('add-model-form').style.display = 'none';
            // Clear form
            document.getElementById('new-model-name').value = '';
            document.getElementById('new-model-model').value = '';
            document.getElementById('new-model-endpoint').value = '';
            document.getElementById('new-model-apikey').value = '';
            document.getElementById('new-model-description').value = '';
        }

        async function addModelProfile() {
            const profile = {
                name: document.getElementById('new-model-name').value.trim(),
                provider: document.getElementById('new-model-provider').value,
                model: document.getElementById('new-model-model').value.trim(),
                endpoint: document.getElementById('new-model-endpoint').value.trim(),
                api_key: document.getElementById('new-model-apikey').value,
                description: document.getElementById('new-model-description').value.trim()
            };

            if (!profile.name || !profile.model) {
                alert('Name and Model are required');
                return;
            }

            try {
                await fetchWithAuth('/api/models', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(profile)
                });
                hideAddModelForm();
                loadModelProfiles();
            } catch (e) {
                alert('Failed to add model: ' + e.message);
            }
        }

        // MCP Management Functions
        async function loadMCPServers() {
            try {
                const resp = await fetchWithAuth('/api/mcp/servers');
                const data = await resp.json();
                const container = document.getElementById('mcp-servers-list');

                if (!data.servers || data.servers.length === 0) {
                    container.innerHTML = '<p style="color:var(--text-secondary);">No MCP servers configured.</p>';
                    return;
                }

                container.innerHTML = data.servers.map(s => `
                    <div class="settings-row" style="background:var(--bg-primary);padding:12px;border-radius:8px;margin-bottom:8px;">
                        <div style="flex:1;">
                            <div style="font-weight:bold;display:flex;align-items:center;gap:8px;">
                                ${escapeHtml(s.name)}
                                ${s.connected ? '<span style="background:var(--accent-green);color:var(--bg-primary);padding:2px 8px;border-radius:4px;font-size:10px;">CONNECTED</span>' : s.enabled ? '<span style="background:var(--accent-yellow);color:var(--bg-primary);padding:2px 8px;border-radius:4px;font-size:10px;">DISCONNECTED</span>' : '<span style="background:var(--bg-tertiary);padding:2px 8px;border-radius:4px;font-size:10px;">DISABLED</span>'}
                            </div>
                            <div style="font-size:12px;color:var(--text-secondary);margin-top:4px;">
                                ${escapeHtml(s.command)} ${s.args ? escapeHtml(s.args.join(' ')) : ''} ${s.description ? '- ' + escapeHtml(s.description) : ''}
                            </div>
                        </div>
                        <div style="display:flex;gap:8px;">
                            ${s.enabled ? `<button class="btn btn-secondary" onclick="toggleMCPServer('${s.name}', 'disable')" style="padding:4px 12px;font-size:12px;">Disable</button>` : `<button class="btn btn-secondary" onclick="toggleMCPServer('${s.name}', 'enable')" style="padding:4px 12px;font-size:12px;">Enable</button>`}
                            ${s.enabled ? `<button class="btn btn-secondary" onclick="toggleMCPServer('${s.name}', 'reconnect')" style="padding:4px 12px;font-size:12px;">Reconnect</button>` : ''}
                            <button class="btn btn-secondary" onclick="deleteMCPServer('${s.name}')" style="padding:4px 12px;font-size:12px;color:var(--accent-red);">Delete</button>
                        </div>
                    </div>
                `).join('');
            } catch (e) {
                console.error('Failed to load MCP servers:', e);
            }
        }

        async function loadMCPTools() {
            try {
                const resp = await fetchWithAuth('/api/mcp/tools');
                const data = await resp.json();
                const container = document.getElementById('mcp-tools-list');

                let html = '';

                if (data.builtin_tools && data.builtin_tools.length > 0) {
                    html += '<div style="margin-bottom:12px;"><strong>Built-in Tools:</strong></div>';
                    html += data.builtin_tools.map(t => `
                        <div style="background:var(--bg-primary);padding:8px 12px;border-radius:4px;margin-bottom:4px;font-size:12px;">
                            <span style="color:var(--accent-blue);">${t.name}</span>
                            <span style="color:var(--text-secondary);margin-left:8px;">${t.description || ''}</span>
                        </div>
                    `).join('');
                }

                if (data.mcp_tools && data.mcp_tools.length > 0) {
                    html += '<div style="margin:12px 0;"><strong>MCP Tools:</strong></div>';
                    html += data.mcp_tools.map(t => `
                        <div style="background:var(--bg-primary);padding:8px 12px;border-radius:4px;margin-bottom:4px;font-size:12px;">
                            <span style="color:var(--accent-purple);">${t.name}</span>
                            <span style="color:var(--text-secondary);margin-left:8px;">${t.description || ''}</span>
                            <span style="color:var(--accent-cyan);margin-left:8px;font-size:10px;">[${t.server}]</span>
                        </div>
                    `).join('');
                }

                if (!html) {
                    html = '<p style="color:var(--text-secondary);">No tools available.</p>';
                }

                container.innerHTML = html;
            } catch (e) {
                console.error('Failed to load MCP tools:', e);
            }
        }

        async function toggleMCPServer(name, action) {
            try {
                await fetchWithAuth('/api/mcp/servers', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, action })
                });
                loadMCPServers();
                loadMCPTools();
            } catch (e) {
                alert('Failed to ' + action + ' MCP server: ' + e.message);
            }
        }

        async function deleteMCPServer(name) {
            if (!confirm('Delete MCP server "' + name + '"?')) return;
            try {
                await fetchWithAuth('/api/mcp/servers?name=' + encodeURIComponent(name), {
                    method: 'DELETE'
                });
                loadMCPServers();
                loadMCPTools();
            } catch (e) {
                alert('Failed to delete MCP server: ' + e.message);
            }
        }

        function showAddMCPForm() {
            document.getElementById('add-mcp-form').style.display = 'block';
        }

        function hideAddMCPForm() {
            document.getElementById('add-mcp-form').style.display = 'none';
            // Clear form
            document.getElementById('new-mcp-name').value = '';
            document.getElementById('new-mcp-command').value = '';
            document.getElementById('new-mcp-args').value = '';
            document.getElementById('new-mcp-description').value = '';
            document.getElementById('new-mcp-enabled').checked = true;
        }

        async function addMCPServer() {
            const argsStr = document.getElementById('new-mcp-args').value.trim();
            const args = argsStr ? argsStr.split(',').map(a => a.trim()).filter(a => a) : [];

            const server = {
                name: document.getElementById('new-mcp-name').value.trim(),
                command: document.getElementById('new-mcp-command').value.trim(),
                args: args,
                description: document.getElementById('new-mcp-description').value.trim(),
                enabled: document.getElementById('new-mcp-enabled').checked
            };

            if (!server.name || !server.command) {
                alert('Name and Command are required');
                return;
            }

            try {
                const resp = await fetchWithAuth('/api/mcp/servers', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(server)
                });
                const data = await resp.json();
                hideAddMCPForm();
                loadMCPServers();
                loadMCPTools();
                if (data.warning) {
                    alert(data.warning);
                }
            } catch (e) {
                alert('Failed to add MCP server: ' + e.message);
            }
        }

        // === Feature Permissions ===
        let userPermissions = {};

        async function loadUserPermissions() {
            try {
                const resp = await fetchWithAuth('/api/auth/permissions');
                if (resp.ok) {
                    const data = await resp.json();
                    userPermissions = data.features || {};
                    applyFeaturePermissions();
                }
            } catch (e) {
                console.warn('Failed to load permissions:', e);
            }
        }

        function hasFeature(name) {
            if (!userPermissions || Object.keys(userPermissions).length === 0) return true;
            return userPermissions[name] === true;
        }

        function applyFeaturePermissions() {
            const featureMap = {
                'topology': 'topology',
                'reports': 'reports',
                'helm': 'helm',
                'security': 'security_scanning',
                'templates': 'templates',
            };
            document.querySelectorAll('.sidebar-item[data-view]').forEach(item => {
                const view = item.getAttribute('data-view');
                const feature = featureMap[view];
                if (feature && !hasFeature(feature)) {
                    item.style.display = 'none';
                }
            });
        }

        // === Roles Management ===
        async function loadRoles() {
            try {
                const resp = await fetchWithAuth('/api/roles');
                if (!resp.ok) return;
                const roles = await resp.json();
                const container = document.getElementById('roles-list-container');
                if (!container) return;

                let html = '<table class="data-table" style="width:100%;"><thead><tr><th>Role</th><th>Type</th><th>Features</th><th>Actions</th></tr></thead><tbody>';
                for (const role of roles) {
                    const type = role.is_custom ? '<span style="color:var(--accent-color);">Custom</span>' : 'Built-in';
                    const featureCount = role.allowed_features ? (role.allowed_features.includes('*') ? 'All' : role.allowed_features.length) : 0;
                    const actions = role.is_custom ? `<button class="btn btn-sm" onclick="editRole('${escapeHtml(role.name)}')">Edit</button> <button class="btn btn-sm btn-danger" onclick="deleteRole('${escapeHtml(role.name)}')">Delete</button>` : '<span style="color:var(--text-secondary);">Protected</span>';
                    html += `<tr><td><strong>${escapeHtml(role.name)}</strong></td><td>${type}</td><td>${featureCount}</td><td>${actions}</td></tr>`;
                }
                html += '</tbody></table>';
                container.innerHTML = html;
            } catch (e) {
                console.error('Failed to load roles:', e);
            }
        }

        async function showCreateRoleModal() {
            const allFeatures = ['dashboard','topology','reports','metrics','helm','terminal','rbac_viewer','network_policy','event_timeline','templates','ai_assistant','security_scanning','audit_logs','settings_general','settings_ai','settings_metrics','settings_mcp','settings_notifications'];

            let checkboxes = allFeatures.map(f => `<label style="display:block;margin:4px 0;"><input type="checkbox" value="${f}" checked> ${f.replace(/_/g, ' ')}</label>`).join('');

            const modal = document.createElement('div');
            modal.className = 'modal-overlay';
            modal.innerHTML = `<div class="modal-content" style="max-width:500px;max-height:80vh;overflow-y:auto;">
                <h3>Create Custom Role</h3>
                <div class="form-group"><label>Role Name</label><input type="text" id="new-role-name" class="form-control" placeholder="e.g., developer"></div>
                <div class="form-group"><label>Description</label><input type="text" id="new-role-desc" class="form-control" placeholder="e.g., Developer with limited access"></div>
                <div class="form-group"><label>Allowed Features</label><div id="new-role-features" style="max-height:300px;overflow-y:auto;border:1px solid var(--border-color);padding:8px;border-radius:4px;">${checkboxes}</div></div>
                <div style="display:flex;gap:8px;margin-top:16px;">
                    <button class="btn btn-primary" onclick="createRole()">Create</button>
                    <button class="btn" onclick="this.closest('.modal-overlay').remove()">Cancel</button>
                </div>
            </div>`;
            document.body.appendChild(modal);
        }

        async function createRole() {
            const name = document.getElementById('new-role-name').value.trim();
            const desc = document.getElementById('new-role-desc').value.trim();
            if (!name) { showToast('Role name is required', 'error'); return; }

            const features = [];
            document.querySelectorAll('#new-role-features input:checked').forEach(cb => features.push(cb.value));

            try {
                const resp = await fetchWithAuth('/api/roles', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, description: desc, allowed_features: features, is_custom: true })
                });
                if (resp.ok) {
                    showToast('Role created successfully');
                    document.querySelector('.modal-overlay').remove();
                    loadRoles();
                } else {
                    const err = await resp.text();
                    showToast(err, 'error');
                }
            } catch (e) {
                showToast('Failed to create role', 'error');
            }
        }

        async function deleteRole(name) {
            if (!confirm(`Delete role "${name}"?`)) return;
            try {
                const resp = await fetchWithAuth(`/api/roles/${name}`, { method: 'DELETE' });
                if (resp.ok) {
                    showToast('Role deleted');
                    loadRoles();
                } else {
                    showToast(await resp.text(), 'error');
                }
            } catch (e) {
                showToast('Failed to delete role', 'error');
            }
        }

        // === Tool Approval Settings ===
        async function loadToolApprovalSettings() {
            try {
                const resp = await fetchWithAuth('/api/settings/tool-approval');
                if (!resp.ok) return;
                const policy = await resp.json();

                const setToggle = (id, active) => {
                    const el = document.getElementById(id);
                    if (el) el.classList.toggle('active', active);
                };
                setToggle('ta-auto-approve-ro', policy.auto_approve_read_only !== false);
                setToggle('ta-require-write', policy.require_approval_for_write !== false);
                setToggle('ta-block-dangerous', policy.block_dangerous === true);
                setToggle('ta-require-unknown', policy.require_approval_for_unknown !== false);

                const timeout = document.getElementById('ta-timeout');
                if (timeout) timeout.value = policy.approval_timeout_seconds || 60;

                const patterns = document.getElementById('ta-blocked-patterns');
                if (patterns) patterns.value = (policy.blocked_patterns || []).join('\n');
            } catch (e) {
                console.error('Failed to load tool approval settings:', e);
            }
        }

        function toggleToolApproval(el) {
            el.classList.toggle('active');
        }

        async function saveToolApprovalSettings() {
            const policy = {
                auto_approve_read_only: document.getElementById('ta-auto-approve-ro')?.classList.contains('active') ?? true,
                require_approval_for_write: document.getElementById('ta-require-write')?.classList.contains('active') ?? true,
                block_dangerous: document.getElementById('ta-block-dangerous')?.classList.contains('active') ?? false,
                require_approval_for_unknown: document.getElementById('ta-require-unknown')?.classList.contains('active') ?? true,
                approval_timeout_seconds: parseInt(document.getElementById('ta-timeout')?.value) || 60,
                blocked_patterns: (document.getElementById('ta-blocked-patterns')?.value || '').split('\n').filter(l => l.trim()),
            };
            try {
                const resp = await fetchWithAuth('/api/settings/tool-approval', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(policy)
                });
                if (resp.ok) showToast('Tool approval settings saved');
                else showToast('Failed to save settings', 'error');
            } catch (e) {
                showToast('Failed to save settings', 'error');
            }
        }

        // === Agent Settings ===
        async function loadAgentSettings() {
            try {
                const resp = await fetchWithAuth('/api/settings/agent');
                if (!resp.ok) return;
                const data = await resp.json();

                const maxIter = document.getElementById('agent-max-iterations');
                if (maxIter) { maxIter.value = data.max_iterations || 10; document.getElementById('agent-max-iter-val').textContent = maxIter.value; }

                const effort = document.getElementById('agent-reasoning-effort');
                if (effort) effort.value = data.reasoning_effort || 'medium';

                const temp = document.getElementById('agent-temperature');
                if (temp) { temp.value = Math.round((data.temperature || 0.7) * 100); document.getElementById('agent-temp-val').textContent = (temp.value/100).toFixed(1); }

                const tokens = document.getElementById('agent-max-tokens');
                if (tokens) tokens.value = data.max_tokens || 4096;
            } catch (e) {
                console.error('Failed to load agent settings:', e);
            }
        }

        async function saveAgentSettings() {
            const settings = {
                max_iterations: parseInt(document.getElementById('agent-max-iterations')?.value) || 10,
                reasoning_effort: document.getElementById('agent-reasoning-effort')?.value || 'medium',
                temperature: parseInt(document.getElementById('agent-temperature')?.value || '70') / 100,
                max_tokens: parseInt(document.getElementById('agent-max-tokens')?.value) || 4096,
            };
            try {
                const resp = await fetchWithAuth('/api/settings/agent', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(settings)
                });
                if (resp.ok) showToast('Agent settings saved');
                else showToast('Failed to save settings', 'error');
            } catch (e) {
                showToast('Failed to save settings', 'error');
            }
        }

        // Admin User Management Functions
        async function loadAdminUsers() {
            try {
                const resp = await fetchWithAuth('/api/admin/users');
                if (!resp.ok) {
                    if (resp.status === 403) {
                        document.getElementById('admin-users-list').innerHTML = '<p style="color:var(--accent-red);">Access denied. Admin role required.</p>';
                        return;
                    }
                    throw new Error('Failed to load users');
                }
                const data = await resp.json();
                const container = document.getElementById('admin-users-list');

                if (!data.users || data.users.length === 0) {
                    container.innerHTML = '<p style="color:var(--text-secondary);">No users found.</p>';
                    return;
                }

                container.innerHTML = data.users.map(u => `
                    <div class="settings-row" style="background:var(--bg-primary);padding:12px;border-radius:8px;margin-bottom:8px;">
                        <div style="flex:1;">
                            <div style="font-weight:bold;display:flex;align-items:center;gap:8px;">
                                ${escapeHtml(u.username)}
                                <span style="background:${u.role === 'admin' ? 'var(--accent-red)' : u.role === 'user' ? 'var(--accent-blue)' : 'var(--bg-tertiary)'};color:${u.role === 'admin' || u.role === 'user' ? '#fff' : 'var(--text-primary)'};padding:2px 8px;border-radius:4px;font-size:10px;text-transform:uppercase;">${u.role}</span>
                                <span style="background:var(--bg-tertiary);padding:2px 8px;border-radius:4px;font-size:10px;">${u.source || 'local'}</span>
                            </div>
                            <div style="font-size:12px;color:var(--text-secondary);margin-top:4px;">
                                ${u.email ? escapeHtml(u.email) + ' · ' : ''}Last login: ${u.last_login ? new Date(u.last_login).toLocaleString() : 'Never'}
                            </div>
                        </div>
                        <div style="display:flex;gap:8px;">
                            ${u.source === 'local' ? `
                                <button class="btn btn-secondary" onclick="showResetPasswordModal('${escapeHtml(u.username)}')" style="padding:4px 12px;font-size:12px;">Reset Password</button>
                                <button class="btn btn-secondary" onclick="deleteUser('${escapeHtml(u.username)}')" style="padding:4px 12px;font-size:12px;color:var(--accent-red);">Delete</button>
                            ` : '<span style="font-size:11px;color:var(--text-secondary);">External user</span>'}
                        </div>
                    </div>
                `).join('');
            } catch (e) {
                console.error('Failed to load admin users:', e);
                document.getElementById('admin-users-list').innerHTML = '<p style="color:var(--accent-red);">Failed to load users.</p>';
            }
        }

        async function loadAuthStatus() {
            try {
                const resp = await fetchWithAuth('/api/admin/status');
                if (!resp.ok) return;
                const data = await resp.json();

                // Update current auth mode display
                const currentModeEl = document.getElementById('current-auth-mode');
                if (currentModeEl) {
                    const modeLabels = {
                        'local': 'Local (Username/Password)',
                        'token': 'Kubernetes Token',
                        'oidc': 'OIDC/OAuth SSO',
                        'ldap': 'LDAP/Active Directory'
                    };
                    currentModeEl.textContent = modeLabels[data.auth_mode] || data.auth_mode || 'Unknown';
                }

                // Set the auth mode select to current value
                const authModeSelect = document.getElementById('auth-mode');
                if (authModeSelect && data.auth_mode) {
                    authModeSelect.value = data.auth_mode;
                    onAuthModeChange(data.auth_mode);
                }

                // Load SSO settings if available
                if (data.oidc_configured) {
                    // Try to load OIDC settings
                    try {
                        const oidcResp = await fetchWithAuth('/api/settings/auth');
                        if (oidcResp.ok) {
                            const authConfig = await oidcResp.json();
                            if (authConfig.oidc) {
                                const oidc = authConfig.oidc;
                                if (document.getElementById('oidc-provider-url')) {
                                    document.getElementById('oidc-provider-url').value = oidc.provider_url || '';
                                }
                                if (document.getElementById('oidc-client-id')) {
                                    document.getElementById('oidc-client-id').value = oidc.client_id || '';
                                }
                                if (document.getElementById('oidc-scopes')) {
                                    document.getElementById('oidc-scopes').value = oidc.scopes || 'openid email profile';
                                }
                            }
                            if (authConfig.ldap) {
                                const ldap = authConfig.ldap;
                                if (document.getElementById('ldap-server-url')) {
                                    document.getElementById('ldap-server-url').value = ldap.server_url || '';
                                }
                                if (document.getElementById('ldap-bind-dn')) {
                                    document.getElementById('ldap-bind-dn').value = ldap.bind_dn || '';
                                }
                                if (document.getElementById('ldap-user-search-base')) {
                                    document.getElementById('ldap-user-search-base').value = ldap.user_search_base || '';
                                }
                            }
                        }
                    } catch (configErr) {
                        console.log('Auth config not available:', configErr);
                    }
                }
            } catch (e) {
                console.error('Failed to load auth status:', e);
            }
        }

        function showAddUserForm() {
            document.getElementById('add-user-form').style.display = 'block';
        }

        function hideAddUserForm() {
            document.getElementById('add-user-form').style.display = 'none';
            document.getElementById('new-user-username').value = '';
            document.getElementById('new-user-password').value = '';
            document.getElementById('new-user-email').value = '';
            document.getElementById('new-user-role').value = 'viewer';
        }

        async function addUser() {
            const user = {
                username: document.getElementById('new-user-username').value.trim(),
                password: document.getElementById('new-user-password').value,
                email: document.getElementById('new-user-email').value.trim(),
                role: document.getElementById('new-user-role').value
            };

            if (!user.username || !user.password) {
                alert('Username and password are required');
                return;
            }

            try {
                const resp = await fetchWithAuth('/api/admin/users', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(user)
                });

                if (!resp.ok) {
                    const error = await resp.text();
                    throw new Error(error);
                }

                hideAddUserForm();
                loadAdminUsers();
                alert('User created successfully');
            } catch (e) {
                alert('Failed to create user: ' + e.message);
            }
        }

        async function deleteUser(username) {
            if (!confirm('Delete user "' + username + '"? This action cannot be undone.')) return;

            try {
                const resp = await fetchWithAuth('/api/admin/users/' + encodeURIComponent(username), {
                    method: 'DELETE'
                });

                if (!resp.ok) {
                    const error = await resp.text();
                    throw new Error(error);
                }

                loadAdminUsers();
                alert('User deleted successfully');
            } catch (e) {
                alert('Failed to delete user: ' + e.message);
            }
        }

        function showResetPasswordModal(username) {
            const newPassword = prompt('Enter new password for ' + username + ':');
            if (!newPassword) return;

            resetUserPassword(username, newPassword);
        }

        async function resetUserPassword(username, newPassword) {
            try {
                const resp = await fetchWithAuth('/api/admin/reset-password', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ username, new_password: newPassword })
                });

                if (!resp.ok) {
                    const error = await resp.text();
                    throw new Error(error);
                }

                alert('Password reset successfully');
            } catch (e) {
                alert('Failed to reset password: ' + e.message);
            }
        }

        async function loadSettings() {
            try {
                const resp = await fetchWithAuth('/api/settings');
                const data = await resp.json();
                currentLanguage = data.language || 'ko';
                document.getElementById('setting-language').value = currentLanguage;
                document.getElementById('setting-log-level').value = data.log_level || 'info';
                // Load timezone setting
                if (data.timezone) {
                    appTimezone = data.timezone;
                    localStorage.setItem('k13d_timezone', appTimezone);
                }
                const tzSelect = document.getElementById('setting-timezone');
                if (tzSelect) tzSelect.value = appTimezone || 'auto';
                if (data.llm) {
                    const provider = data.llm.provider || 'upstage';
                    document.getElementById('setting-llm-provider').value = provider;

                    // Set model and endpoint with defaults based on provider
                    const defaults = {
                        'upstage': { model: 'solar-pro2', endpoint: 'https://api.upstage.ai/v1' },
                        'openai': { model: 'gpt-4', endpoint: 'https://api.openai.com/v1' },
                        'ollama': { model: 'qwen2.5:3b', endpoint: 'http://localhost:11434' },
                        'gemini': { model: 'gemini-pro', endpoint: 'https://generativelanguage.googleapis.com/v1beta' },
                        'anthropic': { model: 'claude-3-opus', endpoint: 'https://api.anthropic.com' }
                    };
                    const providerDefaults = defaults[provider] || { model: '', endpoint: '' };

                    document.getElementById('setting-llm-model').value = data.llm.model || providerDefaults.model;
                    document.getElementById('setting-llm-endpoint').value = data.llm.endpoint || providerDefaults.endpoint;
                    currentLLMModel = data.llm.model || providerDefaults.model;

                    // Load reasoning effort setting
                    if (data.llm.reasoning_effort) {
                        reasoningEffort = data.llm.reasoning_effort;
                        localStorage.setItem('k13d_reasoning_effort', reasoningEffort);
                    }
                } else {
                    // No LLM config from server, set Upstage defaults
                    document.getElementById('setting-llm-provider').value = 'upstage';
                    document.getElementById('setting-llm-model').value = 'solar-pro2';
                    document.getElementById('setting-llm-endpoint').value = 'https://api.upstage.ai/v1';
                    currentLLMModel = 'solar-pro2';
                }
                // Update endpoint placeholder based on current provider
                updateEndpointPlaceholder();
                // Load local settings
                updateSettingsUI();
                // Update AI panel status
                updateAIStatus();
                // Update UI language based on loaded settings
                updateUILanguage();
                // Load Prometheus settings
                loadPrometheusSettings();
            } catch (e) {
                console.error('Failed to load settings:', e);
            }
        }

        // Prometheus Settings Functions
        async function loadPrometheusSettings() {
            try {
                const resp = await fetchWithAuth('/api/prometheus/settings');
                const data = await resp.json();

                document.getElementById('prometheus-expose-metrics').checked = data.expose_metrics || false;
                document.getElementById('prometheus-external-url').value = data.external_url || '';
                document.getElementById('prometheus-collect-k8s').checked = data.collect_k8s_metrics !== false;
                document.getElementById('prometheus-collection-interval').value = data.collection_interval || 60;

                updatePrometheusExposeInfo();
                updatePrometheusStatus(data.expose_metrics, data.external_url);
            } catch (e) {
                console.error('Failed to load Prometheus settings:', e);
            }
        }

        function updatePrometheusExposeInfo() {
            const isChecked = document.getElementById('prometheus-expose-metrics').checked;
            document.getElementById('prometheus-expose-info').style.display = isChecked ? 'block' : 'none';
        }

        function updatePrometheusStatus(exposeEnabled, externalUrl) {
            const statusEl = document.getElementById('prometheus-status');
            if (!statusEl) return;

            if (externalUrl) {
                statusEl.classList.add('connected');
                statusEl.classList.remove('disconnected');
                statusEl.querySelector('span').textContent = 'Prometheus Connected';
            } else if (exposeEnabled) {
                statusEl.classList.remove('connected', 'disconnected');
                statusEl.querySelector('span').textContent = 'Prometheus: Exposing';
            } else {
                statusEl.classList.remove('connected');
                statusEl.classList.add('disconnected');
                statusEl.querySelector('span').textContent = 'Metrics Source';
            }

            // Check metrics-server availability
            fetchWithAuth('/api/metrics/nodes').then(resp => resp.json()).then(data => {
                if (!data.error && data.items && data.items.length > 0) {
                    // Check if real CPU/Memory data exists
                    const hasMetrics = data.items.some(n => (n.cpu || 0) > 0 || (n.memory || 0) > 0);
                    if (hasMetrics) {
                        statusEl.classList.add('connected');
                        statusEl.classList.remove('disconnected');
                        const currentText = statusEl.querySelector('span').textContent;
                        if (!currentText.includes('Prometheus')) {
                            statusEl.querySelector('span').textContent = 'metrics-server: Connected';
                        }
                    } else {
                        if (!statusEl.classList.contains('connected')) {
                            statusEl.querySelector('span').textContent = 'metrics-server: N/A';
                        }
                    }
                }
            }).catch(() => {});
        }

        async function testPrometheusConnection() {
            const url = document.getElementById('prometheus-external-url').value;
            const username = document.getElementById('prometheus-username').value;
            const password = document.getElementById('prometheus-password').value;
            const resultEl = document.getElementById('prometheus-test-result');

            if (!url) {
                resultEl.style.display = 'block';
                resultEl.style.background = 'rgba(247, 118, 142, 0.1)';
                resultEl.style.color = 'var(--accent-red)';
                resultEl.innerHTML = 'Please enter a Prometheus URL';
                return;
            }

            resultEl.style.display = 'block';
            resultEl.style.background = 'var(--bg-primary)';
            resultEl.style.color = 'var(--text-secondary)';
            resultEl.innerHTML = 'Testing connection...';

            try {
                const resp = await fetchWithAuth('/api/prometheus/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url, username, password })
                });
                const data = await resp.json();

                if (data.success) {
                    resultEl.style.background = 'rgba(158, 206, 106, 0.1)';
                    resultEl.style.color = 'var(--accent-green)';
                    resultEl.innerHTML = `✓ Connected successfully! Prometheus version: ${data.version || 'unknown'}`;
                } else {
                    resultEl.style.background = 'rgba(247, 118, 142, 0.1)';
                    resultEl.style.color = 'var(--accent-red)';
                    resultEl.innerHTML = `✗ Connection failed: ${data.error}`;
                }
            } catch (e) {
                resultEl.style.background = 'rgba(247, 118, 142, 0.1)';
                resultEl.style.color = 'var(--accent-red)';
                resultEl.innerHTML = `✗ Error: ${e.message}`;
            }
        }

        async function savePrometheusSettings() {
            const settings = {
                expose_metrics: document.getElementById('prometheus-expose-metrics').checked,
                external_url: document.getElementById('prometheus-external-url').value,
                username: document.getElementById('prometheus-username').value,
                password: document.getElementById('prometheus-password').value,
                collect_k8s_metrics: document.getElementById('prometheus-collect-k8s').checked,
                collection_interval: parseInt(document.getElementById('prometheus-collection-interval').value)
            };

            try {
                await fetchWithAuth('/api/prometheus/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(settings)
                });
                showToast(t('msg_settings_saved') || 'Settings saved');
                updatePrometheusStatus(settings.expose_metrics, settings.external_url);
            } catch (e) {
                showToast('Failed to save Prometheus settings', 'error');
            }
        }

        async function cleanupOldMetrics() {
            try {
                await fetchWithAuth('/api/metrics/collect', { method: 'POST' });
                showToast('Metrics cleanup initiated');
            } catch (e) {
                showToast('Failed to cleanup metrics', 'error');
            }
        }

        function toggleMetricsAutoRefresh() {
            const checkbox = document.getElementById('metrics-auto-refresh');
            if (checkbox.checked) {
                if (!metricsInterval) {
                    metricsInterval = setInterval(loadMetrics, 30000);
                }
            } else {
                if (metricsInterval) {
                    clearInterval(metricsInterval);
                    metricsInterval = null;
                }
            }
        }

        // Load version info for About page
        async function loadVersionInfo() {
            try {
                const resp = await fetch('/api/version');
                const data = await resp.json();

                const versionEl = document.getElementById('about-version');
                const buildTimeEl = document.getElementById('about-build-time');
                const gitCommitEl = document.getElementById('about-git-commit');

                if (versionEl) {
                    versionEl.textContent = data.version || 'dev';
                    // Add badge for dev version
                    if (data.version === 'dev') {
                        versionEl.innerHTML = '<span style="color: var(--accent-yellow);">dev</span> <span style="font-size: 10px; color: var(--text-muted);">(development build)</span>';
                    }
                }
                if (buildTimeEl) {
                    if (data.build_time && data.build_time !== 'unknown') {
                        // Format the date nicely
                        const date = new Date(data.build_time);
                        if (!isNaN(date.getTime())) {
                            buildTimeEl.textContent = date.toLocaleString();
                        } else {
                            buildTimeEl.textContent = data.build_time;
                        }
                    } else {
                        buildTimeEl.textContent = '-';
                    }
                }
                if (gitCommitEl) {
                    if (data.git_commit && data.git_commit !== 'unknown') {
                        // Show shortened commit hash
                        const shortCommit = data.git_commit.substring(0, 7);
                        gitCommitEl.textContent = shortCommit;
                        gitCommitEl.title = data.git_commit;
                    } else {
                        gitCommitEl.textContent = '-';
                    }
                }
            } catch (e) {
                console.error('Failed to load version info:', e);
            }
        }

        // Update AI Assistant panel with model name and connection status
        async function updateAIStatus() {
            const statusDot = document.getElementById('ai-status-dot');
            const modelBadge = document.getElementById('ai-model-badge');

            if (!statusDot || !modelBadge) return;

            // Show checking state
            statusDot.className = 'ai-status-dot checking';
            statusDot.title = 'Checking connection...';

            try {
                // Get LLM settings to display model name
                const settingsResp = await fetchWithAuth('/api/settings');
                const settings = await settingsResp.json();

                if (settings.llm && settings.llm.model) {
                    currentLLMModel = settings.llm.model;
                    modelBadge.textContent = currentLLMModel;
                    modelBadge.title = `${settings.llm.provider || 'openai'}: ${currentLLMModel}`;
                } else {
                    modelBadge.textContent = 'Not configured';
                    modelBadge.title = 'AI model not configured';
                    statusDot.className = 'ai-status-dot disconnected';
                    statusDot.title = 'AI not configured';
                    llmConnected = false;
                    return;
                }

                // Ping test - try to check LLM connection
                const pingResp = await fetchWithAuth('/api/ai/ping');
                if (pingResp.ok) {
                    statusDot.className = 'ai-status-dot connected';
                    statusDot.title = 'Connected';
                    llmConnected = true;
                } else {
                    statusDot.className = 'ai-status-dot disconnected';
                    statusDot.title = 'Connection failed';
                    llmConnected = false;
                }
            } catch (e) {
                console.error('Failed to check AI status:', e);
                statusDot.className = 'ai-status-dot disconnected';
                statusDot.title = 'Connection error';
                modelBadge.textContent = 'Error';
                llmConnected = false;
            }
        }

        function updateSettingsUI() {
            const streamingToggle = document.getElementById('setting-streaming');
            const autoRefreshToggle = document.getElementById('setting-auto-refresh');
            const intervalSelect = document.getElementById('setting-refresh-interval');

            if (streamingToggle) {
                streamingToggle.classList.toggle('active', useStreaming);
            }
            if (autoRefreshToggle) {
                autoRefreshToggle.classList.toggle('active', autoRefreshEnabled);
            }
            if (intervalSelect) {
                intervalSelect.value = autoRefreshInterval;
            }
        }

        function toggleStreamingSetting() {
            useStreaming = !useStreaming;
            localStorage.setItem('k13d_use_streaming', useStreaming);
            updateSettingsUI();
        }

        function toggleAutoRefreshSetting() {
            toggleAutoRefresh();
            updateSettingsUI();
        }

        function setAutoRefreshIntervalSetting(value) {
            setAutoRefreshInterval(parseInt(value));
            updateSettingsUI();
        }

        function toggleReasoningEffort() {
            reasoningEffort = reasoningEffort === 'minimal' ? 'high' : 'minimal';
            localStorage.setItem('k13d_reasoning_effort', reasoningEffort);
            updateReasoningEffortUI();
            // Save to server config
            saveReasoningEffortToServer();
        }

        function updateReasoningEffortUI() {
            const toggle = document.getElementById('reasoning-effort-toggle');
            const status = document.getElementById('reasoning-effort-status');
            const section = document.getElementById('reasoning-effort-section');
            const provider = document.getElementById('setting-llm-provider')?.value;

            // Show/hide section based on provider (only for upstage)
            if (section) {
                section.style.display = (provider === 'upstage') ? 'block' : 'none';
            }

            if (toggle) {
                toggle.classList.toggle('active', reasoningEffort === 'high');
            }
            if (status) {
                status.textContent = reasoningEffort === 'high'
                    ? 'Current: high (deeper reasoning enabled)'
                    : 'Current: minimal (default)';
            }
        }

        async function saveReasoningEffortToServer() {
            try {
                await fetchWithAuth('/api/config', {
                    method: 'PATCH',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        llm: { reasoning_effort: reasoningEffort }
                    })
                });
            } catch (e) {
                console.error('Failed to save reasoning effort:', e);
            }
        }

        // SSO/Authentication settings handlers
        function onAuthModeChange(mode) {
            const oidcSection = document.getElementById('oidc-settings');
            const ldapSection = document.getElementById('ldap-settings');
            const oauthRoleSection = document.getElementById('oauth-role-settings');

            // Hide all sections first
            if (oidcSection) oidcSection.style.display = 'none';
            if (ldapSection) ldapSection.style.display = 'none';
            if (oauthRoleSection) oauthRoleSection.style.display = 'none';

            // Show relevant section based on mode
            if (mode === 'oidc') {
                if (oidcSection) oidcSection.style.display = 'block';
                if (oauthRoleSection) oauthRoleSection.style.display = 'block';
            } else if (mode === 'ldap') {
                if (ldapSection) ldapSection.style.display = 'block';
            }
        }

        function toggleAllowPasswordLogin() {
            const toggle = document.getElementById('allow-password-login');
            if (toggle) {
                toggle.classList.toggle('active');
            }
        }

        function toggleEnableSignup() {
            const toggle = document.getElementById('enable-signup');
            if (toggle) {
                toggle.classList.toggle('active');
            }
        }

        async function testLDAPConnection() {
            const btn = event.target;
            const originalText = btn.textContent;
            btn.textContent = 'Testing...';
            btn.disabled = true;

            try {
                const config = {
                    server_url: document.getElementById('ldap-server-url')?.value || '',
                    bind_dn: document.getElementById('ldap-bind-dn')?.value || '',
                    bind_password: document.getElementById('ldap-bind-password')?.value || '',
                    user_search_base: document.getElementById('ldap-user-search-base')?.value || '',
                    user_search_filter: document.getElementById('ldap-user-search-filter')?.value || ''
                };

                const resp = await fetchWithAuth('/api/settings/auth/ldap/test', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(config)
                });

                const result = await resp.json();
                if (result.success) {
                    alert('LDAP connection successful!\n\nServer: ' + config.server_url);
                } else {
                    alert('LDAP connection failed:\n' + (result.error || 'Unknown error'));
                }
            } catch (e) {
                alert('LDAP connection test failed:\n' + e.message);
            } finally {
                btn.textContent = originalText;
                btn.disabled = false;
            }
        }

        function getAuthSettings() {
            const mode = document.getElementById('auth-mode')?.value || 'local';
            const settings = { mode };

            if (mode === 'oidc') {
                settings.oidc = {
                    provider_url: document.getElementById('oidc-provider-url')?.value || '',
                    client_id: document.getElementById('oidc-client-id')?.value || '',
                    client_secret: document.getElementById('oidc-client-secret')?.value || '',
                    scopes: document.getElementById('oidc-scopes')?.value || 'openid profile email',
                    redirect_uri: document.getElementById('oidc-redirect-uri')?.value || ''
                };
                settings.oauth_roles = {
                    roles_claim: document.getElementById('oauth-roles-claim')?.value || 'roles',
                    admin_roles: document.getElementById('oauth-admin-roles')?.value || '',
                    allowed_roles: document.getElementById('oauth-allowed-roles')?.value || ''
                };
                settings.allow_password_login = document.getElementById('allow-password-login')?.classList.contains('active') || false;
                settings.enable_signup = document.getElementById('enable-signup')?.classList.contains('active') || false;
            } else if (mode === 'ldap') {
                settings.ldap = {
                    server_url: document.getElementById('ldap-server-url')?.value || '',
                    bind_dn: document.getElementById('ldap-bind-dn')?.value || '',
                    bind_password: document.getElementById('ldap-bind-password')?.value || '',
                    user_search_base: document.getElementById('ldap-user-search-base')?.value || '',
                    user_search_filter: document.getElementById('ldap-user-search-filter')?.value || '(uid={{username}})',
                    group_search_base: document.getElementById('ldap-group-search-base')?.value || '',
                    group_search_filter: document.getElementById('ldap-group-search-filter')?.value || '',
                    admin_group: document.getElementById('ldap-admin-group')?.value || ''
                };
            }

            return settings;
        }

        async function saveAuthSettings() {
            try {
                const authSettings = getAuthSettings();
                const resp = await fetchWithAuth('/api/settings/auth', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(authSettings)
                });

                if (!resp.ok) {
                    const error = await resp.text();
                    throw new Error(error);
                }

                alert('Authentication settings saved!\n\nNote: Server restart may be required for changes to take effect.');
            } catch (e) {
                alert('Failed to save authentication settings:\n' + e.message);
            }
        }

        async function saveSettings() {
            try {
                // Save general settings (including timezone)
                const newTimezone = document.getElementById('setting-timezone')?.value || 'auto';
                await fetchWithAuth('/api/settings', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        language: document.getElementById('setting-language').value,
                        log_level: document.getElementById('setting-log-level').value,
                        timezone: newTimezone
                    })
                });
                // Apply timezone immediately
                appTimezone = newTimezone;
                localStorage.setItem('k13d_timezone', appTimezone);

                // Save LLM settings
                const apiKey = document.getElementById('setting-llm-apikey').value;
                const llmResp = await fetchWithAuth('/api/settings/llm', {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        provider: document.getElementById('setting-llm-provider').value,
                        model: document.getElementById('setting-llm-model').value,
                        endpoint: document.getElementById('setting-llm-endpoint').value,
                        api_key: apiKey,
                        reasoning_effort: reasoningEffort
                    })
                });

                if (!llmResp.ok) {
                    const errData = await llmResp.json().catch(() => ({}));
                    const errMsg = errData.message || errData.error || `LLM settings error (${llmResp.status})`;
                    showToast(errMsg, 'error');
                    return;
                }

                // Update current language for AI responses
                currentLanguage = document.getElementById('setting-language').value;
                updateUILanguage();

                closeSettings();
                showToast(t('msg_settings_saved'));

                // Update AI status (model name and connection status)
                updateAIStatus();
            } catch (e) {
                alert('Failed to save settings');
            }
        }

        // Audit logs and reports
        let auditFilter = { onlyLLM: false, onlyErrors: false };

        async function showAuditLogs() {
            document.getElementById('audit-modal').classList.add('active');
            // Sync filter checkboxes
            document.getElementById('audit-filter-llm').checked = auditFilter.onlyLLM;
            document.getElementById('audit-filter-errors').checked = auditFilter.onlyErrors;
            loadAuditModalData();
        }

        function closeAuditModal() {
            document.getElementById('audit-modal').classList.remove('active');
        }

        async function loadAuditModalData() {
            const body = document.getElementById('audit-modal-body');
            body.innerHTML = '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--text-secondary);">Loading audit logs...</td></tr>';

            try {
                let params = new URLSearchParams();
                if (auditFilter.onlyLLM) params.append('only_llm', 'true');
                if (auditFilter.onlyErrors) params.append('only_errors', 'true');

                const resp = await fetchWithAuth('/api/audit?' + params.toString());
                if (!resp.ok) {
                    const errText = await resp.text();
                    throw new Error(errText || `HTTP ${resp.status}`);
                }
                const data = await resp.json();

                document.getElementById('audit-entry-count').textContent =
                    `Showing ${data.logs ? data.logs.length : 0} entries`;

                if (data.logs && data.logs.length > 0) {
                    body.innerHTML = data.logs.map(log => {
                        const isLLM = log.action_type === 'llm' || log.llm_tool;
                        const statusBadge = log.success
                            ? '<span style="color: var(--accent-green);">✓</span>'
                            : '<span style="color: var(--accent-red);">✗</span>';
                        const actionBadge = getActionBadge(log.action, log.action_type);
                        const llmDetails = isLLM && log.llm_tool
                            ? `<div style="margin-top:5px;padding:5px;background:var(--bg-tertiary);border-radius:4px;font-size:11px;">
                                <strong>LLM Tool:</strong> ${escapeHtml(log.llm_tool)}<br>
                                <strong>Command:</strong> <code style="color:var(--accent-yellow);">${escapeHtml(log.llm_command || 'N/A')}</code><br>
                                <strong>Approved:</strong> ${log.llm_approved ? 'Yes' : 'No'}
                                ${log.llm_request ? `<br><strong>Question:</strong> ${escapeHtml(truncateText(log.llm_request, 100))}` : ''}
                              </div>`
                            : '';
                        const errorInfo = log.error_msg
                            ? `<div style="color:var(--accent-red);margin-top:3px;font-size:11px;">Error: ${escapeHtml(log.error_msg)}</div>`
                            : '';

                        return `
                            <tr style="${!log.success ? 'background: rgba(239,68,68,0.1);' : (isLLM ? 'background: rgba(59,130,246,0.05);' : '')}">
                                <td style="white-space:nowrap;padding:8px 12px;">${formatDateTime(log.timestamp)}</td>
                                <td style="padding:8px 12px;">${escapeHtml(log.user || 'anonymous')}</td>
                                <td style="padding:8px 12px;color:var(--accent-cyan);">${escapeHtml(log.k8s_user || '-')}</td>
                                <td style="padding:8px 12px;">${actionBadge}</td>
                                <td style="padding:8px 12px;">${escapeHtml(log.resource)}</td>
                                <td style="padding:8px 12px;"><span style="padding:2px 6px;border-radius:3px;background:var(--bg-tertiary);font-size:11px;">${escapeHtml(log.source || 'unknown')}</span></td>
                                <td style="text-align:center;padding:8px 12px;">${statusBadge}</td>
                                <td style="padding:8px 12px;">
                                    ${escapeHtml(log.details)}
                                    ${llmDetails}
                                    ${errorInfo}
                                </td>
                            </tr>
                        `;
                    }).join('');
                } else {
                    body.innerHTML =
                        '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--text-secondary);">No audit logs found</td></tr>';
                }
            } catch (e) {
                console.error('Failed to load audit logs:', e);
                body.innerHTML =
                    '<tr><td colspan="8" style="text-align:center;padding:40px;color:var(--accent-red);">Failed to load audit logs</td></tr>';
            }
        }

        function toggleAuditFilter(filterName) {
            auditFilter[filterName] = !auditFilter[filterName];
            loadAuditModalData();
        }

        function getActionBadge(action, actionType) {
            const colors = {
                'llm': { bg: 'rgba(59,130,246,0.2)', color: 'var(--accent-blue)', icon: '🤖' },
                'mutation': { bg: 'rgba(234,179,8,0.2)', color: 'var(--accent-yellow)', icon: '⚡' },
                'auth': { bg: 'rgba(139,92,246,0.2)', color: 'var(--accent-purple)', icon: '🔐' },
                'config': { bg: 'rgba(34,197,94,0.2)', color: 'var(--status-running)', icon: '⚙️' }
            };
            const style = colors[actionType] || colors['mutation'];
            return `<span style="padding:2px 8px;border-radius:4px;background:${style.bg};color:${style.color};font-size:12px;">${style.icon} ${escapeHtml(action)}</span>`;
        }

        function truncateText(text, maxLen) {
            if (!text) return '';
            return text.length > maxLen ? text.substring(0, maxLen) + '...' : text;
        }

        // ==========================================
        // Topology View Functions
        // ==========================================

        let topologyGraph = null;
        let topologyData = null;
        let topologySelectedNode = null;
        let topologyFocusNodeId = null; // When set, show subgraph around this resource

        const topologyStatusColors = {
            running: '#9ece6a',
            pending: '#e0af68',
            failed: '#f7768e',
            succeeded: '#7aa2f7',
            unknown: '#a9b1d6',
        };

        const topologyKindShapes = {
            Deployment: 'rect',
            ReplicaSet: 'rect',
            StatefulSet: 'rect',
            DaemonSet: 'rect',
            Pod: 'circle',
            Service: 'diamond',
            Ingress: 'diamond',
            Job: 'rect',
            CronJob: 'rect',
            ConfigMap: 'triangle',
            Secret: 'triangle',
            PVC: 'rect',
            HPA: 'diamond',
        };

        const topologyKindLabels = {
            Deployment: 'Deploy',
            ReplicaSet: 'RS',
            StatefulSet: 'STS',
            DaemonSet: 'DS',
            Pod: 'Pod',
            Service: 'Svc',
            Ingress: 'Ing',
            Job: 'Job',
            CronJob: 'CJ',
            ConfigMap: 'CM',
            Secret: 'Sec',
            PVC: 'PVC',
            HPA: 'HPA',
        };

        const topologyEdgeStyles = {
            owns: { lineDash: 0, stroke: '#565f89' },
            selects: { lineDash: [5, 5], stroke: '#7aa2f7' },
            mounts: { lineDash: [2, 4], stroke: '#bb9af7' },
            routes: { lineDash: 0, stroke: '#9ece6a' },
            scales: { lineDash: [8, 4], stroke: '#e0af68' },
        };

        function hideTopologyView() {
            const topoContainer = document.getElementById('topology-container');
            const mainPanel = document.querySelector('.main-panel');
            if (topoContainer) topoContainer.style.display = 'none';
            if (mainPanel) mainPanel.style.display = '';
        }

        function showTopology() {
            currentResource = 'topology';
            document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
            const topoNav = document.querySelector('.nav-item[data-resource="topology"]');
            if (topoNav) topoNav.classList.add('active');

            // Hide main panel, custom views and overview, show topology
            hideOverviewPanel();
            hideAllCustomViews();
            const mainPanel = document.querySelector('.main-panel');
            const topoContainer = document.getElementById('topology-container');
            if (mainPanel) mainPanel.style.display = 'none';
            if (topoContainer) topoContainer.style.display = 'flex';

            // Sync namespace select
            syncTopologyNamespaces();

            loadTopology();
        }

        function syncTopologyNamespaces() {
            const srcSelect = document.getElementById('namespace-select');
            const topoSelect = document.getElementById('topology-ns-select');
            if (!srcSelect || !topoSelect) return;

            // Copy options from main namespace select
            topoSelect.innerHTML = '';
            for (const opt of srcSelect.options) {
                const newOpt = document.createElement('option');
                newOpt.value = opt.value;
                newOpt.textContent = opt.textContent;
                topoSelect.appendChild(newOpt);
            }
            topoSelect.value = srcSelect.value;
        }

        function onTopologyNamespaceChange() {
            loadTopology();
        }

        async function loadTopology() {
            const namespace = document.getElementById('topology-ns-select')?.value || '';
            const kindFilter = document.getElementById('topology-kind-filter')?.value || '';
            const graphContainer = document.getElementById('topology-graph');
            if (!graphContainer) return;

            try {
                const resp = await fetchWithAuth(`/api/topology/?namespace=${encodeURIComponent(namespace)}`);
                const data = await resp.json();
                topologyData = data;

                // Filter based on checkboxes
                const showCM = document.getElementById('topology-show-configmaps')?.checked;
                const showSec = document.getElementById('topology-show-secrets')?.checked;

                let filteredNodes = data.nodes || [];
                let filteredEdges = data.edges || [];

                if (!showCM) {
                    const cmIds = new Set(filteredNodes.filter(n => n.kind === 'ConfigMap').map(n => n.id));
                    filteredNodes = filteredNodes.filter(n => n.kind !== 'ConfigMap');
                    filteredEdges = filteredEdges.filter(e => !cmIds.has(e.source) && !cmIds.has(e.target));
                }
                if (!showSec) {
                    const secIds = new Set(filteredNodes.filter(n => n.kind === 'Secret').map(n => n.id));
                    filteredNodes = filteredNodes.filter(n => n.kind !== 'Secret');
                    filteredEdges = filteredEdges.filter(e => !secIds.has(e.source) && !secIds.has(e.target));
                }

                // Kind filter: show selected kind + all connected resources
                if (kindFilter) {
                    const kindNodeIds = new Set(filteredNodes.filter(n => n.kind === kindFilter).map(n => n.id));
                    const connectedIds = new Set(kindNodeIds);
                    filteredEdges.forEach(e => {
                        if (kindNodeIds.has(e.source)) connectedIds.add(e.target);
                        if (kindNodeIds.has(e.target)) connectedIds.add(e.source);
                    });
                    filteredNodes = filteredNodes.filter(n => connectedIds.has(n.id));
                    filteredEdges = filteredEdges.filter(e => connectedIds.has(e.source) && connectedIds.has(e.target));
                }

                // Resource focus: show subgraph reachable from the focused resource
                if (topologyFocusNodeId) {
                    const focusResult = extractSubgraph(filteredNodes, filteredEdges, topologyFocusNodeId);
                    filteredNodes = focusResult.nodes;
                    filteredEdges = focusResult.edges;
                }

                renderTopologyGraph(filteredNodes, filteredEdges);

                // After rendering, highlight the focused node
                if (topologyFocusNodeId && topologyGraph) {
                    try {
                        topologyGraph.setElementState(topologyFocusNodeId, ['selected']);
                    } catch (e) { /* node may not exist */ }
                }
            } catch (err) {
                graphContainer.innerHTML = `<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--accent-red);">Failed to load topology: ${escapeHtml(err.message)}</div>`;
            }
        }

        // Extract the connected subgraph reachable from a root node (BFS in both directions)
        function extractSubgraph(nodes, edges, rootId) {
            const visited = new Set([rootId]);
            const queue = [rootId];
            // Walk up: find ancestors (who owns/selects/routes to this node)
            // Walk down: find descendants (what this node owns/selects/routes)
            while (queue.length > 0) {
                const current = queue.shift();
                edges.forEach(e => {
                    if (e.source === current && !visited.has(e.target)) {
                        visited.add(e.target);
                        queue.push(e.target);
                    }
                    if (e.target === current && !visited.has(e.source)) {
                        visited.add(e.source);
                        queue.push(e.source);
                    }
                });
            }
            return {
                nodes: nodes.filter(n => visited.has(n.id)),
                edges: edges.filter(e => visited.has(e.source) && visited.has(e.target)),
            };
        }

        function renderTopologyGraph(nodes, edges) {
            const container = document.getElementById('topology-graph');
            if (!container) return;

            // Destroy previous graph
            if (topologyGraph) {
                topologyGraph.destroy();
                topologyGraph = null;
            }

            if (!nodes || nodes.length === 0) {
                container.innerHTML = '<div style="display:flex;align-items:center;justify-content:center;height:100%;color:var(--text-secondary);">No resources found in this namespace</div>';
                return;
            }

            // Clear container
            container.innerHTML = '';

            // Transform data for G6
            const g6Nodes = nodes.map(n => ({
                id: n.id,
                data: {
                    kind: n.kind,
                    name: n.name,
                    namespace: n.namespace,
                    status: n.status,
                    info: n.info || {},
                    kindLabel: topologyKindLabels[n.kind] || n.kind,
                },
            }));

            const g6Edges = edges.map((e, i) => ({
                id: `edge-${i}`,
                source: e.source,
                target: e.target,
                data: { type: e.type },
            }));

            const statusColor = (status) => topologyStatusColors[status] || topologyStatusColors.unknown;

            topologyGraph = new G6.Graph({
                container,
                autoFit: 'view',
                padding: [40, 40, 40, 40],
                data: { nodes: g6Nodes, edges: g6Edges },
                node: {
                    type: (d) => {
                        const shape = topologyKindShapes[d.data?.kind] || 'circle';
                        return shape;
                    },
                    style: {
                        size: (d) => {
                            const kind = d.data?.kind;
                            if (kind === 'Pod') return 36;
                            if (kind === 'Service' || kind === 'Ingress' || kind === 'HPA') return 40;
                            if (kind === 'ConfigMap' || kind === 'Secret') return 36;
                            return [110, 44]; // rect: wider for label text
                        },
                        fill: (d) => {
                            const color = statusColor(d.data?.status);
                            return color + '33'; // 20% opacity
                        },
                        stroke: (d) => statusColor(d.data?.status),
                        lineWidth: 2,
                        labelText: (d) => {
                            const label = d.data?.kindLabel || '';
                            const name = d.data?.name || '';
                            const kind = d.data?.kind;
                            // Shorter truncation for non-rect shapes
                            const isCompact = (kind === 'Pod' || kind === 'Service' || kind === 'Ingress' || kind === 'HPA' || kind === 'ConfigMap' || kind === 'Secret');
                            const maxLen = isCompact ? 10 : 14;
                            const shortName = name.length > maxLen ? name.substring(0, maxLen - 2) + '..' : name;
                            return isCompact ? `${label}\n${shortName}` : `${label}: ${shortName}`;
                        },
                        labelFill: '#c0caf5',
                        labelFontSize: (d) => {
                            const kind = d.data?.kind;
                            const isCompact = (kind === 'Pod' || kind === 'ConfigMap' || kind === 'Secret');
                            return isCompact ? 9 : 10;
                        },
                        labelFontFamily: 'SF Mono, Monaco, Consolas, Liberation Mono, monospace',
                        labelPlacement: (d) => {
                            // Place label below for small shapes so text doesn't overflow
                            const kind = d.data?.kind;
                            if (kind === 'Pod' || kind === 'Service' || kind === 'Ingress' || kind === 'HPA' || kind === 'ConfigMap' || kind === 'Secret') return 'bottom';
                            return 'center';
                        },
                        labelMaxLines: 2,
                        labelWordWrap: true,
                        labelWordWrapWidth: 100,
                        labelOffsetY: (d) => {
                            const kind = d.data?.kind;
                            if (kind === 'Pod' || kind === 'Service' || kind === 'Ingress' || kind === 'HPA' || kind === 'ConfigMap' || kind === 'Secret') return 8;
                            return 0;
                        },
                    },
                    state: {
                        highlight: {
                            stroke: '#7aa2f7',
                            lineWidth: 3,
                            shadowColor: '#7aa2f7',
                            shadowBlur: 10,
                        },
                        dim: {
                            opacity: 0.3,
                        },
                        selected: {
                            stroke: '#7dcfff',
                            lineWidth: 3,
                            shadowColor: '#7dcfff',
                            shadowBlur: 12,
                        },
                    },
                },
                edge: {
                    type: 'line',
                    style: {
                        stroke: (d) => {
                            const es = topologyEdgeStyles[d.data?.type] || topologyEdgeStyles.owns;
                            return es.stroke;
                        },
                        lineDash: (d) => {
                            const es = topologyEdgeStyles[d.data?.type] || topologyEdgeStyles.owns;
                            return es.lineDash || 0;
                        },
                        lineWidth: 1,
                        endArrow: true,
                        endArrowSize: 6,
                    },
                    state: {
                        dim: {
                            opacity: 0.15,
                        },
                    },
                },
                layout: {
                    type: 'dagre',
                    rankdir: 'TB',
                    nodesep: 50,
                    ranksep: 70,
                },
                behaviors: [
                    'drag-canvas',
                    'zoom-canvas',
                    'click-select',
                ],
            });

            // Event: node click → show detail
            topologyGraph.on('node:click', (evt) => {
                const nodeId = evt.target.id;
                const nodeData = nodes.find(n => n.id === nodeId);
                if (nodeData) {
                    showTopologyDetail(nodeData);
                }
            });

            // Event: double click → navigate to dashboard
            topologyGraph.on('node:dblclick', (evt) => {
                const nodeId = evt.target.id;
                const nodeData = nodes.find(n => n.id === nodeId);
                if (nodeData) {
                    topologyNavigateToDashboardForNode(nodeData);
                }
            });

            // Event: canvas click → close detail
            topologyGraph.on('canvas:click', () => {
                closeTopologyDetail();
            });

            topologyGraph.render();

            // Force fit-view after render to reset zoom/pan state.
            // G6 autoFit:'view' may not reliably reset viewport when
            // re-creating graphs in the same container (e.g. switching
            // from focused subgraph back to full topology).
            requestAnimationFrame(() => {
                if (topologyGraph) {
                    try { topologyGraph.fitView(); } catch(e) {}
                }
            });
        }

        function showTopologyDetail(nodeData) {
            topologySelectedNode = nodeData;
            const detail = document.getElementById('topology-detail');
            const title = document.getElementById('topology-detail-title');
            const body = document.getElementById('topology-detail-body');

            if (!detail || !body) return;

            title.textContent = `${nodeData.kind}: ${nodeData.name}`;

            let html = '';
            html += `<div class="topology-detail-field">
                <div class="label">Kind</div>
                <div class="value">${escapeHtml(nodeData.kind)}</div>
            </div>`;
            html += `<div class="topology-detail-field">
                <div class="label">Name</div>
                <div class="value">${escapeHtml(nodeData.name)}</div>
            </div>`;
            html += `<div class="topology-detail-field">
                <div class="label">Namespace</div>
                <div class="value">${escapeHtml(nodeData.namespace)}</div>
            </div>`;
            html += `<div class="topology-detail-field">
                <div class="label">Status</div>
                <div class="value"><span class="topology-status-badge ${nodeData.status}">${escapeHtml(nodeData.status)}</span></div>
            </div>`;

            // Show info fields
            if (nodeData.info) {
                for (const [key, val] of Object.entries(nodeData.info)) {
                    html += `<div class="topology-detail-field">
                        <div class="label">${escapeHtml(key)}</div>
                        <div class="value">${escapeHtml(val)}</div>
                    </div>`;
                }
            }

            // Show connections
            if (topologyData) {
                const incoming = (topologyData.edges || []).filter(e => e.target === nodeData.id);
                const outgoing = (topologyData.edges || []).filter(e => e.source === nodeData.id);
                if (incoming.length > 0) {
                    html += `<div class="topology-detail-field">
                        <div class="label">Incoming (${incoming.length})</div>
                        <div class="value" style="font-size:11px;">`;
                    for (const e of incoming) {
                        html += `${escapeHtml(e.source)} <span style="color:var(--text-secondary);">(${e.type})</span><br>`;
                    }
                    html += `</div></div>`;
                }
                if (outgoing.length > 0) {
                    html += `<div class="topology-detail-field">
                        <div class="label">Outgoing (${outgoing.length})</div>
                        <div class="value" style="font-size:11px;">`;
                    for (const e of outgoing) {
                        html += `${escapeHtml(e.target)} <span style="color:var(--text-secondary);">(${e.type})</span><br>`;
                    }
                    html += `</div></div>`;
                }
            }

            body.innerHTML = html;
            detail.classList.add('active');
        }

        function closeTopologyDetail() {
            const detail = document.getElementById('topology-detail');
            if (detail) detail.classList.remove('active');
            topologySelectedNode = null;
        }

        function topologyNavigateToDashboard() {
            if (!topologySelectedNode) return;
            topologyNavigateToDashboardForNode(topologySelectedNode);
        }

        function topologyNavigateToDashboardForNode(nodeData) {
            const kindToResource = {
                Pod: 'pods',
                Deployment: 'deployments',
                ReplicaSet: 'replicasets',
                StatefulSet: 'statefulsets',
                DaemonSet: 'daemonsets',
                Service: 'services',
                Ingress: 'ingresses',
                Job: 'jobs',
                CronJob: 'cronjobs',
                ConfigMap: 'configmaps',
                Secret: 'secrets',
                PVC: 'persistentvolumeclaims',
                HPA: 'hpas',
            };
            const resource = kindToResource[nodeData.kind];
            if (resource) {
                // Set namespace if available
                if (nodeData.namespace) {
                    const nsSelect = document.getElementById('namespace-select');
                    if (nsSelect) nsSelect.value = nodeData.namespace;
                    currentNamespace = nodeData.namespace;
                }
                switchResource(resource);
            }
        }

        function topologyFitView() {
            if (topologyGraph) {
                topologyGraph.fitView();
            }
        }

        // Show topology focused on a specific resource (called from dashboard Topo button)
        function showTopologyForResource(kind, name, namespace) {
            topologyFocusNodeId = `${kind}/${namespace}/${name}`;

            // Set namespace in topology view
            const nsSelect = document.getElementById('topology-ns-select');
            if (nsSelect) nsSelect.value = namespace || '';

            // Clear kind filter when focusing on a specific resource
            const kindFilter = document.getElementById('topology-kind-filter');
            if (kindFilter) kindFilter.value = '';

            showTopology();
        }

        // Clear the focused resource and show the full topology
        function clearTopologyFocus() {
            topologyFocusNodeId = null;
            const kindFilter = document.getElementById('topology-kind-filter');
            if (kindFilter) kindFilter.value = '';
            loadTopology();
        }

        function filterTopologyGraph(query) {
            if (!topologyGraph || !topologyData) return;

            const q = query.toLowerCase().trim();
            if (!q) {
                // Clear filter: reset all states
                topologyGraph.getNodeData().forEach(n => {
                    topologyGraph.setElementState(n.id, []);
                });
                topologyGraph.getEdgeData().forEach(e => {
                    topologyGraph.setElementState(e.id, []);
                });
                topologyGraph.draw();
                return;
            }

            const matchedIds = new Set();
            (topologyData.nodes || []).forEach(n => {
                if (n.name.toLowerCase().includes(q) || n.kind.toLowerCase().includes(q)) {
                    matchedIds.add(n.id);
                }
            });

            topologyGraph.getNodeData().forEach(n => {
                topologyGraph.setElementState(n.id, matchedIds.has(n.id) ? ['highlight'] : ['dim']);
            });
            topologyGraph.getEdgeData().forEach(e => {
                const connected = matchedIds.has(e.source) || matchedIds.has(e.target);
                topologyGraph.setElementState(e.id, connected ? [] : ['dim']);
            });
            topologyGraph.draw();
        }

        async function showReports() {
            document.getElementById('reports-modal').classList.add('active');
            // Reset status/preview on open
            document.getElementById('report-status').innerHTML = '';
            document.getElementById('report-preview').innerHTML = '';
        }

        function closeReportsModal() {
            document.getElementById('reports-modal').classList.remove('active');
        }

        // Build sections query string from report checkboxes
        function getReportSections() {
            const mapping = {
                'report-sec-workloads': 'workloads',
                'report-sec-nodes': 'nodes,namespaces',
                'report-sec-security': 'security',
                'report-sec-trivy': 'security_full',
                'report-sec-finops': 'finops',
                'report-sec-events': 'events',
                'report-sec-metrics': 'metrics',
            };
            const parts = [];
            for (const [id, value] of Object.entries(mapping)) {
                if (document.getElementById(id)?.checked) parts.push(value);
            }
            return parts.join(',');
        }

        function getReportIncludeAI() {
            return document.getElementById('report-sec-ai')?.checked ?? false;
        }

        function reportSelectAll() {
            document.querySelectorAll('[id^="report-sec-"]').forEach(cb => cb.checked = true);
        }

        function reportSelectNone() {
            document.querySelectorAll('[id^="report-sec-"]').forEach(cb => cb.checked = false);
        }

        // Preview report in new window
        async function previewReport() {
            const includeAI = getReportIncludeAI();
            const sections = getReportSections();
            const statusEl = document.getElementById('report-status');

            if (!sections && !includeAI) {
                statusEl.innerHTML = `<div style="color: var(--accent-yellow);">Please select at least one section.</div>`;
                return;
            }

            if (includeAI && !llmConnected) {
                statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    AI is not connected. Please configure LLM settings first, or uncheck "AI Analysis".
                </div>`;
                return;
            }

            statusEl.innerHTML = `<div style="color: var(--accent-blue);">
                <span class="loading-dots"><span></span><span></span><span></span></span>
                Generating report preview... This may take a moment.
            </div>`;

            try {
                const url = `/api/reports/preview?ai=${includeAI}&sections=${encodeURIComponent(sections)}`;
                const resp = await fetchWithAuth(url);

                if (!resp.ok) throw new Error('Failed to generate report');

                const html = await resp.text();

                // Open in new window
                const previewWindow = window.open('', '_blank', 'width=1200,height=800');
                previewWindow.document.write(html);
                previewWindow.document.close();

                statusEl.innerHTML = `<div style="color: var(--accent-green);">
                    Report preview opened in new window
                </div>`;
            } catch (e) {
                statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    Failed to generate preview: ${e.message}
                </div>`;
            }
        }

        // Download report
        async function downloadReport(format) {
            const includeAI = getReportIncludeAI();
            const sections = getReportSections();
            const statusEl = document.getElementById('report-status');

            if (!sections && !includeAI) {
                statusEl.innerHTML = `<div style="color: var(--accent-yellow);">Please select at least one section.</div>`;
                return;
            }

            if (includeAI && !llmConnected) {
                statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    AI is not connected. Please configure LLM settings first, or uncheck "AI Analysis".
                </div>`;
                return;
            }

            statusEl.innerHTML = `<div style="color: var(--accent-blue);">
                <span class="loading-dots"><span></span><span></span><span></span></span>
                Generating ${format.toUpperCase()} report...
            </div>`;

            try {
                const url = `/api/reports?format=${format}&ai=${includeAI}&download=true&sections=${encodeURIComponent(sections)}`;
                const resp = await fetch(url, {
                    headers: { 'Authorization': `Bearer ${authToken}` }
                });

                if (!resp.ok) throw new Error('Failed to generate report');

                const blob = await resp.blob();
                const filename = resp.headers.get('Content-Disposition')?.match(/filename=(.+)/)?.[1]
                    || `k13d-report-${new Date().toISOString().slice(0,10)}.${format}`;

                // Trigger download
                const downloadUrl = URL.createObjectURL(blob);
                const a = document.createElement('a');
                a.href = downloadUrl;
                a.download = filename;
                document.body.appendChild(a);
                a.click();
                document.body.removeChild(a);
                URL.revokeObjectURL(downloadUrl);

                statusEl.innerHTML = `<div style="color: var(--accent-green);">
                    ✓ Report downloaded: ${filename}
                </div>`;

                if (format === 'html') {
                    document.getElementById('report-preview').innerHTML = `
                        <p style="color: var(--text-secondary); margin-top: 10px;">
                            💡 <strong>Tip:</strong> Open the HTML file in your browser and use Print → Save as PDF to create a PDF version.
                        </p>
                    `;
                }
            } catch (e) {
                statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    ✕ Failed to download report: ${e.message}
                </div>`;
            }
        }

        async function generateReport(format) {
            const includeAI = getReportIncludeAI();
            const sections = getReportSections();
            const statusEl = document.getElementById('report-status');
            const previewEl = document.getElementById('report-preview');

            if (!sections && !includeAI) {
                statusEl.innerHTML = `<div style="color: var(--accent-yellow);">Please select at least one section.</div>`;
                return;
            }

            if (includeAI && !llmConnected) {
                statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    AI is not connected. Please configure LLM settings first, or uncheck "AI Analysis".
                </div>`;
                return;
            }

            statusEl.innerHTML = `<div style="color: var(--accent-blue);">
                <span class="loading-dots"><span></span><span></span><span></span></span>
                Generating report... This may take a moment.
            </div>`;
            previewEl.innerHTML = '';

            try {
                const url = `/api/reports?format=${format}&ai=${includeAI}&sections=${encodeURIComponent(sections)}`;

                if (format === 'json') {
                    // View JSON in preview
                    const resp = await fetchWithAuth(url);
                    const report = await resp.json();

                    statusEl.innerHTML = `<div style="color: var(--accent-green);">
                        ✓ Report generated successfully at ${new Date(report.generated_at).toLocaleString()}
                    </div>`;

                    // Calculate total potential savings
                    const totalSavings = (report.finops_analysis?.cost_optimizations || [])
                        .reduce((sum, opt) => sum + (opt.estimated_saving || 0), 0);

                    // Show summary with FinOps
                    previewEl.innerHTML = `
                        <div style="background: var(--bg-tertiary); padding: 20px; border-radius: 8px; margin-top: 20px;">
                            <h3 style="margin-bottom: 15px;">📈 Report Summary</h3>
                            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 12px;">
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: var(--accent-blue);">${report.node_summary?.total || 0}</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Nodes (${report.node_summary?.ready || 0} Ready)</div>
                                </div>
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: var(--accent-green);">${report.workloads?.total_pods || 0}</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Pods (${report.workloads?.running_pods || 0} Running)</div>
                                </div>
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: var(--accent-purple);">${report.workloads?.total_deployments || 0}</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Deployments</div>
                                </div>
                                <div style="background: var(--bg-secondary); padding: 15px; border-radius: 6px; text-align: center;">
                                    <div style="font-size: 22px; font-weight: bold; color: ${report.health_score >= 90 ? 'var(--accent-green)' : report.health_score >= 70 ? 'var(--accent-yellow)' : 'var(--accent-red)'};">${Math.round(report.health_score || 0)}%</div>
                                    <div style="font-size: 11px; color: var(--text-secondary);">Health Score</div>
                                </div>
                            </div>

                            <!-- FinOps Section -->
                            <div style="margin-top: 25px; background: linear-gradient(135deg, #1a472a 0%, #2d5a3d 100%); padding: 20px; border-radius: 8px; border: 1px solid #4caf50;">
                                <h3 style="margin-bottom: 15px; color: #9ece6a;">💰 FinOps Cost Analysis</h3>
                                <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; margin-bottom: 15px;">
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #9ece6a;">$${(report.finops_analysis?.total_estimated_monthly_cost || 0).toFixed(2)}</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">Est. Monthly Cost</div>
                                    </div>
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #7dcfff;">${(report.finops_analysis?.resource_efficiency?.cpu_requests_vs_capacity || 0).toFixed(1)}%</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">CPU Utilization</div>
                                    </div>
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #bb9af7;">${(report.finops_analysis?.resource_efficiency?.memory_requests_vs_capacity || 0).toFixed(1)}%</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">Memory Utilization</div>
                                    </div>
                                    <div style="background: rgba(0,0,0,0.3); padding: 15px; border-radius: 6px; text-align: center;">
                                        <div style="font-size: 24px; font-weight: bold; color: #f7768e;">$${totalSavings.toFixed(2)}</div>
                                        <div style="font-size: 11px; color: var(--text-secondary);">Potential Savings/mo</div>
                                    </div>
                                </div>

                                ${(report.finops_analysis?.cost_optimizations || []).length > 0 ? `
                                    <h4 style="margin: 15px 0 10px 0; color: #e0af68;">⚡ Cost Optimization Recommendations</h4>
                                    <div style="max-height: 200px; overflow-y: auto;">
                                        ${(report.finops_analysis?.cost_optimizations || []).slice(0, 5).map(opt => `
                                            <div style="background: rgba(0,0,0,0.2); padding: 10px; border-radius: 4px; margin-bottom: 8px; border-left: 3px solid ${opt.priority === 'high' ? '#f7768e' : opt.priority === 'medium' ? '#e0af68' : '#9ece6a'};">
                                                <div style="display: flex; justify-content: space-between; align-items: center;">
                                                    <span style="font-weight: bold; color: ${opt.priority === 'high' ? '#f7768e' : opt.priority === 'medium' ? '#e0af68' : '#9ece6a'};">[${opt.priority.toUpperCase()}] ${escapeHtml(opt.category)}</span>
                                                    <span style="color: #9ece6a; font-weight: bold;">Save $${(opt.estimated_saving || 0).toFixed(2)}/mo</span>
                                                </div>
                                                <div style="font-size: 12px; margin-top: 5px; color: var(--text-secondary);">${escapeHtml(opt.description)}</div>
                                            </div>
                                        `).join('')}
                                    </div>
                                ` : '<p style="color: var(--text-secondary);">No optimization recommendations at this time.</p>'}
                            </div>

                            ${report.ai_analysis ? `
                                <div style="margin-top: 20px;">
                                    <h4 style="margin-bottom: 10px;">🤖 AI Analysis with FinOps Insights</h4>
                                    <div style="background: var(--bg-primary); padding: 15px; border-radius: 6px; white-space: pre-wrap; font-size: 13px; max-height: 300px; overflow-y: auto; border-left: 3px solid var(--accent-blue);">
                                        ${escapeHtml(report.ai_analysis)}
                                    </div>
                                </div>
                            ` : ''}

                            <div style="margin-top: 20px;">
                                <h4 style="margin-bottom: 10px;">📊 Cost by Namespace (Top 5)</h4>
                                <table style="width: 100%; font-size: 12px;">
                                    <tr style="background: var(--bg-secondary);"><th style="padding: 8px;">Namespace</th><th style="padding: 8px;">Pods</th><th style="padding: 8px;">CPU</th><th style="padding: 8px;">Memory</th><th style="padding: 8px;">Est. Cost</th></tr>
                                    ${(report.finops_analysis?.cost_by_namespace || []).slice(0, 5).map(ns => `
                                        <tr><td style="padding: 8px;">${escapeHtml(ns.namespace)}</td><td style="padding: 8px;">${ns.pod_count}</td><td style="padding: 8px;">${escapeHtml(ns.cpu_requests)}</td><td style="padding: 8px;">${escapeHtml(ns.memory_requests)}</td><td style="padding: 8px;">$${(ns.estimated_cost || 0).toFixed(2)}</td></tr>
                                    `).join('')}
                                </table>
                            </div>

                            <div style="margin-top: 20px;">
                                <h4 style="margin-bottom: 10px;">🐳 Top Container Images</h4>
                                <table style="width: 100%; font-size: 12px;">
                                    <tr style="background: var(--bg-secondary);"><th style="padding: 8px;">Image</th><th style="padding: 8px;">Tag</th><th style="padding: 8px;">Pods</th></tr>
                                    ${(report.images || []).slice(0, 8).map(img => `
                                        <tr><td style="padding: 8px;">${escapeHtml(img.repository)}</td><td style="padding: 8px;">${escapeHtml(img.tag)}</td><td style="padding: 8px;">${img.pod_count}</td></tr>
                                    `).join('')}
                                </table>
                            </div>
                        </div>
                    `;
                }
            } catch (e) {
                statusEl.innerHTML = `<div style="color: var(--accent-red);">
                    ✕ Failed to generate report: ${e.message}
                </div>`;
            }
        }

        // Note: Auto-refresh is now handled by startAutoRefresh() in init()
        // with user-configurable interval settings

        // Global search across all resources
        let searchTimeout = null;
        let searchSelectedIndex = -1;
        let searchResults = [];

        function handleGlobalSearchInput(event) {
            const query = event.target.value.trim();

            // Clear previous timeout
            if (searchTimeout) {
                clearTimeout(searchTimeout);
            }

            if (query.length < 2) {
                hideSearchResults();
                return;
            }

            // Debounce search
            searchTimeout = setTimeout(() => {
                performGlobalSearch(query);
            }, 300);
        }

        function handleGlobalSearchKeydown(event) {
            const resultsDiv = document.getElementById('search-results');
            const items = resultsDiv.querySelectorAll('.search-result-item');

            switch (event.key) {
                case 'ArrowDown':
                    event.preventDefault();
                    searchSelectedIndex = Math.min(searchSelectedIndex + 1, items.length - 1);
                    updateSearchSelection(items);
                    break;
                case 'ArrowUp':
                    event.preventDefault();
                    searchSelectedIndex = Math.max(searchSelectedIndex - 1, 0);
                    updateSearchSelection(items);
                    break;
                case 'Enter':
                    event.preventDefault();
                    if (searchSelectedIndex >= 0 && searchResults[searchSelectedIndex]) {
                        navigateToSearchResult(searchResults[searchSelectedIndex]);
                    }
                    break;
                case 'Escape':
                    hideSearchResults();
                    event.target.blur();
                    break;
            }
        }

        function updateSearchSelection(items) {
            items.forEach((item, idx) => {
                if (idx === searchSelectedIndex) {
                    item.style.background = 'var(--bg-tertiary)';
                    item.scrollIntoView({ block: 'nearest' });
                } else {
                    item.style.background = '';
                }
            });
        }

        async function performGlobalSearch(query) {
            const resultsDiv = document.getElementById('search-results');
            resultsDiv.innerHTML = '<div class="search-loading">Searching...</div>';
            resultsDiv.style.display = 'block';

            try {
                const response = await fetch(`/api/search?q=${encodeURIComponent(query)}&namespace=${currentNamespace || ''}`, {
                    headers: {
                        'Authorization': `Bearer ${authToken}`
                    }
                });

                if (!response.ok) throw new Error('Search failed');

                const data = await response.json();
                searchResults = data.results || [];
                searchSelectedIndex = -1;

                if (searchResults.length === 0) {
                    resultsDiv.innerHTML = '<div class="search-no-results">No results found</div>';
                } else {
                    resultsDiv.innerHTML = searchResults.map((result, idx) => `
                        <div class="search-result-item" onclick="navigateToSearchResult(searchResults[${idx}])">
                            <span class="search-result-kind ${result.kind.toLowerCase()}">${result.kind}</span>
                            <div class="search-result-info">
                                <div class="search-result-name">${escapeHtml(result.name)}</div>
                                ${result.namespace ? `<div class="search-result-namespace">${escapeHtml(result.namespace)}</div>` : ''}
                            </div>
                            ${result.status ? `<span class="search-result-status ${result.status.toLowerCase().replace(/\s/g, '')}">${result.status}</span>` : ''}
                        </div>
                    `).join('');
                }
            } catch (e) {
                resultsDiv.innerHTML = '<div class="search-no-results">Search error</div>';
                console.error('Search error:', e);
            }
        }

        function navigateToSearchResult(result) {
            hideSearchResults();
            document.getElementById('global-search').value = '';

            // Map kind to resource type
            const kindToResource = {
                'Pod': 'pods',
                'Deployment': 'deployments',
                'Service': 'services',
                'StatefulSet': 'statefulsets',
                'DaemonSet': 'daemonsets',
                'ConfigMap': 'configmaps',
                'Secret': 'secrets',
                'Ingress': 'ingresses',
                'Node': 'nodes',
                'Namespace': 'namespaces',
                'ReplicaSet': 'replicasets',
                'Job': 'jobs',
                'CronJob': 'cronjobs'
            };

            const resourceType = kindToResource[result.kind] || result.kind.toLowerCase() + 's';

            // Switch namespace if needed
            if (result.namespace && result.namespace !== currentNamespace) {
                currentNamespace = result.namespace;
                document.getElementById('namespace-select').value = result.namespace;
            }

            // Switch to the resource type
            switchResource(resourceType);

            // Set filter to highlight the specific resource
            setTimeout(() => {
                document.getElementById('filter-input').value = result.name;
                currentFilter = result.name.toLowerCase();
                applyFilterAndSort();
            }, 500);
        }

        function showSearchResults() {
            const query = document.getElementById('global-search').value.trim();
            if (query.length >= 2 && searchResults.length > 0) {
                document.getElementById('search-results').style.display = 'block';
            }
        }

        function hideSearchResults() {
            document.getElementById('search-results').style.display = 'none';
            searchSelectedIndex = -1;
        }

        // Hide search results when clicking outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('.search-container')) {
                hideSearchResults();
            }
        });

        // Filter functionality
        let currentFilter = '';
        let cachedData = [];

        function handleFilter(event) {
            currentFilter = event.target.value.trim().toLowerCase();
            // Use the new filtering system that works with sorting/pagination
            currentPage = 1;
            applyFilterAndSort();
        }

        // Legacy filterTable for compatibility (now uses new system)
        function filterTable(query) {
            document.getElementById('filter-input').value = query;
            currentPage = 1;
            applyFilterAndSort();
        }

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            // Check if command bar is open
            const commandBarOpen = document.getElementById('command-bar-overlay').classList.contains('active');
            const yamlEditorOpen = document.getElementById('yaml-editor-modal').classList.contains('active');

            // Handle command bar input separately
            if (commandBarOpen) {
                handleCommandBarKeydown(e);
                return;
            }

            // Handle YAML editor shortcuts
            if (yamlEditorOpen) {
                handleYamlEditorKeydown(e);
                return;
            }

            // Ignore if in input/textarea (except for specific shortcuts)
            if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
                if (e.key === 'Escape') {
                    e.target.blur();
                }
                return;
            }

            // Check for modifiers
            const isMeta = e.metaKey || e.ctrlKey;
            const isAlt = e.altKey;

            // Alt+number for namespace switching
            if (isAlt && e.key >= '0' && e.key <= '9') {
                e.preventDefault();
                switchToRecentNamespace(parseInt(e.key));
                return;
            }

            switch (e.key.toLowerCase()) {
                case 'k':
                    if (isMeta) {
                        e.preventDefault();
                        document.getElementById('global-search').focus();
                    }
                    break;
                case 'f':
                    if (isMeta) {
                        e.preventDefault();
                        toggleColumnFilters();
                    }
                    break;
                case '/':
                    e.preventDefault();
                    document.getElementById('filter-input').focus();
                    break;
                case ':':
                    e.preventDefault();
                    openCommandBar();
                    break;
                case 'r':
                    e.preventDefault();
                    refreshData();
                    break;
                case 'a':
                    e.preventDefault();
                    toggleAIPanel();
                    break;
                case 'b':
                    e.preventDefault();
                    toggleSidebar();
                    break;
                case 'd':
                    e.preventDefault();
                    toggleDebugMode();
                    break;
                case 'e':
                    e.preventDefault();
                    openYamlEditor();
                    break;
                case 'n':
                    e.preventDefault();
                    showNamespaceIndicator();
                    break;
                case '1':
                    e.preventDefault();
                    switchResource('pods');
                    break;
                case '2':
                    e.preventDefault();
                    switchResource('deployments');
                    break;
                case '3':
                    e.preventDefault();
                    switchResource('services');
                    break;
                case '4':
                    e.preventDefault();
                    switchResource('nodes');
                    break;
                case 's':
                    e.preventDefault();
                    showSettings();
                    break;
                case '?':
                    e.preventDefault();
                    showShortcuts();
                    break;
                case 'escape':
                    closeAllModals();
                    hideNamespaceIndicator();
                    break;
            }
        });

        function toggleAIPanel() {
            const panel = document.getElementById('ai-panel');
            const handle = document.getElementById('resize-handle');
            const btn = document.getElementById('ai-toggle-btn');
            const isHidden = panel.style.display === 'none';
            panel.style.display = isHidden ? 'flex' : 'none';
            handle.style.display = isHidden ? 'block' : 'none';
            if (btn) btn.classList.toggle('active', isHidden);
            localStorage.setItem('k13d_ai_panel', isHidden ? 'open' : 'closed');
        }

        // Restore AI panel state on load
        (function initAIPanelState() {
            const saved = localStorage.getItem('k13d_ai_panel');
            if (saved === 'closed') {
                const panel = document.getElementById('ai-panel');
                const handle = document.getElementById('resize-handle');
                const btn = document.getElementById('ai-toggle-btn');
                if (panel) panel.style.display = 'none';
                if (handle) handle.style.display = 'none';
                if (btn) btn.classList.remove('active');
            } else {
                const btn = document.getElementById('ai-toggle-btn');
                if (btn) btn.classList.add('active');
            }
        })();

        function closeAllModals() {
            document.querySelectorAll('.modal-overlay').forEach(m => m.classList.remove('active'));
            closeCommandBar();
            closeYamlEditor();
        }

        // ==================== Command Bar (TUI-style : mode) ====================
        const commandDefinitions = [
            // Resource commands
            { name: 'pods', alias: ['po', 'pod'], desc: 'View Pods', action: () => switchResource('pods') },
            { name: 'deployments', alias: ['deploy', 'dep'], desc: 'View Deployments', action: () => switchResource('deployments') },
            { name: 'services', alias: ['svc', 'service'], desc: 'View Services', action: () => switchResource('services') },
            { name: 'statefulsets', alias: ['sts'], desc: 'View StatefulSets', action: () => switchResource('statefulsets') },
            { name: 'daemonsets', alias: ['ds'], desc: 'View DaemonSets', action: () => switchResource('daemonsets') },
            { name: 'replicasets', alias: ['rs'], desc: 'View ReplicaSets', action: () => switchResource('replicasets') },
            { name: 'configmaps', alias: ['cm'], desc: 'View ConfigMaps', action: () => switchResource('configmaps') },
            { name: 'secrets', alias: ['sec'], desc: 'View Secrets', action: () => switchResource('secrets') },
            { name: 'ingresses', alias: ['ing'], desc: 'View Ingresses', action: () => switchResource('ingresses') },
            { name: 'jobs', alias: ['job'], desc: 'View Jobs', action: () => switchResource('jobs') },
            { name: 'cronjobs', alias: ['cj'], desc: 'View CronJobs', action: () => switchResource('cronjobs') },
            { name: 'nodes', alias: ['no', 'node'], desc: 'View Nodes', action: () => switchResource('nodes') },
            { name: 'namespaces', alias: ['ns'], desc: 'View Namespaces', action: () => switchResource('namespaces') },
            { name: 'pvcs', alias: ['pvc'], desc: 'View PVCs', action: () => switchResource('pvcs') },
            { name: 'pvs', alias: ['pv'], desc: 'View PVs', action: () => switchResource('pvs') },
            { name: 'events', alias: ['ev'], desc: 'View Events', action: () => switchResource('events') },
            { name: 'serviceaccounts', alias: ['sa'], desc: 'View Service Accounts', action: () => switchResource('serviceaccounts') },
            { name: 'roles', alias: ['role'], desc: 'View Roles', action: () => switchResource('roles') },
            { name: 'rolebindings', alias: ['rb'], desc: 'View RoleBindings', action: () => switchResource('rolebindings') },
            { name: 'clusterroles', alias: ['cr'], desc: 'View ClusterRoles', action: () => switchResource('clusterroles') },
            { name: 'clusterrolebindings', alias: ['crb'], desc: 'View ClusterRoleBindings', action: () => switchResource('clusterrolebindings') },
            // Actions
            { name: 'refresh', alias: ['r', 'reload'], desc: 'Refresh current data', action: () => refreshData() },
            { name: 'ai', alias: ['assistant', 'chat'], desc: 'Toggle AI Panel', action: () => toggleAIPanel() },
            { name: 'settings', alias: ['config', 'set'], desc: 'Open Settings', action: () => showSettings() },
            { name: 'help', alias: ['?', 'h'], desc: 'Show Shortcuts', action: () => showShortcuts() },
            { name: 'yaml', alias: ['edit', 'create'], desc: 'Open YAML Editor', action: () => openYamlEditor() },
            { name: 'metrics', alias: ['metric'], desc: 'Show Metrics View', action: () => document.getElementById('metrics-tab')?.click() },
            { name: 'audit', alias: ['log', 'history'], desc: 'Show Audit Logs', action: () => document.getElementById('audit-tab')?.click() },
        ];

        let commandSelectedIndex = 0;
        let filteredCommands = [];

        function openCommandBar() {
            const overlay = document.getElementById('command-bar-overlay');
            const input = document.getElementById('command-input');
            overlay.classList.add('active');
            input.value = '';
            input.focus();
            commandSelectedIndex = 0;
            updateCommandSuggestions('');
        }

        function closeCommandBar() {
            document.getElementById('command-bar-overlay').classList.remove('active');
        }

        function handleCommandBarKeydown(e) {
            const input = document.getElementById('command-input');

            switch (e.key) {
                case 'Escape':
                    e.preventDefault();
                    closeCommandBar();
                    break;
                case 'ArrowDown':
                    e.preventDefault();
                    commandSelectedIndex = Math.min(commandSelectedIndex + 1, filteredCommands.length - 1);
                    renderCommandSuggestions();
                    break;
                case 'ArrowUp':
                    e.preventDefault();
                    commandSelectedIndex = Math.max(commandSelectedIndex - 1, 0);
                    renderCommandSuggestions();
                    break;
                case 'Tab':
                    e.preventDefault();
                    if (filteredCommands.length > 0) {
                        input.value = filteredCommands[commandSelectedIndex].name;
                        updateCommandSuggestions(input.value);
                    }
                    break;
                case 'Enter':
                    e.preventDefault();
                    executeSelectedCommand();
                    break;
                default:
                    // Let input handle it, then update suggestions
                    setTimeout(() => updateCommandSuggestions(input.value), 0);
            }
        }

        function updateCommandSuggestions(query) {
            query = query.toLowerCase().trim();

            if (!query) {
                filteredCommands = commandDefinitions.slice(0, 15);
            } else {
                filteredCommands = commandDefinitions.filter(cmd => {
                    if (cmd.name.startsWith(query)) return true;
                    if (cmd.alias.some(a => a.startsWith(query))) return true;
                    if (cmd.desc.toLowerCase().includes(query)) return true;
                    return false;
                }).slice(0, 10);
            }

            commandSelectedIndex = 0;
            renderCommandSuggestions();
        }

        function renderCommandSuggestions() {
            const container = document.getElementById('command-suggestions');

            if (filteredCommands.length === 0) {
                container.innerHTML = '<div class="command-suggestion" style="color: var(--text-secondary);">No matching commands</div>';
                return;
            }

            container.innerHTML = filteredCommands.map((cmd, i) => `
                <div class="command-suggestion ${i === commandSelectedIndex ? 'selected' : ''}"
                     onclick="executeCommand(${i})"
                     onmouseover="commandSelectedIndex = ${i}; renderCommandSuggestions();">
                    <div>
                        <span class="command-suggestion-name">${cmd.name}</span>
                        <span class="command-suggestion-desc"> - ${cmd.desc}</span>
                    </div>
                    <span class="command-suggestion-shortcut">${cmd.alias[0] || ''}</span>
                </div>
            `).join('');
        }

        function executeCommand(index) {
            if (filteredCommands[index]) {
                closeCommandBar();
                filteredCommands[index].action();
            }
        }

        function executeSelectedCommand() {
            const input = document.getElementById('command-input').value.trim().toLowerCase();

            // First try exact match
            const exactMatch = commandDefinitions.find(cmd =>
                cmd.name === input || cmd.alias.includes(input)
            );

            if (exactMatch) {
                closeCommandBar();
                exactMatch.action();
                return;
            }

            // Otherwise execute selected suggestion
            if (filteredCommands[commandSelectedIndex]) {
                closeCommandBar();
                filteredCommands[commandSelectedIndex].action();
            }
        }

        // Click outside to close command bar
        document.getElementById('command-bar-overlay')?.addEventListener('click', (e) => {
            if (e.target.id === 'command-bar-overlay') {
                closeCommandBar();
            }
        });

        // ==================== Namespace Quick Switcher ====================
        let recentNamespaces = [];
        let namespaceIndicatorTimeout = null;

        function trackNamespaceUsage(ns) {
            // Remove if already exists
            recentNamespaces = recentNamespaces.filter(n => n !== ns);
            // Add to front
            if (ns) {
                recentNamespaces.unshift(ns);
            }
            // Keep max 9
            recentNamespaces = recentNamespaces.slice(0, 9);
            // Save to localStorage
            localStorage.setItem('k13d-recent-namespaces', JSON.stringify(recentNamespaces));
        }

        function loadRecentNamespaces() {
            try {
                const saved = localStorage.getItem('k13d-recent-namespaces');
                if (saved) {
                    recentNamespaces = JSON.parse(saved);
                }
            } catch (e) {
                console.error('Failed to load recent namespaces:', e);
            }
        }

        function switchToRecentNamespace(index) {
            if (index === 0) {
                // All namespaces
                document.getElementById('namespace-select').value = '';
                currentNamespace = '';
                onNamespaceChange();
                showToast('Switched to All Namespaces');
                return;
            }

            const ns = recentNamespaces[index - 1];
            if (ns) {
                document.getElementById('namespace-select').value = ns;
                currentNamespace = ns;
                onNamespaceChange();
                showToast(`Switched to namespace: ${ns}`);
            }
        }

        function showNamespaceIndicator() {
            const indicator = document.getElementById('namespace-indicator');

            // Use recent namespaces, or fall back to available namespaces from selector
            let nsList = recentNamespaces.slice(0, 9);
            if (nsList.length === 0) {
                const nsSelect = document.getElementById('namespace-select');
                if (nsSelect) {
                    for (const opt of nsSelect.options) {
                        if (opt.value && nsList.length < 9) {
                            nsList.push(opt.value);
                        }
                    }
                }
            }

            // Build namespace keys
            let html = `
                <div class="namespace-key ${!currentNamespace ? 'current' : ''}" onclick="switchToRecentNamespace(0)">
                    <span class="namespace-key-num">0</span>
                    <span class="namespace-key-name">All</span>
                </div>
            `;

            for (let i = 0; i < 9; i++) {
                const ns = nsList[i];
                const isCurrent = ns && ns === currentNamespace;
                html += `
                    <div class="namespace-key ${isCurrent ? 'current' : ''} ${!ns ? 'disabled' : ''}"
                         onclick="${ns ? `switchToNamespaceByName('${ns}')` : ''}"
                         style="${!ns ? 'opacity: 0.3; cursor: default;' : ''}">
                        <span class="namespace-key-num">${i + 1}</span>
                        <span class="namespace-key-name">${ns || '-'}</span>
                    </div>
                `;
            }

            indicator.innerHTML = html;
            indicator.classList.add('active');

            // Auto hide after 3 seconds
            if (namespaceIndicatorTimeout) {
                clearTimeout(namespaceIndicatorTimeout);
            }
            namespaceIndicatorTimeout = setTimeout(hideNamespaceIndicator, 3000);
        }

        function switchToNamespaceByName(ns) {
            document.getElementById('namespace-select').value = ns;
            currentNamespace = ns;
            trackNamespaceUsage(ns);
            onNamespaceChange();
            showToast(`Switched to namespace: ${ns}`);
            hideNamespaceIndicator();
        }

        function hideNamespaceIndicator() {
            document.getElementById('namespace-indicator').classList.remove('active');
            if (namespaceIndicatorTimeout) {
                clearTimeout(namespaceIndicatorTimeout);
                namespaceIndicatorTimeout = null;
            }
        }

        // Track namespace changes
        const originalOnNamespaceChange = typeof onNamespaceChange === 'function' ? onNamespaceChange : null;

        // ==================== YAML Editor ====================
        const yamlTemplates = [
            {
                title: 'Pod',
                desc: 'Basic Pod template',
                yaml: `apiVersion: v1
kind: Pod
metadata:
  name: my-pod
  namespace: default
  labels:
    app: my-app
spec:
  containers:
  - name: main
    image: nginx:latest
    ports:
    - containerPort: 80
    resources:
      limits:
        memory: "128Mi"
        cpu: "500m"`
            },
            {
                title: 'Deployment',
                desc: 'Deployment with replicas',
                yaml: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-deployment
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: main
        image: nginx:latest
        ports:
        - containerPort: 80`
            },
            {
                title: 'Service',
                desc: 'ClusterIP Service',
                yaml: `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  selector:
    app: my-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  type: ClusterIP`
            },
            {
                title: 'ConfigMap',
                desc: 'Configuration data',
                yaml: `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  config.json: |
    {
      "key": "value"
    }
  APP_ENV: production`
            },
            {
                title: 'Secret',
                desc: 'Opaque Secret',
                yaml: `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
type: Opaque
stringData:
  username: admin
  password: changeme`
            },
            {
                title: 'Ingress',
                desc: 'HTTP Ingress rule',
                yaml: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  rules:
  - host: example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: my-service
            port:
              number: 80`
            },
            {
                title: 'CronJob',
                desc: 'Scheduled job',
                yaml: `apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-cronjob
  namespace: default
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: job
            image: busybox
            command: ["echo", "Hello"]
          restartPolicy: OnFailure`
            },
            {
                title: 'PVC',
                desc: 'Persistent Volume Claim',
                yaml: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi`
            }
        ];

        let yamlEditorMode = 'create'; // 'create' or 'edit'
        let yamlEditingResource = null;

        function openYamlEditor(existingYaml = null, resourceInfo = null) {
            const modal = document.getElementById('yaml-editor-modal');
            const textarea = document.getElementById('yaml-editor-content');
            const modeLabel = document.getElementById('yaml-editor-mode');
            const nsSelect = document.getElementById('yaml-editor-namespace');

            // Populate namespace select
            nsSelect.innerHTML = document.getElementById('namespace-select').innerHTML;
            nsSelect.value = currentNamespace || '';

            // Render templates
            renderYamlTemplates();

            if (existingYaml) {
                textarea.value = existingYaml;
                yamlEditorMode = 'edit';
                modeLabel.textContent = 'Edit';
                modeLabel.style.background = 'var(--accent-yellow)';
                yamlEditingResource = resourceInfo;
            } else {
                textarea.value = '';
                yamlEditorMode = 'create';
                modeLabel.textContent = 'Create';
                modeLabel.style.background = 'var(--accent-blue)';
                yamlEditingResource = null;
            }

            updateYamlEditorStatus('valid', 'Ready');
            modal.classList.add('active');
            textarea.focus();
        }

        function closeYamlEditor() {
            document.getElementById('yaml-editor-modal').classList.remove('active');
        }

        function renderYamlTemplates() {
            const container = document.getElementById('yaml-template-list');
            container.innerHTML = yamlTemplates.map((tpl, i) => `
                <div class="yaml-template-item" onclick="loadYamlTemplate(${i})">
                    <div class="yaml-template-item-title">${tpl.title}</div>
                    <div class="yaml-template-item-desc">${tpl.desc}</div>
                </div>
            `).join('');
        }

        function loadYamlTemplate(index) {
            const tpl = yamlTemplates[index];
            if (tpl) {
                const textarea = document.getElementById('yaml-editor-content');
                // Replace namespace in template
                const ns = document.getElementById('yaml-editor-namespace').value || 'default';
                let yaml = tpl.yaml.replace(/namespace: default/g, `namespace: ${ns}`);
                textarea.value = yaml;
                updateYamlEditorStatus('valid', 'Template loaded');
            }
        }

        function validateYaml() {
            const yaml = document.getElementById('yaml-editor-content').value;

            if (!yaml.trim()) {
                updateYamlEditorStatus('invalid', 'YAML is empty');
                return false;
            }

            // Basic validation
            if (!yaml.includes('apiVersion:')) {
                updateYamlEditorStatus('invalid', 'Missing apiVersion');
                return false;
            }
            if (!yaml.includes('kind:')) {
                updateYamlEditorStatus('invalid', 'Missing kind');
                return false;
            }
            if (!yaml.includes('metadata:')) {
                updateYamlEditorStatus('invalid', 'Missing metadata');
                return false;
            }

            updateYamlEditorStatus('valid', 'YAML is valid');
            return true;
        }

        function formatYaml() {
            // Simple formatting - just normalize indentation
            const textarea = document.getElementById('yaml-editor-content');
            const yaml = textarea.value;

            try {
                // Basic cleanup
                let formatted = yaml
                    .replace(/\t/g, '  ')  // Tabs to spaces
                    .replace(/  +$/gm, '') // Trailing spaces
                    .replace(/\n{3,}/g, '\n\n'); // Multiple blank lines

                textarea.value = formatted;
                updateYamlEditorStatus('valid', 'Formatted');
            } catch (e) {
                updateYamlEditorStatus('invalid', 'Format error: ' + e.message);
            }
        }

        async function applyYaml() {
            const yaml = document.getElementById('yaml-editor-content').value;
            const dryRun = document.getElementById('yaml-dry-run').checked;
            const namespace = document.getElementById('yaml-editor-namespace').value || 'default';

            if (!validateYaml()) {
                return;
            }

            updateYamlEditorStatus('valid', dryRun ? 'Validating (dry-run)...' : 'Applying...');

            try {
                const resp = await fetchWithAuth('/api/k8s/apply', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        yaml: yaml,
                        namespace: namespace,
                        dryRun: dryRun
                    })
                });

                const result = await resp.json();

                if (result.error) {
                    updateYamlEditorStatus('invalid', 'Error: ' + result.error);
                    return;
                }

                if (dryRun) {
                    updateYamlEditorStatus('valid', 'Dry-run successful! Uncheck "Dry Run" to apply.');
                    showToast('Dry-run validation passed', 'success');
                } else {
                    updateYamlEditorStatus('valid', 'Applied successfully!');
                    showToast('Resource applied successfully', 'success');
                    // Refresh data
                    refreshData();
                    // Close editor after short delay
                    setTimeout(closeYamlEditor, 1500);
                }
            } catch (e) {
                updateYamlEditorStatus('invalid', 'Error: ' + e.message);
            }
        }

        function updateYamlEditorStatus(state, message) {
            const status = document.getElementById('yaml-editor-status');
            status.className = 'yaml-editor-status ' + state;
            status.querySelector('.status-text').textContent = message;
        }

        function handleYamlEditorKeydown(e) {
            const isMeta = e.metaKey || e.ctrlKey;

            if (e.key === 'Escape') {
                e.preventDefault();
                closeYamlEditor();
                return;
            }

            if (isMeta && e.key === 'Enter') {
                e.preventDefault();
                applyYaml();
                return;
            }

            if (isMeta && e.shiftKey && e.key.toLowerCase() === 'f') {
                e.preventDefault();
                formatYaml();
                return;
            }
        }

        // Edit existing resource YAML
        function editResourceYaml(resource, item) {
            // Get full YAML from API
            const ns = item.namespace || currentNamespace;
            const name = item.name;

            fetchWithAuth(`/api/k8s/${resource}/${name}?namespace=${ns}&format=yaml`)
                .then(resp => resp.text())
                .then(yaml => {
                    openYamlEditor(yaml, { resource, name, namespace: ns });
                })
                .catch(e => {
                    showToast('Failed to load YAML: ' + e.message, 'error');
                });
        }

        // Initialize
        loadRecentNamespaces();

        // ==================== Chat History (localStorage) ====================
        const CHAT_STORAGE_KEY = 'k13d-chat-history';
        const MAX_CHATS = 50;
        let chatHistory = [];
        let currentChatId = null;

        function generateChatId() {
            return 'chat-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
        }

        function loadChatHistory() {
            try {
                const saved = localStorage.getItem(CHAT_STORAGE_KEY);
                if (saved) {
                    chatHistory = JSON.parse(saved);
                }
            } catch (e) {
                console.error('Failed to load chat history:', e);
                chatHistory = [];
            }
            renderChatHistoryList();

            // Load most recent chat or create new one
            if (chatHistory.length > 0) {
                loadChat(chatHistory[0].id);
            } else {
                createNewChat();
            }
        }

        function saveChatHistory() {
            try {
                // Limit to MAX_CHATS
                if (chatHistory.length > MAX_CHATS) {
                    chatHistory = chatHistory.slice(0, MAX_CHATS);
                }
                localStorage.setItem(CHAT_STORAGE_KEY, JSON.stringify(chatHistory));
            } catch (e) {
                console.error('Failed to save chat history:', e);
            }
        }

        function createNewChat() {
            const chat = {
                id: generateChatId(),
                title: 'New Chat',
                messages: [],
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString()
            };

            chatHistory.unshift(chat);
            saveChatHistory();
            loadChat(chat.id);
            renderChatHistoryList();
        }

        function loadChat(chatId) {
            const chat = chatHistory.find(c => c.id === chatId);
            if (!chat) return;

            currentChatId = chatId;

            // Clear and restore messages
            const container = document.getElementById('ai-messages');
            container.innerHTML = '';

            if (chat.messages.length === 0) {
                // Show welcome message for new chats
                container.innerHTML = `
                    <div class="message assistant">
                        <div class="message-content">
                            Welcome to k13d! I can help you manage your Kubernetes cluster.
                            <br><br>
                            Try asking:
                            <br>- "Show me all pods"
                            <br>- "Create an nginx pod"
                            <br>- "Scale deployment to 3 replicas"
                            <br><br>
                            <strong>Tip:</strong> Click any resource row to add it as context for AI analysis!
                        </div>
                    </div>
                `;
            } else {
                // Restore messages
                chat.messages.forEach(msg => {
                    addMessageToDOM(msg.content, msg.isUser, false);
                });
            }

            renderChatHistoryList();
        }

        function saveCurrentChatMessage(content, isUser) {
            const chat = chatHistory.find(c => c.id === currentChatId);
            if (!chat) return;

            chat.messages.push({
                content: content,
                isUser: isUser,
                timestamp: new Date().toISOString()
            });

            // Update title from first user message
            if (isUser && chat.title === 'New Chat') {
                chat.title = generateChatTitle(content);
            }

            chat.updatedAt = new Date().toISOString();

            // Move to top
            chatHistory = chatHistory.filter(c => c.id !== chat.id);
            chatHistory.unshift(chat);

            saveChatHistory();
            renderChatHistoryList();
        }

        function deleteChat(chatId, event) {
            event.stopPropagation();

            if (!confirm('Delete this chat?')) return;

            chatHistory = chatHistory.filter(c => c.id !== chatId);
            saveChatHistory();

            if (currentChatId === chatId) {
                if (chatHistory.length > 0) {
                    loadChat(chatHistory[0].id);
                } else {
                    createNewChat();
                }
            }

            renderChatHistoryList();
        }

        function renderChatHistoryList(filter = '') {
            const container = document.getElementById('chat-history-list');
            let filtered = chatHistory;

            if (filter) {
                const lowerFilter = filter.toLowerCase();
                filtered = chatHistory.filter(c =>
                    c.title.toLowerCase().includes(lowerFilter) ||
                    c.messages.some(m => m.content.toLowerCase().includes(lowerFilter))
                );
            }

            if (filtered.length === 0) {
                container.innerHTML = `
                    <div class="chat-history-empty">
                        <div class="chat-history-empty-icon">💬</div>
                        <div>${filter ? 'No matching chats' : 'No chat history yet'}</div>
                        <div style="margin-top: 8px; font-size: 11px;">Start a new conversation!</div>
                    </div>
                `;
                return;
            }

            container.innerHTML = filtered.map(chat => {
                const date = new Date(chat.updatedAt);
                const dateStr = formatChatDate(date);
                const msgCount = chat.messages.length;
                const isActive = chat.id === currentChatId;

                return `
                    <div class="chat-history-item ${isActive ? 'active' : ''}" onclick="loadChat('${chat.id}')">
                        <div class="chat-history-title">${escapeHtml(chat.title)}</div>
                        <div class="chat-history-meta">
                            <span>${dateStr}</span>
                            <span>${msgCount} message${msgCount !== 1 ? 's' : ''}</span>
                        </div>
                        <button class="chat-history-edit" onclick="startRenameChat('${chat.id}', event)" title="Rename">✏️</button>
                        <button class="chat-history-delete" onclick="deleteChat('${chat.id}', event)" title="Delete">🗑️</button>
                    </div>
                `;
            }).join('');
        }

        function formatChatDate(date) {
            const now = new Date();
            const diff = now - date;
            const days = Math.floor(diff / (1000 * 60 * 60 * 24));

            if (days === 0) {
                return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
            } else if (days === 1) {
                return 'Yesterday';
            } else if (days < 7) {
                return date.toLocaleDateString([], { weekday: 'short' });
            } else {
                return date.toLocaleDateString([], { month: 'short', day: 'numeric' });
            }
        }

        function filterChatHistory(query) {
            renderChatHistoryList(query);
        }

        // Generate a meaningful chat title from the first message
        function generateChatTitle(content) {
            // Remove markdown, code blocks, and extra whitespace
            let title = content
                .replace(/```[\s\S]*?```/g, '')  // Remove code blocks
                .replace(/`[^`]+`/g, '')          // Remove inline code
                .replace(/\*\*([^*]+)\*\*/g, '$1') // Remove bold
                .replace(/\*([^*]+)\*/g, '$1')     // Remove italic
                .replace(/#+\s*/g, '')             // Remove headers
                .replace(/\n/g, ' ')               // Replace newlines
                .replace(/\s+/g, ' ')              // Collapse whitespace
                .trim();

            // If it starts with common question words, keep them
            const questionPatterns = [
                /^(show|list|get|create|delete|scale|restart|describe|explain|why|what|how|can|help|find|check|monitor|deploy|update|patch|edit|fix|debug)/i
            ];

            // Extract the main intent (first meaningful phrase)
            const words = title.split(' ');
            let titleWords = [];
            let charCount = 0;

            for (const word of words) {
                if (charCount + word.length > 35) break;
                titleWords.push(word);
                charCount += word.length + 1;
            }

            title = titleWords.join(' ');

            // Capitalize first letter
            if (title.length > 0) {
                title = title.charAt(0).toUpperCase() + title.slice(1);
            }

            // Add ellipsis if truncated
            if (words.length > titleWords.length) {
                title += '...';
            }

            return title || 'New Chat';
        }

        // Rename chat functionality
        function startRenameChat(chatId, event) {
            event.stopPropagation();
            const chat = chatHistory.find(c => c.id === chatId);
            if (!chat) return;

            const item = event.target.closest('.chat-history-item');
            const titleEl = item.querySelector('.chat-history-title');
            const currentTitle = chat.title;

            // Replace title with input
            titleEl.innerHTML = `<input type="text" class="chat-history-rename-input" value="${escapeHtml(currentTitle)}" />`;
            const input = titleEl.querySelector('input');
            input.focus();
            input.select();

            // Handle save on Enter or blur
            const saveRename = () => {
                const newTitle = input.value.trim() || 'New Chat';
                chat.title = newTitle;
                chat.updatedAt = new Date().toISOString();
                saveChatHistory();
                renderChatHistoryList();
            };

            input.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    saveRename();
                } else if (e.key === 'Escape') {
                    renderChatHistoryList();
                }
            });

            input.addEventListener('blur', saveRename);
        }

        function toggleChatHistory() {
            const sidebar = document.getElementById('chat-history-sidebar');
            const panel = document.getElementById('ai-panel');

            sidebar.classList.toggle('open');
            panel.classList.toggle('history-open');
        }

        // Add message to DOM (without saving)
        function addMessageToDOM(content, isUser, scroll = true) {
            const container = document.getElementById('ai-messages');
            const div = document.createElement('div');
            div.className = `message ${isUser ? 'user' : 'assistant'}`;

            let formattedContent = content;
            if (!isUser) {
                formattedContent = formatResourceLinks(marked.parse(content));
            }

            div.innerHTML = `<div class="message-content">${formattedContent}</div>`;
            container.appendChild(div);

            if (scroll) {
                container.scrollTop = container.scrollHeight;
            }
        }

        // Initialize chat history on load
        loadChatHistory();

        // ==================== K8s Safety Guardrails ====================
        const GUARDRAILS_STORAGE_KEY = 'k13d-guardrails';
        let guardrailsConfig = {
            enabled: true,
            strictMode: false,  // Block all dangerous operations
            autoAnalyze: true,  // Auto-analyze AI responses for safety
            currentNamespace: 'default',
            recentAnalysis: null,
            analysisHistory: []
        };

        // Risk level styling
        const RISK_STYLES = {
            safe: { color: 'var(--accent-green)', icon: '✓', label: 'Safe' },
            warning: { color: 'var(--accent-yellow)', icon: '⚠', label: 'Warning' },
            dangerous: { color: 'var(--accent-red)', icon: '⚡', label: 'Dangerous' },
            critical: { color: '#ff4757', icon: '☠', label: 'Critical' }
        };

        function loadGuardrailsConfig() {
            try {
                const saved = localStorage.getItem(GUARDRAILS_STORAGE_KEY);
                if (saved) {
                    guardrailsConfig = { ...guardrailsConfig, ...JSON.parse(saved) };
                }
            } catch (e) {
                console.error('Failed to load guardrails config:', e);
            }
            updateGuardrailsUI();
        }

        function saveGuardrailsConfig() {
            try {
                localStorage.setItem(GUARDRAILS_STORAGE_KEY, JSON.stringify(guardrailsConfig));
            } catch (e) {
                console.error('Failed to save guardrails config:', e);
            }
        }

        // Analyze K8s command/action safety via backend API
        async function analyzeK8sSafety(command, namespace = null) {
            if (!guardrailsConfig.enabled) {
                return { safe: true, riskLevel: 'safe', allowed: true };
            }

            try {
                const response = await fetchWithAuth('/api/safety/analyze', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        command: command,
                        namespace: namespace || guardrailsConfig.currentNamespace || currentNamespace
                    })
                });

                if (!response.ok) {
                    console.error('Safety analysis failed:', response.status);
                    return { safe: true, riskLevel: 'safe', allowed: true }; // Fail open
                }

                const analysis = await response.json();

                // Store in history
                guardrailsConfig.recentAnalysis = analysis;
                guardrailsConfig.analysisHistory.unshift({
                    command: command,
                    analysis: analysis,
                    timestamp: Date.now()
                });
                if (guardrailsConfig.analysisHistory.length > 50) {
                    guardrailsConfig.analysisHistory.pop();
                }
                saveGuardrailsConfig();

                updateGuardrailsUI(analysis);

                return {
                    ...analysis,
                    allowed: !analysis.requires_approval || !guardrailsConfig.strictMode
                };
            } catch (e) {
                console.error('Safety analysis error:', e);
                return { safe: true, riskLevel: 'safe', allowed: true }; // Fail open
            }
        }

        // Quick client-side check for common dangerous patterns
        function checkGuardrails(message) {
            if (!guardrailsConfig.enabled) {
                return { allowed: true };
            }

            const lowerMessage = message.toLowerCase();

            // Critical patterns that should always be flagged
            const criticalPatterns = [
                { pattern: 'delete namespace', reason: 'Deleting a namespace removes ALL resources within it' },
                { pattern: 'delete ns ', reason: 'Deleting a namespace removes ALL resources within it' },
                { pattern: '--all-namespaces', reason: 'Operation affects ALL namespaces in the cluster' },
                { pattern: 'drain node', reason: 'Draining a node evicts all pods' },
                { pattern: 'delete node', reason: 'Deleting a node removes it from the cluster' },
                { pattern: '--force --grace-period=0', reason: 'Force deletion bypasses graceful termination' },
                { pattern: 'rm -rf', reason: 'Recursive file deletion is dangerous' },
            ];

            // Check critical patterns
            for (const { pattern, reason } of criticalPatterns) {
                if (lowerMessage.includes(pattern)) {
                    return {
                        allowed: !guardrailsConfig.strictMode,
                        requireConfirmation: true,
                        riskLevel: 'critical',
                        reason: reason,
                        dangerous: true
                    };
                }
            }

            // Dangerous patterns that need confirmation
            const dangerousPatterns = [
                { pattern: 'delete deployment', reason: 'Deleting deployments stops all pods' },
                { pattern: 'delete statefulset', reason: 'StatefulSet deletion can cause data issues' },
                { pattern: 'delete service', reason: 'Deleting services breaks connectivity' },
                { pattern: 'delete pvc', reason: 'PVC deletion can cause data loss' },
                { pattern: 'delete secret', reason: 'Deleting secrets can break applications' },
                { pattern: 'scale --replicas=0', reason: 'Scaling to zero stops all pods' },
            ];

            for (const { pattern, reason } of dangerousPatterns) {
                if (lowerMessage.includes(pattern)) {
                    return {
                        allowed: true,
                        requireConfirmation: true,
                        riskLevel: 'dangerous',
                        reason: reason
                    };
                }
            }

            // Warning patterns
            const warningPatterns = [
                { pattern: 'delete pod', reason: 'Pod deletion causes temporary unavailability' },
                { pattern: 'scale ', reason: 'Scaling changes running pod count' },
                { pattern: 'rollout restart', reason: 'Restart causes temporary unavailability' },
                { pattern: 'apply ', reason: 'Applying changes modifies cluster state' },
                { pattern: 'patch ', reason: 'Patching modifies resource configuration' },
            ];

            for (const { pattern, reason } of warningPatterns) {
                if (lowerMessage.includes(pattern)) {
                    return {
                        allowed: true,
                        requireConfirmation: true,
                        riskLevel: 'warning',
                        reason: reason
                    };
                }
            }

            // Check for production namespace indicators
            const productionIndicators = ['prod', 'production', 'live', 'main', 'master'];
            for (const indicator of productionIndicators) {
                if (lowerMessage.includes(indicator)) {
                    return {
                        allowed: true,
                        requireConfirmation: true,
                        riskLevel: 'warning',
                        reason: 'Possible production environment detected'
                    };
                }
            }

            return { allowed: true, riskLevel: 'safe' };
        }

        // Show safety confirmation dialog
        function showSafetyConfirmation(analysis, onConfirm, onCancel) {
            const style = RISK_STYLES[analysis.riskLevel] || RISK_STYLES.warning;

            const modal = document.createElement('div');
            modal.className = 'modal-overlay';
            modal.id = 'safety-confirmation-modal';
            modal.innerHTML = `
                <div class="modal" style="max-width: 500px;">
                    <div class="modal-header" style="background: ${style.color}20; border-bottom: 2px solid ${style.color};">
                        <h2 style="color: ${style.color};">${style.icon} ${style.label}: Safety Check Required</h2>
                        <button class="modal-close" onclick="closeSafetyConfirmation(false)">&times;</button>
                    </div>
                    <div class="modal-body" style="padding: 20px;">
                        <div style="margin-bottom: 16px;">
                            <strong style="color: ${style.color};">Risk Level:</strong> ${analysis.riskLevel.toUpperCase()}
                        </div>

                        ${analysis.explanation ? `
                        <div style="margin-bottom: 16px; padding: 12px; background: var(--bg-tertiary); border-radius: 8px;">
                            ${analysis.explanation}
                        </div>
                        ` : ''}

                        ${analysis.warnings && analysis.warnings.length > 0 ? `
                        <div style="margin-bottom: 16px;">
                            <strong>Warnings:</strong>
                            <ul style="margin: 8px 0; padding-left: 20px; color: var(--accent-yellow);">
                                ${analysis.warnings.map(w => `<li>${w}</li>`).join('')}
                            </ul>
                        </div>
                        ` : ''}

                        ${analysis.recommendations && analysis.recommendations.length > 0 ? `
                        <div style="margin-bottom: 16px;">
                            <strong>Recommendations:</strong>
                            <ul style="margin: 8px 0; padding-left: 20px; color: var(--text-secondary);">
                                ${analysis.recommendations.map(r => `<li>${r}</li>`).join('')}
                            </ul>
                        </div>
                        ` : ''}

                        <div style="margin-top: 20px; padding: 12px; background: ${style.color}10; border: 1px solid ${style.color}40; border-radius: 8px;">
                            <strong>Do you want to proceed with this action?</strong>
                        </div>
                    </div>
                    <div class="modal-footer" style="display: flex; gap: 12px; justify-content: flex-end;">
                        <button class="btn btn-secondary" onclick="closeSafetyConfirmation(false)">Cancel</button>
                        <button class="btn" style="background: ${style.color};" onclick="closeSafetyConfirmation(true)">
                            Proceed Anyway
                        </button>
                    </div>
                </div>
            `;

            document.body.appendChild(modal);

            // Store callbacks
            window._safetyConfirmCallbacks = { onConfirm, onCancel };
        }

        function closeSafetyConfirmation(confirmed) {
            const modal = document.getElementById('safety-confirmation-modal');
            if (modal) {
                modal.remove();
            }

            const callbacks = window._safetyConfirmCallbacks;
            if (callbacks) {
                if (confirmed && callbacks.onConfirm) {
                    callbacks.onConfirm();
                } else if (!confirmed && callbacks.onCancel) {
                    callbacks.onCancel();
                }
                delete window._safetyConfirmCallbacks;
            }
        }

        function updateGuardrailsUI(analysis = null) {
            const indicator = document.getElementById('guardrails-indicator');
            const limitDisplay = document.getElementById('guardrails-limit');

            if (!guardrailsConfig.enabled) {
                indicator.className = 'guardrails-indicator warning';
                indicator.innerHTML = '<span class="dot"></span><span>Protection Off</span>';
                limitDisplay.textContent = 'K8s Safety: Disabled';
            } else if (analysis) {
                const style = RISK_STYLES[analysis.risk_level] || RISK_STYLES.safe;
                indicator.className = `guardrails-indicator ${analysis.risk_level || 'safe'}`;
                indicator.innerHTML = `<span class="dot" style="background: ${style.color};"></span><span>${style.label}</span>`;
                limitDisplay.textContent = `Last: ${analysis.category || 'read-only'} | ${analysis.affected_scope || 'pod'}`;
            } else {
                indicator.className = 'guardrails-indicator safe';
                indicator.innerHTML = '<span class="dot"></span><span>Protected</span>';
                limitDisplay.textContent = 'K8s Safety: Active';
            }
        }

        // Initialize guardrails
        loadGuardrailsConfig();

        // ==================== Ollama Auto-Detection ====================
        let selectedOllamaModel = null;
        let ollamaModels = [];

        async function checkOllamaStatus() {
            const statusDot = document.getElementById('ollama-status-dot');
            const statusText = document.getElementById('ollama-status-text');
            const notInstalled = document.getElementById('ollama-not-installed');
            const installed = document.getElementById('ollama-installed');

            statusDot.style.background = '#888';
            statusText.textContent = 'Checking Ollama status...';

            try {
                // Check Ollama status through backend proxy (avoids CSP/CORS issues)
                const response = await fetchWithAuth('/api/llm/ollama/status');
                if (response.ok) {
                    const data = await response.json();
                    if (data.running) {
                        ollamaModels = data.models || [];
                        statusDot.style.background = 'var(--accent-green)';
                        statusText.textContent = `Ollama running - ${ollamaModels.length} model(s) available`;
                        notInstalled.style.display = 'none';
                        installed.style.display = 'block';
                        renderOllamaModels();
                        return;
                    }
                }
                statusDot.style.background = 'var(--accent-yellow)';
                statusText.textContent = 'Ollama not detected';
                notInstalled.style.display = 'block';
                installed.style.display = 'none';
            } catch (e) {
                statusDot.style.background = 'var(--accent-yellow)';
                statusText.textContent = 'Ollama not detected';
                notInstalled.style.display = 'block';
                installed.style.display = 'none';
            }
        }

        function renderOllamaModels() {
            const container = document.getElementById('ollama-models-list');
            const useBtn = document.getElementById('use-ollama-btn');

            if (ollamaModels.length === 0) {
                container.innerHTML = '<span style="color:var(--text-secondary);font-size:12px;">No models installed. Pull a model to get started.</span>';
                useBtn.disabled = true;
                return;
            }

            container.innerHTML = ollamaModels.map(m => {
                const name = m.name || m;
                const size = m.size ? `(${formatBytes(m.size)})` : '';
                const isSelected = selectedOllamaModel === name;
                return `<button class="btn ${isSelected ? 'btn-primary' : 'btn-secondary'}"
                    onclick="selectOllamaModel('${name}')"
                    style="font-size:11px;padding:4px 10px;">
                    ${name} ${size}
                </button>`;
            }).join('');

            useBtn.disabled = !selectedOllamaModel;
        }

        function selectOllamaModel(modelName) {
            selectedOllamaModel = modelName;
            renderOllamaModels();
        }

        function useOllamaModel() {
            if (!selectedOllamaModel) return;

            // Set LLM settings to use Ollama
            document.getElementById('setting-llm-provider').value = 'ollama';
            document.getElementById('setting-llm-model').value = selectedOllamaModel;
            document.getElementById('setting-llm-endpoint').value = 'http://localhost:11434';
            document.getElementById('setting-llm-apikey').value = '';

            updateEndpointPlaceholder();
            showToast(`Configured to use Ollama model: ${selectedOllamaModel}`, 'success');
        }

        function showOllamaInstallInstructions(os) {
            const container = document.getElementById('ollama-install-instructions');
            container.style.display = 'block';

            let instructions = '';
            switch (os) {
                case 'macos':
                    instructions = `
                        <div style="margin-bottom:8px;color:var(--accent-blue);">macOS Installation:</div>
                        <div style="margin-bottom:8px;">Option 1 - Homebrew:</div>
                        <code style="display:block;background:#000;padding:8px;border-radius:4px;margin-bottom:8px;">brew install ollama</code>
                        <div style="margin-bottom:8px;">Option 2 - Direct Download:</div>
                        <a href="https://ollama.ai/download" target="_blank" style="color:var(--accent-cyan);">Download from ollama.ai →</a>
                        <div style="margin-top:12px;color:var(--text-secondary);">After installation, start Ollama:</div>
                        <code style="display:block;background:#000;padding:8px;border-radius:4px;margin-top:4px;">ollama serve</code>
                    `;
                    break;
                case 'linux':
                    instructions = `
                        <div style="margin-bottom:8px;color:var(--accent-blue);">Linux Installation:</div>
                        <div style="margin-bottom:4px;">Run this command in terminal:</div>
                        <code style="display:block;background:#000;padding:8px;border-radius:4px;margin-bottom:8px;word-break:break-all;">curl -fsSL https://ollama.ai/install.sh | sh</code>
                        <div style="margin-top:12px;color:var(--text-secondary);">After installation, start Ollama:</div>
                        <code style="display:block;background:#000;padding:8px;border-radius:4px;margin-top:4px;">ollama serve</code>
                    `;
                    break;
                case 'windows':
                    instructions = `
                        <div style="margin-bottom:8px;color:var(--accent-blue);">Windows Installation:</div>
                        <div style="margin-bottom:8px;">Download the installer from:</div>
                        <a href="https://ollama.ai/download" target="_blank" style="color:var(--accent-cyan);">Download from ollama.ai →</a>
                        <div style="margin-top:12px;color:var(--text-secondary);">After installation, Ollama will start automatically.</div>
                    `;
                    break;
            }

            container.innerHTML = instructions;
        }

        function showOllamaPullDialog() {
            const dialog = document.getElementById('ollama-pull-dialog');
            dialog.style.display = dialog.style.display === 'none' ? 'block' : 'none';
        }

        async function pullOllamaModel(modelName) {
            if (!modelName) {
                modelName = document.getElementById('ollama-custom-model').value.trim();
            }
            if (!modelName) {
                showToast('Please enter a model name', 'error');
                return;
            }

            const statusDiv = document.getElementById('ollama-pull-status');
            statusDiv.style.display = 'block';
            statusDiv.innerHTML = `<span style="color:var(--accent-yellow);">⏳ Pulling ${modelName}... This may take several minutes.</span>`;

            try {
                // Use our backend proxy for pulling
                const response = await fetchWithAuth('/api/llm/ollama/pull', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ model: modelName })
                });

                const result = await response.json();

                if (result.error) {
                    statusDiv.innerHTML = `<span style="color:var(--accent-red);">❌ Error: ${result.error}</span>`;
                } else {
                    statusDiv.innerHTML = `<span style="color:var(--accent-green);">✅ Model ${modelName} pulled successfully!</span>`;
                    // Refresh model list
                    setTimeout(checkOllamaStatus, 1000);
                }
            } catch (e) {
                // If backend doesn't support, show manual instructions
                statusDiv.innerHTML = `
                    <span style="color:var(--accent-yellow);">Run this command in your terminal:</span>
                    <code style="display:block;background:#000;padding:8px;border-radius:4px;margin-top:4px;">ollama pull ${modelName}</code>
                    <button class="btn btn-secondary" onclick="checkOllamaStatus()" style="margin-top:8px;font-size:11px;">Refresh Status</button>
                `;
            }
        }

        function formatBytes(bytes) {
            if (!bytes) return '';
            const gb = bytes / (1024 * 1024 * 1024);
            if (gb >= 1) return gb.toFixed(1) + 'GB';
            const mb = bytes / (1024 * 1024);
            return mb.toFixed(0) + 'MB';
        }

        // K8s Safety Guardrails settings UI
        function toggleGuardrailsSetting() {
            const toggle = document.getElementById('guardrails-toggle');
            guardrailsConfig.enabled = !guardrailsConfig.enabled;
            toggle.classList.toggle('active', guardrailsConfig.enabled);
            saveGuardrailsConfig();
            updateGuardrailsUI();
        }

        function toggleStrictMode() {
            const toggle = document.getElementById('guardrails-strict-toggle');
            guardrailsConfig.strictMode = !guardrailsConfig.strictMode;
            toggle.classList.toggle('active', guardrailsConfig.strictMode);
            saveGuardrailsConfig();
            showToast(guardrailsConfig.strictMode ?
                'Strict mode enabled - dangerous operations will be blocked' :
                'Strict mode disabled - dangerous operations will require confirmation',
                guardrailsConfig.strictMode ? 'warning' : 'info');
        }

        function toggleAutoAnalyze() {
            const toggle = document.getElementById('guardrails-auto-analyze');
            guardrailsConfig.autoAnalyze = !guardrailsConfig.autoAnalyze;
            toggle.classList.toggle('active', guardrailsConfig.autoAnalyze);
            saveGuardrailsConfig();
        }

        function clearGuardrailsHistory() {
            guardrailsConfig.analysisHistory = [];
            guardrailsConfig.recentAnalysis = null;
            saveGuardrailsConfig();
            updateGuardrailsHistoryUI();
            updateGuardrailsUI();
            showToast('Safety check history cleared', 'success');
        }

        function updateGuardrailsHistoryUI() {
            const historyDiv = document.getElementById('guardrails-history');
            if (!historyDiv) return;

            if (!guardrailsConfig.analysisHistory || guardrailsConfig.analysisHistory.length === 0) {
                historyDiv.innerHTML = '<div style="color:var(--text-secondary); font-size:13px;">No recent checks</div>';
                return;
            }

            const html = guardrailsConfig.analysisHistory.slice(0, 10).map(item => {
                const style = RISK_STYLES[item.analysis.risk_level] || RISK_STYLES.safe;
                const time = formatTime(item.timestamp);
                const cmd = item.command.length > 50 ? item.command.substring(0, 47) + '...' : item.command;
                return `
                    <div style="display:flex; align-items:center; gap:8px; padding:6px 0; border-bottom:1px solid var(--border-color);">
                        <span style="color:${style.color}; font-size:14px;">${style.icon}</span>
                        <span style="flex:1; font-size:12px; font-family:monospace; color:var(--text-secondary);" title="${item.command}">${cmd}</span>
                        <span style="font-size:11px; color:var(--text-secondary);">${time}</span>
                    </div>
                `;
            }).join('');

            historyDiv.innerHTML = html;
        }

        function loadGuardrailsSettingsUI() {
            document.getElementById('guardrails-toggle').classList.toggle('active', guardrailsConfig.enabled);
            document.getElementById('guardrails-strict-toggle').classList.toggle('active', guardrailsConfig.strictMode || false);
            document.getElementById('guardrails-auto-analyze').classList.toggle('active', guardrailsConfig.autoAnalyze !== false);
            updateGuardrailsHistoryUI();
        }

        // Check Ollama on settings open
        const originalShowSettings = typeof showSettings === 'function' ? showSettings : null;

        // Initialize Ollama check when LLM tab is opened
        function onLLMTabOpened() {
            checkOllamaStatus();
            loadGuardrailsSettingsUI();
        }

        // Shortcuts modal
        function showShortcuts() {
            document.getElementById('shortcuts-modal').classList.add('active');
        }

        function closeShortcuts() {
            document.getElementById('shortcuts-modal').classList.remove('active');
        }

        // Resource detail modal
        let selectedResource = null;

        // Generate resource-specific overview HTML
        function generateResourceOverview(resource, item) {
            switch (resource) {
                case 'pods':
                    return generatePodOverview(item);
                case 'deployments':
                    return generateDeploymentOverview(item);
                case 'services':
                    return generateServiceOverview(item);
                case 'statefulsets':
                    return generateStatefulSetOverview(item);
                case 'daemonsets':
                    return generateDaemonSetOverview(item);
                case 'nodes':
                    return generateNodeOverview(item);
                case 'configmaps':
                    return generateConfigMapOverview(item);
                case 'secrets':
                    return generateSecretOverview(item);
                case 'ingresses':
                    return generateIngressOverview(item);
                case 'jobs':
                    return generateJobOverview(item);
                case 'cronjobs':
                    return generateCronJobOverview(item);
                case 'pvcs':
                    return generatePVCOverview(item);
                case 'pvs':
                    return generatePVOverview(item);
                default:
                    return generateDefaultOverview(item);
            }
        }

        // Default overview (key-value pairs)
        function generateDefaultOverview(item) {
            const html = Object.entries(item).map(([key, value]) =>
                `<div class="property-label">${key}</div><div class="property-value">${escapeHtml(String(value || '-'))}</div>`
            ).join('');
            return `<div class="property-grid">${html}</div>`;
        }

        // Pod Overview
        function generatePodOverview(item) {
            const statusColor = item.status === 'Running' ? 'var(--accent-green)' :
                               item.status === 'Pending' ? 'var(--accent-yellow)' :
                               item.status === 'Failed' || item.status === 'Error' ? 'var(--accent-red)' : 'var(--text-secondary)';
            const restarts = parseInt(item.restarts) || 0;
            const restartColor = restarts > 5 ? 'var(--accent-red)' : restarts > 0 ? 'var(--accent-yellow)' : 'var(--accent-green)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📦 Container Status</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: var(--accent-green);">${escapeHtml(item.ready || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Restarts</span>
                                <span class="stat-value" style="color: ${restartColor};">${restarts}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🖥️ Node & Network</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Node</span>
                                <span class="stat-value">${escapeHtml(item.node || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Pod IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.ip || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="overview-actions">
                    <button class="btn btn-secondary" onclick="openLogViewerDirect('${escapeHtml(item.name)}', '${escapeHtml(item.namespace || '')}')">📋 View Logs</button>
                </div>
            `;
        }

        // Deployment Overview
        function generateDeploymentOverview(item) {
            const ready = item.ready || '0/0';
            const [readyCount, totalCount] = ready.split('/').map(n => parseInt(n) || 0);
            const healthPercent = totalCount > 0 ? Math.round((readyCount / totalCount) * 100) : 0;
            const healthColor = healthPercent === 100 ? 'var(--accent-green)' : healthPercent >= 50 ? 'var(--accent-yellow)' : 'var(--accent-red)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${healthColor}20; color: ${healthColor}; border: 1px solid ${healthColor}40;">
                        <span class="status-dot" style="background: ${healthColor};"></span>
                        ${healthPercent === 100 ? 'Healthy' : healthPercent > 0 ? 'Degraded' : 'Unavailable'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Replicas</div>
                        <div class="overview-card-content">
                            <div class="overview-progress">
                                <div class="progress-bar" style="width: ${healthPercent}%; background: ${healthColor};"></div>
                            </div>
                            <div class="overview-stat" style="margin-top: 8px;">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: ${healthColor};">${ready}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Up-to-date</span>
                                <span class="stat-value">${escapeHtml(item.upToDate || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Available</span>
                                <span class="stat-value">${escapeHtml(item.available || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // Service Overview
        function generateServiceOverview(item) {
            const typeColors = {
                'ClusterIP': 'var(--accent-blue)',
                'NodePort': 'var(--accent-purple)',
                'LoadBalancer': 'var(--accent-green)',
                'ExternalName': 'var(--accent-yellow)'
            };
            const typeColor = typeColors[item.type] || 'var(--text-secondary)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${typeColor}20; color: ${typeColor}; border: 1px solid ${typeColor}40;">
                        ${escapeHtml(item.type || 'Unknown')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">🌐 Network</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Cluster IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.clusterIP || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">External IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.externalIP || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🔌 Ports</div>
                        <div class="overview-card-content">
                            <div class="ports-list">${escapeHtml(item.ports || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // StatefulSet Overview
        function generateStatefulSetOverview(item) {
            const ready = item.ready || '0/0';
            const [readyCount, totalCount] = ready.split('/').map(n => parseInt(n) || 0);
            const healthPercent = totalCount > 0 ? Math.round((readyCount / totalCount) * 100) : 0;
            const healthColor = healthPercent === 100 ? 'var(--accent-green)' : healthPercent >= 50 ? 'var(--accent-yellow)' : 'var(--accent-red)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${healthColor}20; color: ${healthColor}; border: 1px solid ${healthColor}40;">
                        <span class="status-dot" style="background: ${healthColor};"></span>
                        ${healthPercent === 100 ? 'Healthy' : healthPercent > 0 ? 'Degraded' : 'Unavailable'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Replicas</div>
                        <div class="overview-card-content">
                            <div class="overview-progress">
                                <div class="progress-bar" style="width: ${healthPercent}%; background: ${healthColor};"></div>
                            </div>
                            <div class="overview-stat" style="margin-top: 8px;">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: ${healthColor};">${ready}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // DaemonSet Overview
        function generateDaemonSetOverview(item) {
            const ready = parseInt(item.ready) || 0;
            const desired = parseInt(item.desired) || 0;
            const healthPercent = desired > 0 ? Math.round((ready / desired) * 100) : 0;
            const healthColor = healthPercent === 100 ? 'var(--accent-green)' : healthPercent >= 50 ? 'var(--accent-yellow)' : 'var(--accent-red)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${healthColor}20; color: ${healthColor}; border: 1px solid ${healthColor}40;">
                        <span class="status-dot" style="background: ${healthColor};"></span>
                        ${healthPercent === 100 ? 'Healthy' : healthPercent > 0 ? 'Degraded' : 'Unavailable'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Node Coverage</div>
                        <div class="overview-card-content">
                            <div class="overview-progress">
                                <div class="progress-bar" style="width: ${healthPercent}%; background: ${healthColor};"></div>
                            </div>
                            <div class="overview-stat" style="margin-top: 8px;">
                                <span class="stat-label">Desired</span>
                                <span class="stat-value">${desired}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Current</span>
                                <span class="stat-value">${escapeHtml(item.current || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Ready</span>
                                <span class="stat-value" style="color: ${healthColor};">${ready}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Available</span>
                                <span class="stat-value">${escapeHtml(item.available || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // Node Overview
        function generateNodeOverview(item) {
            const statusColor = item.status === 'Ready' ? 'var(--accent-green)' : 'var(--accent-red)';
            const roles = item.roles || '-';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                    <div class="overview-roles">
                        ${roles.split(',').map(r => `<span class="role-badge">${escapeHtml(r.trim())}</span>`).join('')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">💻 System Info</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Version</span>
                                <span class="stat-value">${escapeHtml(item.version || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">OS</span>
                                <span class="stat-value">${escapeHtml(item.os || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Arch</span>
                                <span class="stat-value">${escapeHtml(item.arch || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📦 Capacity</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">CPU</span>
                                <span class="stat-value">${escapeHtml(item.cpu || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Memory</span>
                                <span class="stat-value">${escapeHtml(item.memory || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Pods</span>
                                <span class="stat-value">${escapeHtml(item.pods || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🌐 Network</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Internal IP</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.internalIP || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // ConfigMap Overview
        function generateConfigMapOverview(item) {
            return `
                <div class="overview-cards">
                    <div class="overview-card" style="grid-column: span 2;">
                        <div class="overview-card-title">📝 Data</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Keys</span>
                                <span class="stat-value">${escapeHtml(item.data || '0')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // Secret Overview
        function generateSecretOverview(item) {
            const typeColors = {
                'Opaque': 'var(--accent-blue)',
                'kubernetes.io/service-account-token': 'var(--accent-purple)',
                'kubernetes.io/dockerconfigjson': 'var(--accent-green)',
                'kubernetes.io/tls': 'var(--accent-yellow)'
            };
            const typeColor = typeColors[item.type] || 'var(--text-secondary)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${typeColor}20; color: ${typeColor}; border: 1px solid ${typeColor}40;">
                        🔒 ${escapeHtml(item.type || 'Unknown')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card" style="grid-column: span 2;">
                        <div class="overview-card-title">🔐 Data</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Keys</span>
                                <span class="stat-value">${escapeHtml(item.data || '0')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // Ingress Overview
        function generateIngressOverview(item) {
            return `
                <div class="overview-cards">
                    <div class="overview-card" style="grid-column: span 2;">
                        <div class="overview-card-title">🌐 Routing</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Class</span>
                                <span class="stat-value">${escapeHtml(item.class || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Hosts</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.hosts || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Address</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.address || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // Job Overview
        function generateJobOverview(item) {
            const statusColor = item.status === 'Complete' ? 'var(--accent-green)' :
                               item.status === 'Running' ? 'var(--accent-blue)' :
                               item.status === 'Failed' ? 'var(--accent-red)' : 'var(--text-secondary)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status || 'Unknown')}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">📊 Completion</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Completions</span>
                                <span class="stat-value">${escapeHtml(item.completions || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Duration</span>
                                <span class="stat-value">${escapeHtml(item.duration || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // CronJob Overview
        function generateCronJobOverview(item) {
            const suspendColor = item.suspend === 'True' ? 'var(--accent-yellow)' : 'var(--accent-green)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${suspendColor}20; color: ${suspendColor}; border: 1px solid ${suspendColor}40;">
                        ${item.suspend === 'True' ? '⏸️ Suspended' : '▶️ Active'}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">⏰ Schedule</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Schedule</span>
                                <span class="stat-value" style="font-family: monospace;">${escapeHtml(item.schedule || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Last Schedule</span>
                                <span class="stat-value">${escapeHtml(item.lastSchedule || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Active Jobs</span>
                                <span class="stat-value">${escapeHtml(item.active || '0')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🐳 Container Image</div>
                        <div class="overview-card-content">
                            <div class="image-tag" title="${escapeHtml(item.image || '-')}">${escapeHtml(item.image || '-')}</div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // PVC Overview
        function generatePVCOverview(item) {
            const statusColor = item.status === 'Bound' ? 'var(--accent-green)' :
                               item.status === 'Pending' ? 'var(--accent-yellow)' : 'var(--accent-red)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">💾 Storage</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Capacity</span>
                                <span class="stat-value">${escapeHtml(item.capacity || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Access Modes</span>
                                <span class="stat-value">${escapeHtml(item.accessModes || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Storage Class</span>
                                <span class="stat-value">${escapeHtml(item.storageClass || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🔗 Volume</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Volume</span>
                                <span class="stat-value" style="font-family: monospace; font-size: 11px;">${escapeHtml(item.volume || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Namespace</span>
                                <span class="stat-value">${escapeHtml(item.namespace || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        // PV Overview
        function generatePVOverview(item) {
            const statusColor = item.status === 'Available' ? 'var(--accent-green)' :
                               item.status === 'Bound' ? 'var(--accent-blue)' :
                               item.status === 'Released' ? 'var(--accent-yellow)' : 'var(--accent-red)';

            return `
                <div class="resource-overview-header">
                    <div class="overview-status-badge" style="background: ${statusColor}20; color: ${statusColor}; border: 1px solid ${statusColor}40;">
                        <span class="status-dot" style="background: ${statusColor};"></span>
                        ${escapeHtml(item.status)}
                    </div>
                </div>
                <div class="overview-cards">
                    <div class="overview-card">
                        <div class="overview-card-title">💾 Storage</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Capacity</span>
                                <span class="stat-value">${escapeHtml(item.capacity || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Access Modes</span>
                                <span class="stat-value">${escapeHtml(item.accessModes || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Reclaim Policy</span>
                                <span class="stat-value">${escapeHtml(item.reclaimPolicy || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">🔗 Claim</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Claim</span>
                                <span class="stat-value" style="font-family: monospace; font-size: 11px;">${escapeHtml(item.claim || '-')}</span>
                            </div>
                            <div class="overview-stat">
                                <span class="stat-label">Storage Class</span>
                                <span class="stat-value">${escapeHtml(item.storageClass || '-')}</span>
                            </div>
                        </div>
                    </div>
                    <div class="overview-card">
                        <div class="overview-card-title">📋 Metadata</div>
                        <div class="overview-card-content">
                            <div class="overview-stat">
                                <span class="stat-label">Age</span>
                                <span class="stat-value">${escapeHtml(item.age || '-')}</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        function showResourceDetail(item) {
            selectedResource = item;
            document.getElementById('detail-title').textContent = `${currentResource.slice(0, -1)}: ${item.name}`;

            // Overview tab - use resource-specific generator
            const overviewHtml = generateResourceOverview(currentResource, item);
            document.getElementById('detail-overview').innerHTML = overviewHtml;

            // YAML tab - will be loaded on demand
            document.getElementById('detail-yaml').innerHTML = `<div class="yaml-viewer" style="color: var(--text-secondary);">Click the YAML tab to load...</div>`;
            document.getElementById('detail-yaml').dataset.loaded = 'false';

            // Events tab - will be loaded on demand
            document.getElementById('detail-events').innerHTML = '<p style="color: var(--text-secondary);">Click the Events tab to load...</p>';
            document.getElementById('detail-events').dataset.loaded = 'false';

            // Related Pods tab - only for Services, Deployments, StatefulSets, DaemonSets, ReplicaSets
            const podsTab = document.getElementById('detail-pods-tab');
            const podsContent = document.getElementById('detail-pods');
            const workloadResources = ['services', 'deployments', 'statefulsets', 'daemonsets', 'replicasets'];
            if (workloadResources.includes(currentResource)) {
                podsTab.style.display = 'inline-block';
                podsContent.innerHTML = '<p style="color: var(--text-secondary);">Click the Related Pods tab to load...</p>';
                podsContent.dataset.loaded = 'false';
            } else {
                podsTab.style.display = 'none';
            }

            document.getElementById('detail-modal').classList.add('active');
            switchDetailTab('overview');
        }

        async function switchDetailTab(tab) {
            document.querySelectorAll('.detail-tab').forEach(t => t.classList.remove('active'));
            document.querySelector(`.detail-tab[onclick*="${tab}"]`).classList.add('active');

            document.getElementById('detail-overview').style.display = tab === 'overview' ? 'block' : 'none';
            document.getElementById('detail-yaml').style.display = tab === 'yaml' ? 'block' : 'none';
            document.getElementById('detail-events').style.display = tab === 'events' ? 'block' : 'none';
            document.getElementById('detail-pods').style.display = tab === 'pods' ? 'block' : 'none';

            // Load YAML on demand
            if (tab === 'yaml' && selectedResource) {
                const yamlEl = document.getElementById('detail-yaml');
                if (yamlEl.dataset.loaded !== 'true') {
                    yamlEl.innerHTML = `<div class="yaml-viewer" style="color: var(--text-secondary);">Loading YAML...</div>`;
                    try {
                        let url;
                        if (selectedResource._isCR) {
                            // Custom Resource: use CRD API
                            const crdName = selectedResource._crdName;
                            const ns = selectedResource.namespace ? `&namespace=${encodeURIComponent(selectedResource.namespace)}` : '';
                            url = `/api/crd/${crdName}/instances/${encodeURIComponent(selectedResource.name)}?format=yaml${ns}`;
                        } else {
                            // Built-in resource: use k8s API
                            const ns = selectedResource.namespace || '';
                            url = `/api/k8s/${currentResource}?name=${encodeURIComponent(selectedResource.name)}&namespace=${encodeURIComponent(ns)}&format=yaml`;
                        }
                        const response = await fetchWithAuth(url);
                        if (!response.ok) {
                            throw new Error(await response.text());
                        }
                        const yaml = await response.text();
                        yamlEl.innerHTML = `<pre class="yaml-viewer">${escapeHtml(yaml)}</pre>`;
                        yamlEl.dataset.loaded = 'true';
                    } catch (error) {
                        yamlEl.innerHTML = `<div class="yaml-viewer" style="color: var(--accent-red);">Error loading YAML: ${escapeHtml(error.message)}</div>`;
                    }
                }
            }

            // Load Events on demand
            if (tab === 'events' && selectedResource) {
                const eventsEl = document.getElementById('detail-events');
                if (eventsEl.dataset.loaded !== 'true') {
                    eventsEl.innerHTML = '<p style="color: var(--text-secondary);">Loading events...</p>';
                    try {
                        const ns = selectedResource.namespace || '';
                        const url = `/api/k8s/events?namespace=${encodeURIComponent(ns)}`;
                        const response = await fetch(url, { credentials: 'include' });
                        if (!response.ok) {
                            throw new Error(await response.text());
                        }
                        const data = await response.json();
                        // Filter events related to this resource
                        const resourceName = selectedResource.name;
                        const relatedEvents = (data.items || []).filter(e =>
                            (e.involvedObject && e.involvedObject.name === resourceName) ||
                            (e.message && e.message.includes(resourceName))
                        );

                        if (relatedEvents.length === 0) {
                            eventsEl.innerHTML = '<p style="color: var(--text-secondary);">No events found for this resource.</p>';
                        } else {
                            const eventsHtml = relatedEvents.map(e => `
                                <div class="event-item" style="padding: 8px; margin-bottom: 8px; border-left: 3px solid ${e.type === 'Warning' ? 'var(--accent-yellow)' : 'var(--accent-green)'}; background: var(--bg-secondary);">
                                    <div style="display: flex; justify-content: space-between; margin-bottom: 4px;">
                                        <span style="font-weight: 500; color: ${e.type === 'Warning' ? 'var(--accent-yellow)' : 'var(--accent-green)'}">${escapeHtml(e.reason || 'Unknown')}</span>
                                        <span style="color: var(--text-secondary); font-size: 12px;">${escapeHtml(e.lastSeen || '')}</span>
                                    </div>
                                    <div style="color: var(--text-primary); font-size: 13px;">${escapeHtml(e.message || '')}</div>
                                    ${e.count > 1 ? `<div style="color: var(--text-secondary); font-size: 11px; margin-top: 4px;">Count: ${e.count}</div>` : ''}
                                </div>
                            `).join('');
                            eventsEl.innerHTML = eventsHtml;
                        }
                        eventsEl.dataset.loaded = 'true';
                    } catch (error) {
                        eventsEl.innerHTML = `<p style="color: var(--accent-red);">Error loading events: ${escapeHtml(error.message)}</p>`;
                    }
                }
            }

            // Load Related Pods on demand (for Services, Deployments, etc.)
            if (tab === 'pods' && selectedResource) {
                const podsEl = document.getElementById('detail-pods');
                if (podsEl.dataset.loaded !== 'true') {
                    podsEl.innerHTML = '<p style="color: var(--text-secondary);">Loading related pods...</p>';
                    try {
                        const ns = selectedResource.namespace || '';
                        // First, get the resource's YAML to extract the selector
                        const yamlUrl = `/api/k8s/${currentResource}?name=${encodeURIComponent(selectedResource.name)}&namespace=${encodeURIComponent(ns)}&format=yaml`;
                        const yamlResp = await fetch(yamlUrl, { credentials: 'include' });
                        if (!yamlResp.ok) {
                            throw new Error('Failed to fetch resource details');
                        }
                        const yamlText = await yamlResp.text();

                        // Parse selector from YAML (simple parsing for common patterns)
                        let labelSelector = '';
                        if (currentResource === 'services') {
                            // Services use spec.selector
                            const selectorMatch = yamlText.match(/spec:\s*[\s\S]*?selector:\s*\n((?:\s+\w+:\s*\S+\n?)+)/);
                            if (selectorMatch) {
                                const selectorLines = selectorMatch[1].trim().split('\n');
                                const selectors = [];
                                for (const line of selectorLines) {
                                    const match = line.trim().match(/^(\S+):\s*(.+)$/);
                                    if (match && !match[1].startsWith('match')) {
                                        selectors.push(`${match[1]}=${match[2].trim()}`);
                                    }
                                }
                                labelSelector = selectors.join(',');
                            }
                        } else {
                            // Deployments, StatefulSets, etc. use spec.selector.matchLabels
                            const matchLabelsMatch = yamlText.match(/matchLabels:\s*\n((?:\s+\w[\w.-]*:\s*\S+\n?)+)/);
                            if (matchLabelsMatch) {
                                const labelLines = matchLabelsMatch[1].trim().split('\n');
                                const selectors = [];
                                for (const line of labelLines) {
                                    const match = line.trim().match(/^([\w.-]+):\s*(.+)$/);
                                    if (match) {
                                        selectors.push(`${match[1]}=${match[2].trim()}`);
                                    }
                                }
                                labelSelector = selectors.join(',');
                            }
                        }

                        if (!labelSelector) {
                            podsEl.innerHTML = '<p style="color: var(--text-secondary);">No selector found for this resource.</p>';
                            podsEl.dataset.loaded = 'true';
                            return;
                        }

                        // Fetch pods with the label selector
                        const podsUrl = `/api/k8s/pods?namespace=${encodeURIComponent(ns)}&labelSelector=${encodeURIComponent(labelSelector)}`;
                        const podsResp = await fetchWithAuth(podsUrl);
                        const podsData = await podsResp.json();

                        if (!podsData.items || podsData.items.length === 0) {
                            podsEl.innerHTML = `
                                <p style="color: var(--text-secondary);">No pods found matching selector:</p>
                                <code style="display: block; padding: 8px; background: var(--bg-secondary); border-radius: 4px; font-size: 12px; margin-top: 8px;">${escapeHtml(labelSelector)}</code>
                            `;
                            podsEl.dataset.loaded = 'true';
                            return;
                        }

                        // Render pods table
                        let podsHtml = `
                            <div style="margin-bottom: 12px;">
                                <span style="color: var(--text-secondary); font-size: 12px;">Selector: </span>
                                <code style="padding: 2px 6px; background: var(--bg-secondary); border-radius: 3px; font-size: 11px;">${escapeHtml(labelSelector)}</code>
                                <span style="color: var(--text-secondary); font-size: 12px; margin-left: 12px;">${podsData.items.length} pod(s)</span>
                            </div>
                            <table class="data-table" style="font-size: 12px;">
                                <thead>
                                    <tr>
                                        <th>NAME</th>
                                        <th>STATUS</th>
                                        <th>READY</th>
                                        <th>RESTARTS</th>
                                        <th>NODE</th>
                                        <th>AGE</th>
                                        <th>ACTIONS</th>
                                    </tr>
                                </thead>
                                <tbody>
                        `;

                        for (const pod of podsData.items) {
                            const statusColor = pod.status === 'Running' ? 'var(--accent-green)' :
                                               pod.status === 'Pending' ? 'var(--accent-yellow)' :
                                               pod.status === 'Failed' ? 'var(--accent-red)' : 'var(--text-secondary)';

                            podsHtml += `
                                <tr style="cursor: pointer;" onclick="viewPodFromDetail('${escapeHtml(pod.name)}', '${escapeHtml(pod.namespace || '')}')">
                                    <td style="color: var(--accent-blue);">${escapeHtml(pod.name)}</td>
                                    <td><span style="color: ${statusColor};">${escapeHtml(pod.status)}</span></td>
                                    <td>${escapeHtml(pod.ready || '-')}</td>
                                    <td>${escapeHtml(pod.restarts || '0')}</td>
                                    <td style="color: var(--text-secondary);">${escapeHtml(pod.node || '-')}</td>
                                    <td style="color: var(--text-secondary);">${escapeHtml(pod.age || '-')}</td>
                                    <td class="resource-actions" onclick="event.stopPropagation();">
                                        <button class="resource-action-btn" onclick="openLogViewerDirect('${escapeHtml(pod.name)}', '${escapeHtml(pod.namespace || '')}')" title="View Logs">📋</button>
                                    </td>
                                </tr>
                            `;
                        }

                        podsHtml += '</tbody></table>';
                        podsEl.innerHTML = podsHtml;
                        podsEl.dataset.loaded = 'true';
                    } catch (error) {
                        podsEl.innerHTML = `<p style="color: var(--accent-red);">Error loading related pods: ${escapeHtml(error.message)}</p>`;
                    }
                }
            }
        }

        // Helper function to view pod details from the related pods tab
        function viewPodFromDetail(podName, namespace) {
            closeDetail();
            // Switch to pods view and find the pod
            switchResource('pods');
            setTimeout(() => {
                // Try to find and highlight the pod in the table
                const rows = document.querySelectorAll('#table-body tr');
                for (const row of rows) {
                    const nameCell = row.querySelector('td:first-child');
                    if (nameCell && nameCell.textContent.trim() === podName) {
                        row.click();
                        row.scrollIntoView({ behavior: 'smooth', block: 'center' });
                        break;
                    }
                }
            }, 500);
        }

        // Helper function to open log viewer directly (without row context)
        function openLogViewerDirect(podName, namespace) {
            openLogViewer(podName, namespace, ['default']);
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function closeDetail() {
            document.getElementById('detail-modal').classList.remove('active');
            selectedResource = null;
        }

        function analyzeWithAI() {
            if (selectedResource) {
                const msg = `Analyze this ${currentResource.slice(0, -1)}: ${selectedResource.name} in namespace ${selectedResource.namespace || 'N/A'}. Current status: ${selectedResource.status || 'unknown'}`;
                document.getElementById('ai-input').value = msg;
                closeDetail();
                document.getElementById('ai-input').focus();
            }
        }

        // Override renderTable to include click handlers and cache data
        const originalRenderTable = renderTable;
        renderTable = function(resource, items) {
            cachedData = items || [];
            originalRenderTable(resource, items);
            addRowClickHandlers();
        };

        // ==========================================
        // Sidebar Toggle
        // ==========================================
        function toggleSidebar() {
            const sidebar = document.getElementById('sidebar');
            const hamburger = document.getElementById('hamburger-btn');
            const isMobile = window.innerWidth <= 768;

            if (isMobile) {
                const isOpen = sidebar.classList.contains('mobile-open');
                sidebar.classList.toggle('mobile-open', !isOpen);
                hamburger.classList.toggle('active', !isOpen);
                // Show/hide overlay
                let overlay = document.getElementById('sidebar-overlay');
                if (!isOpen) {
                    if (!overlay) {
                        overlay = document.createElement('div');
                        overlay.id = 'sidebar-overlay';
                        overlay.style.cssText = 'position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.5);z-index:199;';
                        overlay.onclick = function() { toggleSidebar(); };
                        document.body.appendChild(overlay);
                    }
                    overlay.style.display = 'block';
                } else if (overlay) {
                    overlay.style.display = 'none';
                }
            } else {
                sidebarCollapsed = !sidebarCollapsed;
                sidebar.classList.toggle('collapsed', sidebarCollapsed);
                hamburger.classList.toggle('active', sidebarCollapsed);
                localStorage.setItem('k13d_sidebar_collapsed', sidebarCollapsed);
            }
        }

        // Close mobile sidebar when a nav item is clicked
        function closeMobileSidebar() {
            if (window.innerWidth <= 768) {
                const sidebar = document.getElementById('sidebar');
                if (sidebar.classList.contains('mobile-open')) {
                    toggleSidebar();
                }
            }
        }

        // ==========================================
        // Debug Mode (MCP Tool Calling)
        // ==========================================
        let debugLogs = [];

        function toggleDebugMode() {
            debugMode = !debugMode;
            const panel = document.getElementById('debug-panel');
            const toggle = document.getElementById('debug-toggle');

            panel.classList.toggle('active', debugMode);
            toggle.style.background = debugMode ? 'var(--accent-purple)' : 'transparent';
            localStorage.setItem('k13d_debug_mode', debugMode);
        }

        function addDebugLog(type, title, content) {
            if (!debugMode) return;

            const timestamp = new Date().toLocaleTimeString();
            debugLogs.push({ type, title, content, timestamp });

            const container = document.getElementById('debug-content');
            const entry = document.createElement('div');
            entry.className = `debug-entry ${type}`;
            entry.innerHTML = `
                <div class="debug-entry-header">
                    <span>${title}</span>
                    <span>${timestamp}</span>
                </div>
                <div class="debug-entry-body">${typeof content === 'object' ? JSON.stringify(content, null, 2) : content}</div>
            `;
            container.appendChild(entry);
            container.scrollTop = container.scrollHeight;
        }

        function clearDebugLogs() {
            debugLogs = [];
            document.getElementById('debug-content').innerHTML = `
                <div style="color: var(--text-secondary); text-align: center; padding: 20px;">
                    Debug logs cleared. Tool calls will appear here.
                </div>
            `;
        }

        // ==========================================
        // AI Context Management
        // ==========================================
        function addToAIContext(item) {
            // Check if already exists
            const exists = aiContextItems.find(i => i.name === item.name && i.namespace === item.namespace);
            if (exists) return;

            aiContextItems.push({
                type: currentResource,
                name: item.name,
                namespace: item.namespace || '',
                data: item
            });

            renderContextChips();
        }

        function removeFromAIContext(index) {
            aiContextItems.splice(index, 1);
            renderContextChips();
        }

        function clearAIContext() {
            aiContextItems = [];
            renderContextChips();
        }

        function renderContextChips() {
            const container = document.getElementById('context-chips');
            if (aiContextItems.length === 0) {
                container.innerHTML = '<span style="font-size: 11px; color: var(--text-secondary);">Context: Click resources to add</span>';
                return;
            }

            container.innerHTML = aiContextItems.map((item, index) => `
                <span class="context-chip">
                    ${item.type.slice(0, -1)}: ${item.name}
                    <span class="remove" onclick="event.stopPropagation(); removeFromAIContext(${index})">×</span>
                </span>
            `).join('') + `<span class="context-chip" style="background: var(--bg-tertiary); cursor: pointer;" onclick="clearAIContext()">Clear all</span>`;
        }

        function getContextForAI() {
            if (aiContextItems.length === 0) return '';

            return '\n\nContext from selected resources:\n' + aiContextItems.map(item => {
                return `[${item.type}] ${item.name}${item.namespace ? ` (ns: ${item.namespace})` : ''}: ${JSON.stringify(item.data)}`;
            }).join('\n');
        }

        // Update addRowClickHandlers - explicit button for AI context only
        function addRowClickHandlers() {
            document.querySelectorAll('#table-body tr').forEach((row, index) => {
                // Left click - show detail modal (but not if clicking on action buttons)
                row.onclick = (e) => {
                    // Ignore clicks on action buttons or their container
                    if (e.target.closest('.resource-actions') || e.target.closest('.resource-action-btn')) {
                        return;
                    }
                    const item = cachedData[index];
                    if (item) {
                        showResourceDetail(item);
                    }
                };

                // Add explicit + button for adding to context
                const firstCell = row.querySelector('td:first-child');
                if (firstCell && !firstCell.querySelector('.add-context-btn')) {
                    const btn = document.createElement('button');
                    btn.className = 'add-context-btn';
                    btn.textContent = '+';
                    btn.title = 'Add to AI context';
                    btn.onclick = (e) => {
                        e.stopPropagation();
                        const item = cachedData[index];
                        if (item) {
                            addToAIContext(item);
                            // Visual feedback
                            btn.textContent = '✓';
                            btn.style.background = 'var(--accent-green)';
                            setTimeout(() => {
                                btn.textContent = '+';
                                btn.style.background = '';
                            }, 1000);
                        }
                    };
                    firstCell.prepend(btn);
                    firstCell.prepend(document.createTextNode(' '));
                }
            });
        }

        // Override sendMessage to include context (uses agentic mode)
        const originalSendMessage = sendMessage;
        sendMessage = async function() {
            const input = document.getElementById('ai-input');
            let message = input.value.trim();
            if (!message || isLoading) return;

            // Add context if available
            const contextStr = getContextForAI();
            if (contextStr) {
                message += contextStr;
            }

            // Log request in debug mode
            addDebugLog('request', 'AI Request', { message, context: aiContextItems });

            isLoading = true;
            document.getElementById('send-btn').disabled = true;
            input.value = '';

            addMessage(message.split('\n\nContext from selected resources:')[0], true);

            // Use agentic mode
            await sendMessageAgentic(message);

            isLoading = false;
            document.getElementById('send-btn').disabled = false;
        };

        // ==========================================
        // Terminal Functions (xterm.js + WebSocket)
        // ==========================================
        let currentTerminal = null;
        let currentTerminalWs = null;
        let terminalFitAddon = null;
        let terminalReconnectAttempts = 0;
        let terminalReconnectTimer = null;
        let terminalHeartbeatInterval = null;
        let terminalPodName = null;
        let terminalNamespace = null;
        let terminalContainer = null;
        let terminalShouldReconnect = true;

        function openTerminal(podName, namespace, container = '') {
            // Store connection params for reconnection
            terminalPodName = podName;
            terminalNamespace = namespace;
            terminalContainer = container;
            terminalShouldReconnect = true;
            const modal = document.getElementById('terminal-modal');
            document.getElementById('terminal-pod-name').textContent = podName;
            document.getElementById('terminal-container-name').textContent = container || 'default';

            modal.classList.add('active');

            // Initialize xterm.js
            const terminalEl = document.getElementById('terminal-container');
            terminalEl.innerHTML = '';

            currentTerminal = new Terminal({
                cursorBlink: true,
                fontSize: 14,
                fontFamily: "'SF Mono', 'Monaco', 'Menlo', monospace",
                theme: {
                    background: '#1a1b26',
                    foreground: '#c0caf5',
                    cursor: '#c0caf5',
                    selection: '#33467c',
                    black: '#15161e',
                    red: '#f7768e',
                    green: '#9ece6a',
                    yellow: '#e0af68',
                    blue: '#7aa2f7',
                    magenta: '#bb9af7',
                    cyan: '#7dcfff',
                    white: '#a9b1d6'
                }
            });

            terminalFitAddon = new FitAddon.FitAddon();
            currentTerminal.loadAddon(terminalFitAddon);

            if (typeof WebLinksAddon !== 'undefined') {
                const webLinksAddon = new WebLinksAddon.WebLinksAddon();
                currentTerminal.loadAddon(webLinksAddon);
            }

            currentTerminal.open(terminalEl);
            terminalFitAddon.fit();

            // Connect WebSocket with reconnection support
            connectTerminalWebSocket();
        }

        function connectTerminalWebSocket() {
            // Clean up existing connection
            if (terminalHeartbeatInterval) {
                clearInterval(terminalHeartbeatInterval);
                terminalHeartbeatInterval = null;
            }
            if (currentTerminalWs) {
                currentTerminalWs.onclose = null; // Prevent reconnection loop
                currentTerminalWs.close();
                currentTerminalWs = null;
            }

            // Build WebSocket URL with auth token (WebSocket cannot set headers)
            const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsParams = new URLSearchParams();
            if (terminalContainer) wsParams.set('container', terminalContainer);
            if (authToken && authToken !== 'anonymous') wsParams.set('token', authToken);
            const wsQuery = wsParams.toString() ? '?' + wsParams.toString() : '';
            const wsUrl = `${wsProtocol}//${window.location.host}/api/terminal/${terminalNamespace}/${terminalPodName}${wsQuery}`;

            currentTerminalWs = new WebSocket(wsUrl);

            currentTerminalWs.onopen = () => {
                // Reset reconnect attempts on successful connection
                terminalReconnectAttempts = 0;

                if (currentTerminal) {
                    currentTerminal.writeln('\x1b[32m● Connected to pod: ' + terminalPodName + '\x1b[0m');
                    currentTerminal.writeln('');

                    const dims = terminalFitAddon.proposeDimensions();
                    if (dims) {
                        currentTerminalWs.send(JSON.stringify({ type: 'resize', cols: dims.cols, rows: dims.rows }));
                    }
                }

                // Start heartbeat/keepalive (ping every 30 seconds)
                terminalHeartbeatInterval = setInterval(() => {
                    if (currentTerminalWs && currentTerminalWs.readyState === WebSocket.OPEN) {
                        currentTerminalWs.send(JSON.stringify({ type: 'ping' }));
                    }
                }, 30000);
            };

            currentTerminalWs.onmessage = (event) => {
                try {
                    const msg = JSON.parse(event.data);
                    if (msg.type === 'output') {
                        if (currentTerminal) currentTerminal.write(msg.data);
                    } else if (msg.type === 'error') {
                        if (currentTerminal) currentTerminal.writeln('\x1b[31mError: ' + msg.data + '\x1b[0m');
                    } else if (msg.type === 'pong') {
                        // Heartbeat response received
                    }
                } catch (e) {
                    if (currentTerminal) currentTerminal.write(event.data);
                }
            };

            currentTerminalWs.onclose = (event) => {
                // Clear heartbeat
                if (terminalHeartbeatInterval) {
                    clearInterval(terminalHeartbeatInterval);
                    terminalHeartbeatInterval = null;
                }

                if (!currentTerminal) return;

                // Show disconnection message
                currentTerminal.writeln('\x1b[33m\r\n● Connection closed.\x1b[0m');

                // Attempt reconnection with exponential backoff
                if (terminalShouldReconnect) {
                    const delay = Math.min(1000 * Math.pow(2, terminalReconnectAttempts), 30000); // Max 30s
                    terminalReconnectAttempts++;

                    currentTerminal.writeln('\x1b[90mReconnecting in ' + (delay/1000) + 's... (attempt ' + terminalReconnectAttempts + ')\x1b[0m');

                    terminalReconnectTimer = setTimeout(() => {
                        if (terminalShouldReconnect && currentTerminal) {
                            currentTerminal.writeln('\x1b[36m● Reconnecting...\x1b[0m');
                            connectTerminalWebSocket();
                        }
                    }, delay);
                }
            };

            currentTerminalWs.onerror = (err) => {
                if (!currentTerminal) return;

                // Only show detailed error on first connection attempt
                if (terminalReconnectAttempts === 0) {
                    const isRemoteAccess = window.location.hostname !== 'localhost' && window.location.hostname !== '127.0.0.1';
                    currentTerminal.writeln('\x1b[31m');
                    currentTerminal.writeln('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
                    currentTerminal.writeln('  ✗ WebSocket Connection Failed');
                    currentTerminal.writeln('━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━');
                    currentTerminal.writeln('\x1b[0m');
                    if (isRemoteAccess) {
                        currentTerminal.writeln('\x1b[33mYou are accessing from a remote IP address.\x1b[0m');
                        currentTerminal.writeln('\x1b[33mWebSocket connections require additional configuration.\x1b[0m');
                        currentTerminal.writeln('');
                        currentTerminal.writeln('\x1b[36mSolution 1: Enable development mode\x1b[0m');
                        currentTerminal.writeln('  export K13D_DEV=true');
                        currentTerminal.writeln('');
                        currentTerminal.writeln('\x1b[36mSolution 2: Allow your origin explicitly\x1b[0m');
                        currentTerminal.writeln('  export K13D_WS_ALLOWED_ORIGINS="' + window.location.origin + '"');
                        currentTerminal.writeln('');
                        currentTerminal.writeln('\x1b[90mThen restart k13d server.\x1b[0m');
                    } else {
                        currentTerminal.writeln('\x1b[33mFailed to connect to terminal WebSocket.\x1b[0m');
                        currentTerminal.writeln('\x1b[33mPlease check if the pod is running and accessible.\x1b[0m');
                    }
                }
            };

            currentTerminal.onData((data) => {
                if (currentTerminalWs && currentTerminalWs.readyState === WebSocket.OPEN) {
                    currentTerminalWs.send(JSON.stringify({ type: 'input', data: data }));
                }
            });

            window.addEventListener('resize', handleTerminalResize);
            currentTerminal.onResize(({ cols, rows }) => {
                if (currentTerminalWs && currentTerminalWs.readyState === WebSocket.OPEN) {
                    currentTerminalWs.send(JSON.stringify({ type: 'resize', cols, rows }));
                }
            });

            setTimeout(() => terminalFitAddon.fit(), 100);
        }

        function handleTerminalResize() { if (terminalFitAddon) terminalFitAddon.fit(); }

        function closeTerminal() {
            // Disable reconnection
            terminalShouldReconnect = false;

            // Clear timers
            if (terminalReconnectTimer) {
                clearTimeout(terminalReconnectTimer);
                terminalReconnectTimer = null;
            }
            if (terminalHeartbeatInterval) {
                clearInterval(terminalHeartbeatInterval);
                terminalHeartbeatInterval = null;
            }

            // Close WebSocket
            if (currentTerminalWs) {
                currentTerminalWs.onclose = null; // Prevent reconnection
                currentTerminalWs.close();
                currentTerminalWs = null;
            }

            // Dispose terminal
            if (currentTerminal) {
                currentTerminal.dispose();
                currentTerminal = null;
            }

            // Remove event listeners
            window.removeEventListener('resize', handleTerminalResize);

            // Hide modal
            document.getElementById('terminal-modal').classList.remove('active');

            // Reset state
            terminalReconnectAttempts = 0;
            terminalPodName = null;
            terminalNamespace = null;
            terminalContainer = null;
        }

        // ==========================================
        // Log Viewer Functions
        // ==========================================
        let currentLogPod = null, currentLogNamespace = null, currentLogContainer = null;
        let currentLogPods = []; // For multi-pod logging
        let logEventSource = null, logFollowMode = true, allLogs = [], ansiUp = null;
        let isMultiPodMode = false;
        let podColorMap = {};
        let hiddenPods = new Set();

        const POD_COLORS = [
            { name: 'blue', class: 'log-pod-0' },
            { name: 'green', class: 'log-pod-1' },
            { name: 'yellow', class: 'log-pod-2' },
            { name: 'purple', class: 'log-pod-3' },
            { name: 'cyan', class: 'log-pod-4' },
            { name: 'red', class: 'log-pod-5' },
            { name: 'teal', class: 'log-pod-6' },
            { name: 'orange', class: 'log-pod-7' }
        ];

        // Helper function to open log viewer from row button click
        function openLogViewerFromRow(btn, podName, namespace) {
            const row = btn.closest('tr');
            let containers = ['default'];
            if (row && row.dataset.containers) {
                try {
                    containers = JSON.parse(row.dataset.containers);
                } catch (e) {
                    console.warn('Failed to parse containers:', e);
                }
            }
            openLogViewer(podName, namespace, containers);
        }

        // Open multi-pod log viewer for a workload (deployment, replicaset, etc.)
        async function openMultiPodLogViewer(workloadName, namespace, labelSelector) {
            isMultiPodMode = true;
            currentLogNamespace = namespace;
            currentLogPods = [];
            podColorMap = {};
            hiddenPods.clear();

            document.getElementById('log-pod-name').textContent = workloadName;
            document.getElementById('log-container-name').textContent = '(multiple pods)';

            // Hide container select for multi-pod mode
            document.getElementById('log-container-select').style.display = 'none';

            document.getElementById('log-viewer-modal').classList.add('active');
            if (typeof AnsiUp !== 'undefined') { ansiUp = new AnsiUp(); ansiUp.use_classes = true; }
            // Ensure Follow button shows correct state
            document.getElementById('log-follow-btn').classList.toggle('active', logFollowMode);

            // Fetch pods for this workload
            try {
                const resp = await fetchWithAuth(`/api/k8s/pods?namespace=${namespace}&labelSelector=${encodeURIComponent(labelSelector)}`);
                const data = await resp.json();
                currentLogPods = (data.items || []).map(p => p.name);

                // Assign colors to pods
                currentLogPods.forEach((pod, idx) => {
                    podColorMap[pod] = POD_COLORS[idx % POD_COLORS.length];
                });

                // Show pod legend
                renderPodLegend();
                await loadMultiPodLogs();
            } catch (e) {
                document.getElementById('log-content').innerHTML = `<p style="color: var(--accent-red);">Error: ${e.message}</p>`;
            }
        }

        function renderPodLegend() {
            const legend = document.getElementById('log-pod-legend');
            if (currentLogPods.length <= 1) {
                legend.style.display = 'none';
                return;
            }

            legend.style.display = 'flex';
            legend.innerHTML = currentLogPods.map((pod, idx) => {
                const color = podColorMap[pod];
                const shortName = pod.length > 30 ? pod.substring(0, 27) + '...' : pod;
                const hidden = hiddenPods.has(pod) ? 'hidden' : '';
                return `<div class="log-pod-legend-item ${hidden}" onclick="togglePodVisibility('${pod}')" title="${pod}">
                    <span class="log-pod-legend-dot legend-${color.class.replace('log-', '')}"></span>
                    <span>${shortName}</span>
                </div>`;
            }).join('');
        }

        function togglePodVisibility(podName) {
            if (hiddenPods.has(podName)) {
                hiddenPods.delete(podName);
            } else {
                hiddenPods.add(podName);
            }
            renderPodLegend();
            // Re-render logs with filter
            filterLogs();
        }

        async function openLogViewer(podName, namespace, containers = []) {
            isMultiPodMode = false;
            currentLogPod = podName;
            currentLogNamespace = namespace;
            currentLogPods = [podName];
            podColorMap = { [podName]: POD_COLORS[0] };

            document.getElementById('log-pod-name').textContent = podName;
            document.getElementById('log-pod-legend').style.display = 'none';
            document.getElementById('log-container-select').style.display = '';

            // Filter out 'default' placeholder - use actual container names only
            const validContainers = containers.filter(c => c && c !== 'default');

            const containerSelect = document.getElementById('log-container-select');
            if (validContainers.length > 0) {
                containerSelect.innerHTML = validContainers.map((c, i) => `<option value="${c}" ${i === 0 ? 'selected' : ''}>${c}</option>`).join('');
                currentLogContainer = validContainers[0];
                document.getElementById('log-container-name').textContent = currentLogContainer;
            } else {
                // No containers specified - let the backend use the default container
                containerSelect.innerHTML = '<option value="">default</option>';
                currentLogContainer = '';
                document.getElementById('log-container-name').textContent = 'default';
            }

            document.getElementById('log-viewer-modal').classList.add('active');
            if (typeof AnsiUp !== 'undefined') { ansiUp = new AnsiUp(); ansiUp.use_classes = true; }
            // Ensure Follow button shows correct state
            document.getElementById('log-follow-btn').classList.toggle('active', logFollowMode);
            await loadLogs();
        }

        function switchLogContainer() {
            currentLogContainer = document.getElementById('log-container-select').value;
            document.getElementById('log-container-name').textContent = currentLogContainer;
            loadLogs();
        }

        async function loadMultiPodLogs() {
            const tailLines = document.getElementById('log-lines-select').value;
            const logContent = document.getElementById('log-content');
            logContent.innerHTML = '<p style="color: var(--text-secondary);">Loading logs from multiple pods...</p>';
            allLogs = [];

            try {
                // Fetch logs from all pods in parallel
                const logPromises = currentLogPods.map(async (pod) => {
                    try {
                        const url = `/api/pods/${currentLogNamespace}/${pod}/logs?tailLines=${Math.floor(tailLines / currentLogPods.length)}`;
                        const resp = await fetchWithAuth(url);
                        const text = await resp.text();
                        return text.split('\n').filter(l => l.trim()).map(line => ({ pod, line }));
                    } catch (e) {
                        return [{ pod, line: `[Error fetching logs: ${e.message}]` }];
                    }
                });

                const results = await Promise.all(logPromises);
                const allPodLogs = results.flat();

                // Sort by timestamp if present, otherwise keep order
                // For now, interleave logs from different pods
                logContent.innerHTML = '';
                allPodLogs.forEach(({ pod, line }) => {
                    appendLogLine(line, pod);
                });
                // Auto-scroll to bottom after loading logs
                logContent.scrollTop = logContent.scrollHeight;

            } catch (e) {
                logContent.innerHTML = `<p style="color: var(--accent-red);">Error loading logs: ${e.message}</p>`;
            }
        }

        async function loadLogs() {
            if (isMultiPodMode) {
                return loadMultiPodLogs();
            }

            const tailLines = document.getElementById('log-lines-select').value;
            const logContent = document.getElementById('log-content');
            logContent.innerHTML = '<p style="color: var(--text-secondary);">Loading logs...</p>';
            allLogs = [];
            if (logEventSource) { logEventSource.close(); logEventSource = null; }

            try {
                // Always use follow=false to get plain text response
                // SSE streaming is not properly supported in this fetch pattern
                let url = `/api/pods/${currentLogNamespace}/${currentLogPod}/logs?tailLines=${tailLines}&follow=false`;
                if (currentLogContainer) {
                    url += `&container=${currentLogContainer}`;
                }
                const resp = await fetchWithAuth(url);
                if (!resp.ok) {
                    const errorText = await resp.text();
                    throw new Error(errorText || `HTTP ${resp.status}`);
                }
                const text = await resp.text();
                logContent.innerHTML = '';
                if (text.trim()) {
                    text.split('\n').forEach(line => { if (line.trim()) appendLogLine(line, currentLogPod); });
                } else {
                    logContent.innerHTML = '<p style="color: var(--text-secondary);">No logs available for this pod.</p>';
                }
                // Auto-scroll to bottom after loading logs
                logContent.scrollTop = logContent.scrollHeight;
            } catch (e) {
                logContent.innerHTML = `<p style="color: var(--accent-red);">Error loading logs: ${e.message}</p>`;
            }
        }

        function appendLogLine(line, podName = null) {
            const logContent = document.getElementById('log-content');
            const pod = podName || currentLogPod;
            allLogs.push({ line, pod });

            const div = document.createElement('div');
            div.className = 'log-line';
            div.dataset.pod = pod;

            // Add pod color class for multi-pod mode
            const podColor = podColorMap[pod];
            if (podColor && currentLogPods.length > 1) {
                div.classList.add(podColor.class);
            }

            // Detect log level
            const lineLower = line.toLowerCase();
            if (lineLower.includes('error') || lineLower.includes('fatal') || lineLower.includes('panic')) {
                div.classList.add('error');
            } else if (lineLower.includes('warn') || lineLower.includes('warning')) {
                div.classList.add('warn');
            }

            // Add pod tag for multi-pod mode
            if (currentLogPods.length > 1) {
                const podTag = document.createElement('span');
                podTag.className = 'log-pod-tag';
                // Show short pod name (last part after last dash or first 15 chars)
                const shortPod = pod.split('-').slice(-2).join('-').substring(0, 15);
                podTag.textContent = shortPod;
                podTag.title = pod;
                div.appendChild(podTag);
            }

            // Create content wrapper
            const content = document.createElement('span');
            content.className = 'log-line-content';
            content.innerHTML = ansiUp ? ansiUp.ansi_to_html(line) : escapeHtml(line);
            div.appendChild(content);

            logContent.appendChild(div);
            if (logFollowMode) logContent.scrollTop = logContent.scrollHeight;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function reloadLogs() { loadLogs(); }
        function toggleLogFollow() {
            logFollowMode = !logFollowMode;
            document.getElementById('log-follow-btn').classList.toggle('active', logFollowMode);
            loadLogs();
        }

        function filterLogs() {
            const searchTerm = document.getElementById('log-search-input').value.toLowerCase();
            let matchCount = 0;
            document.querySelectorAll('#log-content .log-line').forEach(lineEl => {
                const pod = lineEl.dataset.pod;
                const isPodHidden = hiddenPods.has(pod);
                const matchesSearch = searchTerm === '' || lineEl.textContent.toLowerCase().includes(searchTerm);
                const visible = !isPodHidden && matchesSearch;
                lineEl.style.display = visible ? 'flex' : 'none';
                if (visible && searchTerm) matchCount++;
            });
            document.getElementById('log-match-count').textContent = searchTerm ? `${matchCount} matches` : '';
        }

        function downloadLogs() {
            const logsText = allLogs.map(l => {
                if (typeof l === 'object') {
                    return currentLogPods.length > 1 ? `[${l.pod}] ${l.line}` : l.line;
                }
                return l;
            }).join('\n');
            const blob = new Blob([logsText], { type: 'text/plain' });
            const a = document.createElement('a'); a.href = URL.createObjectURL(blob);
            const dateStr = new Date().toISOString().slice(0, 19).replace(/[T:]/g, '-');
            const podName = currentLogPod || currentLogPods[0] || 'unknown';
            const filename = currentLogPods.length > 1
                ? `${podName}-multi-${dateStr}.log`
                : `${podName}-${dateStr}.log`;
            a.download = filename; a.click();
        }

        function closeLogViewer() {
            document.getElementById('log-viewer-modal').classList.remove('active');
            if (logEventSource) { logEventSource.close(); logEventSource = null; }
            allLogs = []; logFollowMode = true; // Reset to default (follow mode on)
        }

        // ==========================================
        // Metrics Functions
        // ==========================================
        let cpuChart = null, memoryChart = null, llmUsageChart = null;
        let metricsHistory = { cpu: [], memory: [], timestamps: [], pods: [], nodes: [] };
        let llmUsageHistory = { requests: [], tokens: [], timestamps: [] };
        let metricsInterval = null;
        let metricsHistoryLoaded = false;
        let metricsTimeRangeMinutes = 30; // Default time range in minutes

        function setMetricsTimeRangeMinutes(minutes) {
            metricsTimeRangeMinutes = minutes;
            // Update active button state
            document.querySelectorAll('.time-range-btn').forEach(btn => {
                btn.classList.remove('active');
                if (parseInt(btn.dataset.minutes) === minutes) {
                    btn.classList.add('active');
                }
            });
            // Reload historical data with new time range
            loadHistoricalMetrics();
            loadLLMUsageStats();
        }

        function setMetricsTimeRange(hours) {
            metricsTimeRangeMinutes = hours * 60;
            // Update active button state
            document.querySelectorAll('.time-range-btn').forEach(btn => {
                btn.classList.remove('active');
                if (parseInt(btn.dataset.hours) === hours) {
                    btn.classList.add('active');
                }
            });
            // Reload historical data with new time range
            loadHistoricalMetrics();
            loadLLMUsageStats();
        }

        async function showMetrics() {
            document.getElementById('metrics-modal').classList.add('active');
            const metricsNsSelect = document.getElementById('metrics-namespace-select');
            metricsNsSelect.innerHTML = document.getElementById('namespace-select').innerHTML;

            // Load Prometheus status
            try {
                const resp = await fetchWithAuth('/api/prometheus/settings');
                const data = await resp.json();
                if (!data.error) {
                    updatePrometheusStatus(data.expose_metrics, data.external_url);
                }
            } catch (e) {
                console.error('Failed to load Prometheus status:', e);
            }

            // Load historical metrics first, then real-time
            await loadHistoricalMetrics();
            await loadMetrics();
            await loadLLMUsageStats();

            // Set up auto-refresh interval if checkbox is checked
            const autoRefresh = document.getElementById('metrics-auto-refresh');
            if (autoRefresh && autoRefresh.checked) {
                metricsInterval = setInterval(loadMetrics, 30000);
            }
        }

        async function loadHistoricalMetrics() {
            try {
                const resp = await fetchWithAuth(`/api/metrics/history/cluster?minutes=${metricsTimeRangeMinutes}&limit=100`);
                const data = await resp.json();

                if (!data.error && data.items && data.items.length > 0) {
                    // Sort by timestamp ascending
                    const sorted = data.items.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));

                    metricsHistory.timestamps = sorted.map(m => formatTimeShort(m.timestamp));
                    metricsHistory.cpu = sorted.map(m => m.used_cpu_millis || 0);
                    metricsHistory.memory = sorted.map(m => m.used_memory_mb || 0);
                    metricsHistory.pods = sorted.map(m => m.running_pods || 0);
                    metricsHistory.nodes = sorted.map(m => m.ready_nodes || m.total_nodes || 0);

                    // Check if metrics-server data is available (all zeros means unavailable)
                    const hasCPUData = metricsHistory.cpu.some(v => v > 0);
                    const hasMemData = metricsHistory.memory.some(v => v > 0);

                    metricsHistoryLoaded = true;
                    updateMetricsCharts(hasCPUData, hasMemData);

                    // Update summary from latest metrics
                    const latest = sorted[sorted.length - 1];
                    if (latest) {
                        if (hasCPUData) {
                            document.getElementById('metrics-total-cpu').textContent = `${latest.used_cpu_millis || 0}m`;
                        } else {
                            document.getElementById('metrics-total-cpu').textContent = 'N/A';
                            document.getElementById('metrics-total-cpu').title = 'Install metrics-server for CPU data';
                        }
                        if (hasMemData) {
                            document.getElementById('metrics-total-memory').textContent = formatBytes((latest.used_memory_mb || 0) * 1024 * 1024);
                        } else {
                            document.getElementById('metrics-total-memory').textContent = 'N/A';
                            document.getElementById('metrics-total-memory').title = 'Install metrics-server for memory data';
                        }
                        document.getElementById('metrics-total-pods').textContent = latest.running_pods || 0;
                    }
                } else {
                    // No data collected yet
                    const cpuEl = document.getElementById('metrics-total-cpu');
                    const memEl = document.getElementById('metrics-total-memory');
                    const podEl = document.getElementById('metrics-total-pods');
                    if (cpuEl) cpuEl.textContent = 'Collecting...';
                    if (memEl) memEl.textContent = 'Collecting...';
                    if (podEl) podEl.textContent = '0';
                }
            } catch (e) {
                console.error('Failed to load historical metrics:', e);
            }
        }

        async function loadMetrics() {
            const namespace = document.getElementById('metrics-namespace-select').value;
            try {
                // Load pod metrics
                const url = namespace ? `/api/metrics/pods?namespace=${namespace}` : '/api/metrics/pods';
                const resp = await fetchWithAuth(url);
                const data = await resp.json();

                if (data.error) {
                    document.getElementById('metrics-cpu-value').textContent = 'N/A';
                    document.getElementById('metrics-mem-value').textContent = 'N/A';
                    document.getElementById('metrics-pods-value').textContent = 'N/A';
                    // Also update legacy elements for backward compatibility
                    const legacyCpu = document.getElementById('metrics-total-cpu');
                    const legacyMem = document.getElementById('metrics-total-memory');
                    const legacyPods = document.getElementById('metrics-total-pods');
                    if (legacyCpu) legacyCpu.textContent = 'N/A';
                    if (legacyMem) legacyMem.textContent = 'N/A';
                    if (legacyPods) legacyPods.textContent = 'N/A';
                    return;
                }

                const totalCpu = data.items?.reduce((sum, p) => sum + (p.cpu || 0), 0) || 0;
                const totalMem = data.items?.reduce((sum, p) => sum + (p.memory || 0), 0) || 0;
                const podCount = data.items?.length || 0;

                // Update new dashboard stat cards
                document.getElementById('metrics-cpu-value').textContent = `${totalCpu.toFixed(0)}m`;
                document.getElementById('metrics-mem-value').textContent = formatBytes(totalMem * 1024 * 1024);
                document.getElementById('metrics-pods-value').textContent = podCount;

                // Also update legacy elements for backward compatibility
                const legacyCpu = document.getElementById('metrics-total-cpu');
                const legacyMem = document.getElementById('metrics-total-memory');
                const legacyPods = document.getElementById('metrics-total-pods');
                if (legacyCpu) legacyCpu.textContent = `${totalCpu.toFixed(0)}m`;
                if (legacyMem) legacyMem.textContent = formatBytes(totalMem * 1024 * 1024);
                if (legacyPods) legacyPods.textContent = podCount;

                // Append real-time data point to history
                metricsHistory.timestamps.push(formatTimeShort(new Date()));
                metricsHistory.cpu.push(totalCpu);
                metricsHistory.memory.push(totalMem);
                metricsHistory.pods.push(podCount);
                // Keep last known node count for real-time updates
                metricsHistory.nodes.push(metricsHistory.nodes.length > 0 ? metricsHistory.nodes[metricsHistory.nodes.length - 1] : 0);
                const maxHistory = 100;
                while (metricsHistory.timestamps.length > maxHistory) {
                    metricsHistory.timestamps.shift();
                    metricsHistory.cpu.shift();
                    metricsHistory.memory.shift();
                    metricsHistory.pods.shift();
                    metricsHistory.nodes.shift();
                }
                updateMetricsCharts();
                updateTopConsumers(data.items || []);

                // Load node health info
                await loadNodeHealth();
            } catch (e) { console.error('Failed to load metrics:', e); }
        }

        async function loadNodeHealth() {
            try {
                const resp = await fetchWithAuth('/api/metrics/nodes');
                const data = await resp.json();

                // Also get node list for status info
                const nodesResp = await fetchWithAuth('/api/nodes');
                const nodesData = await nodesResp.json();

                const nodeHealthGrid = document.getElementById('node-health-grid');
                if (!nodeHealthGrid) return;

                // Build node info map
                const nodeInfo = {};
                if (nodesData.items) {
                    nodesData.items.forEach(node => {
                        const readyCondition = node.status?.conditions?.find(c => c.type === 'Ready');
                        nodeInfo[node.metadata.name] = {
                            ready: readyCondition?.status === 'True',
                            capacity: {
                                cpu: parseCpuToMillicores(node.status?.capacity?.cpu || '0'),
                                memory: parseMemoryToMB(node.status?.capacity?.memory || '0')
                            }
                        };
                    });
                }

                // Update nodes stat card
                const totalNodes = nodesData.items?.length || 0;
                const readyNodes = Object.values(nodeInfo).filter(n => n.ready).length;
                document.getElementById('metrics-nodes-value').textContent = `${readyNodes}/${totalNodes}`;

                // Update last nodes value in history for real-time sync
                if (metricsHistory.nodes.length > 0) {
                    metricsHistory.nodes[metricsHistory.nodes.length - 1] = readyNodes;
                }

                // Build node health cards
                if (data.items && data.items.length > 0) {
                    const cards = data.items.map(node => {
                        const info = nodeInfo[node.name] || { ready: true, capacity: { cpu: 4000, memory: 8192 } };
                        const cpuPercent = Math.min(100, (node.cpu / info.capacity.cpu) * 100);
                        const memPercent = Math.min(100, (node.memory / info.capacity.memory) * 100);
                        const status = !info.ready ? 'critical' : (cpuPercent > 80 || memPercent > 80) ? 'warning' : 'healthy';

                        return `
                            <div class="node-health-card">
                                <div class="node-name">
                                    <span class="node-status ${status}"></span>
                                    <span>${escapeHtml(node.name)}</span>
                                </div>
                                <div class="usage-bar-container">
                                    <div class="usage-bar-label">
                                        <span>CPU</span>
                                        <span>${node.cpu}m / ${info.capacity.cpu}m (${cpuPercent.toFixed(0)}%)</span>
                                    </div>
                                    <div class="usage-bar">
                                        <div class="fill cpu ${cpuPercent > 80 ? 'high' : ''}" style="width: ${cpuPercent}%"></div>
                                    </div>
                                </div>
                                <div class="usage-bar-container">
                                    <div class="usage-bar-label">
                                        <span>Memory</span>
                                        <span>${formatBytes(node.memory * 1024 * 1024)} / ${formatBytes(info.capacity.memory * 1024 * 1024)} (${memPercent.toFixed(0)}%)</span>
                                    </div>
                                    <div class="usage-bar">
                                        <div class="fill memory ${memPercent > 80 ? 'high' : ''}" style="width: ${memPercent}%"></div>
                                    </div>
                                </div>
                            </div>
                        `;
                    }).join('');
                    nodeHealthGrid.innerHTML = cards;
                } else {
                    nodeHealthGrid.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">No node metrics available</p>';
                }
            } catch (e) {
                console.error('Failed to load node health:', e);
                const nodeHealthGrid = document.getElementById('node-health-grid');
                if (nodeHealthGrid) {
                    nodeHealthGrid.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">Failed to load node health</p>';
                }
            }
        }

        function parseCpuToMillicores(cpuStr) {
            if (!cpuStr) return 0;
            if (cpuStr.endsWith('m')) {
                return parseInt(cpuStr) || 0;
            }
            // Assume it's in cores, convert to millicores
            return (parseFloat(cpuStr) || 0) * 1000;
        }

        function parseMemoryToMB(memStr) {
            if (!memStr) return 0;
            const units = { 'Ki': 1/1024, 'Mi': 1, 'Gi': 1024, 'Ti': 1024*1024, 'K': 1/1000, 'M': 1, 'G': 1000, 'T': 1000000 };
            for (const [unit, multiplier] of Object.entries(units)) {
                if (memStr.endsWith(unit)) {
                    return (parseFloat(memStr) || 0) * multiplier;
                }
            }
            // Assume bytes
            return (parseInt(memStr) || 0) / (1024 * 1024);
        }

        function updateMetricsCharts(hasCPUData, hasMemData) {
            // Default to checking history data if not passed
            if (hasCPUData === undefined) hasCPUData = metricsHistory.cpu.some(v => v > 0);
            if (hasMemData === undefined) hasMemData = metricsHistory.memory.some(v => v > 0);

            const opts = {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: false },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                    }
                },
                scales: {
                    x: {
                        ticks: { color: '#a9b1d6', maxTicksLimit: 10 },
                        grid: { color: '#414868' }
                    },
                    y: {
                        ticks: { color: '#a9b1d6' },
                        grid: { color: '#414868' },
                        beginAtZero: true
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            };

            // Update chart titles based on data availability
            const cpuTitle = document.querySelector('#cpu-chart')?.closest('.metric-card')?.querySelector('h4');
            const memTitle = document.querySelector('#memory-chart')?.closest('.metric-card')?.querySelector('h4');

            const cpuCtx = document.getElementById('cpu-chart')?.getContext('2d');
            if (cpuCtx) {
                // Choose data: CPU if available, otherwise Pod Count
                const chartData = hasCPUData ? metricsHistory.cpu : metricsHistory.pods;
                const chartLabel = hasCPUData ? 'CPU (millicores)' : 'Running Pods';
                const chartColor = hasCPUData ? '#7dcfff' : '#9ece6a';
                const chartBg = hasCPUData ? 'rgba(125,207,255,0.1)' : 'rgba(158,206,106,0.1)';

                if (cpuTitle) {
                    cpuTitle.textContent = hasCPUData ? 'CPU Usage Over Time' : 'Running Pods Over Time';
                    if (!hasCPUData) cpuTitle.title = 'Install metrics-server for CPU data';
                }

                if (cpuChart) {
                    cpuChart.data.labels = metricsHistory.timestamps;
                    cpuChart.data.datasets[0].data = chartData;
                    cpuChart.data.datasets[0].label = chartLabel;
                    cpuChart.data.datasets[0].borderColor = chartColor;
                    cpuChart.data.datasets[0].backgroundColor = chartBg;
                    cpuChart.update();
                } else {
                    cpuChart = new Chart(cpuCtx, {
                        type: 'line',
                        data: {
                            labels: metricsHistory.timestamps,
                            datasets: [{
                                label: chartLabel,
                                data: chartData,
                                borderColor: chartColor,
                                backgroundColor: chartBg,
                                fill: true,
                                tension: 0.4,
                                pointRadius: 0,
                                pointHoverRadius: 4
                            }]
                        },
                        options: opts
                    });
                }
            }

            const memCtx = document.getElementById('memory-chart')?.getContext('2d');
            if (memCtx) {
                // Choose data: Memory if available, otherwise Node Count
                const chartData = hasMemData ? metricsHistory.memory : metricsHistory.nodes;
                const chartLabel = hasMemData ? 'Memory (MB)' : 'Ready Nodes';
                const chartColor = hasMemData ? '#bb9af7' : '#e0af68';
                const chartBg = hasMemData ? 'rgba(187,154,247,0.1)' : 'rgba(224,175,104,0.1)';

                if (memTitle) {
                    memTitle.textContent = hasMemData ? 'Memory Usage Over Time' : 'Ready Nodes Over Time';
                    if (!hasMemData) memTitle.title = 'Install metrics-server for Memory data';
                }

                if (memoryChart) {
                    memoryChart.data.labels = metricsHistory.timestamps;
                    memoryChart.data.datasets[0].data = chartData;
                    memoryChart.data.datasets[0].label = chartLabel;
                    memoryChart.data.datasets[0].borderColor = chartColor;
                    memoryChart.data.datasets[0].backgroundColor = chartBg;
                    memoryChart.update();
                } else {
                    memoryChart = new Chart(memCtx, {
                        type: 'line',
                        data: {
                            labels: metricsHistory.timestamps,
                            datasets: [{
                                label: chartLabel,
                                data: chartData,
                                borderColor: chartColor,
                                backgroundColor: chartBg,
                                fill: true,
                                tension: 0.4,
                                pointRadius: 0,
                                pointHoverRadius: 4
                            }]
                        },
                        options: opts
                    });
                }
            }
        }

        function updateTopConsumers(pods) {
            const topCpu = [...pods].sort((a, b) => (b.cpu || 0) - (a.cpu || 0)).slice(0, 5);
            document.getElementById('top-cpu-list').innerHTML = topCpu.map(p => `<div style="display:flex;justify-content:space-between;padding:8px;border-bottom:1px solid var(--border-color);"><span style="font-size:12px;">${escapeHtml(p.name)}</span><span style="font-size:12px;color:var(--accent-cyan);">${p.cpu||0}m</span></div>`).join('');
            const topMem = [...pods].sort((a, b) => (b.memory || 0) - (a.memory || 0)).slice(0, 5);
            document.getElementById('top-memory-list').innerHTML = topMem.map(p => `<div style="display:flex;justify-content:space-between;padding:8px;border-bottom:1px solid var(--border-color);"><span style="font-size:12px;">${escapeHtml(p.name)}</span><span style="font-size:12px;color:var(--accent-purple);">${formatBytes((p.memory||0)*1024*1024)}</span></div>`).join('');
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024, sizes = ['B', 'Ki', 'Mi', 'Gi'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
        }

        function formatNumber(num) {
            if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
            if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
            return num.toString();
        }

        // ==========================================
        // LLM Usage Functions
        // ==========================================
        async function loadLLMUsageStats() {
            try {
                const resp = await fetchWithAuth(`/api/llm/usage/stats?minutes=${metricsTimeRangeMinutes}`);
                const data = await resp.json();

                if (data.error) {
                    document.getElementById('llm-total-requests').textContent = 'N/A';
                    document.getElementById('llm-total-tokens').textContent = 'N/A';
                    document.getElementById('llm-prompt-tokens').textContent = 'N/A';
                    document.getElementById('llm-completion-tokens').textContent = 'N/A';
                    return;
                }

                // Update summary stats
                document.getElementById('llm-total-requests').textContent = formatNumber(data.total_requests || 0);
                document.getElementById('llm-total-tokens').textContent = formatNumber(data.total_tokens || 0);
                document.getElementById('llm-prompt-tokens').textContent = formatNumber(data.prompt_tokens || 0);
                document.getElementById('llm-completion-tokens').textContent = formatNumber(data.completion_tokens || 0);

                // Update time series chart
                if (data.hourly && data.hourly.length > 0) {
                    llmUsageHistory.timestamps = data.hourly.map(h => {
                        const d = new Date(h.hour);
                        return d.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
                    });
                    llmUsageHistory.requests = data.hourly.map(h => h.requests || 0);
                    llmUsageHistory.tokens = data.hourly.map(h => h.total_tokens || 0);
                    updateLLMUsageChart();
                }

                // Update model breakdown list
                if (data.by_model && data.by_model.length > 0) {
                    document.getElementById('llm-model-list').innerHTML = data.by_model.map(m =>
                        `<div style="display:flex;justify-content:space-between;padding:8px;border-bottom:1px solid var(--border-color);">
                            <span style="font-size:12px;">${escapeHtml(m.model || 'unknown')}</span>
                            <span style="font-size:12px;color:var(--accent-yellow);">${formatNumber(m.total_tokens || 0)} tokens</span>
                        </div>`
                    ).join('');
                } else {
                    document.getElementById('llm-model-list').innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">No LLM usage data</p>';
                }
            } catch (e) {
                console.error('Failed to load LLM usage stats:', e);
                document.getElementById('llm-model-list').innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">Failed to load data</p>';
            }
        }

        function updateLLMUsageChart() {
            const opts = {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: { display: true, position: 'top', labels: { color: '#a9b1d6', font: { size: 10 } } },
                    tooltip: { mode: 'index', intersect: false }
                },
                scales: {
                    x: {
                        ticks: { color: '#a9b1d6', maxTicksLimit: 8 },
                        grid: { color: '#414868' }
                    },
                    y: {
                        type: 'linear',
                        position: 'left',
                        ticks: { color: '#e0af68' },
                        grid: { color: '#414868' },
                        beginAtZero: true,
                        title: { display: true, text: 'Requests', color: '#e0af68' }
                    },
                    y1: {
                        type: 'linear',
                        position: 'right',
                        ticks: { color: '#9ece6a' },
                        grid: { drawOnChartArea: false },
                        beginAtZero: true,
                        title: { display: true, text: 'Tokens', color: '#9ece6a' }
                    }
                },
                interaction: { mode: 'nearest', axis: 'x', intersect: false }
            };

            const ctx = document.getElementById('llm-usage-chart')?.getContext('2d');
            if (ctx) {
                if (llmUsageChart) {
                    llmUsageChart.data.labels = llmUsageHistory.timestamps;
                    llmUsageChart.data.datasets[0].data = llmUsageHistory.requests;
                    llmUsageChart.data.datasets[1].data = llmUsageHistory.tokens;
                    llmUsageChart.update();
                } else {
                    llmUsageChart = new Chart(ctx, {
                        type: 'line',
                        data: {
                            labels: llmUsageHistory.timestamps,
                            datasets: [
                                {
                                    label: 'Requests',
                                    data: llmUsageHistory.requests,
                                    borderColor: '#e0af68',
                                    backgroundColor: 'rgba(224,175,104,0.1)',
                                    fill: false,
                                    tension: 0.4,
                                    pointRadius: 2,
                                    pointHoverRadius: 4,
                                    yAxisID: 'y'
                                },
                                {
                                    label: 'Tokens',
                                    data: llmUsageHistory.tokens,
                                    borderColor: '#9ece6a',
                                    backgroundColor: 'rgba(158,206,106,0.1)',
                                    fill: true,
                                    tension: 0.4,
                                    pointRadius: 0,
                                    pointHoverRadius: 4,
                                    yAxisID: 'y1'
                                }
                            ]
                        },
                        options: opts
                    });
                }
            }
        }

        function closeMetrics() {
            document.getElementById('metrics-modal').classList.remove('active');
            if (metricsInterval) { clearInterval(metricsInterval); metricsInterval = null; }
            metricsHistoryLoaded = false;
            // Reset history to avoid stale data on reopen
            metricsHistory = { cpu: [], memory: [], timestamps: [], pods: [], nodes: [] };
            // Destroy all charts so they're recreated fresh on next open
            if (cpuChart) { cpuChart.destroy(); cpuChart = null; }
            if (memoryChart) { memoryChart.destroy(); memoryChart = null; }
            if (llmUsageChart) { llmUsageChart.destroy(); llmUsageChart = null; }
        }

        async function collectMetricsNow() {
            try {
                const resp = await fetchWithAuth('/api/metrics/collect', { method: 'POST' });
                const data = await resp.json();
                if (data.success) {
                    // Reload data after collection completes
                    await loadHistoricalMetrics();
                    await loadMetrics();
                }
            } catch (e) {
                console.error('Failed to trigger metrics collection:', e);
            }
        }

        // ==========================================
        // Port Forward Functions
        // ==========================================
        let currentPfPod = null, currentPfNamespace = null, activePortForwards = [];

        function openPortForward(podName, namespace) {
            currentPfPod = podName; currentPfNamespace = namespace;
            document.getElementById('pf-target').value = `${namespace}/${podName}`;
            document.getElementById('portforward-modal').classList.add('active');
            loadActivePortForwards();
        }

        async function startPortForward() {
            const localPort = document.getElementById('pf-local-port').value;
            const remotePort = document.getElementById('pf-remote-port').value;
            if (!localPort || !remotePort) { alert('Please enter both ports'); return; }
            try {
                const resp = await fetchWithAuth('/api/portforward/start', { method: 'POST', headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ namespace: currentPfNamespace, pod: currentPfPod, localPort: parseInt(localPort), remotePort: parseInt(remotePort) }) });
                const data = await resp.json();
                if (data.error) alert('Error: ' + data.error);
                else { showToast(`Port forward started: localhost:${localPort}`); loadActivePortForwards(); }
            } catch (e) { alert('Failed: ' + e.message); }
        }

        async function loadActivePortForwards() {
            try {
                const resp = await fetchWithAuth('/api/portforward/list');
                activePortForwards = (await resp.json()).items || [];
                renderPortForwardList();
            } catch (e) { console.error(e); }
        }

        function renderPortForwardList() {
            const list = document.getElementById('portforward-list');
            if (activePortForwards.length === 0) { list.innerHTML = '<p style="color:var(--text-secondary);text-align:center;padding:20px;">No active port forwards</p>'; return; }
            list.innerHTML = activePortForwards.map(pf => `<div class="portforward-item"><div class="info"><div class="ports">localhost:${parseInt(pf.localPort)||0} → :${parseInt(pf.remotePort)||0}</div><div class="target">${escapeHtml(pf.namespace)}/${escapeHtml(pf.pod)}</div></div><div class="status"><span class="status-dot ${pf.active?'active':'stopped'}"></span><button onclick="stopPortForward('${escapeHtml(pf.id)}')">Stop</button></div></div>`).join('');
        }

        async function stopPortForward(id) { try { await fetchWithAuth(`/api/portforward/${id}`, { method: 'DELETE' }); loadActivePortForwards(); } catch (e) { console.error(e); } }
        function closePortForward() { document.getElementById('portforward-modal').classList.remove('active'); }

        // ==========================================
        // AI-Dashboard Interactive Actions
        // ==========================================
        function executeAIAction(action) {
            switch(action.type) {
                case 'navigate': navigateToResource(action.target, action.params); break;
                case 'highlight': highlightResource(action.target, action.params); break;
                case 'open_terminal': openTerminal(action.params.pod, action.params.namespace, action.params.container); break;
                case 'show_logs': fetchPodContainers(action.params.pod, action.params.namespace).then(c => openLogViewer(action.params.pod, action.params.namespace, c)); break;
                case 'show_metrics': showMetrics(); break;
                case 'port_forward': openPortForward(action.params.pod, action.params.namespace); break;
            }
            showToast(`AI Action: ${action.type}`);
        }

        function navigateToResource(resource, params) {
            switchResource(resource);
            if (params.namespace) { document.getElementById('namespace-select').value = params.namespace; currentNamespace = params.namespace; }
            if (params.filter) { document.getElementById('filter-input').value = params.filter; }
            loadData();
        }

        function highlightResource(resourceType, params) {
            setTimeout(() => {
                document.querySelectorAll('#table-body tr').forEach(row => {
                    const nameCell = row.querySelector('td:first-child');
                    if (nameCell && nameCell.textContent.includes(params.name)) {
                        row.classList.add('ai-highlight');
                        row.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    }
                });
            }, 500);
        }

        async function fetchPodContainers(podName, namespace) {
            try { const resp = await fetchWithAuth(`/api/k8s/pods/${namespace}/${podName}`); return (await resp.json()).containers || ['default']; }
            catch (e) { return ['default']; }
        }

        function showToast(message, type) {
            const toast = document.createElement('div');
            toast.className = 'ai-action-toast'; toast.textContent = message;
            if (type === 'error') {
                toast.style.background = 'var(--accent-red)';
                toast.style.color = '#fff';
            }
            document.body.appendChild(toast);
            setTimeout(() => toast.remove(), type === 'error' ? 5000 : 3000);
        }

        // ==========================================
        // Enhanced renderTable with action buttons
        // ==========================================
        // Add ACTIONS column to workload and networking types
        ['pods', 'deployments', 'daemonsets', 'statefulsets', 'replicasets', 'services', 'ingresses'].forEach(resource => {
            if (!tableHeaders[resource].includes('ACTIONS')) {
                tableHeaders[resource].push('ACTIONS');
            }
        });

        // Override renderTableBody to add action buttons for pods
        const baseRenderTableBody = renderTableBody;
        renderTableBody = function(resource, items) {
            const headers = tableHeaders[resource];
            if (!items || items.length === 0) {
                document.getElementById('table-body').innerHTML =
                    `<tr><td colspan="${headers ? headers.length : 1}" style="text-align:center;padding:40px;">No ${resource} found</td></tr>`;
                return;
            }

            document.getElementById('table-body').innerHTML = items.map((item, index) => {
                switch (resource) {
                    case 'pods':
                        const podContainers = item.containers || ['default'];
                        const podContainersJson = JSON.stringify(podContainers).replace(/'/g, "\\'");
                        return `<tr data-index="${index}" data-containers='${podContainersJson}'><td>${item.name}</td><td>${item.namespace}</td><td>${item.ready}</td><td class="status-${item.status.toLowerCase()}">${item.status}</td><td>${item.restarts}</td><td>${item.age}</td><td>${item.ip || '-'}</td><td class="resource-actions"><button class="resource-action-btn terminal" onclick="event.stopPropagation(); openTerminal('${item.name}', '${item.namespace}')">Terminal</button><button class="resource-action-btn logs" onclick="event.stopPropagation(); openLogViewerFromRow(this, '${item.name}', '${item.namespace}')">Logs</button><button class="resource-action-btn portforward" onclick="event.stopPropagation(); openPortForward('${item.name}', '${item.namespace}')">Forward</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Pod', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'deployments':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.ready}</td><td>${item.upToDate || item.up_to_date || '-'}</td><td>${item.available || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Deployment', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'daemonsets':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.desired || '-'}</td><td>${item.current || '-'}</td><td>${item.ready || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('DaemonSet', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'statefulsets':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.ready || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('StatefulSet', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'replicasets':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.desired || '-'}</td><td>${item.current || '-'}</td><td>${item.ready || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn logs" onclick="event.stopPropagation(); openMultiPodLogViewer('${item.name}', '${item.namespace}', '${item.selector || 'app=' + item.name}')">Logs</button><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('ReplicaSet', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'jobs':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.completions || '-'}</td><td>${item.duration || '-'}</td><td>${item.age}</td></tr>`;
                    case 'cronjobs':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.schedule || '-'}</td><td>${item.suspend ? 'Yes' : 'No'}</td><td>${item.active || 0}</td><td>${item.lastSchedule || '-'}</td></tr>`;
                    case 'services':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.type}</td><td>${item.clusterIP}</td><td>${item.ports}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Service', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'ingresses':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.class || item.ingressClass || '-'}</td><td>${item.hosts || '-'}</td><td>${item.address || '-'}</td><td>${item.age}</td><td class="resource-actions"><button class="resource-action-btn topo" onclick="event.stopPropagation(); showTopologyForResource('Ingress', '${item.name}', '${item.namespace}')">Topo</button></td></tr>`;
                    case 'networkpolicies':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.podSelector || '-'}</td><td>${item.age}</td></tr>`;
                    case 'configmaps':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.data || item.dataCount || 0}</td><td>${item.age}</td></tr>`;
                    case 'secrets':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.type || '-'}</td><td>${item.data || item.dataCount || 0}</td><td>${item.age}</td></tr>`;
                    case 'serviceaccounts':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.secrets || 0}</td><td>${item.age}</td></tr>`;
                    case 'persistentvolumes':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.capacity || '-'}</td><td>${item.accessModes || '-'}</td><td>${item.reclaimPolicy || '-'}</td><td class="status-${(item.status || '').toLowerCase()}">${item.status || '-'}</td><td>${item.claim || '-'}</td></tr>`;
                    case 'persistentvolumeclaims':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td class="status-${(item.status || '').toLowerCase()}">${item.status || '-'}</td><td>${item.volume || '-'}</td><td>${item.capacity || '-'}</td><td>${item.accessModes || '-'}</td></tr>`;
                    case 'nodes':
                        return `<tr data-index="${index}"><td>${item.name}</td><td class="status-${(item.status || '').toLowerCase()}">${item.status}</td><td>${item.roles || '-'}</td><td>${item.version || '-'}</td><td>${item.age}</td></tr>`;
                    case 'namespaces':
                        return `<tr data-index="${index}"><td>${item.name}</td><td class="status-active">${item.status}</td><td>${item.age}</td></tr>`;
                    case 'events':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.type}</td><td>${item.reason}</td><td>${item.message?.substring(0, 50) || '-'}...</td><td>${item.count}</td><td>${item.lastSeen}</td></tr>`;
                    case 'roles':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.age}</td></tr>`;
                    case 'rolebindings':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.role || '-'}</td><td>${item.age}</td></tr>`;
                    case 'clusterroles':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.age}</td></tr>`;
                    case 'clusterrolebindings':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.role || '-'}</td><td>${item.age}</td></tr>`;
                    case 'hpa':
                        return `<tr data-index="${index}"><td>${item.name}</td><td>${item.namespace}</td><td>${item.reference || '-'}</td><td>${item.minReplicas || '-'}</td><td>${item.maxReplicas || '-'}</td><td>${item.replicas || '-'}</td><td>${item.age}</td></tr>`;
                    default:
                        // Handle CRDs and unknown types with fallback
                        if (resource.startsWith('crd:')) {
                            return baseRenderTableBody.call(this, resource, [item]).replace(/<tbody[^>]*>|<\/tbody>/g, '');
                        }
                        // Generic fallback for any unhandled resource
                        const values = (headers || ['NAME']).map(h => {
                            const key = h.toLowerCase().replace(/[- ]/g, '');
                            return item[key] || item[h] || item.name || '-';
                        });
                        return `<tr data-index="${index}">${values.map(v => `<td>${v}</td>`).join('')}</tr>`;
                }
            }).join('');
            addRowClickHandlers();
        };

        // Add Metrics nav item
        setTimeout(() => {
            // Find Monitoring section by its title text
            const navSections = document.querySelectorAll('.nav-section');
            let monitoringSection = null;
            for (const section of navSections) {
                const title = section.querySelector('.nav-title');
                if (title && title.textContent.trim() === 'Monitoring') {
                    monitoringSection = section;
                    break;
                }
            }
            if (monitoringSection && !document.querySelector('[onclick="showMetrics()"]')) {
                const metricsItem = document.createElement('div');
                metricsItem.className = 'nav-item';
                metricsItem.onclick = showMetrics;
                metricsItem.innerHTML = '<span>Metrics</span>';
                const firstChild = monitoringSection.querySelector('.nav-item');
                if (firstChild) monitoringSection.insertBefore(metricsItem, firstChild);
            }
        }, 100);

        // ==========================================
        // Cluster Overview
        // ==========================================

        function loadOverviewData() {
            loadClusterOverview();
            loadRecentEvents();
        }

        async function loadClusterOverview() {
            try {
                const resp = await fetchWithAuth('/api/overview');
                if (!resp.ok) return;
                const data = await resp.json();

                // Update context badge
                const ctxEl = document.getElementById('overview-context');
                if (ctxEl && data.context) {
                    ctxEl.textContent = data.context;
                }

                // Update cards
                const nodesEl = document.getElementById('ov-nodes-ready');
                if (nodesEl) nodesEl.textContent = `${data.nodes?.ready || 0}/${data.nodes?.total || 0}`;

                const podsEl = document.getElementById('ov-pods-running');
                if (podsEl) podsEl.textContent = `${data.pods?.running || 0}/${data.pods?.total || 0}`;

                const deployEl = document.getElementById('ov-deploy-healthy');
                if (deployEl) deployEl.textContent = `${data.deployments?.healthy || 0}/${data.deployments?.total || 0}`;

                const nsEl = document.getElementById('ov-namespaces');
                if (nsEl) nsEl.textContent = `${data.namespaces || 0}`;
            } catch (e) {
                console.error('Failed to load cluster overview:', e);
            }
        }

        async function loadRecentEvents() {
            try {
                const resp = await fetchWithAuth('/api/k8s/events?namespace=');
                if (!resp.ok) return;
                const data = await resp.json();
                const eventsEl = document.getElementById('overview-events');
                if (!eventsEl) return;

                const events = (data.items || []).slice(0, 10);
                if (events.length === 0) {
                    eventsEl.innerHTML = '<p class="text-muted">No recent events</p>';
                    return;
                }

                eventsEl.innerHTML = events.map(evt => {
                    const typeLower = (evt.type || 'normal').toLowerCase();
                    const typeClass = typeLower === 'warning' ? 'warning' : 'normal';
                    const msg = K13D?.Utils?.escapeHtml ? K13D.Utils.escapeHtml(evt.message || '') : (evt.message || '');
                    return `<div class="overview-event-item ${typeClass}">
                        <span class="event-type ${typeClass}">${evt.type || 'Normal'}</span>
                        <span class="event-message">${msg.substring(0, 120)}${msg.length > 120 ? '...' : ''}</span>
                        <span class="event-time">${evt.lastSeen || evt.age || ''}</span>
                    </div>`;
                }).join('');
            } catch (e) {
                console.error('Failed to load recent events:', e);
            }
        }

        function showOverviewPanel() {
            closeMobileSidebar();
            currentResource = 'overview';
            document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
            const nav = document.querySelector('.nav-item[data-resource="overview"]');
            if (nav) nav.classList.add('active');
            hideTopologyView();
            hideAllCustomViews();
            const mainPanel = document.querySelector('.main-panel');
            if (mainPanel) mainPanel.style.display = 'none';
            // Hide AI panel on Overview
            const aiPanel = document.getElementById('ai-panel');
            const resizeHandle = document.getElementById('resize-handle');
            if (aiPanel) aiPanel.style.display = 'none';
            if (resizeHandle) resizeHandle.style.display = 'none';
            const btn = document.getElementById('ai-toggle-btn');
            if (btn) btn.classList.remove('active');
            // Show overview container
            const container = document.getElementById('overview-container');
            if (container) container.style.display = 'flex';
            loadOverviewData();
        }

        function hideOverviewPanel() {
            const container = document.getElementById('overview-container');
            if (container) container.style.display = 'none';
            // Restore AI panel state
            const saved = localStorage.getItem('k13d_ai_panel');
            if (saved !== 'closed') {
                const aiPanel = document.getElementById('ai-panel');
                const resizeHandle = document.getElementById('resize-handle');
                if (aiPanel) aiPanel.style.display = 'flex';
                if (resizeHandle) resizeHandle.style.display = 'block';
                const btn = document.getElementById('ai-toggle-btn');
                if (btn) btn.classList.add('active');
            }
        }

        // ============================
        // Custom View Helpers
        // ============================
        const customViewIds = ['overview-container','metrics-dashboard-container','topology-tree-container','applications-container','validate-container','rbac-viz-container','netpol-viz-container','timeline-container','templates-container'];

        function hideAllCustomViews() {
            customViewIds.forEach(id => {
                const el = document.getElementById(id);
                if (el) el.style.display = 'none';
            });
        }

        function showCustomView(containerId, resource) {
            currentResource = resource;
            document.querySelectorAll('.nav-item').forEach(i => i.classList.remove('active'));
            const nav = document.querySelector(`.nav-item[data-resource="${resource}"]`);
            if (nav) nav.classList.add('active');
            hideOverviewPanel();
            hideTopologyView();
            hideAllCustomViews();
            const mainPanel = document.querySelector('.main-panel');
            if (mainPanel) mainPanel.style.display = 'none';
            const container = document.getElementById(containerId);
            if (container) container.style.display = 'flex';
            // Sync namespace selects
            syncCustomViewNamespaces();
        }

        function syncCustomViewNamespaces() {
            const src = document.getElementById('namespace-select');
            if (!src) return;
            ['metrics-dash-ns-select','topo-tree-ns-select','apps-ns-select','validate-ns-select','rbac-viz-ns-select','netpol-viz-ns-select','timeline-ns-select'].forEach(id => {
                const sel = document.getElementById(id);
                if (!sel) return;
                const prev = sel.value;
                sel.innerHTML = '';
                // For validate, add a placeholder instead of "All Namespaces"
                const needsPlaceholder = (id === 'validate-ns-select');
                for (const opt of src.options) {
                    const o = document.createElement('option');
                    o.value = opt.value;
                    if (needsPlaceholder && opt.value === '') {
                        o.textContent = '-- Select Namespace --';
                    } else {
                        o.textContent = opt.textContent;
                    }
                    sel.appendChild(o);
                }
                // Restore previous selection, or use currentNamespace if available
                if (prev) {
                    sel.value = prev;
                } else if (currentNamespace) {
                    sel.value = currentNamespace;
                } else {
                    sel.value = '';
                }
            });
        }

        // ============================
        // Metrics Dashboard View
        // ============================
        function showMetricsDashboard() {
            showCustomView('metrics-dashboard-container', 'metrics');
            loadMetricsDashData();
        }

        async function loadMetricsDashData() {
            const body = document.getElementById('metrics-dash-body');
            const ns = document.getElementById('metrics-dash-ns-select')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading metrics...</div>';
            try {
                const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
                const resp = await fetchWithAuth(`/api/pulse${params}`);
                const d = await resp.json();
                const cpuPct = d.cpu_avail && d.cpu_capacity_milli > 0 ? Math.round(d.cpu_used_milli / d.cpu_capacity_milli * 100) : 0;
                const memPct = d.mem_avail && d.mem_capacity_mib > 0 ? Math.round(d.mem_used_mib / d.mem_capacity_mib * 100) : 0;
                const barColor = pct => pct > 80 ? 'var(--accent-red)' : pct > 60 ? 'var(--accent-yellow)' : 'var(--accent-green)';

                body.innerHTML = `
                    <div class="pulse-grid">
                        <div class="pulse-card">
                            <div class="pulse-card-title">Pods</div>
                            <div class="pulse-card-value">${d.pods_running}<span style="font-size:14px;color:var(--text-secondary);">/${d.pods_total}</span></div>
                            <div class="pulse-card-sub" style="color:var(--accent-green);">${d.pods_running} Running</div>
                            ${d.pods_pending > 0 ? `<div class="pulse-card-sub" style="color:var(--accent-yellow);">${d.pods_pending} Pending</div>` : ''}
                            ${d.pods_failed > 0 ? `<div class="pulse-card-sub" style="color:var(--accent-red);">${d.pods_failed} Failed</div>` : ''}
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Deployments</div>
                            <div class="pulse-card-value">${d.deploys_ready}<span style="font-size:14px;color:var(--text-secondary);">/${d.deploys_total}</span></div>
                            <div class="pulse-card-sub">${d.deploys_ready} Ready${d.deploys_updating > 0 ? `, ${d.deploys_updating} Updating` : ''}</div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">StatefulSets</div>
                            <div class="pulse-card-value">${d.sts_ready}<span style="font-size:14px;color:var(--text-secondary);">/${d.sts_total}</span></div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">DaemonSets</div>
                            <div class="pulse-card-value">${d.ds_ready}<span style="font-size:14px;color:var(--text-secondary);">/${d.ds_total}</span></div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Jobs</div>
                            <div class="pulse-card-value">${d.jobs_complete}<span style="font-size:14px;color:var(--text-secondary);">/${d.jobs_total}</span></div>
                            <div class="pulse-card-sub">${d.jobs_active || 0} Active, ${d.jobs_failed || 0} Failed</div>
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Nodes</div>
                            <div class="pulse-card-value">${d.nodes_ready}<span style="font-size:14px;color:var(--text-secondary);">/${d.nodes_total}</span></div>
                            ${d.nodes_not_ready > 0 ? `<div class="pulse-card-sub" style="color:var(--accent-red);">${d.nodes_not_ready} Not Ready</div>` : '<div class="pulse-card-sub" style="color:var(--accent-green);">All Ready</div>'}
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">CPU Usage</div>
                            ${d.cpu_avail ? `
                                <div class="pulse-card-value">${cpuPct}%</div>
                                <div class="pulse-bar"><div class="pulse-bar-fill" style="width:${cpuPct}%;background:${barColor(cpuPct)};"></div></div>
                                <div class="pulse-card-sub">${d.cpu_used_milli}m / ${d.cpu_capacity_milli}m</div>
                            ` : `
                                <div class="pulse-card-value" style="font-size:14px;color:var(--text-secondary);">N/A</div>
                                <div class="pulse-card-sub" style="color:var(--accent-yellow);">metrics-server not available</div>
                            `}
                        </div>
                        <div class="pulse-card">
                            <div class="pulse-card-title">Memory Usage</div>
                            ${d.mem_avail ? `
                                <div class="pulse-card-value">${memPct}%</div>
                                <div class="pulse-bar"><div class="pulse-bar-fill" style="width:${memPct}%;background:${barColor(memPct)};"></div></div>
                                <div class="pulse-card-sub">${d.mem_used_mib}Mi / ${d.mem_capacity_mib}Mi</div>
                            ` : `
                                <div class="pulse-card-value" style="font-size:14px;color:var(--text-secondary);">N/A</div>
                                <div class="pulse-card-sub" style="color:var(--accent-yellow);">metrics-server not available</div>
                            `}
                        </div>
                    </div>
                    <div style="display:flex;gap:10px;margin-bottom:16px;">
                        <button onclick="showMetrics()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-blue);cursor:pointer;font-size:12px;">Historical Charts</button>
                        <button onclick="showValidateView()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-yellow);cursor:pointer;font-size:12px;">Run Validation</button>
                        <button onclick="showApplicationsView()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-purple);cursor:pointer;font-size:12px;">Applications</button>
                    </div>
                    ${d.events && d.events.length > 0 ? `
                    <div class="pulse-events">
                        <h3>Recent Events</h3>
                        ${d.events.map(e => `
                            <div class="pulse-event-item">
                                <span class="pulse-event-badge ${e.type === 'Warning' ? 'warning' : 'normal'}">${escapeHtml(e.type || 'Normal')}</span>
                                <span style="color:var(--accent-cyan);font-family:monospace;">${escapeHtml(e.reason || '')}</span>
                                <span style="flex:1;">${escapeHtml(e.message || '')}</span>
                                <span style="color:var(--text-muted);flex-shrink:0;">${escapeHtml(e.age || '')}</span>
                            </div>
                        `).join('')}
                    </div>` : ''}
                `;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load metrics: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================
        // Topology Tree View
        // ============================
        function showTopologyTreeView() {
            showCustomView('topology-tree-container', 'topology-tree');
            loadTopologyTreeData();
        }

        async function loadTopologyTreeData() {
            const body = document.getElementById('topo-tree-body');
            const type = document.getElementById('topo-tree-type-select')?.value || 'deploy';
            const ns = document.getElementById('topo-tree-ns-select')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading topology tree...</div>';
            try {
                let params = `?type=${encodeURIComponent(type)}`;
                if (ns) params += `&namespace=${encodeURIComponent(ns)}`;
                const resp = await fetchWithAuth(`/api/xray${params}`);
                const data = await resp.json();
                if (!data.nodes || data.nodes.length === 0) {
                    body.innerHTML = '<div class="loading-placeholder">No resources found for this type/namespace.</div>';
                    return;
                }
                body.innerHTML = `<div class="xray-tree">${data.nodes.map(n => renderXRayNode(n, 0)).join('')}</div>`;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load topology tree: ${escapeHtml(e.message)}</div>`;
            }
        }

        function renderXRayNode(node, depth) {
            const hasChildren = node.children && node.children.length > 0;
            const statusClass = (node.status || '').toLowerCase().replace(/\s+/g, '');
            const kindIcons = {Deployment:'⊞',StatefulSet:'⊟',DaemonSet:'⊠',ReplicaSet:'◫',Pod:'◉',Job:'⧫',CronJob:'⏱',Service:'◎',ConfigMap:'⊡',Secret:'⊗'};
            const icon = kindIcons[node.kind] || '◇';
            const id = `xray-${depth}-${(node.name||'').replace(/[^a-z0-9]/gi,'-')}`;
            return `
                <div class="xray-node">
                    <div class="xray-node-header" onclick="toggleXRayNode('${id}')">
                        <span class="xray-toggle">${hasChildren ? '▼' : '·'}</span>
                        <span class="xray-icon">${icon}</span>
                        <span class="xray-kind">${escapeHtml(node.kind)}</span>
                        <span class="xray-name">${escapeHtml(node.name)}</span>
                        <span class="xray-status ${statusClass}">${escapeHtml(node.status || '')}</span>
                    </div>
                    ${hasChildren ? `<div class="xray-children" id="${id}">${node.children.map(c => renderXRayNode(c, depth + 1)).join('')}</div>` : ''}
                </div>`;
        }

        function toggleXRayNode(id) {
            const el = document.getElementById(id);
            if (!el) return;
            const isHidden = el.style.display === 'none';
            el.style.display = isHidden ? '' : 'none';
            const header = el.previousElementSibling;
            if (header) {
                const toggle = header.querySelector('.xray-toggle');
                if (toggle) toggle.textContent = isHidden ? '▼' : '▶';
            }
        }

        // ============================
        // Applications View
        // ============================
        function showApplicationsView() {
            showCustomView('applications-container', 'applications');
            loadApplicationsData();
        }

        async function loadApplicationsData() {
            const body = document.getElementById('applications-body');
            const ns = document.getElementById('apps-ns-select')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading applications...</div>';
            try {
                const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
                const resp = await fetchWithAuth(`/api/applications${params}`);
                const apps = await resp.json();
                if (!apps || apps.length === 0) {
                    body.innerHTML = '<div class="loading-placeholder">No applications found.</div>';
                    return;
                }
                body.innerHTML = `<div class="apps-grid">${apps.map(app => {
                    const resourceChips = Object.entries(app.resources || {}).map(([kind, items]) =>
                        `<span class="app-resource-chip">${escapeHtml(kind)} (${items.length})</span>`
                    ).join('');
                    return `
                        <div class="app-card">
                            <div class="app-card-header">
                                <span class="app-card-name">${escapeHtml(app.name)}</span>
                                <span class="app-card-badge ${app.status || 'healthy'}">${escapeHtml(app.status || 'healthy')}</span>
                            </div>
                            <div class="app-card-meta">
                                ${app.version ? `<span>v${escapeHtml(app.version)}</span>` : ''}
                                ${app.component ? `<span>${escapeHtml(app.component)}</span>` : ''}
                                ${app.podCount !== undefined ? `<span>Pods: ${app.readyPods || 0}/${app.podCount}</span>` : ''}
                            </div>
                            <div class="app-card-resources">${resourceChips}</div>
                        </div>`;
                }).join('')}</div>`;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load applications: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================
        // Validate View
        // ============================
        function showValidateView() {
            showCustomView('validate-container', 'validate');
            // Auto-select current namespace if validate select is empty
            const sel = document.getElementById('validate-ns-select');
            if (sel && !sel.value && currentNamespace) {
                sel.value = currentNamespace;
            }
            // Always call loadValidateData - it shows a placeholder if no namespace is selected
            loadValidateData();
        }

        async function loadValidateData() {
            const body = document.getElementById('validate-body');
            const ns = document.getElementById('validate-ns-select')?.value;
            if (!ns) {
                body.innerHTML = '<div class="loading-placeholder">Select a namespace to run cross-resource validation.</div>';
                return;
            }
            body.innerHTML = '<div class="loading-placeholder">Running validation...</div>';
            try {
                const resp = await fetchWithAuth(`/api/validate?namespace=${encodeURIComponent(ns)}`);
                const data = await resp.json();
                const findings = data.findings || [];
                const critCount = findings.filter(f => f.severity === 'critical').length;
                const warnCount = findings.filter(f => f.severity === 'warning').length;
                const infoCount = findings.filter(f => f.severity === 'info').length;
                const rc = data.resource_counts || {};
                const scanned = data.resources_scanned || 0;

                body.innerHTML = `
                    <div class="validate-summary">
                        <div class="validate-summary-card">
                            <div class="count" style="color:var(--text-primary);">${scanned}</div>
                            <div class="label">Scanned</div>
                        </div>
                        <div class="validate-summary-card">
                            <div class="count" style="color:var(--accent-red);">${critCount}</div>
                            <div class="label">Critical</div>
                        </div>
                        <div class="validate-summary-card">
                            <div class="count" style="color:var(--accent-yellow);">${warnCount}</div>
                            <div class="label">Warning</div>
                        </div>
                        <div class="validate-summary-card">
                            <div class="count" style="color:var(--accent-blue);">${infoCount}</div>
                            <div class="label">Info</div>
                        </div>
                    </div>
                    <div style="margin-bottom:12px;padding:10px;background:var(--bg-tertiary);border-radius:6px;font-size:12px;color:var(--text-secondary);display:flex;gap:12px;flex-wrap:wrap;">
                        ${rc.pods != null ? `<span>Pods: ${rc.pods}</span>` : ''}
                        ${rc.services != null ? `<span>Services: ${rc.services}</span>` : ''}
                        ${rc.deployments != null ? `<span>Deployments: ${rc.deployments}</span>` : ''}
                        ${rc.statefulsets != null ? `<span>StatefulSets: ${rc.statefulsets}</span>` : ''}
                        ${rc.configmaps != null ? `<span>ConfigMaps: ${rc.configmaps}</span>` : ''}
                        ${rc.secrets != null ? `<span>Secrets: ${rc.secrets}</span>` : ''}
                        ${rc.ingresses != null ? `<span>Ingresses: ${rc.ingresses}</span>` : ''}
                        ${rc.hpas != null ? `<span>HPAs: ${rc.hpas}</span>` : ''}
                    </div>
                    ${findings.length === 0 ? '<div class="loading-placeholder" style="color:var(--accent-green);">No issues found. All resources look healthy!</div>' :
                    findings.map(f => `
                        <div class="validate-finding">
                            <div class="validate-finding-header">
                                <span class="validate-severity ${f.severity || 'info'}">${escapeHtml(f.severity || 'info')}</span>
                                <span class="validate-finding-title">${escapeHtml(f.title || '')}</span>
                            </div>
                            <div class="validate-finding-resource">${escapeHtml(f.resource || '')}</div>
                            <div class="validate-finding-details">${escapeHtml(f.details || '')}</div>
                            ${f.suggestions && f.suggestions.length > 0 ? `
                                <div class="validate-finding-suggestions">
                                    <ul style="margin:0;padding-left:16px;">
                                        ${f.suggestions.map(s => `<li>${escapeHtml(s)}</li>`).join('')}
                                    </ul>
                                </div>` : ''}
                        </div>
                    `).join('')}
                    <div style="margin-top:16px;padding-top:12px;border-top:1px solid var(--border-color);display:flex;gap:10px;">
                        <button onclick="showReports()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-blue);cursor:pointer;font-size:12px;">Generate Full Report (incl. FinOps)</button>
                        <button onclick="showMetricsDashboard()" style="padding:8px 16px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-secondary);color:var(--accent-green);cursor:pointer;font-size:12px;">View Metrics</button>
                    </div>`;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Validation failed: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================

        // ============================
        // Helm View
        // ============================
        function showHelmView() {
            showCustomView('helm-container', 'helm');
            loadHelmData();
        }

        async function loadHelmData() {
            const body = document.getElementById('helm-body');
            const ns = document.getElementById('helm-ns-select')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading Helm releases...</div>';
            try {
                let params = ns ? `?namespace=${encodeURIComponent(ns)}` : '?all=true';
                const resp = await fetchWithAuth(`/api/helm/releases${params}`);
                const data = await resp.json();
                const items = data.items || [];
                if (items.length === 0) {
                    body.innerHTML = '<div class="loading-placeholder">No Helm releases found.</div>';
                    return;
                }
                body.innerHTML = `
                    <table class="helm-table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Namespace</th>
                                <th>Revision</th>
                                <th>Status</th>
                                <th>Chart</th>
                                <th>App Version</th>
                                <th>Updated</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            ${items.map(r => `
                                <tr>
                                    <td style="font-weight:600;color:var(--accent-cyan);cursor:pointer;" onclick="showHelmReleaseDetail('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">${escapeHtml(r.name)}</td>
                                    <td>${escapeHtml(r.namespace || '-')}</td>
                                    <td>${r.revision || '-'}</td>
                                    <td><span class="helm-status ${(r.status || '').toLowerCase()}">${escapeHtml(r.status || '-')}</span></td>
                                    <td style="font-family:monospace;">${escapeHtml(r.chart || '-')}</td>
                                    <td>${escapeHtml(r.appVersion || '-')}</td>
                                    <td style="color:var(--text-secondary);">${r.updated ? new Date(r.updated).toLocaleString() : '-'}</td>
                                    <td class="helm-actions">
                                        <button onclick="showHelmReleaseDetail('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">Details</button>
                                        <button onclick="helmRollback('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">Rollback</button>
                                        <button style="color:var(--accent-red);" onclick="helmUninstall('${escapeHtml(r.name)}','${escapeHtml(r.namespace || '')}')">Uninstall</button>
                                    </td>
                                </tr>
                            `).join('')}
                        </tbody>
                    </table>
                    <div id="helm-detail-area"></div>`;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load Helm releases: ${escapeHtml(e.message)}</div>`;
            }
        }

        async function showHelmReleaseDetail(name, namespace) {
            const area = document.getElementById('helm-detail-area');
            if (!area) return;
            area.innerHTML = '<div class="loading-placeholder">Loading release details...</div>';
            try {
                const nsParam = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
                const [valuesResp, historyResp] = await Promise.all([
                    fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/values${nsParam}&all=true`),
                    fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/history${nsParam}`)
                ]);
                const values = await valuesResp.json();
                const history = await historyResp.json();
                const historyItems = history.items || [];

                area.innerHTML = `
                    <div class="helm-detail-panel">
                        <h3>Release: ${escapeHtml(name)} (${escapeHtml(namespace || 'default')})</h3>
                        <div style="display:flex;gap:16px;margin-bottom:16px;">
                            <button onclick="showHelmDetailTab('values','${escapeHtml(name)}','${escapeHtml(namespace)}')" style="padding:6px 14px;border-radius:6px;border:1px solid var(--border-color);background:var(--accent-blue);color:#fff;cursor:pointer;">Values</button>
                            <button onclick="showHelmDetailTab('history','${escapeHtml(name)}','${escapeHtml(namespace)}')" style="padding:6px 14px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-tertiary);color:var(--text-primary);cursor:pointer;">History</button>
                            <button onclick="showHelmDetailTab('manifest','${escapeHtml(name)}','${escapeHtml(namespace)}')" style="padding:6px 14px;border-radius:6px;border:1px solid var(--border-color);background:var(--bg-tertiary);color:var(--text-primary);cursor:pointer;">Manifest</button>
                        </div>
                        <div id="helm-detail-content">
                            <pre>${escapeHtml(JSON.stringify(values, null, 2))}</pre>
                        </div>
                        ${historyItems.length > 0 ? `
                        <div style="margin-top:16px;">
                            <h4 style="margin-bottom:8px;">Revision History</h4>
                            <table class="helm-table" style="font-size:12px;">
                                <thead><tr><th>Rev</th><th>Status</th><th>Chart</th><th>Description</th><th>Updated</th></tr></thead>
                                <tbody>
                                    ${historyItems.map(h => `
                                        <tr>
                                            <td>${h.revision || '-'}</td>
                                            <td><span class="helm-status ${(h.status || '').toLowerCase()}">${escapeHtml(h.status || '')}</span></td>
                                            <td style="font-family:monospace;">${escapeHtml(h.chart || '-')}</td>
                                            <td>${escapeHtml(h.description || '-')}</td>
                                            <td style="color:var(--text-secondary);">${h.updated ? new Date(h.updated).toLocaleString() : '-'}</td>
                                        </tr>
                                    `).join('')}
                                </tbody>
                            </table>
                        </div>` : ''}
                    </div>`;
            } catch (e) {
                area.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load details: ${escapeHtml(e.message)}</div>`;
            }
        }

        async function showHelmDetailTab(tab, name, namespace) {
            const content = document.getElementById('helm-detail-content');
            if (!content) return;
            content.innerHTML = '<div class="loading-placeholder">Loading...</div>';
            const nsParam = namespace ? `?namespace=${encodeURIComponent(namespace)}` : '';
            try {
                if (tab === 'values') {
                    const resp = await fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/values${nsParam}&all=true`);
                    const data = await resp.json();
                    content.innerHTML = `<pre>${escapeHtml(JSON.stringify(data, null, 2))}</pre>`;
                } else if (tab === 'manifest') {
                    const resp = await fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/manifest${nsParam}`);
                    const text = await resp.text();
                    content.innerHTML = `<pre>${escapeHtml(text)}</pre>`;
                } else if (tab === 'history') {
                    const resp = await fetchWithAuth(`/api/helm/release/${encodeURIComponent(name)}/history${nsParam}`);
                    const data = await resp.json();
                    content.innerHTML = `<pre>${escapeHtml(JSON.stringify(data, null, 2))}</pre>`;
                }
            } catch (e) {
                content.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
            }
        }

        async function helmRollback(name, namespace) {
            const revision = prompt(`Rollback "${name}" to which revision? (Enter revision number)`);
            if (!revision) return;
            try {
                await fetchWithAuth('/api/helm/rollback', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({name, namespace, revision: parseInt(revision)})
                });
                alert(`Release "${name}" rolled back to revision ${revision}`);
                loadHelmData();
            } catch (e) {
                alert('Rollback failed: ' + e.message);
            }
        }

        async function helmUninstall(name, namespace) {
            if (!confirm(`Uninstall Helm release "${name}" from "${namespace || 'default'}"?`)) return;
            try {
                await fetchWithAuth('/api/helm/uninstall', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({name, namespace})
                });
                alert(`Release "${name}" uninstalled`);
                loadHelmData();
            } catch (e) {
                alert('Uninstall failed: ' + e.message);
            }
        }

        // ============================
        // Multi-Cluster Context Switcher
        // ============================
        async function loadClusterContexts() {
            try {
                const resp = await fetchWithAuth('/api/contexts');
                const data = await resp.json();
                const list = document.getElementById('cluster-dropdown-list');
                const nameEl = document.getElementById('cluster-name');
                if (!list) return;
                if (data.currentContext) nameEl.textContent = data.currentContext;
                list.innerHTML = (data.contexts || []).map((ctx, i) => `
                    <div class="cluster-dropdown-item ${ctx.name === data.currentContext ? 'active' : ''}" data-ctx-index="${i}">
                        <span class="ctx-icon"></span>
                        <div style="flex:1;">
                            <div style="font-weight:${ctx.name === data.currentContext ? '600' : '400'}">${escapeHtml(ctx.name)}</div>
                            <div style="font-size:11px;color:var(--text-secondary);">${escapeHtml(ctx.cluster || '')}</div>
                        </div>
                        ${ctx.name === data.currentContext ? '<span style="color:var(--accent-green);">●</span>' : ''}
                    </div>
                `).join('');
                // Use event delegation instead of inline onclick to prevent XSS
                list.querySelectorAll('[data-ctx-index]').forEach(el => {
                    el.addEventListener('click', () => {
                        const idx = parseInt(el.dataset.ctxIndex, 10);
                        const ctxName = (data.contexts || [])[idx]?.name;
                        if (ctxName) switchClusterContext(ctxName);
                    });
                });
            } catch (e) { console.warn('Failed to load contexts:', e); }
        }

        function toggleClusterDropdown() {
            const dd = document.getElementById('cluster-dropdown');
            dd.classList.toggle('active');
            if (dd.classList.contains('active')) loadClusterContexts();
        }

        async function switchClusterContext(name) {
            try {
                const resp = await fetchWithAuth('/api/contexts/switch', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({context: name})
                });
                if (!resp.ok) {
                    const errText = await resp.text();
                    throw new Error(errText || `HTTP ${resp.status}`);
                }
                document.getElementById('cluster-name').textContent = name;
                document.getElementById('cluster-dropdown').classList.remove('active');
                showToast(`Switched to context: ${name}`);
                // Reload namespaces (different cluster = different namespaces)
                currentNamespace = '';
                document.getElementById('namespace-select').value = '';
                await loadNamespaces();
                syncCustomViewNamespaces();
                // Reload all data for new cluster
                loadData();
            } catch (e) { alert('Failed to switch context: ' + e.message); }
        }

        // Close dropdown when clicking outside
        document.addEventListener('click', (e) => {
            const sw = document.getElementById('cluster-switcher');
            if (sw && !sw.contains(e.target)) {
                document.getElementById('cluster-dropdown')?.classList.remove('active');
            }
        });

        // ============================
        // RBAC Visualization View
        // ============================
        function showRBACVizView() {
            showCustomView('rbac-viz-container', 'rbac-viz');
            loadRBACVizData();
        }

        async function loadRBACVizData() {
            const body = document.getElementById('rbac-viz-body');
            const ns = document.getElementById('rbac-viz-ns-select')?.value || '';
            const filter = document.getElementById('rbac-viz-filter')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading RBAC data...</div>';
            try {
                let url = '/api/rbac/visualization';
                const params = [];
                if (ns) params.push(`namespace=${encodeURIComponent(ns)}`);
                if (filter) params.push(`subject_kind=${encodeURIComponent(filter)}`);
                if (params.length) url += '?' + params.join('&');
                const resp = await fetchWithAuth(url);
                const data = await resp.json();
                if (!data.subjects || data.subjects.length === 0) {
                    body.innerHTML = '<div class="loading-placeholder">No RBAC bindings found</div>';
                    return;
                }
                body.innerHTML = data.subjects.map(s => {
                    const iconClass = s.kind === 'ServiceAccount' ? 'sa' : s.kind === 'User' ? 'user' : 'group';
                    const initial = s.kind === 'ServiceAccount' ? 'SA' : s.kind === 'User' ? 'U' : 'G';
                    return `<div class="rbac-card">
                        <div class="rbac-card-header">
                            <div class="rbac-subject-icon ${iconClass}">${initial}</div>
                            <div>
                                <div style="font-weight:600;">${escapeHtml(s.name)}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">${escapeHtml(s.kind)}${s.namespace ? ' · ' + escapeHtml(s.namespace) : ''}</div>
                            </div>
                        </div>
                        <div style="display:flex;flex-wrap:wrap;gap:4px;">
                            ${(s.roles || []).map(r => `<span class="rbac-role-badge ${r.cluster_scope ? 'cluster' : ''}">${r.cluster_scope ? '⊕ ' : ''}${escapeHtml(r.role_name)}</span>`).join('')}
                        </div>
                    </div>`;
                }).join('');
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed to load RBAC: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================
        // Network Policy Visualization
        // ============================
        function showNetPolVizView() {
            showCustomView('netpol-viz-container', 'netpol-viz');
            loadNetPolVizData();
        }

        async function loadNetPolVizData() {
            const body = document.getElementById('netpol-viz-body');
            const ns = document.getElementById('netpol-viz-ns-select')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading network policies...</div>';
            try {
                const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
                const resp = await fetchWithAuth(`/api/netpol/visualization${params}`);
                const data = await resp.json();
                if (!data.policies || data.policies.length === 0) {
                    body.innerHTML = '<div class="loading-placeholder">No network policies found</div>';
                    return;
                }
                body.innerHTML = data.policies.map(p => `
                    <div class="netpol-card">
                        <div class="netpol-card-header">
                            <div>
                                <div style="font-weight:600;">${escapeHtml(p.name)}</div>
                                <div style="font-size:11px;color:var(--text-secondary);">${escapeHtml(p.namespace)}</div>
                            </div>
                            <div class="netpol-selector">Selector: ${escapeHtml(p.pod_selector || '*')}</div>
                        </div>
                        <div class="netpol-direction">
                            <div class="netpol-direction-col">
                                <div class="netpol-direction-label">↓ Ingress (${(p.ingress_rules || []).length} rules)</div>
                                ${(p.ingress_rules || []).map(r => `<div class="netpol-rule">${escapeHtml(r)}</div>`).join('') || '<div style="font-size:12px;color:var(--text-secondary);">No ingress rules</div>'}
                            </div>
                            <div class="netpol-direction-col">
                                <div class="netpol-direction-label">↑ Egress (${(p.egress_rules || []).length} rules)</div>
                                ${(p.egress_rules || []).map(r => `<div class="netpol-rule">${escapeHtml(r)}</div>`).join('') || '<div style="font-size:12px;color:var(--text-secondary);">No egress rules</div>'}
                            </div>
                        </div>
                    </div>
                `).join('');
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================
        // Event Timeline View
        // ============================
        function showTimelineView() {
            showCustomView('timeline-container', 'timeline');
            loadTimelineData();
        }

        async function loadTimelineData() {
            const body = document.getElementById('timeline-body');
            const ns = document.getElementById('timeline-ns-select')?.value || '';
            const hours = document.getElementById('timeline-hours')?.value || '24';
            const warningsOnly = document.getElementById('timeline-warnings-only')?.checked || false;
            body.innerHTML = '<div class="loading-placeholder">Loading events...</div>';
            try {
                const params = new URLSearchParams();
                if (ns) params.set('namespace', ns);
                params.set('hours', hours);
                if (warningsOnly) params.set('warnings_only', 'true');
                const resp = await fetchWithAuth(`/api/events/timeline?${params}`);
                const data = await resp.json();
                const totalEvents = (data.totalNormal || 0) + (data.totalWarning || 0);
                let html = `<div class="timeline-stats">
                    <div class="timeline-stat"><div class="timeline-stat-value">${totalEvents}</div><div class="timeline-stat-label">Total Events</div></div>
                    <div class="timeline-stat"><div class="timeline-stat-value" style="color:var(--accent-green);">${data.totalNormal || 0}</div><div class="timeline-stat-label">Normal</div></div>
                    <div class="timeline-stat"><div class="timeline-stat-value" style="color:var(--accent-red);">${data.totalWarning || 0}</div><div class="timeline-stat-label">Warning</div></div>
                </div>`;
                if (data.windows && data.windows.length > 0) {
                    html += '<div class="timeline-container"><div class="timeline-line"></div>';
                    data.windows.forEach(w => {
                        const dotClass = w.warningCount > 0 ? 'warning' : '';
                        const windowTime = formatTime(w.timestamp);
                        const windowCount = (w.normalCount || 0) + (w.warningCount || 0);
                        html += `<div class="timeline-group">
                            <div class="timeline-dot ${dotClass}"></div>
                            <div class="timeline-time">${escapeHtml(windowTime)} (${windowCount} events)</div>
                            ${(w.events || []).slice(0, 10).map(e => `
                                <div class="timeline-event">
                                    <span class="evt-type ${escapeHtml(e.type)}">${escapeHtml(e.type)}</span>
                                    <span class="evt-reason">${escapeHtml(e.reason || '')}</span>
                                    <span class="evt-msg">${escapeHtml(e.message || '').substring(0, 120)}</span>
                                </div>
                            `).join('')}
                            ${(w.events || []).length > 10 ? `<div style="font-size:11px;color:var(--text-secondary);padding:4px 12px;">...and ${(w.events || []).length - 10} more</div>` : ''}
                        </div>`;
                    });
                    html += '</div>';
                } else {
                    html += '<div class="loading-placeholder">No events in the selected time range</div>';
                }
                body.innerHTML = html;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================
        // GitOps View
        // ============================
        function showGitOpsView() {
            showCustomView('gitops-container', 'gitops');
            loadGitOpsData();
        }

        async function loadGitOpsData() {
            const body = document.getElementById('gitops-body');
            const ns = document.getElementById('gitops-ns-select')?.value || '';
            body.innerHTML = '<div class="loading-placeholder">Loading GitOps status...</div>';
            try {
                const params = ns ? `?namespace=${encodeURIComponent(ns)}` : '';
                const resp = await fetchWithAuth(`/api/gitops/status${params}`);
                const data = await resp.json();
                const argoApps = data.argocd || [];
                const fluxApps = data.flux || [];
                if (argoApps.length === 0 && fluxApps.length === 0) {
                    body.innerHTML = `<div class="gitops-empty">
                        <div style="font-size:48px;margin-bottom:16px;">🔄</div>
                        <h3>No GitOps Resources Found</h3>
                        <p style="margin-top:8px;">${escapeHtml(data.message || 'Install ArgoCD or Flux to enable GitOps features')}</p>
                    </div>`;
                    return;
                }
                let html = '';
                if (argoApps.length > 0) {
                    html += `<h3 style="margin-bottom:12px;display:flex;align-items:center;gap:8px;"><span style="font-size:20px;">🐙</span> ArgoCD Applications (${argoApps.length})</h3>`;
                    html += argoApps.map(a => {
                        const syncStatus = (a.syncStatus || 'unknown').toLowerCase();
                        const dotClass = syncStatus === 'synced' ? 'synced' : syncStatus === 'outofsync' ? 'outofsync' : 'unknown';
                        return `<div class="gitops-card">
                            <div class="gitops-status-dot ${dotClass}"></div>
                            <div class="gitops-info">
                                <div class="gitops-name">${escapeHtml(a.name)}</div>
                                <div class="gitops-meta">${escapeHtml(a.namespace)} · Health: ${escapeHtml(a.status || 'Unknown')}</div>
                                <div class="gitops-repo">${escapeHtml(a.source || '')}</div>
                            </div>
                            <span class="gitops-badge ${dotClass}">${escapeHtml(a.syncStatus || 'Unknown')}</span>
                        </div>`;
                    }).join('');
                }
                if (fluxApps.length > 0) {
                    html += `<h3 style="margin:20px 0 12px;display:flex;align-items:center;gap:8px;"><span style="font-size:20px;">🌊</span> Flux Kustomizations (${fluxApps.length})</h3>`;
                    html += fluxApps.map(f => {
                        const isReady = f.status === 'Ready';
                        const readyClass = isReady ? 'synced' : 'degraded';
                        return `<div class="gitops-card">
                            <div class="gitops-status-dot ${readyClass}"></div>
                            <div class="gitops-info">
                                <div class="gitops-name">${escapeHtml(f.name)}</div>
                                <div class="gitops-meta">${escapeHtml(f.namespace)} · Source: ${escapeHtml(f.source || '')}</div>
                            </div>
                            <span class="gitops-badge ${readyClass}">${isReady ? 'Ready' : 'Not Ready'}</span>
                        </div>`;
                    }).join('');
                }
                body.innerHTML = html;
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
            }
        }

        // ============================
        // Templates View
        // ============================
        let allTemplates = [];

        function showTemplatesView() {
            showCustomView('templates-container', 'templates');
            loadTemplatesData();
        }

        async function loadTemplatesData() {
            const body = document.getElementById('templates-body');
            body.innerHTML = '<div class="loading-placeholder">Loading templates...</div>';
            try {
                const resp = await fetchWithAuth('/api/templates');
                const data = await resp.json();
                allTemplates = data.templates || [];
                renderTemplates(allTemplates);
            } catch (e) {
                body.innerHTML = `<div class="loading-placeholder" style="color:var(--accent-red);">Failed: ${escapeHtml(e.message)}</div>`;
            }
        }

        function filterTemplates() {
            const cat = document.getElementById('templates-category')?.value || '';
            const filtered = cat ? allTemplates.filter(t => t.category === cat) : allTemplates;
            renderTemplates(filtered);
        }

        function renderTemplates(templates) {
            const body = document.getElementById('templates-body');
            if (!templates.length) {
                body.innerHTML = '<div class="loading-placeholder">No templates available</div>';
                return;
            }
            const icons = {webserver:'🌐',database:'💾',cache:'⚡',queue:'📨',monitoring:'📊',batch:'⏱️',networking:'🔗'};
            body.innerHTML = `<div class="templates-grid">${templates.map((t, i) => `
                <div class="template-card" onclick="openTemplateDeploy(${i})">
                    <div class="template-card-icon">${icons[t.category] || '📦'}</div>
                    <div class="template-card-name">${escapeHtml(t.name)}</div>
                    <div class="template-card-desc">${escapeHtml(t.description)}</div>
                    <span class="template-card-category">${escapeHtml(t.category)}</span>
                </div>
            `).join('')}</div>`;
        }

        function openTemplateDeploy(idx) {
            const t = allTemplates[idx];
            if (!t) return;
            document.getElementById('template-deploy-name').textContent = t.name;
            document.getElementById('template-deploy-yaml').value = t.yaml || '';
            document.getElementById('template-deploy-ns').value = 'default';
            document.getElementById('template-deploy-resname').value = '';
            document.getElementById('template-deploy-modal').classList.add('active');
        }

        function closeTemplateDeployModal() {
            document.getElementById('template-deploy-modal').classList.remove('active');
        }

        async function applyTemplate() {
            const yaml = document.getElementById('template-deploy-yaml').value;
            const ns = document.getElementById('template-deploy-ns').value || 'default';
            try {
                const resp = await fetchWithAuth('/api/templates/apply', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({yaml, namespace: ns})
                });
                if (!resp.ok) {
                    const errText = await resp.text();
                    showToast('Deploy failed: ' + errText, 'error');
                    return;
                }
                const data = await resp.json();
                showToast('Template deployed: ' + (data.result || 'success'), 'success');
                closeTemplateDeployModal();
            } catch (e) {
                showToast('Deploy failed: ' + e.message, 'error');
            }
        }


        // ============================
        // Resource Diff Modal
        // ============================
        async function showResourceDiff(kind, name, namespace) {
            document.getElementById('diff-resource-label').textContent = `${kind}/${name} (${namespace || 'default'})`;
            document.getElementById('diff-left').textContent = 'Loading...';
            document.getElementById('diff-right').textContent = 'Loading...';
            document.getElementById('diff-modal').classList.add('active');
            try {
                const resp = await fetchWithAuth('/api/diff', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify({resource: kind, name, namespace})
                });
                const data = await resp.json();
                document.getElementById('diff-left').textContent = data.lastApplied || '(no last-applied annotation found)';
                document.getElementById('diff-right').textContent = data.currentYaml || '(failed to get current)';
            } catch (e) {
                document.getElementById('diff-left').textContent = 'Error: ' + e.message;
                document.getElementById('diff-right').textContent = '';
            }
        }

        function closeDiffModal() {
            document.getElementById('diff-modal').classList.remove('active');
        }

        // ============================
        // AI Auto-Troubleshoot
        // ============================
        async function runAutoTroubleshoot() {
            const ns = currentNamespace || '';
            const aiInput = document.getElementById('ai-input');
            const prompt = ns
                ? `Analyze namespace "${ns}" for issues. Check for CrashLoopBackOff pods, OOMKilled, pending pods, failed deployments, and recent warning events. Provide a diagnosis and remediation steps.`
                : `Analyze the entire cluster for issues. Check all namespaces for CrashLoopBackOff pods, OOMKilled, pending pods, failed deployments, and recent warning events. Provide a diagnosis and remediation steps.`;
            if (aiInput) {
                aiInput.value = prompt;
                sendMessage();
            }
        }

        // ============================
        // Notification Settings
        // ============================
        async function loadNotificationSettings() {
            try {
                const resp = await fetchWithAuth('/api/notifications/config');
                const data = await resp.json();
                if (data.enabled !== undefined) document.getElementById('notif-enabled').checked = data.enabled;
                if (data.provider) document.getElementById('notif-platform').value = data.provider;
                if (data.webhook_url) document.getElementById('notif-webhook-url').value = data.webhook_url;
                if (data.channel) document.getElementById('notif-channel').value = data.channel;
                const evts = data.events || [];
                const evtMap = {'pod_crash': 'notif-evt-crash', 'oom_killed': 'notif-evt-oom', 'node_not_ready': 'notif-evt-node', 'deploy_fail': 'notif-evt-deploy', 'image_pull_fail': 'notif-evt-imagepull'};
                Object.entries(evtMap).forEach(([key, id]) => { const el = document.getElementById(id); if (el) el.checked = evts.includes(key); });
                // Load SMTP settings if email provider
                if (data.smtp) {
                    const s = data.smtp;
                    if (s.host) document.getElementById('notif-smtp-host').value = s.host;
                    if (s.port) document.getElementById('notif-smtp-port').value = s.port;
                    if (s.username) document.getElementById('notif-smtp-username').value = s.username;
                    if (s.from) document.getElementById('notif-smtp-from').value = s.from;
                    if (s.to) document.getElementById('notif-smtp-to').value = s.to.join(', ');
                    document.getElementById('notif-smtp-tls').checked = s.use_tls !== false;
                }
                updateNotifPlaceholder();
                loadNotificationHistory();
            } catch (e) { console.warn('Failed to load notification settings:', e); }
        }

        async function saveNotificationSettings() {
            const provider = document.getElementById('notif-platform').value;
            const payload = {
                enabled: document.getElementById('notif-enabled').checked,
                provider: provider,
                webhook_url: document.getElementById('notif-webhook-url').value,
                channel: document.getElementById('notif-channel').value,
                events: [
                    document.getElementById('notif-evt-crash')?.checked ? 'pod_crash' : '',
                    document.getElementById('notif-evt-oom')?.checked ? 'oom_killed' : '',
                    document.getElementById('notif-evt-node')?.checked ? 'node_not_ready' : '',
                    document.getElementById('notif-evt-deploy')?.checked ? 'deploy_fail' : '',
                    document.getElementById('notif-evt-imagepull')?.checked ? 'image_pull_fail' : ''
                ].filter(Boolean)
            };
            if (provider === 'email') {
                const toStr = document.getElementById('notif-smtp-to').value;
                payload.smtp = {
                    host: document.getElementById('notif-smtp-host').value,
                    port: parseInt(document.getElementById('notif-smtp-port').value) || 587,
                    username: document.getElementById('notif-smtp-username').value,
                    password: document.getElementById('notif-smtp-password').value,
                    from: document.getElementById('notif-smtp-from').value,
                    to: toStr ? toStr.split(',').map(s => s.trim()).filter(Boolean) : [],
                    use_tls: document.getElementById('notif-smtp-tls').checked
                };
            }
            try {
                await fetchWithAuth('/api/notifications/config', {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(payload)
                });
                const result = document.getElementById('notif-test-result');
                result.style.display = 'block';
                result.style.background = 'rgba(46,160,67,0.15)';
                result.style.color = 'var(--accent-green)';
                result.textContent = 'Settings saved!';
                setTimeout(() => { result.style.display = 'none'; }, 3000);
            } catch (e) { alert('Failed to save: ' + e.message); }
        }

        async function testNotification() {
            const result = document.getElementById('notif-test-result');
            result.style.display = 'block';
            result.style.background = 'rgba(56,132,244,0.15)';
            result.style.color = 'var(--accent-blue)';
            result.textContent = 'Sending test notification...';
            try {
                const resp = await fetchWithAuth('/api/notifications/test', {method: 'POST'});
                if (!resp.ok) {
                    const text = await resp.text();
                    throw new Error(text || resp.statusText);
                }
                result.style.background = 'rgba(46,160,67,0.15)';
                result.style.color = 'var(--accent-green)';
                result.textContent = 'Test notification sent!';
            } catch (e) {
                result.style.background = 'rgba(248,81,73,0.15)';
                result.style.color = 'var(--accent-red)';
                result.textContent = 'Failed: ' + e.message;
            }
        }

        function updateNotifPlaceholder() {
            const platform = document.getElementById('notif-platform').value;
            const webhookSection = document.getElementById('notif-webhook-section');
            const smtpSection = document.getElementById('notif-smtp-section');
            if (platform === 'email') {
                webhookSection.style.display = 'none';
                smtpSection.style.display = 'block';
            } else {
                webhookSection.style.display = 'block';
                smtpSection.style.display = 'none';
                const urlInput = document.getElementById('notif-webhook-url');
                const placeholders = {
                    slack: 'https://hooks.slack.com/services/...',
                    discord: 'https://discord.com/api/webhooks/...',
                    teams: 'https://outlook.office.com/webhook/...',
                    custom: 'https://your-webhook-url.com/hook'
                };
                urlInput.placeholder = placeholders[platform] || placeholders.custom;
            }
        }

        async function loadNotificationHistory() {
            const body = document.getElementById('notif-history-body');
            try {
                const resp = await fetchWithAuth('/api/notifications/history');
                const items = await resp.json();
                if (!items || items.length === 0) {
                    body.innerHTML = '<div class="loading-placeholder" style="font-size:12px;">No notifications sent yet.</div>';
                    return;
                }
                body.innerHTML = items.map(h => {
                    const time = h.timestamp ? new Date(h.timestamp).toLocaleString() : '';
                    const icon = h.success ? '<span style="color:var(--accent-green);">&#10003;</span>' : '<span style="color:var(--accent-red);">&#10007;</span>';
                    return `<div style="display:flex;align-items:center;gap:8px;padding:6px 0;border-bottom:1px solid var(--border-color);font-size:12px;">
                        ${icon}
                        <span style="color:var(--text-secondary);min-width:140px;">${escapeHtml(time)}</span>
                        <span style="color:var(--accent-blue);min-width:80px;">${escapeHtml(h.event_type || '')}</span>
                        <span style="flex:1;color:var(--text-primary);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;">${escapeHtml(h.message || '')}</span>
                    </div>`;
                }).join('');
            } catch (e) {
                body.innerHTML = '<div class="loading-placeholder" style="font-size:12px;color:var(--accent-red);">Failed to load history</div>';
            }
        }

        // Init
        init();
        loadClusterContexts();
