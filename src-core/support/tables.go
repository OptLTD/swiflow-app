package support

import (
	"fmt"
	"strings"
)

func MdTable(rows []map[string]any, head []string) string {
	var markdown strings.Builder

	// 写入表头行
	markdown.WriteString("|")
	for _, col := range head {
		markdown.WriteString(" ")
		markdown.WriteString(escapeMarkdown(col))
		markdown.WriteString(" |")
	}
	markdown.WriteString("\n")

	// 写入分隔行
	markdown.WriteString("|")
	for range head {
		markdown.WriteString(" --- |")
	}
	markdown.WriteString("\n")

	// 写入数据行
	for _, row := range rows {
		markdown.WriteString("|")
		for _, col := range head {
			markdown.WriteString(" ")
			if val, ok := row[col]; ok {
				switch v := val.(type) {
				case string:
					markdown.WriteString(escapeMarkdown(v))
				case nil:
					markdown.WriteString("NULL")
				default:
					markdown.WriteString(escapeMarkdown(fmt.Sprintf("%v", v)))
				}
			} else {
				markdown.WriteString("NULL")
			}
			markdown.WriteString(" |")
		}
		markdown.WriteString("\n")
	}

	return markdown.String()
}

func CsvTable(rows []map[string]any, head []string) string {
	var csv strings.Builder

	// 写入表头
	for i, col := range head {
		if i > 0 {
			csv.WriteString(",")
		}
		csv.WriteString(escapeCSV(col))
	}
	csv.WriteString("\n")

	// 写入数据行
	for _, row := range rows {
		for i, col := range head {
			if i > 0 {
				csv.WriteString(",")
			}
			if val, ok := row[col]; ok {
				switch v := val.(type) {
				case string:
					csv.WriteString(escapeCSV(v))
				case nil:
					csv.WriteString("NULL")
				default:
					csv.WriteString(escapeCSV(fmt.Sprintf("%v", v)))
				}
			} else {
				csv.WriteString("NULL")
			}
		}
		csv.WriteString("\n")
	}

	return csv.String()
}

// escapeMarkdown 转义Markdown中的特殊字符
func escapeMarkdown(s string) string {
	// 在Markdown表格中，竖线需要转义
	s = strings.ReplaceAll(s, "|", "\\|")
	// 其他可能需要转义的Markdown特殊字符
	specialChars := []string{"*", "_", "#", "`", ">", "[", "]", "(", ")", "+", "-", ".", "!"}
	for _, char := range specialChars {
		s = strings.ReplaceAll(s, char, "\\"+char)
	}
	return s
}

// escapeCSV escapes special characters in CSV fields
func escapeCSV(s string) string {
	if strings.ContainsAny(s, ",\"\n") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}
