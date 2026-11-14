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

完整预期写在工具的代码中，与被测代码分开。规范的行是 `<检查> <状态>`，以及每条失败一行的 `<检查> fail <J> <项> <类型> <来源>[#序号] @ <站点>`，次数由相同的行数表示（按 Review 147 的 B-147-4 加入站点）。站点中的夹具临时目录写作 `<fx>`、本配置写作 `<cfg>`，其余原样；只有失败的细节文字不参加比较。规范行由结构化的被测实例生成，不过滤任何失败；被测实例本身另外完整保存（§7.4）。外层只在被测实例进入终态之后判定，规则见 `internal/controls/controls.go` 的 `Judge`：

- 有进入记录，结果与完整预期逐行相同，外层的准备与清理都成功 → **命中**；
- 没有进入记录、结果有任何不同（包括多出的真实错误）、准备失败，或适用而没有执行 → **不命中**；
- 实例没有进入终态，或外层的清理失败或未知 → **未决**；
- 适用表中预先写定的平台 → **不适用**。

| 对照 | 变体 | 夹具与预期的来源 |
|---|---|---|
| LC-2 | 改一个字节；多一个文件；含一个符号链接 | 夹具模块 `example.com/lc2`。go.sum 的 h1，与两个改动内容应得的 h1，都在工具之外按 Hash1 的定义独立计算（`lc23.go` 的常量）。预期分别为 J-4（并记下应得的 h1）、J-4、J-2（不计算 h1） |
| LC-3 | 名字不在 M 中；重新读取得到不同的字节；句柄不是普通文件；真实 `Close` 成功之后注入 | J-4、J-4、J-2，以及 J-1（来源为注入 #1，另记“注入单独保存 #1”）。真实 `Close` 失败时不注入（V-29；单元测试另有反向的用例） |
| LC-4 | 77 个字节变体（`decode.LC4Variants`） | 每个变体写定成功（附逐字段的解码值）或失败（J-3、错误种类与解码器停下的位置 `At` 作为站点）；变体清单见 `internal/decode/variants.go` |
| LC-5 | 计划 §5.3 的八个用例 | 夹具 R_f 与 Q；Q 的 h1 独立计算。投影按 §5.3 的表逐项生成，并比较三种调用的次数、“第 3 种调用失败时没有来自它的派生记录”，以及实例记录的来源链能否解析（`provenance ok`，§7.4）（`lc5.go`） |
| LC-6 | 健康；加入带 `exec.Command(...).Run()` 的 `!nowebui` 文件；加入 `//go:build cgo` 的文件 | 夹具主模块 `example.com/lc6`，例外清单为空集；后两项只在第 2 步失败（J-3，“生产变体专有的文件不在例外清单中”） |
| LC-7 | 对象扫描 | 夹具包加上自写的标准库存根（不读取真实的 GOROOT）。五种写法各一个发现，行与列在工具之外由文本检索独立计算；接口派发没有发现；信号实参的分类按 GOOS 写定（linux 上 `syscall.Signal(0x11)` 是“常量 SIGCHLD”，darwin、windows 上是“其他常量”） |
| LC-8 | `W` 之后开始工作单元；`W` 之后才发布的成功结果 | 前者：准入拒绝（J-1，来源为预算），配置的全部检查未评估；后者：第 3 种调用以退出 0 结束而发布在 `W` 之后 → A3 为 J-1，B、S、K 未评估 |
| LC-9 | 控制协议（伪工具） | **未实现**：适用而未执行，即不命中 |

