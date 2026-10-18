# M-14 诊断工具与运行说明

> 状态：**M-14 未解决**（维护事项，[m1-exit-evidence.md §9](m1-exit-evidence.md)）。本工具只用于诊断：它不修改 `Open`、只读保护、错误分类或重试行为，也不决定任何修复。没有复现不等于已经解决。

## 1. 要区分的四件事

| # | 内容 | 性质 |
|---|---|---|
| 1 | **原始 CI 失败** | 事实（GitHub CI 日志） |
| 2 | **辅助只读探测出现同一错误码** | 事实，但只针对辅助探测 |
| 3 | **真实的 `Open` 路径尚未复现** | 事实，限于已运行的次数 |
| 4 | **SQLite 内部机制** | **尚未确认** |

**1. 原始 CI 失败**

- 位置：PR #23 的 [run 39110669282](https://github.com/XKHoshizora/specmetry/actions/runs/39110669282) attempt 1，Go tests (macos-latest)，检出 `4f2393d`。
- 测试：`internal/store/sqlite` 的 `TestSeqIsUniqueAcrossProcesses`。
- 现象：一个辅助进程在 `Open` 中报告 `cannot read the database version: cannot read the database: attempt to write a readonly database (264)`。
- 后续：attempt 2 只重新执行了这个作业，通过。

**2. 辅助只读探测出现同一错误码**

- 做法：在 `Open` 之外、不持有打开锁，循环调用 `Open` 使用的只读版本探测 `readMetaReadOnly`，同时有其他进程在提交。
- 结果：Windows 本机与 macOS CI 都出现了 264，错误文字与原始失败相同；macOS CI 还出现了 1288（`SQLITE_READONLY_CANTINIT`）。没有提交的对照组为 0。
- 限制：这只说明探测函数本身在这种并发下会遇到这些错误码，**不等于 `Open` 会遇到**。`Open` 在打开锁之下调用它，而且只调用一次。

**3. 真实的 `Open` 路径尚未复现**

- 做法：按真实的锁与顺序运行 `Open`，即原测试的模式。
- 结果：在 Windows 本机、Windows CI、macOS CI 上，打开阶段的失败都是 0（§5）。
- 原始失败只出现过一次。

**4. SQLite 内部机制尚未确认**

- 读 modernc 转写的 SQLite 代码得到的假设：只读的 `-shm` 连接在无锁读取 WAL 索引头时，与另一进程的提交重叠，读到不一致的头，于是返回 `SQLITE_READONLY_RECOVERY`。
- 没有对 SQLite 插桩确认是哪一种坏头。macOS 上的 1288 也还没有解释。

## 2. 能确认的保证（限定范围）

只陈述有实际证据的部分：

- **只读探测失败的这一次 `Open` 不写数据**（代码路径，只读审查）：
  - 探测在 `store.go:117 ~ 124` 失败并返回；
  - 这发生在墓碑处理（`:136`）、创建数据库文件（`:140`）、打开读写连接（`:143 ~ 147`）、迁移与任何写事务之前；
  - 探测连接是 `mode=ro&readonly_shm=1`（`readonly.go:173`），在只读事务中执行。
  - 错误归类为 environment（退出码 6）。
- **本工具运行过的场景中，数据库检查都通过**：
  - 每次运行（只读探测）与每一轮（`Open` 路径）结束后，`PRAGMA integrity_check` 都是 `ok`；
  - 行数等于各辅助进程报告已提交的数；
  - `seq` 唯一、从 1 连续，`seq_counter` 等于最后一个。
  - 也就是说，在这些运行中，失败的探测、失败的 `Open` 与写入阶段的 `SQLITE_BUSY` 都没有留下多出来的行。

**不在保证之内**：

- 这些运行之外的场景，例如崩溃、断电或其他程序。
- 用户体验：命令会以退出码 6 误报环境错误。
- SQLite 的内部状态，以及任何关于"无害"的一般结论。

## 3. 工具

| 文件 | 内容 |
|---|---|
| `internal/store/sqlite/m14_diag_test.go` | 两个诊断测试与两个辅助进程 |
| `internal/store/sqlite/store_test.go` | `TestMain` 登记这两个辅助进程（`m14open`、`m14probe`） |
| `.github/workflows/m14-diagnostics.yml` | 在 macOS 与 Windows 上运行这两个测试。不属于 `ci.yml`，后者只把它们作为跳过的测试 |

