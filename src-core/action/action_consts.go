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

	// 把 Bot 工具
	START_BOT_TASK = "start-bot-task"
	QUERY_BOT_TASK = "query-bot-task"
	ABORT_BOT_TASK = "abort-bot-task"

	// MCP工具
	USE_MCP_TOOL = "use-mcp-tool"
	// 自研工具
	USE_SELF_TOOL = "use-self-tool"
	SET_SELF_TOOL = "set-self-tool"
	// 发布为小程序
	PUBLISH_AS_APP = "publish-as-app"
)

const TOOL_RESULT_TAG = "<!-- [tool-result] -->"
