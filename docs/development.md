# 开发指南

本文说明 Specmetry 的本地开发环境、目录结构、常用命令和工程约定。设计依据见 `AGENTS.md` 的文档地图；里程碑与范围以 `docs/implementation-plan.md` 和 `docs/mvp.md` 为准。

## 1. 环境要求

| 工具 | 版本 | 说明 |
|---|---|---|
| Go | `go.mod` 中的 `go` 版本（当前 1.27） | 纯 Go 构建，不需要 cgo |
| Node.js | ≥ 22.12（CI 使用 24） | 只用于构建内嵌的 Dashboard |
| git | 2.32 或更高 | 运行时和测试都需要；`init` 用 `GIT_CONFIG_GLOBAL` / `GIT_CONFIG_SYSTEM` 隔离用户的 git 配置（architecture §6.1.2）。读取 git 对象（git 项目的 `init`、代码引用检查）还要求 git 遵守 `GIT_NO_LAZY_FETCH`，并且 `ls-files` 认识 `--sparse`，运行时探测（PD-26）；git 2.32.0 不满足，测试需要较新的 git。用指定的 git 运行测试时，可设置 `SPECMETRY_TEST_GIT_OBJECTS=safe` 或 `unsafe`，说明预期的探测结论。SHA-256 仓库不受支持（TD-26） |
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
| `internal/config/` | `config.toml` 读取（未知键视为错误；错误只给出行列位置，不回显文件内容）；`[secrets]` 追加规则 |
| `internal/channel/` | CLI 通道识别（architecture §5.1）：`Detect`（stdin、stdout 是否为终端或 Windows 控制台，已核实的 Agent 环境变量只看是否存在且非空，值只用于这次判断，从不保存、记录或输出）、`Guard`（人类专属操作的拦截与交互式确认，没有跳过参数）。防止误操作，不是身份认证 |
| `internal/secrets/` | 密钥识别（architecture §10.8）：14 条内置规则、追加规则校验、按字段扫描（JSON Pointer）、不回显命中文本的拒绝错误。只能降低泄露风险，不能保证检测出所有秘密 |
| `internal/store/sqlite/` | SQLite 存储基础：打开顺序、迁移框架、备份、墓碑、`seq`、只追加触发器、只读检查；迁移 v2 起是业务表（`projects`、`worktrees`、`delivery_targets`） |
| `internal/project/` | 项目登记与 `init`（architecture §6.1.1、§6.1.2）：阶段识别与候选来源（只看路径和大小，不读内容）、marker、exclude 条目、项目与 worktree 的登记、默认交付目标 |
| `internal/source/` | 来源（architecture §6.1.3）：路径解析（`in_repo` / `external`）、分块计算 sha256（可取消，不保存内容）、磁盘观测、来源集合哈希、登记、确认与移除、引用列表 |
| `internal/baseline/` | 读取已确认的基准（当前版本、按编号查找条目），按 TD-14 分配编号（`GOAL-n`、`REQ-nnn`、`AC-<需求号>-n`、`Q-nnn`，永不复用） |
| `internal/proposal/` | 提案（architecture §6.1.4）：静态规则、来源解析、流式片段校验（不保存原文）、代码引用检查、在一个写事务中保存提案、条目、问题与问题的来源引用 |
| `internal/gitx/` | 调用 git CLI（只读）：定位仓库与 worktree、分支与提交、文件列表；不传递 `GIT_DIR` 等定位变量 |
| `internal/domain/derive/` | 推导引擎（纯函数，data-model §6、§6.13、§6.13.1）：`Derive` 是截止 `as_of_seq` 的完整推导，`ChangesSince` 是 `--diff-since` 的 AC 投影。全部输入显式给出，只导入不做 I/O 的标准库包（由测试检查），不读取数据库、git、文件、网络、环境变量或时钟；缺少输入时返回 `*InputError`，不用缺省值或当前值补齐 |
| `internal/ulid/` | ULID 生成（只用于标识，不用于排序） |
| `internal/contract/` | 用内嵌的 `schemas/` 做 JSON Schema 2020-12 校验；错误只给出字段指针和约束，不回显提交的值或意外的键（architecture §10.8） |
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
- 打开顺序（architecture §11.1、§11.2）：
  1. `datadir.Ensure`：不存在则私有地创建；已存在则检查目录和其中所有条目，不私有就在写入前拒绝（退出码 6）。**任何写入方都必须先调用它**；`doctor` 的检查不能代替。
  2. 取得 `.migrate.lock` 上的操作系统排他锁（`flock` / `LockFileEx`），直到打开完成；进程崩溃时由操作系统释放。
  3. 版本检查：用只读连接读取（见下），不兼容或无法确认 → 退出码 6，文件不变；然后才建立读写连接。
  4. 校验并应用墓碑。
  5. 需要迁移时：`VACUUM INTO` 备份（保留最近 3 份），在一个 `BEGIN EXCLUSIVE` 事务中执行全部待执行迁移。