- **适用表**：全部对照在三个平台上都是“适用”，没有写“不适用”的平台。LC-2 的符号链接变体在 Windows 上同样适用：不能创建链接是准备失败，判为不命中（计划 §5.6 第 4 项），不改写为不适用。本机（Windows，没有创建符号链接的权限）上这一变体就是这样；GitHub 的 Windows runner 能否创建，没有取证。
- **LC-1**（R 记录上的正对照）：在 R 的每个配置中，B4 应检查的每个闭包模块（`Result.Modules`，不论快照是否产生）都算出了 h1，即为命中；h1 与 go.sum 不等是 R 的 J-4，不算不命中；闭包未知、有模块没有算出 h1，或结果在 `W` 之后才发布时为未决，并逐项列出（按 Review 147 的 N-147-1 修正，§7.5）。这是对计划 LC-1 一行的实现解释，**提请审查**。
- **LC-10**：`TestMain` 开始时核对 T2 的 688 字节与 SHA-256，写入原始记录 `identity/T2`；不符为第 1 级。

### 2.6 三个域的登记与持有；被测实例的到期点（§5.4）

- **R**：每个配置一个 `collect.Result`，持有检查的登记表、调用记录、I-4 的快照与受控读取，直到进程结束。预算的准入拒绝按配置归入 R 的失败。
- **F**：每个变体一个被测实例，带自己的检查登记表与记录。需要时钟的实例（LC-5、LC-8）使用自己的假时钟，由它给出到期点，不占用包的 `D`。作业在内存中持有每个变体的判定与结构化的实例（`FRecord`），并把它们连同发布的时刻与及时资格写入 `controls.jsonl`（§7.4）。
- **O**：外层的每一次准备与清理都先以 `RegisterOp` 登记，再在锁外执行，最后发布真实的结果。快照点等待它们全部进入终态。

### 2.7 记录与收尾（§6.4）

- **文件**：`raw.jsonl`、`derived.jsonl`、`controls.jsonl`、`closeout.jsonl`，加上 `manifest.txt`。
  - 每个记录文件的第一行是 `{"c9evidence":"<名字>","version":1}`；
  - 记录项是一个 JSON 对象一行，UTF-8、LF；
  - `manifest.txt` 每个记录文件一行 `<名字> <字节数> <SHA-256>`，按名字排序。
- **记录项**：
  - 原始记录有：工具链（守卫的值）、每次调用（参数数组、工具设定的环境、工作目录、两路输出与它们的 SHA-256、退出状态、时刻、控制的失败与注入）、`R/go.sum` 与 `R/go.mod` 的身份、每个模块的 M（名字、字节数、SHA-256、关闭结果）与 h1、T2 的绑定、运行的身份（`GITHUB_SHA`、`GITHUB_REF`、`GITHUB_WORKFLOW_REF`、`GITHUB_WORKFLOW_SHA`、`RUNNER_OS`、`RUNNER_ARCH`）。
  - 派生记录有：闭包、交给 `MatchFile` 的 `Context`、受控读取的消费记录、发现与诊断、与候选的对照、每个配置全部检查的结果与进入记录。
  - 派生记录写明父记录的名称与 SHA-256，只追加：闭包绑定第 3 种调用；`Context` 绑定第 2 种调用；消费记录绑定第 3 种调用与本配置的全部 I-4；发现绑定消费记录与第 3 种调用；对照绑定发现；配置的检查记录绑定以上全部与仓库文件、工具链。收尾之前由 `collect.VerifyProvenance` 核对，缺失或错误的关联是工具缺陷（第 1 级），不因 manifest 完整而得到有效结论（Review 147 的 B-147-4，§7.4）。
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
| 3 | `//go:build` 出现在已闭合的 `/* */` 之后 | 与固定的 `go/build` `parseFileHeader` 一致：写在**下一行**时仍然接受；与块注释写在**同一行**（`/* x */ //go:build a`）时只是该注释行的余下部分，不接受（按 Review 147 §4 第 3 项精化；`cond`、`typecheck` 两处各有用例） |
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
| 16 | LC-1 的判定 | 见 §2.5；按 Review 147 的 N-147-1 改为以应检查的模块集合判定（§7.5） |

## 4. 测试与结果（第一阶段提交 `84c7718` 时；按 Review 147 修复之后的结果见 §7.7）

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

## 7. 按 Review 147 的定向修复（§191；已修复，待 Review 148 复审）

