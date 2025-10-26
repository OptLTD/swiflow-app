package builtin

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"swiflow/ability"
	"swiflow/config"
	"swiflow/model"
	"time"
)

type Py3AliasTool struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Desc string `json:"desc"`
	Args string `json:"args"`

	Code string `json:"code"`
	Deps string `json:"deps"`

	client model.LLMClient `json:"-"`
}

func (a *Py3AliasTool) SetClient(c model.LLMClient) { a.client = c }

func (a *Py3AliasTool) Prompt() string {
	var prompt strings.Builder
	prompt.WriteString(fmt.Sprintf("### **%s**\n", a.UUID))
	prompt.WriteString(fmt.Sprintf("- 描述：%s\n", a.Desc))
	prompt.WriteString(fmt.Sprintf("- 入参：%s\n", a.Args))
	prompt.WriteString("- 示例：\n")
	prompt.WriteString("```xml\n")
	prompt.WriteString("<use-builtin-tool>\n")
	prompt.WriteString(fmt.Sprintf("<desc>%s</desc>\n", a.Name))
	prompt.WriteString(fmt.Sprintf("<tool>%s</tool>\n", a.UUID))
	prompt.WriteString("<args>\n")
	prompt.WriteString("--help\n")
	prompt.WriteString("</args>\n")
	prompt.WriteString("</use-builtin-tool>\n")
	prompt.WriteString("```\n\n")
	return prompt.String()
}

// Handle executes the alias:
// writes code to a temp file,
// installs deps, then runs python3.
func (a *Py3AliasTool) Handle(args string) (string, error) {
	if code := strings.TrimSpace(a.Code); code == "" {
		return "", fmt.Errorf("no python code provided")
	}

	// Determine working directory
	home := config.CurrentHome()
	if strings.TrimSpace(home) == "" {
		home = config.GetWorkHome()
	}
	cmd := &ability.DevCommandAbility{Home: home}

	// Write code to a default file
	filename := fmt.Sprintf("alias-%s.py", a.UUID)
	fullpath := filepath.Join(home, filename)
	if err := os.MkdirAll(filepath.Dir(fullpath), 0755); err != nil {
		return "", fmt.Errorf("prepare dir: %v", err)
	}

	if err := os.WriteFile(fullpath, []byte(a.Code), 0644); err != nil {
		return "", fmt.Errorf("write file: %v", err)
	}
	defer os.Remove(fullpath)

	// Resolve deps: prefer a.Deps -> manager store -> static guess
	deps := strings.TrimSpace(a.Deps)
	if deps == "" {
		deps = guessDeps(a.Code)
		deps = normalizeDeps(deps)
	}

	timeout := 30 * time.Second
	if deps != "" {
		for _, pkg := range strings.Fields(deps) {
			if pkg == "" {
				continue
			}
			args := []string{"-m", "pip", "install", pkg}
			_, _ = cmd.Run("python3", timeout, args...)
		}
	}
	return cmd.Run("python3", timeout, fullpath)
}

// Analyze inspects Python code to determine deps and args using LLM;
func (a *Py3AliasTool) Analyze() (deps string, args string) {
	code := strings.TrimSpace(a.Code)
	if code == "" || a.client == nil {
		return "", ""
	}

	// Prefer LLM analysis when client is available
	messages := []model.Message{
		{Role: "system", Content: parsePrompt()},
		{Role: "user", Content: code},
	}
	choices, err := a.client.Respond("py3-alias", messages)
	if err == nil && len(choices) > 0 {
		text := strings.TrimSpace(choices[0].Message.Content)
		xmlstr := extractXML(text, "py3alias")
		if xmlstr != "" {
			var meta py3AliasXML
			if e := xml.Unmarshal([]byte(xmlstr), &meta); e == nil {
				return normalizeDeps(meta.Deps), sanitizeArgs(meta.Args)
			}
		}
	}

	// Fallback to static extraction
	return guessDeps(code), guessArgs(code)
}

type py3AliasXML struct {
	XMLName xml.Name `xml:"py3alias"`
	Deps    string   `xml:"deps"`
	Args    string   `xml:"args"`
}

