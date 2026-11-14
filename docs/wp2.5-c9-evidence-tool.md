# WP2.5 第三批：C-9 限定取证工具第一阶段的实现记录

> 状态：**第一阶段实现记录（test-only 的独立模块；不合并，不进行正式取证）**（2026-11-14，review-resolution §190）。审查 147 为技术 NO-GO（B-147-1 ~ 4）；定向修复在 PR #56 内进行（§191），待修复之后由 Review 148 复审。
>
> - 依据：[限定取证计划](wp2.5-c9-limited-evidence-plan.md)（PR #55 合并为 `9137279`），[G-4 方法记录](wp2.5-c9-exception-identity-method.md)，[实施计划](wp2.5-implementation-plan.md) §12.9 ~ §12.14，[决策稿](wp2.5-c9-dependency-exception-draft.md) §2.3、§2.7。
> - 本文写定计划 §8.1 要求在实现中固定、在固定实现的审查中核对的事项，并逐项列出没有实现、没有执行与依赖 E-10 (ii) 的部分。
> - **E-10 (ii) 没有确认：真实调用都停止。** 工具中只有进程内的假调用者，以及一个不启动任何进程的停止调用者。本文中的任何测试结果都不是正式 LC 对照的运行证据，也不是真实输入上的取证结果。
> - 不宣布任何 G 项满足，不批准例外清单，不开始第三批的生产实施。路径 1、T-1、L-b1a 保持；H-1 未采纳，原链接方法继续停止。

## 1. 范围

| 计划中的部分（§4.3） | 本阶段 | 位置 |
|---|---|---|
| 输入解析：go.sum（I-1）、第 1 种调用的 JSON、上下文文本 v1、第 3 种调用的 JSON 流（§3.1 ~ §3.4 的第 1 ~ 4 步） | 实现 | `internal/decode` |
| T2 的常量与绑定（LC-10）、三种调用的参数与工具设定的环境 | 实现 | `internal/spec` |
| 路径与版本的守卫 A1-a、A1-c（§12.14 第 1 条） | 实现（读取不执行候选） | `internal/collect/guard.go` |
| 阶段 A、B 的检查，闭包，I-2、I-3 | 实现 | `internal/collect` |
| I-4（Hash1）、受控读取 I-5 与 `ReadDir` 视图 | 实现 | `internal/modsnap` |
| 条件差异检查第 1 ~ 5 步（含 `web/embed.go` 的四项核对） | 实现 | `internal/cond` |
| 装载与类型检查（§12.10 装载第 1 ~ 9 步）、词法预筛、对象引用扫描、X1 / X4 与信号实参的诊断、I-6 (d)、I-7、表的自检 | 实现 | `internal/typecheck` |
| 与静态候选的对照 K、LQ-6 的观察包 | 实现 | `internal/collect/b5.go` |
| 原始记录、派生记录、记录链（写者、诊断任务、截止复核、到期锁存、截止资格、父端的最后消费）、manifest | 实现 | `internal/collect/record.go`、`internal/chain` |
| 三个证据域、外层的判定、四种结论与退出状态 | 实现 | `internal/controls`、`internal/verdict`、`internal/job` |
| 对照 LC-2 ~ LC-8 的夹具与完整预期；LC-1、LC-10 | 实现 | `internal/controls`、`internal/job` |
| 集成接线：`TestMain`、唯一的测试函数、收尾 | 实现 | `internal/job`、`run`（带构建标签） |
| workflow | 只编写，供审查 | `.github/workflows/c9-evidence.yml` |
| §12.14 第 3 ~ 13 条的工具控制协议（管道、回收、pidfd、终止） | **不实现**（真实调用停止） | — |
| LC-9（伪工具） | **不实现**；记为“适用而未执行”，即不命中 | `internal/controls/lc5.go` 的 `LC9` |
| B′、规范表示、批准登记、C 的匹配；仓库代码的规则 | 不实现（LE-9） | — |

## 2. 计划 §8.1 的事项

### 2.1 模块、依赖、导入闭包与工具链（LE-2）

- 模块 `github.com/XKHoshizora/specmetry/prototype/c9evidence`，`go 1.27`，**没有** `toolchain` 行。
- `require golang.org/x/mod v0.42.0`，没有其他 `require`。
- `go.sum` 恰好两行，都是根模块 `go.sum` 中已有的行（第 42、43 行）：
  - `golang.org/x/mod v0.42.0 h1:ICeL3t2L30ljSPHHxDnR4fqVcBt/zR2zJAb9juavmQg=`
  - `golang.org/x/mod v0.42.0/go.mod h1:u5fzIrRInkL8MaO1cbRkUHT64v303YIT5KEqpNr7jRs=`
