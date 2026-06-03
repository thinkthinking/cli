#!/usr/bin/env node
// 构建脚本：从 GitHub Release 下载各平台二进制归档，组装出 5 个平台子包到
// npm/packages/，并同步主包 npm/package/package.json 的 version 与 optionalDependencies。
//
// 设计要点：
//   - 数据源是 GitHub Release v${VERSION} 的归档（tar.gz / zip），与 .goreleaser.yaml
//     的 name_template `thinkthinking_{version}_{os}_{arch}` 对齐。
//     从 Release 下载（而非 goreleaser 的 dist/）使 tag push 与 workflow_dispatch 两条
//     触发路径逻辑统一、彼此解耦。
//   - 子包用 npm 的 os/cpu 字段限定平台；主包用 optionalDependencies 声明它们，
//     npm 安装时自动只装匹配当前系统的那一个，无需 postinstall、无运行时下载。
//   - 子包 package.json 不含 exports 字段，wrapper 才能用 require.resolve 解析包内二进制。
//
// 环境变量：
//   VERSION       要发布的版本号（去 v 前缀），必填。
//   REPO          GitHub 仓库（owner/name），默认 thinkthinking/cli。
//   GITHUB_TOKEN  可选，下载 Release 资产时防限流。
//
// 用法：
//   VERSION=0.1.0 node npm/scripts/build-packages.mjs

import fs from "node:fs";
import path from "node:path";
import os from "node:os";
import https from "node:https";
import { fileURLToPath } from "node:url";
import { execSync } from "node:child_process";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, "..", ".."); // cli/
const MAIN_PKG_DIR = path.join(REPO_ROOT, "npm", "package");
const PACKAGES_DIR = path.join(REPO_ROOT, "npm", "packages");
// 本地归档缓存目录（按版本分目录）。若归档已存在则跳过网络下载，
// 便于本机验证 / 离线 / GitHub 直连慢的场景。CI 上目录为空，正常走下载。
const CACHE_ROOT = path.join(REPO_ROOT, "npm", ".cache");

const BINARY_NAME = "thinkthinking";
const REPO = process.env.REPO || "thinkthinking/cli";

// 平台矩阵：GoReleaser 的 goos/goarch ↔ Node 的 platform/arch ↔ 子包名。
// node/cpu 用于子包 package.json 的 os/cpu 字段（必须是 Node 的命名）。
const PLATFORMS = [
  { goos: "darwin", goarch: "amd64", node: "darwin", cpu: "x64", ext: "tar.gz", exe: false },
  { goos: "darwin", goarch: "arm64", node: "darwin", cpu: "arm64", ext: "tar.gz", exe: false },
  { goos: "linux", goarch: "amd64", node: "linux", cpu: "x64", ext: "tar.gz", exe: false },
  { goos: "linux", goarch: "arm64", node: "linux", cpu: "arm64", ext: "tar.gz", exe: false },
  { goos: "windows", goarch: "amd64", node: "win32", cpu: "x64", ext: "zip", exe: true },
];

function getVersion() {
  const v = (process.env.VERSION || "").replace(/^v/, "").trim();
  if (!v) {
    console.error("[build-packages] 缺少环境变量 VERSION（如 VERSION=0.1.0）");
    process.exit(1);
  }
  return v;
}

// 子包名：@thinkthinking/cli-<node>-<cpu>，如 @thinkthinking/cli-darwin-arm64。
function subPackageName(p) {
  return `@thinkthinking/cli-${p.node}-${p.cpu}`;
}

// 跟随重定向的 HTTPS 下载。
function download(url, destPath, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) return reject(new Error("too many redirects"));
    const headers = { "User-Agent": "thinkthinking-build" };
    if (process.env.GITHUB_TOKEN) {
      headers.Authorization = `Bearer ${process.env.GITHUB_TOKEN}`;
    }
    https
      .get(url, { headers }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          // 重定向到 CDN 时不再带 Authorization（交给 redirect 的 location 自己的鉴权）。
          res.resume();
          return resolve(download(res.headers.location, destPath, redirects + 1));
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`download failed: HTTP ${res.statusCode} for ${url}`));
        }
        const file = fs.createWriteStream(destPath);
        res.pipe(file);
        file.on("finish", () => file.close(() => resolve()));
        file.on("error", reject);
      })
      .on("error", reject);
  });
}

// 解压归档到目标目录（用系统 tar / unzip，避免引入 npm 依赖）。
function extract(archivePath, ext, destDir) {
  if (ext === "zip") {
    execSync(`unzip -o "${archivePath}" -d "${destDir}"`, { stdio: "ignore" });
  } else {
    execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: "ignore" });
  }
}