**`TestM14ReadOnlyProbe`**（辅助只读探测）：

- 测试进程打开数据库并一直保持打开，与原测试相同。
- 探测进程在循环中调用 `readMetaReadOnly`，即 `Open` 使用的探测，但没有数据目录检查，也没有打开锁。
- 遇到 SQLite 错误后立即再探测一次。
- 两个变体：
  - "control, no commits"：没有写入进程；
  - "during commits"：写入进程各自 `Open` 之后在循环中提交。

**`TestM14OpenPath`**（真实的 `Open` 路径）：

- 每一轮一个新的数据库，由测试进程打开并保持。
- 同时启动 `M14_PROCS` 个辅助进程，每个都按真实的方式 `Open`（数据目录检查、打开锁、只读探测、读写连接），再写入 `M14_FACTS` 条事实。
- 默认是原测试的规模（3 个进程，各 100 条），重复 `M14_ROUNDS` 轮。
- 失败按阶段与错误码归类：
  - 只读版本探测；
  - 无法检查；
  - 读写连接的版本读取；
  - 其他打开步骤；
  - 写入。

**开启与上限**：

- 不设置 `SPECMETRY_M14=1` 时，两个测试都跳过，所以常规测试与 CI 不运行它们。
- 每个参数都有上限，超过上限按上限运行：

| 变量 | 默认 | 上限 | 用于 |
|---|---|---|---|
| `M14_SECONDS` | 20 | 300 | 只读探测每个变体的秒数 |
| `M14_WRITERS` | 3 | 16 | 只读探测的写入进程数 |
| `M14_PROBERS` | 3 | 16 | 只读探测的探测进程数 |
| `M14_ROUNDS` | 30 | 1000 | `Open` 路径的轮数 |
| `M14_PROCS` | 3 | 16 | `Open` 路径每轮的辅助进程数 |
| `M14_FACTS` | 100 | 1000 | `Open` 路径每个辅助进程写入的事实数 |
| `M14_SHA` | — | — | 被测提交，写进结果行 |

**结果行**：每个变体一行 `M14-RESULT {...}`（JSON）。内容包括：

- 平台：`GOOS`、`GOARCH`、Go 版本、SQLite 版本（`sqlite_version()`）、驱动版本、`M14_SHA`，在 GitHub 上还有运行器镜像；
- 上限；
- 各结果的计数，零也记录；
- 失败的阶段、错误码与分类；
- 出错的时刻（毫秒）与立即重试的结果；
- 数据库检查发现的问题。

## 4. 运行说明

**本机**（macOS、Linux 或 Windows 的 Git Bash，在仓库根目录中执行）：

```bash
SPECMETRY_M14=1 M14_SHA=$(git rev-parse HEAD) M14_SECONDS=60 M14_ROUNDS=200 \
  go test -tags nowebui -count=1 -v -timeout 25m -run '^TestM14' ./internal/store/sqlite | tee m14.log
grep M14-RESULT m14.log
```

Windows PowerShell：

```powershell
$env:SPECMETRY_M14 = "1"; $env:M14_SHA = git rev-parse HEAD; $env:M14_SECONDS = "60"; $env:M14_ROUNDS = "200"
go test -tags nowebui -count=1 -v -timeout 25m -run '^TestM14' ./internal/store/sqlite | Tee-Object m14.log
Select-String M14-RESULT m14.log
```

- 一次运行大约 4 ~ 5 分钟：两个探测变体各 60 秒，加上 200 轮 `Open` 路径。
- 测试本身只在工具出错时失败，例如辅助进程崩溃或数据库检查发现问题。出现 264 不会让测试失败，它被计入结果。

**GitHub**：

- 工作流 "M-14 diagnostics" 在两种情况下运行：
  - 推送到 `test/m14-diagnostics` 分支、并且改动了被诊断的代码（`internal/store/sqlite/**`、`internal/datadir/**`、`go.mod`、`go.sum`）或工作流本身。只改文档的推送不触发。
  - 手动触发（`workflow_dispatch`，可以给出秒数与轮数）。GitHub 只对默认分支上已有的工作流提供手动触发，所以本 PR 合并之前只能用推送触发。