Review 147 对第一阶段实现为技术 NO-GO（B-147-1 ~ 4）。负责人只授权在 PR #56 内做这四项的定向修复与适用回归，以及 N-147-1 ~ 3 与两处措辞的精化（review-resolution §191）。以下只说明修了什么、由哪些回归覆盖、跑出了什么；阻塞是否关闭由 Review 148 裁定。

### 7.1 B-147-1：父端脱离未返回的分析

- **位置**：`internal/job/job.go` 的 `RunDomains`、`runWork`、`WorkRecord`、`publishVariant`、`runR` 与 `copyState`。
- **修改**：
  - 测试函数调用的 `RunDomains` 先在作业的锁下登记工作任务（`WorkRecord`，状态 registered），再启动它；工作任务在自己的 goroutine 中依次运行 F 域与 R 域。
  - 父端（测试 goroutine）只等工作任务发布“完成”，或时钟到 `W`，取先到的一个。`W` 先到时，父端记下“放弃等待”并返回，所以 `m.Run` 会返回，既有的收尾与 `D` 出口都可以到达。解析器与 `types.Checker` 不被中断，预算不延长，也不依赖 `go test` 或 workflow 的外层超时。
  - 到期之后：工作任务之后的每个单元都被准入拒绝（`W` 之后不准入）；未完成的单元、输入与晚到的结果由作业继续持有，直到进程结束。
  - 每个 F、R 结果在发布的那一次锁下确定及时资格：早于 `W` 而且父端仍在等待，才是及时的。已准入不等于及时：在 `W` 之前准入、在 `W` 之后才发布的结果是晚到的。晚到的命中按未决处理，晚到的配置结果另记一条 J-1（“在 `W` 或之后发布”）；晚到的成功不恢复有效性，真实错误照样保存。
  - 收尾的快照点把未完成的工作任务当作未进入终态（记录链因此把暂定结论降为 2），并写明放弃等待时正在进行的单元。
- **回归**（都经过 `RunDomains` → `Finish` 的实际接线；进程内假调用者、可控时钟与受控阻塞；每个等待都有自己的有界失败出口，阻塞由 `t.Cleanup` 释放；不启动 helper 子进程）：
  - `TestB1471ParentLeavesAStuckAnalysisAtW`：类型检查已准入而一直不返回，时钟到 `W` 时 `RunDomains` 返回，`Finish` 给出非 3、4 的结论。
  - `TestB1471LastUnitCompletesAfterW`：最后一个分析单元在 `W` 之后才返回健康的结果，配置结果记为晚到，结论不是 3 或 4。
  - `TestB1471HealthyTimely`：全部及时完成，结论为 4，工作任务与每个配置都是及时的。
  - `TestB1471RealErrorAfterTheParentStopped`：父端在 `W` 结束等待并作出结论（不是 3 或 4）之后，工作任务才发布真实错误；这个错误作为不及时的结果保留在 R 的记录中，工作任务记为放弃等待、不及时。
  - `TestB1471DExitReachable`：分析与诊断输出都卡住，`Finish` 仍在 `D` 经到期出口返回。

### 7.2 B-147-2：manifest 的持续持有

- **位置**：`internal/chain/writer.go` 的 `Holding`、`writeManifest`、`closeAll`、`markCloseStarted`、`closedHolding`、`Held`。
- **修改**：
  - `Create` 一返回句柄，写者就持有这个对象（任务状态中的 `manifest`），并发布 `Holding{Holder: "writer"}`。
  - 发布的持有状态分开四件事：持有者、`Close` 是否启动、`Close` 的真实结果、是否确认释放（`Close` 返回 nil 才算）。
  - 正常路径只关闭一次；到期出口不启动 `Close`，句柄仍由写者持有；`Close` 出错时释放未知，不重复关闭，对象仍由写者持有。GC 与进程退出都不算确认释放。
  - 记录文件原来在 `Close` 出错之后会把句柄置空，按同一原则改为继续持有，并发布同样的持有状态。这是与 manifest 同一类的缺陷，修改范围只在写者之内。
