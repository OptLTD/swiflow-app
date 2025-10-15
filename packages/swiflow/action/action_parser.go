package action

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"log"
	"os"
	"reflect"
	"regexp"
	"strings"
	"swiflow/errors"
	"swiflow/support"
	"time"

	"github.com/duke-git/lancet/v2/cryptor"
)

type IAct interface {
	Handle(super *SuperAction) any
}

type Input interface {
	Input() (string, string)
}

type Payload struct {
	UUID string `json:"uuid"`
	Home string `json:"home"`

	Time time.Time `json:"time"`
}

func (ctx *Payload) InitHome() error {
	if ctx.Home == "" {
		return fmt.Errorf("undefined home path")
	}
	if _, err := os.Stat(ctx.Home); os.IsNotExist(err) {
		return os.MkdirAll(ctx.Home, fs.ModeDir|0755)
	}
	return nil
}

type SuperAction struct {
	Origin string `json:"origin"`
	ErrMsg error  `json:"errmsg"`

	UseTools []any  `json:"useTools"`
	Thinking string `json:"thinking"`
	Datetime string `json:"datetime"`
	TheMsgId string `json:"theMsgId"`
	WorkerID string `json:"workerId"`

	Context *Context `json:"context"`
	Payload *Payload `json:"payload"`
}

func Errors(err error) *SuperAction {
	return &SuperAction{ErrMsg: err}
}

func Load(data string) []*SuperAction {
	list := []*SuperAction{}
	if strings.TrimSpace(data) == "" {
		return list
	}

	var response *SuperAction
	thinking, datetime := "", ""
	for strings.TrimSpace(data) != "" {
		// 原格式（过于宽松，易误判普通文本为标签）：
		// re := regexp.MustCompile(`(?s)<([^/>]+)>`)
		// 说明：该正则会将 `<` 后所有非 `/>` 的内容当作标签名，
		// 在包含中文、空格或括号时会导致后续动态拼接正则触发 panic。
		re := regexp.MustCompile(`(?s)<([a-zA-Z][a-zA-Z0-9-]*)>`)
		matches := re.FindStringSubmatch(data)
		if len(matches) == 0 || data == "" {
			break
		}

		left, cut := snap(data, matches[1])
		data = strings.TrimSpace(left)
		if len(cut) == 0 {
			break
		}

		// 分割新的对话
		switch matches[1] {
		case DATETIME:
			cut = strings.TrimPrefix(cut, "<datetime>")
			cut = strings.TrimSuffix(cut, "</datetime>")
			datetime = strings.TrimSpace(cut)
			if response != nil {
				response.Datetime = datetime
				response.Thinking = thinking
				list = append(list, response)
				response, thinking = nil, ""
			}
		case THINKING:
			cut = strings.TrimPrefix(cut, "<thinking>")
			cut = strings.TrimSuffix(cut, "</thinking>")
			thinking = strings.TrimSpace(cut)
		default:
			if r := Parse(cut); len(r.UseTools) > 0 {
				// 1. user input 不继续追加
				// 2. 没有前继节点，设置当前
				// 3. 有前继节点，merge data
				if matches[1] == USER_INPUT {
					if response != nil {
						response.Datetime = datetime
						response.Thinking = thinking
						list = append(list, response)
						response, thinking = nil, ""
					}
					list = append(list, r)
				} else if response == nil {
					response = r
				} else {
					response.ErrMsg = r.ErrMsg
					response.UseTools = append(
						response.UseTools, r.UseTools...,
					)
				}
			}
		}
	}

	// 补充最后一个
	if response != nil {
		response.Thinking = thinking
		list = append(list, response)
	}
	return list
}