- 写入只通过 `DB.WriteTx`：它在取得写锁后、执行回调前重新检查 `schema_version`、`min_binary_version` 和 `recursive_triggers`，不满足时不执行回调。
- 新增迁移：在 `internal/store/sqlite/migrations/` 下添加 `NNNN_name.sql`，编号必须连续。迁移需要旧程序拒绝打开时，在迁移中显式设置 `meta.min_binary_version`。
- 建表约束（data-model §1.1）：保存项目数据的表都有 `project_id`（`projects` 用 `id`）；外键只引用同一项目的行；不在建表语句中声明 `ON CONFLICT REPLACE`。`TestEveryTableHasProjectID` 对全部迁移检查第一条。
- 事实表只追加：建表后附加 `AppendOnlyTriggers(table)` 生成的触发器；只有墓碑清理等维护操作可以在 `Maintenance` 中绕过。**不要使用 REPLACE 冲突处理**（`INSERT OR REPLACE`、`REPLACE INTO`、`ON CONFLICT REPLACE`）：它通过删除旧行实现，`TestProductionSQLNeverUsesReplace` 会扫描生产代码和迁移。数据库层由触发器加上每个连接的 `recursive_triggers` 兜底。
- 先后顺序只由 `seq` 决定：在写事务中调用 `NextSeq(ctx, tx)`。
- 连接：每个连接都开启 `recursive_triggers` 并保留 WAL 文件（`SQLITE_FCNTL_PERSIST_WAL`，`journal_size_limit = 0`），所以关闭后 `-wal`（0 字节）和 `-shm` 仍在。
- 只读检查（`readonly.go`；`sqlite.Inspect`、`doctor` 和打开前的版本检查都用它）：SQLite 只读连接（`mode=ro`、`readonly_shm=1`），只在实测不会修改文件的状态下读取，否则报告版本未知（状态表见 architecture §11.5，由 `TestReadOnlyConnectionByFileState` 固定）。**不要**用 `query_only`、`immutable=1` 或 `nolock=1` 做只读检查，也不要自行解析 WAL。
- `doctor` 是非破坏性诊断，不是取证级只读工具：另一个 SQLite 程序在检查期间关闭数据库时，SQLite 可能新建空的 `-wal` / `-shm`，此时报告版本未知（MVP-STO-07，`TestInspectDuringExternalCloseReportsUnknown`）；`-shm` 为空时 `doctor` 不读取、直接报告版本未知，以免连续的同一竞态被漏掉（`TestInspectRepeatedExternalCloseReportsUnknown`、`TestDoctorRepeatedExternalCloseReportsUnknown`）。版本不兼容而拒绝打开时，Specmetry 只允许两种文件副作用（MVP-STO-02）：创建空的 `.migrate.lock`（`TestIncompatibleRefusalMayOnlyCreateCoordinationFile`），以及另一个 SQLite 程序在版本检查期间关闭数据库时只读连接新建的空 `-wal` / `-shm`（`TestOpenDuringExternalCloseOnlyAddsEmptyAuxiliaryFiles`）。写测试时用 `readOnlyHook` 在判断文件状态之后插入外部进程的操作，不要用固定等待。

## 6. 测试