- **回归**（沿真实的写者路径，以假文件系统阻塞或注入）：
  - `TestB1472ManifestCreateThenExpiry`：`Create` 在 `D` 之后返回，之后的决策点走到期出口；不写入，不关闭；对象由写者持有，`Create` 的结果（晚到）保留。
  - `TestB1472ManifestWriteThenExpiry`：`Write` 在 `D` 之后返回；不关闭；对象由写者持有，晚到的写入结果保留。
  - `TestB1472ManifestCloseFails`：`Close` 失败，恰好调用一次；释放未知，对象仍由写者持有，真实错误保留。
  - `TestB1472RecordCloseFailsKeepsTheHandle`：一个记录文件的 `Close` 失败，只有它仍被持有，其余文件确认释放。
  - `TestB1472NormalCloseReleases`：正常关闭一次，确认释放，写者不再持有任何句柄。

### 7.3 B-147-3：完整保存已经发生的错误

- **读取**：`internal/collect/guard.go` 的 `readClosed` 改为 `ReadWhole`，返回结构化的 `ReadResult`，分别保留打开、读取与真实 `Close` 的结果。两个实际的调用者都消费全部结果：守卫（`VERSION` 与候选可执行文件）把每个错误各记一条；A4（`R/go.sum`、`R/go.mod`）把每个错误各记一条 J-1，读取错误不再遮住关闭错误。
- **类型检查的提前出口**：`internal/typecheck/typecheck.go` 的 `markBudgetKeeping`。准入在包或文件处被拒绝时，先把这个包已经观察到的错误（导入环、语言版本、读取、解析、包名、约束行）保存为对应检查的结果，只把没有执行的检查标为“未评估：预算”。预算失败仍是报告中独立的 `Budget` 结果，不覆盖此前的真实失败；主分类的优先级不变。
- **回归**：
  - `TestB1473ReadAndCloseErrorsBothKeptByA4`、`TestB1473ReadAndCloseErrorsBothKeptByTheGuard`：读取与关闭同时失败，两个错误都在。
  - `TestB1473ParseErrorKeptWhenTheNextFileIsRefused`：a.go 有语法错误，处理 b.go 之前到期，`Load` 保留 a.go 的解析错误。
  - `TestB1473ConstraintErrorKeptWhenTheNextFileIsRefused`：a.go 有两条 `//go:build`，`Header` 保留这一错误。
  - `TestB1473VersionErrorKeptWhenAFileIsRefused`：已知的语言版本错误之后到期，`Load` 保留它。
  - `TestB1473BudgetOnlyHealthyControl`：健康的包在 b.go 之前到期，只有预算失败，各检查都是未评估。

### 7.4 B-147-4：来源链与结构化的对照记录

- **来源链**：
  - `collect.Result.Refs` 保存本配置的原始记录引用（仓库文件、工具链、三种调用、每个模块的 I-4）与派生记录的引用。
  - 每条派生记录绑定实际的父记录（§2.7）。
  - `collect.VerifyProvenance` 按追加的顺序核对：每条记录的引用等于它这一行的 SHA-256；每条派生记录至少有一个父项，每个父项都能解析为更早出现、SHA-256 相同的记录；父项属于同一个配置；各类派生记录都绑定了必要的父项。作业在收尾时对 R 的记录调用它，任何问题都是工具缺陷（第 1 级），manifest 完整也不能使结论有效。LC-5 的八个用例与 LC-8 的“W 之后的结果”实例也调用它，结果作为 `provenance ok` 一行进入完整预期（LC-8 的“W 之后的单元”在准入处被拒绝，它的完整预期不含这一行）。
- **F 域的结构化记录**：
  - `controls.Outcome` 携带 `Instance`：完整的检查结果（每条失败的站点、类型、来源、注入序号与细节）、进入记录、假调用者的调用、实例自己的记录行、准入拒绝，以及被测代码的其他原始结果（例如 I-4 的文件清单、受控读取的消费与注入、解码的输出、条件检查的各步、类型检查的完整输出）。
  - 规范行由它生成，保留站点；每条失败一行，不过滤。
  - 作业持有每个变体的 `FRecord`（判定、完整的 `Outcome` 与实例、发布的时刻与资格），并写入 `controls.jsonl`。