func Parse(data string) *SuperAction {
	var msg = &SuperAction{
		Origin: data, UseTools: []any{},
	}

	// 原格式（过于宽松，易误判普通文本为标签）：
	// var tagreg = `(?s)<([^/>]+)>`
	// 说明：仅匹配合法标签名（字母开头，允许数字与连字符）以避免误判。
	var tagreg = `(?s)<([a-zA-Z][a-zA-Z0-9-]*)>`
	re := regexp.MustCompile(tagreg)

	attamp, max, left, curr := 0, 10, data, ""
	for ; left != "" && attamp < max; attamp++ {
		matches := re.FindStringSubmatch(left)
		if len(matches) == 0 && strings.TrimSpace(left) != "" {
			text := strings.TrimSuffix(left, "```xml")
			text = strings.TrimPrefix(text, "```")
			msg.UseTools = append(msg.UseTools, &ToolResult{
				Content: strings.TrimSpace(text),
			})
			break
		}

		text, _, _ := strings.Cut(left, matches[0])
		if text = strings.TrimSpace(text); text != "" {
			left = strings.Replace(left, text, "", 1)
			text = strings.TrimSuffix(text, "```xml")
			text = strings.TrimPrefix(text, "```")
			msg.UseTools = append(msg.UseTools, &ToolResult{
				Content: strings.TrimSpace(text),
			})
		}

		// trim html comment
		re := regexp.MustCompile(`<!--\s*(.*?)\s*-->`)
		comment := re.FindStringSubmatch(matches[0])
		if len(comment) > 1 {
			log.Println("[PARSE] comment", comment[1])
			left = strings.TrimLeft(left, matches[0])
		}

		left, curr = snap(left, matches[1])
		if curr == "" {
			continue
		}
		res := parse(curr)
		switch act := res.(type) {
		case nil:
			continue
		case string:
			log.Println("[PARSE] comment", act)
		case *Context:
			msg.Context = act
		case *Thinking:
			msg.Thinking = act.Content
		case error:
			msg.ErrMsg = act
			return msg
		default:
			msg.UseTools = append(msg.UseTools, act)
		}
	}
	return msg
}

func (act *SuperAction) Hash() []string {
	result := []string{}
	for idx, act := range act.UseTools {
		var identifier string
		switch act := act.(type) {
		case *Memorize:
		case *Annotate:
		case *UserInput:
		case *ToolResult:
		case *Complete:
			identifier = act.XMLName.Local + ":" + act.Content
		case *MakeAsk:
			identifier = act.XMLName.Local + ":" + act.Question
		case *WaitTodo:
			identifier = act.XMLName.Local + ":" + act.UUID
		case *PathListFiles:
			identifier = act.XMLName.Local + ":" + act.Path
		case *FileGetContent:
			identifier = act.XMLName.Local + ":" + act.Path
		case *FilePutContent:
			identifier = act.XMLName.Local + ":" + act.Path
		case *FileReplaceText:
			identifier = act.XMLName.Local + ":" + act.Path
		case *ExecuteCommand:
			identifier = act.XMLName.Local + ":" + act.Command
		case *StartAsyncCmd:
			identifier = act.XMLName.Local + ":" + act.Session
		case *QueryAsyncCmd:
			identifier = act.XMLName.Local + ":" + act.Session
		case *AbortAsyncCmd:
			identifier = act.XMLName.Local + ":" + act.Session
		// Subtask
		case *StartSubtask:
			identifier = act.XMLName.Local + ":" + act.SubAgent
		case *QuerySubtask:
			identifier = act.XMLName.Local + ":" + act.SubAgent
		case *AbortSubtask:
			identifier = act.XMLName.Local + ":" + act.SubAgent
			// MCP工具
		case *UseMcpTool:
			identifier = act.XMLName.Local + ":" + act.Desc + ":" + act.Tool
			identifier += act.Desc + ":" + cryptor.Sha1(act.Args)
		case *UseBuiltinTool:
			identifier = act.XMLName.Local + ":" + act.Desc + ":" + act.Tool
			identifier += act.Desc + ":" + cryptor.Sha1(act.Args)
		default:
			log.Println("[PARSE] undefined action to hash", act)
			result = append(result, fmt.Sprint("tool", idx))
			continue
		}
		result = append(result, cryptor.Sha1(identifier))
	}
	return result
}