- 在同一提交上重跑：`gh run rerun <run-id>`。重跑沿用原来的提交与输入。
- 不要无限重跑；每次运行的结果都按原样记录。

## 5. 实际结果

被测提交 `1d8b73c9e637e339f2e1ebac4c81b5bd029480de`（本工具的提交）。两处都用 `M14_SECONDS=60`、`M14_ROUNDS=200`，其余参数为默认值（3 个写入进程、3 个探测进程、每轮 3 个进程各 100 条）。

| 平台 | 环境 | 只读探测：对照（无提交） | 只读探测：有提交 | `Open` 路径（200 轮，600 次打开） | 数据库检查 |
|---|---|---|---|---|---|
| Windows，本机 | windows/amd64，go1.27.1，SQLite 3.53.4，modernc.org/sqlite v1.60.1 | 24,010 次，全部成功 | 23,609 次：**3 次 264**，在 18,431、27,107、56,110 ms；3 次立即重试都成功。写入进程：39,395 次提交，2 个进程以 `SQLITE_BUSY`（5）停止 | 打开失败 0；写入失败 0 | 无问题 |
| macOS，CI [run 39118832445](https://github.com/XKHoshizora/specmetry/actions/runs/39118832445) | darwin/arm64，go1.27.2，SQLite 3.53.4，modernc.org/sqlite v1.60.1，运行器 macos26 20261013.0424.1 | 689,388 次，全部成功 | 466,837 次：**8 次 264，4 次 1288**（`SQLITE_READONLY_CANTINIT`）；12 次立即重试中 11 次成功、1 次失败。写入进程：288,420 次提交，2 个进程以 `SQLITE_BUSY` 停止 | 打开失败 0；写入失败 0 | 无问题 |
| Windows，CI（同一运行） | windows/amd64，go1.27.2，SQLite 3.53.4，modernc.org/sqlite v1.60.1，运行器 win25-vs2026 20261007.264.1 | 73,932 次，全部成功 | 23,901 次，全部成功。写入进程：15,622 次提交，2 个进程以 `SQLITE_BUSY` 停止 | 打开失败 0；**写入失败 3 次，在 2 轮中**：`database is locked (5) (SQLITE_BUSY)`，分类为 internal | 无问题 |

**结论**：

- **原始失败**（`Open` 中的 264）在三处的 600 次真实打开中都没有复现。
- **辅助探测**：
  - 在有并发提交时，Windows 本机与 macOS CI 都出现了 264，macOS CI 还出现了 1288；Windows CI 这一次没有出现。
  - 对照组在三处都没有出现。
- **另一个现象**：真实 `Open` 路径中，写入阶段的 `SQLITE_BUSY` 在 Windows CI 上出现了（2/200 轮）。它不属于 M-14 的定义（M-14 是打开时的版本探测），错误分类为 internal（退出码 1）。是否另立维护事项，由负责人决定。
- 此前的探索性运行用的是一个临时工具（没有提交），结果没有经本工具复核，只作参考，不计入本节的结果：
  - 辅助探测在有提交时 34,832 次中 6 次 264，对照 10,078 次中 0 次；
  - 真实 `Open` 的 1,600 次打开，0 次失败；
  - 现有测试连续运行 600 次，0 次失败。

## 6. 剩余边界与下一步

- M-14 保持未解决：没有确认的根因，也没有修复。原始失败没有在真实的 `Open` 路径中复现。
- 尚未解释：macOS 上的 1288（`SQLITE_READONLY_CANTINIT`），以及 Windows CI 这一次的辅助探测为 0。
- 下一步建议（都需要负责人另行授权）：
  1. 在隔离环境中对 modernc 的 SQLite 插桩，确认 264 与 1288 各自对应的条件；
  2. 再提出处理选项（例如把探测中的这两个错误码视为瞬时错误，有界地重新探测，或者给出"请重新执行"的提示），对照 MVP-STO-02、STO-07 审查；
  3. 写入阶段的 `SQLITE_BUSY` 另行评估，它与 WP2.3 扫描的并发写入有关。
- 与 M-1 分开：两者在不同的层，没有共同根因的证据。
