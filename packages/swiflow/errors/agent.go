package errors

import "fmt"

var ErrEmptyLlmResponse = fmt.Errorf("empty response of llm")
var ErrListMcpToolsError = fmt.Errorf("list mcp tools error")
var ErrExceededMaximumTurns = fmt.Errorf("exceeded maximum turns")
var ErrTaskTerminatedByUser = fmt.Errorf("task terminated by user")
