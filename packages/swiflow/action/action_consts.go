package action

const (
	// default
	THINKING = "thinking"
	DATETIME = "datetime"
	MEMORIZE = "memorize"
	ANNOTATE = "annotate"
	WAITTODO = "wait-todo"

	// 交互动作
	MAKE_ASK = "make-ask"
	COMPLETE = "complete"

	// 用户输入
	USER_INPUT  = "user-input"
	TOOL_RESULT = "tool-result"

	// 文件操作
	PATH_LIST_FILES   = "path-list-files"
	FILE_GET_CONTENT  = "file-get-content"
	FILE_PUT_CONTENT  = "file-put-content"
	FILE_REPLACE_TEXT = "file-replace-text"

	// 执行命令
	EXECUTE_COMMAND = "execute-command"
	START_ASYNC_CMD = "start-async-cmd"
	QUERY_ASYNC_CMD = "query-async-cmd"
	ABORT_ASYNC_CMD = "abort-async-cmd"

	// subtask
	START_SUBTASK = "start-subtask"
	QUERY_SUBTASK = "query-subtask"
	ABORT_SUBTASK = "abort-subtask"

	// MCP工具
	USE_MCP_TOOL     = "use-mcp-tool"
	GET_MCP_RESOURCE = "get-mcp-resource"
	// builtin
	USE_BUILTIN_TOOL = "use-builtin-tool"
)

const TOOL_RESULT_TAG = "<!-- [tool-result] -->"