- `go.sum` 由固定工具链的 `go mod tidy`（`GOPROXY=off`，只用本机缓存）生成。没有出现新的模块或新的行。
- 实际的导入闭包：标准库，加上 `golang.org/x/mod/sumdb/dirhash`（以固定工具链对工具模块自身执行 `go list -deps` 取得，没有对 R 调用）。
- `golang.org/x/sys` v0.49.0 已获采纳（LE-2），但它只用于 §12.14 的 pidfd。本阶段没有控制协议，所以不导入它，`go.mod` 中也没有它。
- `Hash1` 只消费已经读取的内存字节（`internal/modsnap/hash1.go`）：I-4 先逐个打开、读到 EOF、关闭并保存真实的关闭结果，然后才以名字列表调用 `dirhash.Hash1`，它的打开函数只交出内存中的字节。
- 工具链：本地以负责人决定下载的 go1.27.3 编译与测试（§190 的工具链补充决定；`bin/go.exe` 的 SHA-256 `83b7ba66…999d`），一律以 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` 直接执行这一份 `go.exe`。

### 2.2 R 与工具目录的接线（§4.2）

- R 由 workflow 以环境变量 `C9EVIDENCE_R` 显式给出（`${{ github.workspace }}`）。工具要求它是**整洁的绝对路径**，不从测试的工作目录推断；否则 R 域只记“没有给出 R”，结论不能是 3 或 4（`internal/job/job.go` 的 `runR`）。
- 对 R 的核对：
  - A4 读 `R/go.sum` 与 `R/go.mod`，都是一次读入，保留真实的关闭结果；`go.mod` 只记录身份；
  - A5 对 `R/vendor`、`R/go.work` 做 `Lstat`；
  - B3 要求主模块包的 `Module.Path` 等于 `github.com/XKHoshizora/specmetry`、`Module.Dir` 等于 R（按 `filepath.Clean` 之后的字符串比较，不区分大小写的放宽与符号链接的解析都不做）。
- 三种调用的 `CallSpec`：
  - 可执行文件是 A1-a 核对过的候选的绝对路径；
  - 参数是 `spec.Args` 的原样数组，T2 是其中一个元素；
  - `Cmd.Dir` 为 R；
  - 工具设定的环境只有七个变量：`GOOS`、`GOARCH`、`CGO_ENABLED=0`、`GOFLAGS=`、`GOWORK=off`、`GOTOOLCHAIN=local`、`GOPROXY=off`，原始记录只记它们。
- 记录目录由 `C9EVIDENCE_RECORDS` 给出（`${{ runner.temp }}/c9evidence-records`），运行之前不存在，由写者以 `Mkdir` 建立。
- 独立模块只负责工具自己的构建与测试；它的单元测试从不以 R 运行采集。

### 2.3 实际的消费者（N-143-1）

- `cond.NewContext` 对每个条件与来源只构造**一个** `build.Context`：同一个值既交给 `MatchFile`，也生成派生记录 `ContextRecord`（`GOOS`、`GOARCH`、`CgoEnabled`、`BuildTags`、`ToolTags`、`ReleaseTags`、`Compiler` 与第 2 种调用标准输出的 SHA-256）。
- `OpenFile` 与 `ReadDir` 都由受控读取提供，不回退到 `os.Open` 或 `os.ReadDir`。标准库与仓库的目录没有内容核对，由记录读取的 `Plain` 提供，`OpenFile` 只接受该目录的直接文件。
- 对标签敏感的夹具：`internal/cond/cond_test.go` 的 `TestThirdPartyAndStd`。带 `//go:build amd64.v2` 的文件在宿主默认的 `ToolTags` 下与在含 `amd64.v2` 的 `ToolTags` 下分类不同，第 1 步的一致性核对能发现这一差异。它只是单元测试，不是正式对照；要不要把它加入 LC-6，交固定实现的审查决定（不改变正式对照的数量）。
- 这些都不认证 cmd/go 的视图（G4-Q10 不变）。

### 2.4 Unix 上参数传递的固定源码路径（§1.2 第 2 项）