- 推导引擎（M2 起）测试先行，表驱动，每条规则对应 CX 反例；`ready` 复核器在测试中独立实现。性质测试使用固定的随机种子，每个种子是一个子测试，失败时可以单独重跑，例如 `go test ./internal/domain/derive -run 'TestGapsAndDelivery/seed=17'`。
- 验收项编号写在测试函数的注释中（例如 `// MVP-STO-01`），便于追溯。
- 测试基座 `internal/testutil`：
  - `TempHome`：为测试设置独立且尚未创建的 `SPECMETRY_HOME`。
  - `NewGitRepo`：与本机 git 环境隔离的夹具仓库：不继承任何 `GIT_*` 变量和 `HOME`、`XDG_CONFIG_HOME`，没有系统或全局配置、模板和钩子，固定身份和 SHA-1 对象格式；修改配置和提交之前确认 git 用的是本仓库的 `.git`。支持提交、按固定时间提交（`CommitAt`）、`AddWorktree`（新分支）和 `WorktreeFor`（已有分支）。
  - `BuildFixture`：把 `testdata/fixtures/` 中的项目夹具构造成 git 仓库，提交 ID 在三个平台上相同（见 `testdata/fixtures/README.md`）。
  - `Golden`：golden 文件比较；设置 `SPECMETRY_UPDATE_GOLDEN=1` 重新生成。
  - `RunInPTY` / `RunInPTYSession`：在伪终端中运行命令（Windows 使用 ConPTY），后者还能在提示出现后输入回答，用于通道识别和交互式确认测试。ConPTY 渲染时会去掉行尾空格，所以等待的提示文字不要以空格结尾。
  - `NewCanary` / `FindInTree`：隐私测试中的 canary 搜索。
  - `DispatchHelper` / `HelperCommand`：在 `TestMain` 中注册辅助子进程，用于多进程并发测试。
  - `LoosenPermissions`：放宽文件或目录的权限，用于检查拒绝写入和 `doctor` 的报告。
  - `SnapshotTree` / `AssertTreeUnchanged`：记录目录下每个条目的类型、权限、大小、修改时间和内容哈希，用于断言"拒绝时没有任何改动""只读检查前后完全一致"。`SettledSnapshot` 先等两次快照一致：Windows 的 NTFS 可能稍后才公布另一进程较早写入的修改时间。
  - `factgen`：推导测试的事实前端（WP2.1、WP2.2）。
    - `History` 按生产写入的方式逐次写入事实（同一次逻辑写入共用一个 `seq`），产出与数据库行对应的转储：JSON 列为文本，NULL 为 nil，并带当前投影列，用来检查实现不拿当前投影冒充历史。
    - 审核与人工编辑（`Review` / `Apply`）先计划、再按 `baseline.Apply` 的方式应用：显式变更都来自提案条目，同一提案的 target 至多一次（V-P03），按 data-model §5.5 的阶段与 `item_index` 应用，级联紧跟其原因，后续处理（`FollowUp`）在最后；违反规则的构造会 panic。`Trace` 记录每次写入，用于定位生成历史中的问题。
    - `Random` 用固定种子生成合法事实历史与当前值输入。
    - `factgen/legality` 只读转储行，按冻结的写入规则独立检查历史是否合法，不使用 `History`、推导器或复核器；性质测试与双向对照只使用经它确认的历史。
    - `factgen/toderive` 把转储映射为推导引擎的输入，只做映射，不选择、不过滤、不计算。
  - `testdir.New`：放数据目录的测试目录。Windows 上位于 `%LOCALAPPDATA%\specmetry-test`：`%TEMP%` 可能授予其他账户修改权限（例如编码代理的沙箱账户），按路径规则会被拒绝。
- 存储包的测试用辅助子进程模拟崩溃（写入后不关闭数据库直接退出）、进行中的未提交事务、并发升级和持锁进程。
- 跨用户测试（`crossuser_linux_test.go`）需要以 root 运行，以便启动两个不同用户的进程；普通用户运行时跳过。CI 在 Linux 上单独用 `sudo` 运行它们；本地可在 WSL 中用 `wsl -u root` 运行交叉编译的测试二进制。
- 在 Windows 上验证 Linux 行为（可选）：用 `GOOS=linux go test -c -tags nowebui` 交叉编译测试二进制，在 WSL 中从对应的包目录运行，并把 `TMPDIR` 设为 Linux 文件系统上的目录（`/mnt/c` 不支持 Unix 权限位）。
- 平台相关代码要对三个目标分别运行 lint：`GOOS=linux golangci-lint run ./...`、`GOOS=darwin ...`；CI 的 lint 任务运行在 Linux 上。
- 在 Windows 上创建符号链接需要开发者模式或管理员权限，相关测试在没有权限时跳过（Linux 和 CI 的 Windows 运行器上会执行）；联接（junction）不需要权限，链接检测由它覆盖。

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