- **回归**：
  - `TestVerifyProvenanceHealthy`、`TestVerifyProvenanceRejects`：健康的来源链没有问题；父项缺失、摘要错误、引用另一个配置的记录、绑定了错误的调用、以更晚的记录为父项、行与引用不符，都被拒绝。
  - `TestB1474DerivedRecordsNameResolvableParents`：健康作业的 `derived.jsonl` 中，每条派生记录的父项都能逐项解析。
  - `TestB1474BrokenProvenanceIsInvalid`：加入一条没有父项的派生记录，作业的结论为无效。
  - `TestB1474OnlyTheSiteDiffersIsAMiss`：LC-4 的 `list.malformed` 与 `list.second-malformed` 同为 J-3、`call3-list`、`syntax`，只有解码器停下的位置不同（`$[0]`、`$[1]`）；以对方的期望判定为不命中，以自己的期望判定为命中。两条预期独立写在测试中。
  - `TestB1474SameClassAtTheWrongSiteIsAMiss`：以 LC5-19b 的期望判定 LC5-20a 的实例（同为 B1 的 J-2 不符），判为不命中。两者的检查（`B1/GOWORK`、`B1/GOFLAGS`）与站点都不同，所以它不能单独说明站点参加比较；只有站点不同的情形由上一项覆盖。
  - `TestB1474ExtraRealFailureIsNotFiltered`：夹具多出一个真实失败（同一检查、同一类别），它作为单独的一行出现，判为不命中。
  - `TestB1474InstanceKeepsWholeResults`、`TestB1474FRecordsKeepInstances`、`TestB1474ControlRecordsKeepSites`：实例保留真实的站点、细节、来源与注入序号，以及调用与记录；经过作业的接线之后仍然保留，并写入 `controls.jsonl`。
- **没有做的**：不保存源码全文，不增加新的密码学认证。快照之后才到达的记录，沿用原计划的保存边界：内存中持有，不保证写入记录文件。

### 7.5 非阻塞的精化

- **N-147-1**：`collect.Result.Modules` 记下 B4 应快照的每个闭包模块；LC-1 逐项核对它们是否都算出了 h1，并列出每个没有算出的模块（`TestN1471LC1UsesTheModulesToCheck`）。
- **N-147-2**：`TestN1472TargetTagsReachTheConsumer` 经真实的采集路径（假调用者）验证目标标签进入实际的消费者。主包有一个只由第 2 种调用的 `ToolTags` 才能满足的文件，三种组合的完整预期独立写定：标签存在且文件列出时成立；标签不在而文件列出时第 1 步失败；标签不在、文件也没有列出时成立。同时核对条件检查记录下来的 `Context` 带着第 2 种调用的标签。它只是单元或集成夹具，不扩大正式的 LC 清单。
- **N-147-3**：原始记录 `identity/run` 改为写入实际的 start、`W`、`D`，以及 `W`、`D` 相对 start 的偏移（`1m50s`、`2m0s`），与预算和收尾使用的值相同（`TestN1473WRecord`）。
- **第 3 项的措辞**：见 §3 第 3 项；`cond` 与 `typecheck` 各有同一行、下一行两种情形的用例。

### 7.6 修复之前的回归结果（如实记录）