**没有核对**。真实调用者没有实现，参数不会到达任何进程。Windows 与 Unix 上 T2 作为一个参数的传递，都留待真实调用者的实现与审查。

### 2.5 对照的夹具、完整预期与适用表

完整预期写在工具的代码中，与被测代码分开。规范的行是 `<检查> <状态>`，以及每条失败一行的 `<检查> fail <J> <项> <类型> <来源>[#序号]`，次数由相同的行数表示。站点中的临时路径与细节文字不参加比较。外层只在被测实例进入终态之后判定，规则见 `internal/controls/controls.go` 的 `Judge`：

- 有进入记录，结果与完整预期逐行相同，外层的准备与清理都成功 → **命中**；
- 没有进入记录、结果有任何不同（包括多出的真实错误）、准备失败，或适用而没有执行 → **不命中**；
- 实例没有进入终态，或外层的清理失败或未知 → **未决**；
- 适用表中预先写定的平台 → **不适用**。

| 对照 | 变体 | 夹具与预期的来源 |
|---|---|---|
| LC-2 | 改一个字节；多一个文件；含一个符号链接 | 夹具模块 `example.com/lc2`。go.sum 的 h1，与两个改动内容应得的 h1，都在工具之外按 Hash1 的定义独立计算（`lc23.go` 的常量）。预期分别为 J-4（并记下应得的 h1）、J-4、J-2（不计算 h1） |
| LC-3 | 名字不在 M 中；重新读取得到不同的字节；句柄不是普通文件；真实 `Close` 成功之后注入 | J-4、J-4、J-2，以及 J-1（来源为注入 #1，另记“注入单独保存 #1”）。真实 `Close` 失败时不注入（V-29；单元测试另有反向的用例） |
| LC-4 | 77 个字节变体（`decode.LC4Variants`） | 每个变体写定成功（附逐字段的解码值）或失败（J-3 与错误种类）；变体清单见 `internal/decode/variants.go` |
| LC-5 | 计划 §5.3 的八个用例 | 夹具 R_f 与 Q；Q 的 h1 独立计算。投影按 §5.3 的表逐项生成，并比较三种调用的次数，以及“第 3 种调用失败时没有来自它的派生记录”（`lc5.go`） |
| LC-6 | 健康；加入带 `exec.Command(...).Run()` 的 `!nowebui` 文件；加入 `//go:build cgo` 的文件 | 夹具主模块 `example.com/lc6`，例外清单为空集；后两项只在第 2 步失败（J-3，“生产变体专有的文件不在例外清单中”） |
| LC-7 | 对象扫描 | 夹具包加上自写的标准库存根（不读取真实的 GOROOT）。五种写法各一个发现，行与列在工具之外由文本检索独立计算；接口派发没有发现；信号实参的分类按 GOOS 写定（linux 上 `syscall.Signal(0x11)` 是“常量 SIGCHLD”，darwin、windows 上是“其他常量”） |
| LC-8 | `W` 之后开始工作单元；`W` 之后才发布的成功结果 | 前者：准入拒绝（J-1，来源为预算），配置的全部检查未评估；后者：第 3 种调用以退出 0 结束而发布在 `W` 之后 → A3 为 J-1，B、S、K 未评估 |
| LC-9 | 控制协议（伪工具） | **未实现**：适用而未执行，即不命中 |

- **适用表**：全部对照在三个平台上都是“适用”，没有写“不适用”的平台。LC-2 的符号链接变体在 Windows 上同样适用：不能创建链接是准备失败，判为不命中（计划 §5.6 第 4 项），不改写为不适用。本机（Windows，没有创建符号链接的权限）上这一变体就是这样；GitHub 的 Windows runner 能否创建，没有取证。
- **LC-1**（R 记录上的正对照）：在 R 的每个配置中，闭包里的每个模块都算出了 h1，即为命中；h1 与 go.sum 不等是 R 的 J-4，不算不命中；R 没有到达或没有完成 I-4 时为未决（R 本身已经是未完成）。这是对计划 LC-1 一行的实现解释，**提请审查**。
- **LC-10**：`TestMain` 开始时核对 T2 的 688 字节与 SHA-256，写入原始记录 `identity/T2`；不符为第 1 级。

### 2.6 三个域的登记与持有；被测实例的到期点（§5.4）

