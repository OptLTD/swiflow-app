package server

import "net/http"

// Stable API error codes returned in JSON {"error":"<code>","detail":"...optional"}.
// UI translates via t('errors.<code>'); detail is machine/debug only.
const (
	ErrInvalidJSON              = "invalid_json"
	ErrActRequired              = "act_required"
	ErrUnknownAct               = "unknown_act"
	ErrIDRequired               = "id_required"
	ErrKeyRequired              = "key_required"
	ErrNameRequired             = "name_required"
	ErrSlugRequired             = "slug_required"
	ErrPathRequired             = "path_required"
	ErrMessageRequired          = "message_required"
	ErrListFailed               = "list_failed"
	ErrCreateFailed             = "create_failed"
	ErrUpdateFailed             = "update_failed"
	ErrDeleteFailed             = "delete_failed"
	ErrSaveFailed               = "save_failed"
	ErrReloadFailed             = "reload_failed"
	ErrSyncFailed               = "sync_failed"
	ErrNotFound                 = "not_found"
	ErrNoFieldsToUpdate         = "no_fields_to_update"
	ErrNoFiles                  = "no_files"
	ErrSessionNotFound          = "session_not_found"
	ErrLoadMessagesFailed       = "load_messages_failed"
	ErrSessionWatchUnavailable  = "session_watch_unavailable"
	ErrStreamingUnsupported     = "streaming_unsupported"
	ErrTooManyConcurrentRuns    = "too_many_concurrent_runs"
	ErrWindowBridgeUnavailable  = "window_bridge_unavailable"
	ErrHarnessUnavailable       = "harness_unavailable"
	ErrRunNotFound              = "run_not_found"
	ErrNameAPIKeyRequired       = "name_api_key_required"
	ErrAgentNotFound            = "agent_not_found"
	ErrUnknownAgent             = "unknown_agent"
	ErrUnknownTxtModelProvider  = "unknown_txt_model_provider"
	ErrUnknownImgModelProvider  = "unknown_img_model_provider"
	ErrExecToolsDisabled        = "exec_tools_disabled"
	ErrBrowserToolDisabled      = "browser_tool_disabled"
	ErrDocumentToolDisabled     = "document_tool_disabled"
	ErrNameAndTypeRequired      = "name_and_type_required"
	ErrCmdRequiredForStdio      = "cmd_required_for_stdio"
	ErrURLRequiredForType       = "url_required_for_type"
	ErrInvalidMCPType           = "invalid_mcp_type"
	ErrMCPSyncFailed            = "mcp_sync_failed"
	ErrCronReloadFailed         = "cron_reload_failed"
	ErrCronFieldsRequired       = "cron_fields_required"
	ErrPathIsDirectory          = "path_is_directory"
	ErrInvalidMultipart         = "invalid_multipart"
	ErrMkdirFailed              = "mkdir_failed"
	ErrUploadRequiresMultipart  = "upload_requires_multipart"
	ErrURLMustBeHTTP            = "url_must_be_http"
	ErrFileTooLarge             = "file_too_large"
	ErrInvalidLightAppRuntime   = "invalid_lightapp_runtime"
	ErrCreateAppDirFailed       = "create_app_dir_failed"
	ErrLightAppManagerUnavailable = "lightapp_manager_unavailable"
	ErrKeyAndValueRequired      = "key_and_value_required"
	ErrInvalidRuntimeInstallName = "invalid_runtime_install_name"
	ErrInvalidRuntimeInstallMode = "invalid_runtime_install_mode"
	ErrInstallAlreadyRunning    = "install_already_running"
	ErrScriptNotFound           = "script_not_found"
	ErrInstallStartFailed       = "install_start_failed"
	ErrUnsupportedSearchProvider = "unsupported_search_provider"
	ErrInternalError            = "internal_error"
)

// writeErr writes {"error": code} and optional "detail".
func writeErr(w http.ResponseWriter, status int, code string, detail ...string) {
	body := map[string]string{"error": code}
	if len(detail) > 0 && detail[0] != "" {
		body["detail"] = detail[0]
	}
	writeJSON(w, status, body)
}
