# 开发指南

本文说明 Specmetry 的本地开发环境、目录结构、常用命令和工程约定。设计依据见 `AGENTS.md` 的文档地图；里程碑与范围以 `docs/implementation-plan.md` 和 `docs/mvp.md` 为准。

## 1. 环境要求

| 工具 | 版本 | 说明 |
|---|---|---|
| Go | `go.mod` 中的 `go` 版本（当前 1.27） | 纯 Go 构建，不需要 cgo |
| Node.js | ≥ 22.12（CI 使用 24） | 只用于构建内嵌的 Dashboard |
| git | 任意近期版本 | 运行时和测试都需要 |
| golangci-lint | v2.14.0 | 与 CI 保持一致：`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0` |

开发时建议把 `SPECMETRY_HOME` 指向一个临时目录，避免改动 `~/.specmetry` 中的真实数据：

```bash
export SPECMETRY_HOME="$(mktemp -d)/specmetry-home"
```

## 2. 目录结构

| 路径 | 内容 |
|---|---|
| `cmd/specmetry/` | 二进制入口，只调用 `internal/cli.Execute` |
| `internal/cli/` | cobra 命令。命令返回 `*apperr.Error`，由 `Execute` 统一映射为退出码并输出 |
| `internal/apperr/` | 错误码、退出码、HTTP 状态码映射，以及 `error.v1` 信封（architecture §9.1） |
| `internal/datadir/` | 数据目录解析（`SPECMETRY_HOME`）、私有权限（Unix `0700`/`0600`，Windows 受保护 DACL：当前用户 + SYSTEM）、权限检查 |
| `internal/config/` | `config.toml` 读取（未知键视为错误） |
| `internal/store/sqlite/` | SQLite 存储基础：打开顺序、迁移框架、备份、墓碑、`seq`、只追加触发器、只读检查 |
| `internal/contract/` | 用内嵌的 `schemas/` 做 JSON Schema 2020-12 校验 |
| `internal/testutil/` | 测试基座（见 §6） |
| `internal/buildinfo/` | 版本与构建信息 |
| `schemas/` | 交换契约（规范来源）及示例；`schemas.go` 将其内嵌 |
| `web/` | Dashboard 前端（Vite + React + TypeScript）；`embed.go` 内嵌 `web/dist` |
| `scripts/check.sh` | 在本地运行与 CI 相同的检查 |

## 3. 常用命令

```bash
# 全部检查（与 CI 一致）
sh scripts/check.sh

# Go：不含前端的构建、测试、静态检查
go build -tags nowebui ./...
go test -tags nowebui ./...
go vet -tags nowebui ./...
golangci-lint run ./...
golangci-lint fmt ./...

# 前端（在 web/ 目录下）
npm ci
npm run dev           # 开发服务器
npm run check         # 格式、lint、类型检查、测试、构建
npm run format        # 自动格式化

# 含 Dashboard 的完整二进制（先构建前端）
(cd web && npm run build)
go build -o dist/ ./cmd/specmetry
```

### 构建标签 `nowebui`

`web/embed.go` 用 `//go:embed all:dist` 内嵌前端构建产物。`web/dist` 不进入版本库，因此在没有构建前端时，Go 命令需要加 `-tags nowebui`；此时 `specmetry version` 会显示 `dashboard: not embedded`。`golangci-lint` 已在 `.golangci.yml` 中默认使用该标签。

## 4. 错误与退出码

所有命令错误都使用 `internal/apperr`，映射关系以 architecture §9.1、§9.2 为准：

| 错误码 | 退出码 | HTTP |
|---|---|---|
| `invalid_input` | 2 | 400 |
| `not_found` | 2 | 404 |
| `human_required` | 3 | 403 |
| `not_initialized` | 4 | 404 |
| `stale_precondition` | 5 | 409 |
| `environment` | 6 | 503 |
| `unauthorized` / `forbidden` | —（仅 HTTP） | 401 / 403 |
| `internal` | 1 | 500 |

退出码 10（`check run` 完成但有检查未通过）在 M2 中加入。