// 生成单个平台子包的 package.json 内容。
// 关键：不要加 exports/bin 字段，否则破坏 wrapper 的 require.resolve 子路径解析。
function buildSubPackageJson(p, version) {
  const exeName = p.exe ? `${BINARY_NAME}.exe` : BINARY_NAME;
  return {
    name: subPackageName(p),
    version,
    description: `thinkthinking CLI 平台二进制 (${p.node}/${p.cpu})。由主包 @thinkthinking/cli 通过 optionalDependencies 自动安装，请勿直接依赖。`,
    license: "MIT",
    repository: { type: "git", url: "https://github.com/thinkthinking/cli.git" },
    engines: { node: ">=16" },
    os: [p.node],
    cpu: [p.cpu],
    // Yarn PnP 提示：二进制必须解包到真实路径才能执行。
    preferUnplugged: true,
    files: [exeName],
  };
}

async function buildOne(p, version, tmpDir) {
  const archiveName = `${BINARY_NAME}_${version}_${p.goos}_${p.goarch}.${p.ext}`;

  // 优先用本地缓存的归档（npm/.cache/v<version>/<archive>），存在则跳过下载。
  const cachedArchive = path.join(CACHE_ROOT, `v${version}`, archiveName);
  let archivePath;
  if (fs.existsSync(cachedArchive) && fs.statSync(cachedArchive).size > 0) {
    console.log(`[build-packages] using cached ${cachedArchive}`);
    archivePath = cachedArchive;
  } else {
    const url = `https://github.com/${REPO}/releases/download/v${version}/${archiveName}`;
    archivePath = path.join(tmpDir, archiveName);
    console.log(`[build-packages] downloading ${url}`);
    await download(url, archivePath);
  }

  // 解压到临时子目录，再取出二进制（归档内可能只含二进制本体）。
  const extractDir = path.join(tmpDir, `${p.node}-${p.cpu}`);
  fs.mkdirSync(extractDir, { recursive: true });
  extract(archivePath, p.ext, extractDir);

  const exeName = p.exe ? `${BINARY_NAME}.exe` : BINARY_NAME;
  const extractedBin = path.join(extractDir, exeName);
  if (!fs.existsSync(extractedBin) || fs.statSync(extractedBin).size === 0) {
    throw new Error(`二进制缺失或为空：${extractedBin}（归档 ${archiveName}）`);
  }

  // 组装子包目录。
  const pkgDir = path.join(PACKAGES_DIR, `cli-${p.node}-${p.cpu}`);
  fs.mkdirSync(pkgDir, { recursive: true });

  const destBin = path.join(pkgDir, exeName);
  fs.copyFileSync(extractedBin, destBin);
  // 非 windows 二进制加可执行权限（npm 打包保留可执行位）。
  if (!p.exe) fs.chmodSync(destBin, 0o755);

  const pkgJson = buildSubPackageJson(p, version);
  fs.writeFileSync(
    path.join(pkgDir, "package.json"),
    JSON.stringify(pkgJson, null, 2) + "\n",
    "utf8"
  );

  const sizeMB = (fs.statSync(destBin).size / 1024 / 1024).toFixed(1);
  return { name: pkgJson.name, dir: pkgDir, sizeMB };
}

// 同步主包：version + optionalDependencies 全部指向精确的本次版本号。
function syncMainPackage(version) {
  const pkgPath = path.join(MAIN_PKG_DIR, "package.json");
  const pkg = JSON.parse(fs.readFileSync(pkgPath, "utf8"));
  pkg.version = version;
  pkg.optionalDependencies = {};
  for (const p of PLATFORMS) {
    pkg.optionalDependencies[subPackageName(p)] = version;
  }
  fs.writeFileSync(pkgPath, JSON.stringify(pkg, null, 2) + "\n", "utf8");
  console.log(`[build-packages] synced main package version=${version} + optionalDependencies`);
}

async function main() {
  const version = getVersion();

  // 清空输出目录。
  fs.rmSync(PACKAGES_DIR, { recursive: true, force: true });
  fs.mkdirSync(PACKAGES_DIR, { recursive: true });

  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "tt-build-"));
  const results = [];
  try {
    for (const p of PLATFORMS) {
      results.push(await buildOne(p, version, tmpDir));
    }
  } catch (err) {
    console.error(`[build-packages] 失败：${err.message}`);
    process.exit(1);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }

  syncMainPackage(version);

  console.log(`\n[build-packages] 生成 ${results.length} 个子包 (version=${version}):`);
  for (const r of results) {
    console.log(`  - ${r.name}  (${r.sizeMB} MB)  ${r.dir}`);
  }
}

main();
