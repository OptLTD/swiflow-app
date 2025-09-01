# Swiflow

Swiflow 是一个 Desktop AI 助手，你可以通过自然对话表达需求，Swiflow 会自动制定计划、使用工具来完成任务。你可以让 Swiflow 记住你的喜好、处理问题的方式方法、常见问题的解决方案、行业内的知识信息等等，最终与你道同契合。

## 名字由来

一次无意中想到【花自飘零水自流】，忽略这句诗的上下文，觉得 AI 加持下的工作流程如果也这般轻松惬意就好了，于是把这句诗丢给 Deepseek 给推荐了几个名字。Swiflow是比较中意的一个，读音上也偏向【水自流】，拆开看的话分别是【Swift】+【Flow】，期望大家在处理工作的时候也能轻快流畅。

## 功能特性

### 支持多 Agent

在 Swiflow 中，Agent 对应的概念是Bot，可以为不同的 Bot 设置不同的技能和任务目标，在使用过程中手动切换对应的 Bot 来执行具体任务。

### 自定义工作流

大多数 Agent 的提示词中国都包含 Bot 使命、工作流程、自身流程、可用工具、最佳实践等等部分，Swiflow 把可用工具和自身流程部分提示词与业务相关的Bot使命、工作流程、最佳实践提示词作了切分，任务相关这部分提示词可以在 Bot 设置里自行维护、不断调整以达到最佳效果。

### 支持定时任务

在对话中说"每半个小时帮我检查一下最新邮件"等类似的诉求，Swiflow 即可自动添加定时任务，并到期自动唤醒自己根据设定的目标自驱工作。

### 支持 Memory

在对话中说'请记住刚刚我们讨论的内容，这很有用'，或者在提示词中设定固定的触发逻辑如'当你遇到错误并解决后请记下问题的特点与解决的方式'你也可以记忆管理里手动维护记忆的内容。

### 支持 MCP 协议

在 MCP 管理中添加自己需要的 Mcp Server, 并为 Bot 勾选对应的工具即可为 Bot 开启对应的能力。

## 和其他 Agent 的区别

### 和 Cursor、Copilot、Windsurf等AI IDE的区别？

AI IDE：为专业程序员设计的产品，需要用户有Dev、Debug的能力
Swiflow：为普通人设计产品，无需用户有编程相关知识

> *貌似也有点门槛*, MCP 对使用者环境配置有一定的要求

### 和 V0、Lovable、Bolt 等 App Builder 的区别？

App Builder：多数是 Web App 以展示型功能见长
Swiflow：以 Python 脚本为主，更擅长处理本地文件和数据

### 和 Manus、Coze Space 等 AI Agent的区别？

AI Agent：能力更全面、更通用，解决的是更通用的问题
Swiflow：更 Privacy 更私人订制，更专注解决日常工作的问题

## 项目结构

```
swiflow-app/
├── src-core/          # Go 后端核心代码
├── src-front/         # Vue 前端代码
├── src-tauri/         # Tauri 桌面应用框架
├── src-docs/          # 项目文档
├── public/            # 静态资源文件
└── build-*.sh         # 构建脚本
```

## 开发环境

### 前端开发

```bash
# 安装依赖
yarn install

# 启动开发服务器
yarn dev
```

### 后端开发

```bash
# 进入后端目录
cd src-core

# 运行后端服务
go run . -m serve
```

### 桌面应用开发

```bash
# 启动 Tauri 开发模式
yarn tauri dev
```

## 构建部署

### macOS 构建

```bash
./build-mac.sh
```

### Windows 构建

```bash
# 近构建后端二进制
./build-win.sh
```
> macOs下构建windows安装包
> 参考：[PACKAGE.md](src-docs/PACKAGE.md)



### Docker 部署

```bash
docker-compose up -d
```

## 致谢

我们要感谢以下项目：
- Cline Autonomous coding agent right in your IDE
- Suna Open Source Generalist AI Agent

## 许可证

本项目采用双重许可证：
- 主许可证：见 [LICENSE](LICENSE) 文件
- 附加条款：见 [LICENSE-ADDENDUM.md](LICENSE-ADDENDUM.md) 文件

## 更多信息

- 官方文档：https://swiflow.cc/docs/
- 项目主页：https://swiflow.cc/

---

愿您的工作在 AI 加持下如【花自飘零水自流】般轻松惬意 🌸