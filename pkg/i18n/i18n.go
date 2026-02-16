package i18n

import (
	"strings"
	"sync"
)

type Language string

const (
	EN Language = "en"
	KO Language = "ko"
	ZH Language = "zh"
	JA Language = "ja"
)

var (
	currentLang   Language = EN
	currentLangMu sync.RWMutex
)

func SetLanguage(lang string) {
	currentLangMu.Lock()
	defer currentLangMu.Unlock()
	switch strings.ToLower(lang) {
	case "ko", "korean":
		currentLang = KO
	case "zh", "chinese":
		currentLang = ZH
	case "ja", "japanese":
		currentLang = JA
	default:
		currentLang = EN
	}
}

func GetLanguage() Language {
	currentLangMu.RLock()
	defer currentLangMu.RUnlock()
	return currentLang
}

// translations holds all UI strings keyed by language.
// NOTE: These are hardcoded in-memory maps. For a larger translation set,
// consider loading from embedded JSON/YAML files or using a proper i18n library.
var translations = map[Language]map[string]string{
	EN: {
		"app_title":          "k13d - K8s AI Explorer",
		"dashboard_pods":     "Dashboard (Pods)",
		"ask_ai":             "Ask AI: ",
		"decision_required":  "Decision Required",
		"settings_title":     "LLM Settings",
		"audit_logs":         "Audit Logs",
		"help_title":         "k13d Help & Shortcuts",
		"shortcut_help":      "?:Help",
		"shortcut_cmd":       "/:Command",
		"shortcut_settings":  "s:Settings",
		"shortcut_yaml":      "y:YAML",
		"shortcut_describe":  "d:Describe",
		"shortcut_analyze":   "L:AI Analyze",
		"shortcut_forward":   "Shift-F:Forward",
		"shortcut_ai":        "a:Ask AI",
		"shortcut_quit":      "Ctrl-C:Quit",
		"tab_describe":       "Describe",
		"tab_yaml":           "YAML",
		"tab_events":         "Events",
		"explain_this":       "Explain this resource",
		"beginner_mode":      "Beginner Mode",
		"loading":            "Loading...",
		"error_client":       "K8s Client not initialized",
		"cat_nav":            "General Navigation",
		"cat_dash":           "Dashboard Actions",
		"cat_res":            "Resource Actions",
		"cat_selection":      "Selection",
		"desc_switch":        "Switch resource (e.g., :pods, :svc)",
		"desc_ctx":           "Switch context",
		"desc_audit":         "View enterprise audit logs",
		"desc_filter":        "Filter table rows",
		"desc_regex_filter":  "Regex filter (e.g., /nginx.*/)",
		"desc_clear_filter":  "Clear filter / close panel",
		"desc_yaml":          "View YAML",
		"desc_edit":          "Edit resource in $EDITOR",
		"desc_describe":      "Native resource description",
		"desc_analyze":       "AI-powered analysis",
		"desc_explain":       "AI-powered explanation",
		"desc_shell":         "Shell into pod",
		"desc_scale":         "Scale resource",
		"desc_restart":       "Rollout restart",
		"desc_forward":       "Port Forward",
		"desc_logs":          "Stream logs",
		"desc_delete":        "Delete resource",
		"desc_move_row":      "Move up/down",
		"desc_goto_top":      "Go to first row",
		"desc_goto_bottom":   "Go to last row",
		"desc_page_up":       "Page up (10 rows)",
		"desc_page_down":     "Page down (10 rows)",
		"desc_toggle_select": "Toggle row selection",
		"desc_clear_select":  "Clear all selections",
	},
	KO: {
		"app_title":          "k13d - K8s AI 탐색기",
		"dashboard_pods":     "대시보드 (Pods)",
		"ask_ai":             "AI에게 질문: ",
		"decision_required":  "의사결정 필요",
		"settings_title":     "LLM 설정",
		"audit_logs":         "감사 로그",
		"help_title":         "k13d 도움말 및 단축키",
		"shortcut_help":      "?:도움말",
		"shortcut_cmd":       "/:명령어",
		"shortcut_settings":  "s:설정",
		"shortcut_yaml":      "y:YAML",
		"shortcut_describe":  "d:설명",
		"shortcut_analyze":   "L:AI 분석",
		"shortcut_forward":   "Shift-F:포워딩",
		"shortcut_ai":        "a:AI에게 질문",
		"shortcut_quit":      "Ctrl-C:종료",
		"tab_describe":       "설명",
		"tab_yaml":           "YAML",
		"tab_events":         "이벤트",
		"explain_this":       "이 리소스 설명하기",
		"beginner_mode":      "초보자 모드",
		"loading":            "로딩 중...",
		"error_client":       "K8s 클라이언트가 초기화되지 않았습니다",
		"cat_nav":            "일반 탐색",
		"cat_dash":           "대시보드 작업",
		"cat_res":            "리소스 작업",
		"cat_selection":      "선택",
		"desc_switch":        "리소스 전환 (예: :pods, :svc)",
		"desc_ctx":           "컨텍스트 전환",
		"desc_audit":         "엔터프라이즈 감사 로그 보기",
		"desc_filter":        "테이블 필터링",
		"desc_regex_filter":  "정규식 필터 (예: /nginx.*/)",
		"desc_clear_filter":  "필터 지우기 / 패널 닫기",
		"desc_yaml":          "YAML 보기",
		"desc_edit":          "에디터에서 편집",
		"desc_describe":      "네이티브 리소스 설명 보기",
		"desc_analyze":       "AI 기반 분석",
		"desc_explain":       "AI 기반 설명",
		"desc_shell":         "파드 쉘 접속",
		"desc_scale":         "리소스 스케일링",
		"desc_restart":       "재시작 (Rollout)",
		"desc_forward":       "포트 포워딩",
		"desc_logs":          "로그 스트리밍",
		"desc_delete":        "리소스 삭제",
		"desc_move_row":      "위/아래로 이동",
		"desc_goto_top":      "첫 번째 행으로 이동",
		"desc_goto_bottom":   "마지막 행으로 이동",
		"desc_page_up":       "페이지 위로 (10줄)",
		"desc_page_down":     "페이지 아래로 (10줄)",
		"desc_toggle_select": "행 선택 토글",
		"desc_clear_select":  "모든 선택 해제",
	},
	ZH: {
		"app_title":          "k13d - K8s AI 资源管理器",
		"dashboard_pods":     "仪表板 (Pods)",
		"ask_ai":             "向 AI 提问: ",
		"decision_required":  "需要决策",
		"settings_title":     "LLM 设置",
		"audit_logs":         "审计日志",
		"help_title":         "k13d 帮助与快捷键",
		"shortcut_help":      "?:帮助",
		"shortcut_cmd":       "/:命令",
		"shortcut_settings":  "s:设置",
		"shortcut_yaml":      "y:YAML",
		"shortcut_describe":  "d:详情",
		"shortcut_analyze":   "L:AI 分析",
		"shortcut_forward":   "Shift-F:转发",
		"shortcut_ai":        "a:向 AI 提问",
		"shortcut_quit":      "Ctrl-C:退出",
		"tab_describe":       "详情",
		"tab_yaml":           "YAML",
		"tab_events":         "事件",
		"explain_this":       "解释此资源",
		"beginner_mode":      "入门模式",
		"loading":            "加载中...",
		"error_client":       "K8s 客户端未初始化",
		"cat_nav":            "通用导航",
		"cat_dash":           "仪表板操作",
		"cat_res":            "资源操作",
		"cat_selection":      "选择",
		"desc_switch":        "切换资源 (如 :pods, :svc)",
		"desc_ctx":           "切换上下文",
		"desc_audit":         "查看审计日志",
		"desc_filter":        "过滤表格行",
		"desc_regex_filter":  "正则过滤 (如 /nginx.*/)",
		"desc_clear_filter":  "清除过滤 / 关闭面板",
		"desc_yaml":          "查看 YAML",
		"desc_edit":          "在编辑器中编辑",
		"desc_describe":      "原生资源描述",
		"desc_analyze":       "AI 驱动分析",
		"desc_explain":       "AI 驱动解释",
		"desc_shell":         "进入 Pod Shell",
		"desc_scale":         "调整副本数",
		"desc_restart":       "重启资源",
		"desc_forward":       "端口转发",
		"desc_logs":          "实时日志",
		"desc_delete":        "删除资源",
		"desc_move_row":      "上下移动",
		"desc_goto_top":      "跳转到第一行",
		"desc_goto_bottom":   "跳转到最后一行",
		"desc_page_up":       "向上翻页 (10行)",
		"desc_page_down":     "向下翻页 (10行)",
		"desc_toggle_select": "切换行选择",
		"desc_clear_select":  "清除所有选择",
	},
	JA: {
		"app_title":          "k13d - K8s AI エクスプローラー",
		"dashboard_pods":     "ダッシュボード (Pods)",
		"ask_ai":             "AIに質問: ",
		"decision_required":  "決定が必要",
		"settings_title":     "LLM 設定",
		"audit_logs":         "監査ログ",
		"help_title":         "k13d ヘルプとショートカット",
		"shortcut_help":      "?:ヘルプ",
		"shortcut_cmd":       "/:コマンド",
		"shortcut_settings":  "s:設定",
		"shortcut_yaml":      "y:YAML",
		"shortcut_describe":  "d:詳細",
		"shortcut_analyze":   "L:AI 分析",
		"shortcut_forward":   "Shift-F:フォワード",
		"shortcut_ai":        "a:AIに質問",
		"shortcut_quit":      "Ctrl-C:終了",
		"tab_describe":       "詳細",
		"tab_yaml":           "YAML",
		"tab_events":         "イベント",
		"explain_this":       "このリソースを解説",
		"beginner_mode":      "初心者モード",
		"loading":            "読み込み中...",
		"error_client":       "K8s クライアントが初期化されていません",
		"cat_nav":            "基本ナビゲーション",
		"cat_dash":           "ダッシュボード操作",
		"cat_res":            "リソース操作",
		"cat_selection":      "選択",
		"desc_switch":        "リソース切り替え (:pods, :svc)",
		"desc_ctx":           "コンテキスト切り替え",
		"desc_audit":         "監査ログを表示",
		"desc_filter":        "テーブルをフィルタ",
		"desc_regex_filter":  "正規表現フィルタ (例: /nginx.*/)",
		"desc_clear_filter":  "フィルタをクリア / パネルを閉じる",
		"desc_yaml":          "YAMLを表示",
		"desc_edit":          "エディタで編集",
		"desc_describe":      "ネイティブの詳細表示",
		"desc_analyze":       "AIによる分析",
		"desc_explain":       "AIによる解説",
		"desc_shell":         "Podシェルに接続",
		"desc_scale":         "スケール調整",
		"desc_restart":       "ロールアウト再起動",
		"desc_forward":       "ポートフォワード",
		"desc_logs":          "ログのストリーム",
		"desc_delete":        "リソース削除",
		"desc_move_row":      "上下に移動",
		"desc_goto_top":      "最初の行へ",
		"desc_goto_bottom":   "最後の行へ",
		"desc_page_up":       "ページアップ (10行)",
		"desc_page_down":     "ページダウン (10行)",
		"desc_toggle_select": "行の選択を切り替え",
		"desc_clear_select":  "すべての選択を解除",
	},
}

func T(key string) string {
	currentLangMu.RLock()
	lang := currentLang
	currentLangMu.RUnlock()

	if langMap, ok := translations[lang]; ok {
		if val, ok := langMap[key]; ok {
			return val
		}
	}
	// Fallback to English
	if langMap, ok := translations[EN]; ok {
		if val, ok := langMap[key]; ok {
			return val
		}
	}
	return key
}
