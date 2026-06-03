#!/usr/bin/env node
// Node wrapper：定位随 optionalDependencies 安装的平台子包二进制，
// 把所有参数原样转发给它，保留 stdout/stderr 与 exit code。
// Node 仅作转发壳，不参与业务逻辑，运行时不依赖 Node 之外的下载。

const { spawnSync } = require("child_process");

// 平台/架构 → [子包名, 二进制文件名]。
// 键用 Node 的 `${process.platform}-${process.arch}`（win32 / x64）。
const BINARIES = {
  "darwin-arm64": ["@thinkthinking/cli-darwin-arm64", "thinkthinking"],
  "darwin-x64": ["@thinkthinking/cli-darwin-x64", "thinkthinking"],
  "linux-x64": ["@thinkthinking/cli-linux-x64", "thinkthinking"],
  "linux-arm64": ["@thinkthinking/cli-linux-arm64", "thinkthinking"],
  "win32-x64": ["@thinkthinking/cli-win32-x64", "thinkthinking.exe"],
};

function resolveBinary() {
  const key = `${process.platform}-${process.arch}`;
  const entry = BINARIES[key];
  if (!entry) {
    console.error(
      `[thinkthinking] 不支持的平台：${key}。` +
        `支持的平台：${Object.keys(BINARIES).join(", ")}。`
    );
    process.exit(1);
  }
  const [pkg, file] = entry;
  try {
    // 子包 package.json 不含 exports 字段，可直接解析包内二进制文件路径。
    return require.resolve(`${pkg}/${file}`);
  } catch (err) {
    console.error(
      `[thinkthinking] 找不到平台二进制包 ${pkg}（平台 ${key}）。\n` +
        `可能原因：安装时跳过了 optionalDependencies（如 --no-optional / --omit=optional / --ignore-scripts 无关），\n` +
        `或该平台子包未发布。请重新安装：npm install -g @thinkthinking/cli\n` +
        `原始错误：${err.message}`
    );
    process.exit(1);
  }
}

const binaryPath = resolveBinary();

// 透传 stdio，让 JSON 走 stdout、日志走 stderr 的约定在 wrapper 层也成立。
const result = spawnSync(binaryPath, process.argv.slice(2), { stdio: "inherit" });

if (result.error) {
  console.error(`[thinkthinking] failed to launch binary: ${result.error.message}`);
  process.exit(1);
}
process.exit(result.status === null ? 1 : result.status);