| 日志（草稿目录 `s191/logs/`） | 被测代码 | 结果 |
|---|---|---|
| `pre-fix-regressions.log`（4515 字节，SHA-256 `089c92e3c6bafdfdb519edd20c644a495b1a4f388339e8f48732ae55b86024a9`） | HEAD `b44203e`（树 `04b64c8`），工作区只多出三个未跟踪的回归文件 | 8 项失败、1 项通过：类型检查的三项保留错误的回归失败，健康对照通过；A4 与守卫的两项失败；作业层的三项失败（其中 B-147-1 一项当时没有进入类型检查，见下行） |
| `pre-fix-B1471.log`（2114 字节，SHA-256 `99209fedcab6f129ccbce8a3256838fc87fc98e61def9f5c2ce24ddf136756f1`） | 同上，另加测试夹具的时钟修正（`controls.FixtureRepo` 接收作业的时钟） | 失败：分析卡住时 `RunDomains` 没有在 `W` 返回（10 秒有界出口触发之后释放阻塞） |
| `pre-fix-site-only.log`（1237 字节，SHA-256 `5d5d8a6e1c463a8c9412497e9e38724d38a8569807e5111170b66468364a48f6`） | 第一阶段提交 `84c7718` 的工具模块（子树 `c26eb00c…`），以 `git archive` 导出到草稿目录，另加一个探针测试（SHA-256 `7b41aa71…d7d8`）；不在仓库的工作区中运行 | 失败：两个变体的预期都是 `decode fail J-3 call3-list syntax real`，交叉判定为命中（修复之前站点不参加比较） |

- 第一次运行时，B-147-1 的回归失败在“分析没有进入类型检查”：夹具调用者用的是真实时钟，作业用的是假时钟，调用被判为在 `W` 之后发布，停在 A1-b。这是测试夹具的缺陷，修正之后单独重跑，才按预期失败在缺陷本身。
- B-147-2 的回归与来源核验器的测试在修复之前**不能编译**：当时没有可以观察的持有状态，也没有核验器。它们只有修复之后的结果。
- 修复过程中，记录链的一项新回归在全量运行中失败过一次：它要求出口必须是到期出口 (b)，而时钟越过 `D` 之后，父端也可能先看到诊断任务发布的“到期停止”并按 (a) 消费，两者按五种状态表都得到 2。这是断言过窄，已改为接受这两个出口并只断言结论为 2；修改之后这一组回归连续 30 次通过，全部 Review 147 回归连续 20 次通过（这两次重复运行在提交 `bf0a01c` 之前，没有留下日志）。
- 修复提交 `bf0a01c` 之后，核对实现记录时发现原有的错误站点回归同时改变了检查与站点，于是补了只有站点不同的回归，并更正原测试的注释（提交 `ec07dec`，只改测试文件）。它在修复之前的行为由上表最后一行记录。

### 7.7 修复之后的结果

| 日志 | 被测代码 | 结果 |
|---|---|---|
| `final-full-test.log`（43053 字节，SHA-256 `1f3ac1413d0ffd12896ec9c9eeade1ead9f3658935b7a1590cd94f675defd099`） | 提交 `ec07dec8aa5c56f00d861a6b2a07f0ff66cd9a39`（树 `1baced68…`，工具模块子树 `3b6b26dd…`）；测试前后工作区只有本文件未提交的修改，工具模块内没有修改 | `go test -count=1 -v ./...`：11 个有测试的包全部 ok；顶层测试 158 个通过、0 个失败；子测试 271 个通过、0 个失败；跳过 1 个（`TestTakeNegatives/symbolic_link`，本机没有创建符号链接的权限） |
| `final-static.log`（2771 字节，SHA-256 `e0546c77894c63b426db63cf26b3081ab56449ed437569aea8a9fd1b71462996`） | 同上 | `gofmt -l` 为空；`go vet ./...`、带 `c9evidence_run` 标签的 vet、linux / darwin / windows × amd64 / arm64 的交叉 vet、正式入口只编译、`go mod verify` 都成功；第三方导入闭包仍只有 `golang.org/x/mod/sumdb/dirhash` |
| `final-regress147-x20.log`（2136 字节，SHA-256 `bc53f6ce77c365e208b97ce24a2d547c2d1cba2c6d69bdefea636339b411793f`） | 同上 | `go test -count=20 -run 'B147\|N147\|VerifyProvenance\|ConstraintLineAfterABlockComment' ./...`：Review 147 的全部回归各重复 20 次，各包 ok（不带 `-v`，日志中只有包级结果） |
| `post-fix-full-test.log`（42748 字节，SHA-256 `f24e49842f6193f2530ff0b582b82bc9b273f81c734ea8a1d8f37035e4e7b46b`）、`post-fix-static.log`（2699 字节，SHA-256 `e0b0fd5aff53caf833a1a01b59c6304f17fee7e7fadc62f5d7725cbf8b6a3b24`） | 修复提交 `bf0a01c`（子树 `9dae606b…`），工作区干净 | 同样全部成功（顶层 157 个）；被上面三行取代，只作为历史保留 |