- **R**：每个配置一个 `collect.Result`，持有检查的登记表、调用记录、I-4 的快照与受控读取，直到进程结束。预算的准入拒绝按配置归入 R 的失败。
- **F**：每个变体一个被测实例，带自己的检查登记表与记录。需要时钟的实例（LC-5、LC-8）使用自己的假时钟，由它给出到期点，不占用包的 `D`。判定与全部结果写入 `controls.jsonl`。
- **O**：外层的每一次准备与清理都先以 `RegisterOp` 登记，再在锁外执行，最后发布真实的结果。快照点等待它们全部进入终态。

### 2.7 记录与收尾（§6.4）

- **文件**：`raw.jsonl`、`derived.jsonl`、`controls.jsonl`、`closeout.jsonl`，加上 `manifest.txt`。
  - 每个记录文件的第一行是 `{"c9evidence":"<名字>","version":1}`；
  - 记录项是一个 JSON 对象一行，UTF-8、LF；
  - `manifest.txt` 每个记录文件一行 `<名字> <字节数> <SHA-256>`，按名字排序。
- **记录项**：
  - 原始记录有：工具链（守卫的值）、每次调用（参数数组、工具设定的环境、工作目录、两路输出与它们的 SHA-256、退出状态、时刻、控制的失败与注入）、`R/go.sum` 与 `R/go.mod` 的身份、每个模块的 M（名字、字节数、SHA-256、关闭结果）与 h1、T2 的绑定、运行的身份（`GITHUB_SHA`、`GITHUB_REF`、`GITHUB_WORKFLOW_REF`、`GITHUB_WORKFLOW_SHA`、`RUNNER_OS`、`RUNNER_ARCH`）。
  - 派生记录有：闭包、交给 `MatchFile` 的 `Context`、受控读取的消费记录、发现与诊断、与候选的对照、每个配置全部检查的结果与进入记录。
  - 派生记录写明父记录的名称与 SHA-256，只追加。
  - 记录不含第三方源码的字节。
- **封口**：父端在快照点、在同一次锁下，追加收尾记录项（暂定结论与原因）并封口；写者只写到封口为止，封口之后到达的记录项留在内存中（`chain.Log`）。写者与诊断任务自己的发布（字节数、哈希、资格、终态）不是记录项。
- **诊断行**：都以 `c9evidence: ` 开头，依次为：
  1. `provisional <n> <名字>`；
  2. 失败与未决的计数和逐条；
  3. 写者成功时：每个文件的 `record <名字> <字节数> <SHA-256>`、`manifest manifest.txt <字节数> <SHA-256>`，最后是终结行 `end manifest-sha256 <SHA-256> provisional <n> <名字>`；
  4. 写者失败时：它发布的错误，没有终结行。
- **截止**：写者与诊断任务在每一次建立、写入或关闭之前，都在外层登记表的锁下复核截止；资格在发布的同一次锁下确定（锁存未置且时刻早于 `D` 为及时，否则为晚到并置锁存）；父端的最后消费按第 5 步的唯一顺序 (a)/(b)/(c) 处理（`internal/chain/closeout.go`）。
- **由记录链自身降低暂定结论**（只降不升，原因记在收尾记录项的 `caps` 中，并作为失败行写出）：
  - 降为 1：`m.Run` 非零、日志或登记表的缺陷；
  - 降为 2：快照点上仍有记录没有进入终态、有外层操作没有成功、写者已经发布错误或非成功的终态。
- **退出状态**：最终结论为 3 或 4 时为 0，否则为 1（LE-10）。`TestMain` 只以收尾的返回值调用 `os.Exit`。配置本身不完整（例如没有给出记录目录）时以 2 退出，这时没有任何证据。

### 2.8 workflow（§6.3、§9.1）

`.github/workflows/c9-evidence.yml`：

- **触发**：只在推送 `evidence/c9-run-1` 时运行，没有 `pull_request`、`workflow_dispatch` 或 `schedule`；不设 `paths` 过滤。
- **基本设置**：`permissions: contents: read`；`concurrency` 按分支分组，不取消进行中的运行；三个 runner 的矩阵，`fail-fast: false`；`timeout-minutes: 20` 只是外层兜底。
- **步骤**：
  1. 检出（`actions/checkout@v7.0.1`）；
  2. `actions/setup-go@v7.0.0`，`go-version: 1.27.3`，`cache: false`；作业环境 `GOTOOLCHAIN=local`；
  3. 核对 HEAD 等于 `GITHUB_SHA`、工作区干净，打印 HEAD、树、`GITHUB_WORKFLOW_REF`、`GITHUB_WORKFLOW_SHA` 与 `go version`；
  4. LE-4 的 `go mod download`，打印 `go.mod`、`go.sum` 的 blob，并再次核对工作区干净；
  5. 在 `prototype/c9evidence` 运行 `go test -tags c9evidence_run -count=1 -v ./run/`，以 `C9EVIDENCE_R`、`C9EVIDENCE_RECORDS` 传入 R 与记录目录；
  6. `if: always()` 上传记录目录（`actions/upload-artifact@v7.0.2`，保存 90 天）。
