#!/usr/bin/env node
// postinstall 脚本：根据当前平台/架构，从 GitHub Releases 下载对应的
// thinkthinking 二进制归档，解压后放到 bin/native/。
//
// 设计要点（对应 init.md）：
//   - Go CLI 运行时不依赖 Node，Node 只用于安装与 wrapper。
//   - 支持 macOS / Linux / Windows，Windows 为 .exe。
//   - 环境变量 THINKTHINKING_SKIP_DOWNLOAD=1 跳过下载（CI / 离线场景）。
//   - 环境变量 THINKTHINKING_VERSION 指定版本（默认用 package.json version）。
//   - 下载失败给出清晰错误提示。

const fs = require("fs");
const path = require("path");
const os = require("os");
const https = require("https");
const { execSync } = require("child_process");

const REPO = "thinkthinking/cli";
const BINARY_NAME = "thinkthinking";

// 跳过下载（例如本地用 Go 直接构建，或离线环境）。
if (process.env.THINKTHINKING_SKIP_DOWNLOAD === "1") {
  console.log("[thinkthinking] THINKTHINKING_SKIP_DOWNLOAD=1, skip binary download.");
  process.exit(0);
}

// 平台 / 架构映射到 GoReleaser 的归档命名。
function resolvePlatform() {
  const platformMap = { darwin: "darwin", linux: "linux", win32: "windows" };
  const archMap = { x64: "amd64", arm64: "arm64" };

  const goos = platformMap[process.platform];
  const goarch = archMap[process.arch];

  if (!goos) {
    throw new Error(`unsupported platform: ${process.platform}`);
  }
  if (!goarch) {
    throw new Error(`unsupported arch: ${process.arch}`);
  }
  // Windows 仅发布 amd64。
  if (goos === "windows" && goarch !== "amd64") {
    throw new Error(`unsupported windows arch: ${process.arch} (only amd64 published)`);
  }
  return { goos, goarch };
}

function getVersion() {
  if (process.env.THINKTHINKING_VERSION) {
    return process.env.THINKTHINKING_VERSION.replace(/^v/, "");
  }
  const pkg = JSON.parse(fs.readFileSync(path.join(__dirname, "package.json"), "utf8"));
  return pkg.version;
}

// 构造归档下载 URL 与文件名，与 .goreleaser.yaml 的 name_template 对齐：
//   thinkthinking_{version}_{os}_{arch}.{tar.gz|zip}
function buildDownloadInfo() {
  const { goos, goarch } = resolvePlatform();
  const version = getVersion();
  const ext = goos === "windows" ? "zip" : "tar.gz";
  const archiveName = `${BINARY_NAME}_${version}_${goos}_${goarch}.${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${version}/${archiveName}`;
  return { url, archiveName, goos, ext };
}

// 跟随重定向的 HTTPS 下载。
function download(url, destPath, redirects = 0) {
  return new Promise((resolve, reject) => {
    if (redirects > 5) {
      return reject(new Error("too many redirects"));
    }
    https
      .get(url, { headers: { "User-Agent": "thinkthinking-installer" } }, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          return resolve(download(res.headers.location, destPath, redirects + 1));
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`download failed: HTTP ${res.statusCode} for ${url}`));
        }
        const file = fs.createWriteStream(destPath);
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
        file.on("error", reject);
      })
      .on("error", reject);
  });
}

// 解压归档到目标目录（用系统 tar / unzip，避免引入 npm 依赖）。
function extract(archivePath, ext, destDir) {
  if (ext === "zip") {
    // Windows 与部分 Linux 有 unzip；优先用 PowerShell Expand-Archive 兜底。
    try {
      execSync(`unzip -o "${archivePath}" -d "${destDir}"`, { stdio: "ignore" });
    } catch {
      execSync(
        `powershell -Command "Expand-Archive -Path '${archivePath}' -DestinationPath '${destDir}' -Force"`,
        { stdio: "ignore" }
      );
    }
  } else {
    execSync(`tar -xzf "${archivePath}" -C "${destDir}"`, { stdio: "ignore" });
  }
}

async function main() {
  let info;
  try {
    info = buildDownloadInfo();
  } catch (err) {
    console.error(`[thinkthinking] ${err.message}`);
    process.exit(1);
  }

  const nativeDir = path.join(__dirname, "bin", "native");
  fs.mkdirSync(nativeDir, { recursive: true });

  const tmpArchive = path.join(os.tmpdir(), info.archiveName);

  console.log(`[thinkthinking] downloading ${info.url}`);
  try {
    await download(info.url, tmpArchive);
    extract(tmpArchive, info.ext, nativeDir);
  } catch (err) {
    console.error(`[thinkthinking] install failed: ${err.message}`);
    console.error(
      "[thinkthinking] 你也可以从 GitHub Releases 手动下载二进制并放到 npm/package/bin/native/，" +
        "或设置 THINKTHINKING_SKIP_DOWNLOAD=1 跳过下载后用 `go install` 自行构建。"
    );
    process.exit(1);
  } finally {
    fs.rmSync(tmpArchive, { force: true });
  }

  // 给 macOS/Linux 二进制加可执行权限。
  const binPath = path.join(nativeDir, info.goos === "windows" ? `${BINARY_NAME}.exe` : BINARY_NAME);
  if (info.goos !== "windows" && fs.existsSync(binPath)) {
    fs.chmodSync(binPath, 0o755);
  }

  console.log(`[thinkthinking] installed binary at ${binPath}`);
}

main();
