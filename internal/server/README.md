# internal/server — 未来 Local API / MCP server 入口（预留）

第一期不实现。此目录为未来的本地 HTTP API 与 [MCP](https://modelcontextprotocol.io) server 预留。

## 设想用途

- **Local API**：暴露 `convert` / `draft` 等能力为 HTTP 接口，供 GUI 或其它进程调用。
- **MCP server**：把 thinkthinking 的能力封装为 MCP tools，让支持 MCP 的 Agent 客户端直接调用，无需 shell 出 CLI。

## 设计约束

与 TUI 同理：server 只是**传输层**，业务能力全部来自 `internal/core`，通过 `app.Container` 复用。新增 server 入口不应改动 core。
