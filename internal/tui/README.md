# internal/tui — 未来 TUI 入口（预留）

第一期不实现 TUI。此目录为未来 [Bubble Tea](https://github.com/charmbracelet/bubbletea) 交互式终端界面预留。

## 设计约束

TUI 是**未来入口**，不是核心。所有业务能力都沉淀在 `internal/core`（转换、草稿、配置），TUI 只负责交互与展示，通过 `internal/app` 的 Container 复用同一套 service。

这样未来加 TUI 时无需重写业务逻辑，只需：

1. 在此包实现 Bubble Tea model
2. 从 `app.Container` 取 `Converter` / `DraftService`
3. 新增一个 `thinkthinking tui` 命令挂载到 root

CLI / TUI / GUI 三种入口共享同一个 core，是本项目的架构基线。