- 命令都在 `prototype/c9evidence` 下执行，环境为 `GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=`，工具链为 `C:\Users\hoshi\go\pkg\mod\golang.org\toolchain@v0.0.1-go1.27.3.windows-amd64\bin\go.exe`（SHA-256 `83b7ba66…999d`，`go1.27.3 windows/amd64`）。每份日志的开头写下了命令、环境、工具链、HEAD、树、工具模块的子树与测试前的工作区，结尾写下了退出状态与测试后的工作区。
- 日志保存在仓库之外的会话草稿目录（`C:\Users\hoshi\AppData\Local\Temp\claude\b96df3361c677053\67d963ef-6de7-4877-aebf-656e909830be\scratchpad\s191\logs\`），可供复审读取。日志与被测代码的关系是“测试时读到的 HEAD 与子树”，属于时间上的关联；它们不是不可变产物级的绑定，日志中也没有嵌入测试二进制的哈希。
- 本节之后提交的实现记录是纯文档，不改变工具模块的子树；推送的最终 HEAD 上，工具模块的子树仍是 `3b6b26dd…`。
- 这些日志在会话草稿目录中，重启或清理可能丢失；如须长期保存，要另行决定存放位置。
- 仍然**没有**：Linux、macOS 上的运行（只做了交叉 vet）；`-race`；伪工具；真实的 `go` 调用；正式入口的取证。§5 所列的部分全部不变。

## 8. 开发期间的执行记录（按 Review 147 §6 更正总括的表述）

- **采集器**：没有启动三种真实的调用。正式入口只有停止调用者，它不启动任何进程；正式入口在本地只编译、没有运行。
- **本地 Go 命令**（都以固定的 go1.27.3，在工具模块内执行）：`go build`、`go test`、`go vet`（含交叉 vet）、`gofmt`、`go mod tidy`（`GOPROXY=off`）、`go mod verify`，以及对工具模块自身的 `go list -deps`（取得导入闭包）。下载工具链时执行过 `go version` 与 `go env GOROOT GOVERSION`（§190）。所以“没有运行任何 Go 进程”的说法不成立，上一轮报告中“没有启动任何 `go` 进程”的措辞不准确；准确的说法是：**没有发现采集器启动三种真实调用**。
- **Python**：会话中以 Python 计算夹具的 h1 与 LC-7 的行列（输出用于写定预期）、编辑文件、做链接与格式的检查。其中有失败与修正：一次编辑脚本因 heredoc 中的反斜杠处理而 `AssertionError`，没有写入，改用编辑工具完成。子任务的报告各自写明曾试探性地执行 `python`，没有结果就停止。这里只记录取得的回执，不推断它们之外的行为。
- **Review 147 修复期间的临时探针**：
  - 为挑选只有站点不同的 LC-4 变体，在工具模块的 `internal/decode` 中临时放入一个只列出失败变体的测试文件，运行一次之后删除，没有提交；删除之后工作区中工具模块没有修改。
  - 修复之前的站点行为在仓库之外检查：以 `git archive` 把 `84c7718` 的工具模块导出到草稿目录，加一个探针测试运行（§7.6 最后一行）。
- **开发期间的其他失败与修正**：条件检查测试中块注释与 `"C"` 导入两处预期写错，按固定源码修正；守卫测试的一处预期写错；作业测试中记录文件的计数写错；一次作业测试因记录项带着行尾 LF 而失败，记录链判为缺陷，修正之后通过；一条 shell 命令因多余的 `cat` 等待标准输入而挂起，被停止。最终的测试结果见 §4 与 §7.7，不说成第一次就全部通过。