func (msg *SuperAction) Merge(act *SuperAction) {
	resHashMap := act.Hash()
	result := map[string]any{}
	for idx, item := range act.UseTools {
		hash := resHashMap[idx]
		switch res := item.(type) {
		case *ExecuteCommand:
			result[hash] = res.Result
		case *StartAsyncCmd:
			result[hash] = res.Result
		case *QueryAsyncCmd:
			result[hash] = res.Result
		case *AbortAsyncCmd:
			result[hash] = res.Result
		case *PathListFiles:
			result[hash] = res.Result
		case *FileGetContent:
			result[hash] = res.Result
		case *FilePutContent:
			result[hash] = res.Result
		case *FileReplaceText:
			result[hash] = res.Result
		// Subtask
		case *StartSubtask:
			result[hash] = res.Result
		case *QuerySubtask:
			result[hash] = res.Result
		case *AbortSubtask:
			result[hash] = res.Result
		// MCP工具
		case *UseMcpTool:
			result[hash] = res.Result
		// MCP工具
		case *UseBuiltinTool:
			result[hash] = res.Result
		}
	}

	msgHashMap := msg.Hash()
	for idx, item := range msg.UseTools {
		hash := msgHashMap[idx]
		switch res := item.(type) {
		case *ExecuteCommand:
			res.Result, _ = result[hash]
		case *StartAsyncCmd:
			res.Result, _ = result[hash]
		case *QueryAsyncCmd:
			res.Result, _ = result[hash]
		case *AbortAsyncCmd:
			res.Result, _ = result[hash]
		case *PathListFiles:
			res.Result, _ = result[hash]
		case *FileGetContent:
			res.Result, _ = result[hash]
		case *FilePutContent:
			res.Result, _ = result[hash]
		case *FileReplaceText:
			res.Result, _ = result[hash]
		// 自研Bot工具
		case *StartSubtask:
			res.Result, _ = result[hash]
		case *QuerySubtask:
			res.Result, _ = result[hash]
		case *AbortSubtask:
			res.Result, _ = result[hash]
		// MCP工具
		case *UseMcpTool:
			res.Result, _ = result[hash]
		// 内置工具
		case *UseBuiltinTool:
			res.Result, _ = result[hash]
		}
	}
}

func (act *SuperAction) ToMap() map[string]any {
	result := map[string]any{
		"theMsgId": act.TheMsgId,
		"workerId": act.WorkerID,
		"thinking": act.Thinking,
		"datetime": act.Datetime,
	}
	if act.Context != nil {
		result["context"] = support.ToMap(act.Context)
	}
	if act.ErrMsg != nil {
		result["errmsg"] = act.ErrMsg.Error()
	}
	actions, hash := []any{}, act.Hash()
	for idx, tool := range act.UseTools {
		if act, _ := tool.(IAct); act != nil {
			value := support.ToMap(act)
			value["hash"] = hash[idx]
			actions = append(actions, value)
			continue
		}
		value := support.ToMap(tool)
		actions = append(actions, value)
	}

	result["actions"] = actions
	return result
}

// MarshalJSON implements custom JSON serialization for SuperAction
// This ensures consistent API output format when using json.Marshal or json.Encoder
func (act *SuperAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(act.ToMap())
}

func snap(text string, tag string) (string, string) {
	// 注意：标签名需进行正则转义。原先直接插入会在包含特殊字符时触发 panic：
	// fmt.Sprintf(`(?s)<%s>(.*?)(</%s>|$)`, tag, tag)
	qtag := regexp.QuoteMeta(tag)
	regex := fmt.Sprintf(`(?s)<%s>(.*?)(</%s>|$)`, qtag, qtag)
	result := regexp.MustCompile(regex).FindStringSubmatch(text)
	if len(result) == 0 {
		return text, ""
	}

	text = strings.Replace(text, result[0], "", 1)
	return strings.TrimSpace(text), strings.TrimSpace(result[0])
}

// parse single xmlString
func parse(text string) any {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	// 原格式（过于宽松，易误判普通文本为标签）：
	// var tagreg = `(?s)<([^/>]+)>`
	var tagreg = `(?s)<([a-zA-Z][a-zA-Z0-9-]*)>`
	re := regexp.MustCompile(tagreg)
	matches := re.FindStringSubmatch(text)
	if len(matches) == 0 || matches[1] == "" {
		return fmt.Errorf("unexpected data")
	}
	var detail any
	switch matches[1] {
	case USER_INPUT:
		detail = new(UserInput)
	case THINKING, "think":
		detail = new(Thinking)
	case MAKE_ASK:
		detail = new(MakeAsk)
	case COMPLETE:
		detail = new(Complete)
	case MEMORIZE:
		detail = new(Memorize)
	case ANNOTATE:
		detail = new(Annotate)
	case WAITTODO:
		detail = new(WaitTodo)
	case EXECUTE_COMMAND:
		detail = new(ExecuteCommand)
	case START_ASYNC_CMD:
		detail = new(StartAsyncCmd)
	case QUERY_ASYNC_CMD:
		detail = new(QueryAsyncCmd)
	case ABORT_ASYNC_CMD:
		detail = new(AbortAsyncCmd)

	// file system action
	case PATH_LIST_FILES:
		detail = new(PathListFiles)
	case FILE_GET_CONTENT:
		detail = new(FileGetContent)
	case FILE_PUT_CONTENT:
		detail = new(FilePutContent)
	case FILE_REPLACE_TEXT:
		detail = new(FileReplaceText)
	// 自研Bot工具
	case START_SUBTASK:
		detail = new(StartSubtask)
	case QUERY_SUBTASK:
		detail = new(QuerySubtask)
	case ABORT_SUBTASK:
		detail = new(AbortSubtask)
	// MCP工具
	case USE_MCP_TOOL:
		detail = new(UseMcpTool)
	case USE_BUILTIN_TOOL:
		detail = new(UseBuiltinTool)
	default:
		return fmt.Errorf("%w: %s", errors.ErrUnexpectedTool, matches[1])
	}

	if err := parseAction(text, detail); err != nil {
		return err
	}

	switch act := detail.(type) {
	case *Thinking:
		act.Content = strings.TrimSpace(act.Content)
	case *Memorize:
		act.Content = strings.TrimSpace(act.Content)
	case *Annotate:
		act.Context = support.TrimIndent(act.Context)
	case *Complete:
		act.Content = support.EscapeXml(act.Content)
		act.Content = support.TrimIndent(act.Content)
	case *WaitTodo:
		act.Todo = support.EscapeXml(act.Todo)
		act.Todo = support.TrimIndent(act.Todo)
	case *FileReplaceText:
		act.Diff = support.EscapeXml(act.Diff)
		act.Diff = support.TrimIndent(act.Diff)
	case *FilePutContent:
		act.Data = support.EscapeXml(act.Data)
		act.Data = support.TrimIndent(act.Data)
	case *ExecuteCommand:
		act.Command = support.EscapeXml(act.Command)
		act.Command = support.TrimIndent(act.Command)
	case *StartAsyncCmd:
		act.Command = support.EscapeXml(act.Command)
		act.Command = support.TrimIndent(act.Command)
	}
	return detail
}

