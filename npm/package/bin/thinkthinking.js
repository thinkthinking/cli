#!/usr/bin/env node
// Node wrapper：把所有参数原样转发给已下载的 native 二进制，
// 保留 stdout/stderr 与 exit code。Node 仅作转发壳，不参与业务逻辑。

const path = require("path");
const fs = require("fs");
const { spawnSync } = require("child_process");

const BINARY_NAME = "thinkthinking";
const isWindows = process.platform === "win32";
const binaryFile = isWindows ? `${BINARY_NAME}.exe` : BINARY_NAME;
const binaryPath = path.join(__dirname, "native", binaryFile);

if (!fs.existsSync(binaryPath)) {
  console.error(
    `[thinkthinking] native binary not found at ${binaryPath}\n` +
      `请重新安装（npm install -g @thinkthinking/cli），或检查 postinstall 下载是否失败。`
  );
  process.exit(1);
}

// 透传 stdio，让 JSON 走 stdout、日志走 stderr 的约定在 wrapper 层也成立。
const result = spawnSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error(`[thinkthinking] failed to launch binary: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
