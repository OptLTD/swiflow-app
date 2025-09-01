# 文件类型限制功能

## 概述

为了增强安全性，文件系统操作现在支持文件类型限制。只有特定扩展名的文件才能被读取、写入或修改。

## 支持的文件类型

以下文件扩展名被允许：

### 文本文件
- `.txt` - 纯文本文件
- `.md` - Markdown 文档
- `.rst` - reStructuredText 文档

### 编程语言文件
- `.py` - Python 脚本
- `.js` - JavaScript 文件
- `.ts` - TypeScript 文件
- `.sh` - Shell 脚本
- `.bat` - Windows 批处理文件
- `.ps1` - PowerShell 脚本

### 配置文件
- `.json` - JSON 配置文件
- `.yaml` / `.yml` - YAML 配置文件
- `.toml` - TOML 配置文件
- `.ini` - INI 配置文件
- `.cfg` / `.conf` - 通用配置文件
- `.env` - 环境变量文件
- `.gitignore` - Git 忽略文件

### 数据文件
- `.csv` - 逗号分隔值文件
- `.xml` - XML 文件
- `.sql` - SQL 脚本文件

### 网页文件
- `.html` - HTML 文件
- `.css` - CSS 样式文件

### 其他
- `.dockerfile` - Docker 文件
- `.log` - 日志文件
- `.lock` - 锁定文件

## 实现细节

### 验证函数

```go
func (m *FileSystemAbility) isAllowedFileType() error
```

这个函数检查文件扩展名是否在允许列表中。如果文件没有扩展名或扩展名不在允许列表中，将返回错误。

### 受影响的操作

以下文件系统操作现在会进行文件类型验证：

1. **读取文件** (`Read()`)
2. **写入文件** (`Write()`)
3. **复制文件** (`Copy()`)
4. **重命名文件** (`Rename()`)
5. **替换文件内容** (`Replace()`)

### 错误信息

当尝试操作不支持的文件类型时，会返回详细的错误信息，包括：

- 不支持的文件扩展名
- 所有允许的文件类型列表

示例错误信息：
```
不支持的文件类型: .exe。允许的类型包括: .bat, .cfg, .conf, .css, .csv, .dockerfile, .env, .gitignore, .html, .ini, .js, .json, .lock, .log, .md, .ps1, .py, .rst, .sh, .sql, .toml, .ts, .txt, .xml, .yaml, .yml
```

## 安全考虑

这个限制功能提供了以下安全优势：

1. **防止恶意文件执行** - 阻止 `.exe`、`.dll` 等可执行文件
2. **限制文件类型** - 只允许安全的文本和配置文件类型
3. **清晰的错误信息** - 帮助用户了解允许的文件类型

## 测试

运行以下命令来测试文件类型限制功能：

```bash
# 运行所有文件类型相关测试
go test -v ./ability -run "TestFileTypeRestriction|TestWriteWithFileTypeRestriction|TestReadWithFileTypeRestriction"

# 运行所有测试
go test -v ./ability
```

## 扩展

如果需要添加新的文件类型支持，只需在 `allowedExtensions` 映射中添加新的扩展名：

```go
var allowedExtensions = map[string]bool{
    // ... 现有扩展名 ...
    ".newtype": true,  // 添加新的文件类型
}
``` 