func parseAction(text string, target any) error {
	targetValue := reflect.ValueOf(target)
	targetElem := targetValue.Elem()
	targetType := targetElem.Type()

	content := strings.TrimSpace(text)
	for i := range targetType.NumField() {
		fieldMeta := targetType.Field(i)
		fieldData := targetElem.Field(i)

		tag := fieldMeta.Tag.Get("xml")
		if tag == "" || tag == "-" {
			continue
		}

		tagName := strings.Split(tag, "/")[0]
		if strings.Contains(tagName, ">") {
			_, tagName, _ = strings.Cut(tagName, ">")
		}
		if fieldMeta.Name == "XMLName" {
			for tagName := range strings.SplitSeq(tag, "/") {
				fieldData.Set(reflect.ValueOf(xml.Name{Local: tagName}))
				// 原做法：直接拼接标签名到正则，若标签名包含特殊字符会导致 panic。
				// 修正：使用 QuoteMeta 对标签名进行正则转义。
				q := regexp.QuoteMeta(tagName)
				tagreg := fmt.Sprintf(`(?s)<%s>(.*?)(</%s>|$)`, q, q)
				matches := regexp.MustCompile(tagreg).FindStringSubmatch(content)
				if len(matches) > 1 {
					content = matches[1]
				}
			}
		}

		switch fieldData.Kind() {
		case reflect.String:
			// 为避免特殊字符引发的正则编译错误，这里统一进行转义处理。
			q := regexp.QuoteMeta(tagName)
			tagreg := fmt.Sprintf(`(?s)<%s>(.*?)(</%s>|$)`, q, q)
			matches := regexp.MustCompile(tagreg).FindStringSubmatch(content)
			if len(matches) > 1 {
				fieldData.SetString(matches[1])
			} else if tagName == "content" {
				fieldData.SetString(content)
			}
		case reflect.Interface:
			// 为避免特殊字符引发的正则编译错误，这里统一进行转义处理。
			q := regexp.QuoteMeta(tagName)
			tagreg := fmt.Sprintf(`(?s)<%s>(.*?)(</%s>|$)`, q, q)
			matches := regexp.MustCompile(tagreg).FindStringSubmatch(content)
			if len(matches) > 1 {
				fieldData.Set(reflect.ValueOf(matches[1]))
			}
		case reflect.Slice:
			// 为避免特殊字符引发的正则编译错误，这里统一进行转义处理。
			q := regexp.QuoteMeta(tagName)
			tagreg := fmt.Sprintf("(?s)<%s>(.*?)</%s>", q, q)
			matches := regexp.MustCompile(tagreg).FindAllStringSubmatch(content, -1)
			if len(matches) > 0 {
				options := make([]string, 0, len(matches))
				for _, match := range matches {
					options = append(options, match[1])
				}
				fieldData.Set(reflect.ValueOf(options))
			}
		}
	}
	return nil
}