- 运行分支不存在，本阶段不建立也不推送它。常规 `ci.yml` 没有改动。根模块的 `./...` 不包含这个模块，所以常规 CI 不运行它的测试。

## 3. 实现中的解释与选择（提请审查）

| # | 事项 | 实现的处理 |
|---|---|---|
| 1 | I-4 中名字含 LF 的类别（方法记录只写“即失败”） | 记 J-2（来源异常），这一模块不计算 h1 |
| 2 | 条件差异检查第 4 步中生产变体专有文件导入 `"C"` | 不计为“非标准库的导入”；这样的文件已在第 2 步失败 |
| 3 | `//go:build` 出现在已闭合的 `/* */` 之后 | 与固定的 `go/build` `parseFileHeader` 一致：仍然接受 |
| 4 | I-7 的约束行 | 保留该行原样的字节，只去掉行尾的 LF 与 CR（`go/build` 内部比较的是去掉首尾空白的行） |
| 5 | I-6 (c)：候选是否都经判断 | 分类时记下交给 `MatchFile` 的每个候选（`Judged`），与 H 记录中该目录的候选逐项比较；被文件名规则排除的候选由 `MatchFile` 自己判断，不另读文件头 |
| 6 | B4 中有的模块因 I-2 / I-3 不成立而没有评估 | 任一模块失败即 B4 失败；否则有未评估的模块即 B4 未评估 |
| 7 | `R/go.mod` 的读取错误 | 记入 A4 的 J-1（与 `go.sum` 同为 R 的输入读取） |
| 8 | K 的结果 | 比较已进行即 K 成立；与候选是否一致另记为 `consistent` 与差异清单，结论 3 由它推出（不一致不是 J 类） |
| 9 | LQ-6 | 发现位于候选之外的包，或闭包中出现决策稿 §2.7 列出的包（`spec.WatchPackages`），都记为差异 |
| 10 | 类型检查的工具链版本 | 以完整的 `GOVERSION`（`go1.27.3`）比较模块的语言版本是否高于工具链，与 cmd/go 的做法相同 |
| 11 | 前提不成立的包 | 它与所有（传递地）导入它的包都不交给类型检查，记为未评估，不记为导入失败（`collect/tcadapter.go`） |
| 12 | 表中对象在它的行没有列出的平台上被引用 | 扫描失败（J-3），不丢弃这一引用 |
| 13 | 泛型类型的方法作为调用方 | 写作 `(*pkg.T).M`，不带类型实参；对表中对象与全部候选，与 `FullName` 相同 |
| 14 | 第 3 种调用输出零字节或只有空白 | 解码失败（J-3），不当作“没有包” |
| 15 | 最后一次写入在 `D` 之后才发布 | 诊断任务的终态仍是“完成”，但资格为晚到，最终结论为 2（§6.6 情形 4）；还有写入要做时则走到期出口（情形 5） |
| 16 | LC-1 的判定 | 见 §2.5 |

## 4. 测试与结果

- **命令**（在 `prototype/c9evidence`）：`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS= <go1.27.3 的 go.exe> test -count=1 -v ./...`；另有 `vet ./...`、`vet -tags c9evidence_run ./...`、对 linux、darwin、windows 三个 GOOS（amd64 与 arm64）的交叉 `vet`、`test -tags c9evidence_run -c ./run/`（只编译正式入口）、`gofmt -l`、`mod verify`。
- **结果**（2026-11-14，Windows 本机，go1.27.3 windows/amd64）：
  - 12 个包中 11 个有测试，全部 ok；顶层测试 128 个通过、0 个失败，子测试 268 个通过。
  - 跳过 1 个：`TestTakeNegatives/symbolic_link`，本机没有创建符号链接的权限。
  - vet、交叉 vet 与 gofmt 都干净，正式入口可以编译，`mod verify` 为 all modules verified。
