package support

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/google/jsonschema-go/jsonschema"
	nanoid "github.com/matoous/go-nanoid/v2"
	"golang.org/x/net/html"
)

func UniqueID(args ...int) (string, error) {
	size := 12
	switch {
	case len(args) == 1 && args[0] > 0:
		size = args[0]
	}
	return nanoid.New(size)
}

func Substring(s string, length int) string {
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length])
	}
	return s
}

func EscapeXml(escaped string) string {
	return html.UnescapeString(escaped)
	// replacer := strings.NewReplacer(
	// 	"&lt;", "<",
	// 	"&gt;", ">",
	// 	"&amp;", "&",
	// 	"&quot;", "\"",
	// 	"&apos;", "'",
	// )
	// return replacer.Replace(escaped)
}

func TrimIndent(content string) string {
	lines := strings.Split(content, "\n")
	minIndent := int(^uint(0) >> 1)
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue // 忽略空行
		}
		ltrim := strings.TrimLeft(line, " \t")
		indent := len(line) - len(ltrim)
		if indent < minIndent {
			minIndent = indent
		}
	}

	// 删除每行的最小缩进
	for i, line := range lines {
		if len(line) >= minIndent {
			lines[i] = line[minIndent:]
		}
	}
	return strings.TrimSpace(
		strings.Join(lines, "\n"),
	)
}

func IsHTML(text string) bool {
	re := regexp.MustCompile(`<[^>]+>`)
	if !re.MatchString(text) {
		return false
	}

	_, err := html.Parse(strings.NewReader(text))
	return err == nil
}

func ToHTML(text string) string {
	parts := strings.Split(text, "\n")
	result := []string{}
	for _, part := range parts {
		result = append(result, fmt.Sprintf("<div dir=\"ltr\">%s<br></div>", part))
	}
	return strings.Join(result, "\n")
}

func Quote(text string) string {
	if IsHTML(text) {
		return HtmlQuote(text)
	}
	return TextQuote(text, "> ")
}

func HtmlQuote(text string) string {
	style := strings.Join([]string{
		"border-left: 5px solid #999;",
		"margin: 1em 0; padding: 1em;",
		"background-color: #f9f9f9;",
	}, "")
	return fmt.Sprintf("<blockquote style=\"%s\">%s</blockquote>", style, text)
}

func TextQuote(text string, s string) string {
	parts := strings.Split(text, "\n")
	result := []string{}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			result = append(result, part)
			continue
		}
		result = append(result, s+part)
	}
	return strings.Join(result, "\n")
}

func Capitalize(str string) string {
	var final strings.Builder
	words := strings.Split(str, "-")
	for _, word := range words {
		word = strutil.Capitalize(word)
		final.WriteString(word + " ")
	}
	return strings.TrimSpace(final.String())
}

func ToJson(data any) string {
	result, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return string(result)
}

func ToMap(action any) map[string]any {
	var r = map[string]any{}

	targetValue := reflect.ValueOf(action)
	targetType := targetValue.Elem().Type()
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if field.Name == "XMLName" {
			r["type"] = field.Tag.Get("xml")
			continue
		}
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			tag = field.Tag.Get("xml")
		}
		if tag == "" || tag == "-" {
			continue
		}
		if targetValue.Elem().Field(i).IsZero() {
			continue
		}

		r[tag] = targetValue.Elem().Field(i).Interface()
		if err, ok := r[tag].(error); ok {
			r[tag] = err.Error()
		}
	}
	return r
}

func ToXML(action any, result any) string {
	var s strings.Builder

	var rootName string
	targetValue := reflect.ValueOf(action)
	targetType := targetValue.Elem().Type()
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if field.Name == "XMLName" {
			rootName = field.Tag.Get("xml")
			break
		}
	}

	s.WriteString(fmt.Sprintf("<%s>\n", rootName))
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if field.Name == "XMLName" {
			continue
		}
		tag := field.Tag.Get("xml")
		if tag == "" || tag == "-" {
			continue
		}
		if targetValue.Elem().Field(i).IsZero() {
			continue
		}
		value := fmt.Sprintf("%v", targetValue.Elem().Field(i).Interface())
		s.WriteString(fmt.Sprintf("  <%s>%s</%s>\n", tag, value, tag))
	}
	if Bool(result) {
		tag := "result"
		s.WriteString(fmt.Sprintf("  <%s>%s</%s>\n", tag, result, tag))
	}
	s.WriteString(fmt.Sprintf("</%s>", rootName))
	return s.String()
}

func MaskMiddle(s string) string {
	if s == "" {
		return ""
	}
	n := len(s)
	if n <= 6 {
		return strings.Repeat("*", n)
	}
	head, tail := 4, 4
	return s[:head] + strings.Repeat("*", n-head-tail) + s[n-tail:]
}

// IsNewVer 版本比较函数：比较两个版本号，返回v1是否高于v2
func IsNewVer(v1, v2 string) bool {
	// 移除版本号前缀"v"
	v1 = strings.TrimPrefix(v1, "v")
	v2 = strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var num1, num2 int
		if i < len(parts1) {
			num1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			num2, _ = strconv.Atoi(parts2[i])
		}

		if num1 > num2 {
			return true
		}
		if num1 < num2 {
			return false
		}
	}
	return false // 版本相同或v1小于v2
}

// MapToSchema 将 map[string]any 转为 *jsonschema.Schema
func MapToSchema(m map[string]any) (*jsonschema.Schema, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	schema := new(jsonschema.Schema)
	if err := schema.UnmarshalJSON(b); err != nil {
		return nil, err
	}
	return schema, nil
}