- 非 `*apperr.Error` 的错误一律按 `internal` 处理；cobra 自身的参数错误按 `invalid_input` 处理。
- 人类可读的错误写入 stderr；从 M1 起带 `--json` 的命令把 `error.v1` 信封写入 stdout（architecture §9.1）。

## 5. 存储与迁移

- 数据库：`$SPECMETRY_HOME/specmetry.db`，WAL，`busy_timeout=5000`，单一写连接；写事务使用 `BEGIN IMMEDIATE`（`DB.WriteTx`）。
- 打开顺序：确保目录 → 版本检查（`schema_version` 高于本程序支持的版本，或 `min_binary_version` 高于本程序版本 → 退出码 6，不写入）→ 应用墓碑 → 需要迁移时：获取 `.migrate.lock`、`VACUUM INTO` 备份（保留最近 3 份）、在一个 `BEGIN EXCLUSIVE` 事务中执行全部待执行迁移。
- 新增迁移：在 `internal/store/sqlite/migrations/` 下添加 `NNNN_name.sql`，编号必须连续。迁移需要旧程序拒绝打开时，在迁移中显式设置 `meta.min_binary_version`。
- 事实表只追加：建表后附加 `AppendOnlyTriggers(table)` 生成的触发器；只有墓碑清理等维护操作可以在 `Maintenance` 中绕过。
- 先后顺序只由 `seq` 决定：在写事务中调用 `NextSeq(ctx, tx)`。
- `specmetry doctor` 通过 `sqlite.Inspect` 只读检查数据库，不创建、不迁移、不写入。

## 6. 测试

- 推导引擎（M2 起）测试先行，表驱动，每条规则对应 CX 反例；`ready` 复核器在测试中独立实现。
- 验收项编号写在测试函数的注释中（例如 `// MVP-STO-01`），便于追溯。
- 测试基座 `internal/testutil`：
  - `TempHome`：为测试设置独立且尚未创建的 `SPECMETRY_HOME`。
  - `NewGitRepo`：使用隔离 git 配置的夹具仓库，支持提交和 `AddWorktree`。
  - `Golden`：golden 文件比较；设置 `SPECMETRY_UPDATE_GOLDEN=1` 重新生成。
  - `RunInPTY`：在伪终端中运行命令（Windows 使用 ConPTY），用于通道识别测试。
  - `NewCanary` / `FindInTree`：隐私测试中的 canary 搜索。
  - `DispatchHelper` / `HelperCommand`：在 `TestMain` 中注册辅助子进程，用于多进程并发测试。
  - `LoosenPermissions`：放宽数据目录权限，用于检查 `doctor` 的报告。
- 在 Windows 上验证 Linux 行为（可选）：用 `GOOS=linux go test -c -tags nowebui` 交叉编译测试二进制，在 WSL 中从对应的包目录运行。

## 7. 契约

修改交换格式时，先改 `schemas/`，同步更新 `schemas/examples/valid/` 和 `schemas/examples/invalid-cases.json`；`internal/contract` 的测试会校验所有 schema 能编译、全部有效示例通过、全部无效用例失败。

## 8. 约定

- 行尾统一为 LF（`.gitattributes`）。Go 代码用 `golangci-lint fmt` 格式化，前端用 Prettier。
- 前端依赖使用精确版本（`web/.npmrc` 中 `save-exact=true`），并提交 `package-lock.json`。
- 代码、标识符、CLI 输出、提交信息使用英文；设计文档使用中文。

## 9. CI

`.github/workflows/ci.yml` 在 push 到 `master` 和 pull request 时运行：

| Job | 内容 |
|---|---|
| Go tests | Linux、macOS、Windows 三平台 `go test -tags nowebui`（Linux 额外开启 `-race`） |
| Go lint | `go mod tidy -diff`、`go vet`、golangci-lint |
| Web | `npm ci`、`npm run check`，上传 `web/dist` |
| Build | 三平台下载 `web/dist`，构建完整二进制，运行内嵌测试和冒烟测试（`version`、`doctor`） |