- **各包的覆盖要点**：
  - `decode`：77 个 LC-4 变体与各种严格解码的拒绝；
  - `chain`：§6.6 的七个情形与五种状态表的每一行、半构造、各种写入与关闭的失败、封口之后持续追加、恰好在 `D` 的资格为晚到、晚到的成功不恢复结论；
  - `typecheck`：LC-7 的形式、`unsafe`、`"C"`、`ImportMap`、环、类型错误、语言版本、裁剪函数体、I-6 (d)、I-7、准入、表的自检；
  - `modsnap`：Hash1 与独立计算的 h1 相同、I-4 与受控读取的负向用例、真实关闭错误不被注入掩盖；
  - `cond`：五步与例外的四项核对、对标签敏感的分类；
  - `controls`：LC-2 ~ LC-8 的全部变体判为命中（符号链接一项在本机按设计判为不命中），LC-9 为不命中；
  - `job`：健康时最终结论为 4、退出 0，manifest 与终结行一致；包含 LC-9 时为无效；停止调用者时为未完成；没有 R、`m.Run` 非零、两个域没有运行、T2 不符，各按预期判定。
- **这些都不是**：正式的 LC 对照运行；真实输入上的取证；在 Linux 或 macOS 上的运行（只在 Windows 本机运行过，另两种 GOOS 只做了交叉 `vet`）；`-race` 的检查（本机不可用）。

## 5. 没有实现、没有执行，或依赖 E-10 (ii) 的部分

| 事项 | 状态 | 前提 |
|---|---|---|
| 真实调用者与 §12.14 第 3 ~ 13 条的控制协议（登记先于运行、描述符的责任、唯一的回收者、到期与终止、pidfd 的状态机、成功的判据、`D` 的出口中与工具子进程有关的部分） | 没有实现。停止调用者对每次调用都发布“真实调用已停止（E-10 (ii) 未确认）”，记 J-1 | E-10 (ii) 的确认；实现时导入 `golang.org/x/sys` v0.49.0 |
| LC-9 与平台控制（Linux 的 pidfd、Windows 的进程句柄、darwin 不发终止） | 没有实现，也没有执行；每次正式运行都因此为“无效” | LE-5 ① 伪工具的授权，以及上一项 |
| T2 作为一个参数传到 `go`（Windows 的命令行拼接与拆分，Unix 的参数数组） | 没有核对，也没有运行 | 真实调用者 |
| 真实输入的兼容性：CI 上 `go env`、`go list` 的实际输出与省略，真实模块缓存的内容与类型，真实 GOROOT 的标准库装载与类型检查，三个平台的路径表示（I-3） | 没有执行。解码与核对只经字节夹具与假调用者验证 | 运行的授权（以 E-10 (ii) 为前提）；LE-5 ② 夹具上真实 `go` 调用的授权 |
| 120 秒预算、耗时与内存（LQ-8、G4-Q7） | 没有测量 | 运行 |
| LC-1 在真实记录上的正对照；LC-2 的符号链接在 GitHub runner 上能否创建 | 没有执行 | 运行 |
| 记录目录、artifact 与 §6.5 的完整条件在三个 runner 上的表现；`GITHUB_SHA` 与授权写定的 C、T 的核对 | 没有执行 | 运行 |
| `-race` 下的数据竞争检查 | 没有执行（本机不可用） | Linux 上的运行或另行安排 |

## 6. 下一阶段最少须决定的事项

1. **E-10 (ii)**：是否对实际的三种调用确认“不产生后代”的限定前提，以及是否接受“清理未知即失败（未完成）、如实报告、不作补救”与工具进程可能残留的代价（计划 §10.2）。不确认时，真实调用者与控制协议都不能实现到可以运行。
2. **LE-5 ① 与 ②**：是否允许伪工具（测试二进制的再执行，用于 LC-9 与控制协议的测试），以及是否允许在夹具上进行真实的 `go` 调用（以第 1 项为前提）。
3. 在前两项之后，是否授权实现 §12.14 的控制协议（它须导入 `golang.org/x/sys`）。
4. 本实现记录与固定代码点的独立审查，包括第 3 节列出的解释，以及对标签敏感的夹具是否加入 LC-6。
5. N2 的两项（LE-4 与 G4-Q9 的模块缓存准备；声明的范围），以及运行的授权（写定 C 与 T），都在之后。
