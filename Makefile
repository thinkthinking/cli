# thinkthinking Makefile
#
# 常用目标：build / run / test / lint / fmt / snapshot / release

BINARY      := thinkthinking
CMD_PATH    := ./cmd/thinkthinking
BIN_DIR     := bin
VERSION_PKG := github.com/thinkthinking/cli/internal/version

# 构建期注入的版本信息（本地构建用 git 描述）。
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(VERSION_PKG).Version=$(VERSION) \
	-X $(VERSION_PKG).Commit=$(COMMIT) \
	-X $(VERSION_PKG).Date=$(DATE)

.PHONY: build run test lint fmt vet snapshot release clean tidy

## build: 编译二进制到 bin/
build:
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD_PATH)
	@echo "built $(BIN_DIR)/$(BINARY) ($(VERSION))"

## run: 直接运行（透传参数，例如 make run ARGS="version --pretty"）
run:
	go run $(CMD_PATH) $(ARGS)

## test: 运行全部测试
test:
	go test ./...

## test-v: 详细测试输出
test-v:
	go test -v ./...

## fmt: 格式化代码
fmt:
	gofmt -w .

## vet: 静态检查
vet:
	go vet ./...

## lint: fmt 检查 + vet（CI 用）
lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed:"; gofmt -l .; exit 1)
	go vet ./...

## tidy: 整理依赖
tidy:
	go mod tidy

## snapshot: 本地试构建跨平台产物（不发布）
snapshot:
	goreleaser release --snapshot --clean

## release: 正式发布（需打 tag 并设置 GITHUB_TOKEN）
release:
	goreleaser release --clean

## clean: 清理产物
clean:
	rm -rf $(BIN_DIR) dist