func parsePrompt() string {
	var b strings.Builder
	b.WriteString("你需要分析给定的 Python 代码，提取运行所需的第三方依赖和参数用法提示。\n")
	b.WriteString("- deps: 仅第三方 pip 包名称，使用空格分隔，不包含标准库。\n")
	b.WriteString("- args: 列出所有参数或标志，并为每个参数给出简短中文意义；形式为“name: 作用说明”或“--flag: 作用说明”。不要使用尖括号、方括号或任何额外标记。\n")
	b.WriteString("输出必须是如下结构的 XML：\n")
	b.WriteString("<py3alias>\n  <deps>typer</deps>\n  <args>name: 要问候的名字</args>\n</py3alias>\n")
	return b.String()
}

func extractXML(text, root string) string {
	start := "<" + root + ">"
	end := "</" + root + ">"
	i := strings.Index(text, start)
	j := strings.LastIndex(text, end)
	if i >= 0 && j >= 0 && j+len(end) > i {
		return text[i : j+len(end)]
	}
	return ""
}

// guessDeps extracts import names and filters out common stdlib modules.
func guessDeps(code string) string {
	reImport := regexp.MustCompile(`(?m)^\s*import\s+([a-zA-Z0-9_\.]+)(?:\s+as\s+[a-zA-Z0-9_]+)?`)
	reFrom := regexp.MustCompile(`(?m)^\s*from\s+([a-zA-Z0-9_\.]+)\s+import\s+`)
	mods := map[string]struct{}{}
	add := func(m string) {
		m = strings.TrimSpace(m)
		m = strings.Split(m, ".")[0]
		if m != "" {
			mods[m] = struct{}{}
		}
	}
	for _, m := range reImport.FindAllStringSubmatch(code, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range reFrom.FindAllStringSubmatch(code, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	stdlib := map[string]struct{}{
		"os": {}, "sys": {}, "re": {}, "json": {}, "time": {}, "datetime": {},
		"math": {}, "random": {}, "typing": {}, "pathlib": {}, "logging": {},
		"shutil": {}, "subprocess": {}, "functools": {}, "itertools": {}, "collections": {},
		"dataclasses": {}, "urllib": {}, "http": {}, "hashlib": {}, "base64": {},
		"threading": {}, "multiprocessing": {}, "decimal": {}, "fractions": {}, "statistics": {},
	}
	list := []string{}
	for m := range mods {
		if _, ok := stdlib[m]; ok {
			continue
		}
		list = append(list, m)
	}
	sort.Strings(list)
	return strings.Join(list, " ")
}

// guessArgs provides a minimal usage hint based on argparse/sys.argv patterns.
func guessArgs(code string) string {
	if strings.Contains(code, "argparse") {
		reArg := regexp.MustCompile(`(?m)add_argument\(\s*(['\"])((?:--)?[a-zA-Z0-9_\-]+)\1`)
		flags := map[string]struct{}{}
		for _, m := range reArg.FindAllStringSubmatch(code, -1) {
			if len(m) > 2 {
				flags[m[2]] = struct{}{}
			}
		}
		list := []string{}
		for f := range flags {
			list = append(list, f)
		}
		sort.Strings(list)
		if len(list) > 0 {
			return strings.Join(list, " ")
		}
		return "--help"
	}
	if strings.Contains(code, "sys.argv") {
		return "args"
	}
	return "--help"
}

// sanitizeArgs removes brackets, code fences and squeezes spaces.
func sanitizeArgs(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "`", "")
	s = regexp.MustCompile(`(?s)<\s*([a-zA-Z0-9_\-]+)\s*>`).ReplaceAllString(s, "$1")
	s = regexp.MustCompile(`\[\s*([^\]]+)\s*\]`).ReplaceAllString(s, "$1")
	s = strings.Join(strings.Fields(s), " ")
	return s
}

// normalizeDeps normalizes whitespace and commas in deps
func normalizeDeps(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", " ")
	s = strings.ReplaceAll(s, ";", " ")
	s = strings.Join(strings.Fields(s), " ")
	return s
}
