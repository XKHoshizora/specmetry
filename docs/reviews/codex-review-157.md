# Review 157：PR #59 首次正式取证准备审查

## 1. 结论

**当前准备变更为技术 NO-GO，暂不建议合并固定对象。** 原因是一项限于新增 F-2 文字的阻塞 B-157-1：把 Windows 原文件名不存在，直接推成扩展名查找失败／A②，遗漏了固定源码确实会尝试的追加扩展名分支。这不是对已关闭的第二阶段实现重新判 NO-GO，也不是因尚未运行正式取证而拒绝准备稿。

Windows F-1 接线在已披露的安装布局与受控 runner 前提下基本成立；FR-1～FR-8 已把政策决定、身份固定、执行和结果审查分开。另有两项非阻塞精化：绝对路径检查的 API 含义，以及打印中的子命令状态。未发现本轮改动导致工具模块、判定、预算或既有持有／回收协议的直接回退。

常规 CI **run 46073216782，attempt 1，8/8 success**，已结束，不是待定。本轮只查询一次当前 HEAD 的运行状态，随后读取其八份既有日志；没有轮询、等待或重跑。因此当前合并建议的障碍是下面的文字问题，不是当前常规 CI 未完成。绿色 CI 没有验证新增正式 workflow 的执行路径。

FR 表可以用于负责人讨论下一阶段决定，但应先消除 B-157-1，再形成可合并的准备基准。**即使准备稿之后 GO／合并，也仍不能建立运行分支或执行正式入口**：N2、用于正式 R 的 E-10(ii)、C/T 与运行授权尚未决定。

## 2. 固定对象与实际增量

审查日期：2026-11-28。开始与结束分别核对本地 Git、工作区、GitHub PR 与远程 refs，身份一致。

| 对象 | 核验值 |
|---|---|
| PR／状态 | XKHoshizora/specmetry #59，OPEN |
| 本地及远程分支 | `prep/wp2.5-c9-formal-run` |
| HEAD／PR head | `dcc01bd247a3c04a16103a5bbe5042c5c4a7a34f` |
| HEAD tree | `e079337f7b4aa0725d7a085007dcaa5d993583e2` |
| PR base／远程 master | `c34baf86bac1524bcfdabf2b6b67f8f2169e79b5` |
| 工具模块子树 | `755ffc0c2465959fa5528d07b5c079154e8e4feb` |
| c9-tool-test.yml blob | `4387acdc683a2014c235f3991bf854a30d6c7c2c` |
| c9-evidence.yml blob | `852c032dc4c47c78a0897d13c1f0969ab5869dc6` |
| ci.yml blob | `8ca60930a5407151358046929738c141d8d52f84` |

已读当前 AGENTS、[Review 156](codex-review-156.md) 全文、[处理记录](../review-resolution.md) §206／§207、[限定取证计划](../wp2.5-c9-limited-evidence-plan.md) 的相关规则及新增 §12、[实施计划](../wp2.5-implementation-plan.md) 的分类与工具守卫接点，并沿 workflow → 正式 Collect → 既有 guard 的路径核对。

实际 base→HEAD 为八文件、266 行增加／17 行删除：`cfb26a8` 只改正式 workflow；`dcc01bd` 只改七份文档／状态文件。技术文字变化是 F-2 与 FR 表，其余是授权、历史及状态同步。工具模块、测试 workflow、常规 CI、根模块代码、依赖、Schema、迁移、冻结正文与验收项没有改变。

按 Git 原始字节比较，base 的 `review-resolution.md` **1505379 字节是 HEAD 1515725 字节的完整相同前缀**，§207 只追加，§206 及旧历史未回改。`git diff --check base..HEAD` 无提示。

Review 156 归档仍为 28381 字节，SHA-256 `cc95043d32f24f6310d3e167d168a96529b9638780e4b97470f49d0502c2ec9c`。GitHub PR #58 已合并，merge 为本轮 base；本地合并对象的父提交依次为 `09e81ee3a35a1fdcfd604f509851f1769e43fd3e`、`02d82edfe41722bcb83a5b856c1c92afeef2cc54`，tree 为 `98ada210c1749fd5479e6ca21193c494f43bd555`。本轮没有重新查询全部历史 CI：旧 SQLite failure、仅 Windows 单次重跑、后续最终 HEAD 与 master CI 的成功，按 §206／§207 登记和 Review 156 已取得的范围分别保留，不冒称本轮重新读取了这些历史作业日志，根因与维护事项仍未关闭。

## 3. B-157-1：F-2 的“文件不存在即 A②”缺少查找失败的前提

**位置**：[实施计划](../wp2.5-implementation-plan.md) §12.12，第 1498 行；[处理记录](../review-resolution.md) §207，第 13394 行。均为固定 HEAD `dcc01bd` 的新增文字。

该段写道：没有扩展名，或扩展名不在 `pathExt()` 中，“文件不存在即返回 `*exec.Error`”，继而将其判为 A②。给定名字不存在并不足以推出这个结果。

独立读取固定 Go 1.27.3 提交 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff`：

- [lp_windows.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec/lp_windows.go) 第 37～57 行：`findExecutable` 在原名检查之后，还按顺序尝试 `file + e`。第 45～46 行甚至明确说明要支持 `foo.bat.exe` 这样的名字。
- 同文件第 84～94、132～139 行：扩展名未直接匹配时进入上述查找；只有查找最终返回错误，才包装 `*exec.Error`。
- [exec.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/os/exec/exec.go) 第 477～480 行：查找成功时缓存实际结果，不设置 `cmd.Err`；第 681～687 行：Start 使用缓存的扩展名结果。

**可成立的源码反例，无需运行**：`PATHEXT=.EXE`；`C:\fixture\probe` 不存在，但 `C:\fixture\probe.exe` 是可启动文件。`exec.Command` 接收前一个绝对路径，`findExecutable` 找到后一个文件，`cmd.Err == nil`，Start 可以实际创建进程。类似地，`probe.odd` 不存在但 `probe.odd.exe` 存在，也不能判 A②。不能按新增文字把这种输入当作“确认尚未创建”并据此释放责任。

错误的内层种类也须有前提：若 `PATHEXT` 非空但只有分号等空项，`pathExt()` 可得到空切片；`findExecutable` 第 38～39 行直接返回 `chkStat` 的错误，不保证无扩展名时一定包装 `ErrNotFound`。外层仍可为 `*exec.Error`，但文中对子错误的无条件描述不成立。

**影响与限定**：本轮只改文字，现有实现以实际 `cmd.Err` 和阶段证据分类，不会因这句话自动改成错误释放。第二阶段使用的“已知 `.exe` 扩展名、该文件不存在”输入也不受追加扩展名反例影响。因此不重开 Review 156。但本轮任务正是准确写定 F-2 的分类前提；新增的确定性分类结论与固定源码矛盾，不能作为后续测试输入或责任判断的依据。

**最小修订方向**：保留已知 `.exe` 的 C 类夹具与现有优先级；把另一分支写成“原名及所有适用的追加扩展名候选均未成功解析，`lookExtensions` 实际返回错误并写入 `cmd.Err` 时，才满足 A②”。若保留内层错误种类，补上非空扩展名集合等条件；否则只承诺保存实际错误。同步 §207 的摘要，不改冻结规则、工具代码或历史原始证据，也不要求新增原生实验。

## 4. F-1：接线成立的范围及两项非阻塞精化

### 4.1 实际顺序与失败传播

[c9-evidence.yml](../../.github/workflows/c9-evidence.yml) 第 71～131 行的顺序为 LE-4 → Windows PATH 步骤 → Collect。新步骤仅 Windows 执行；Collect 没有 `always()` 或 `continue-on-error`，前一步失败时不会执行。上传记录步骤仍为 `always()`，没有 artifact 时只警告；这不把缺证变成有效。

新步骤从 `Get-Command go` 的实际 Source 向上取得安装根，读取该目录本身的 LinkType／Target，不硬编码盘符。Junction 分支先检查数量，再对两个路径的文件做 SHA-256 比较；不存在的目标文件、读取／哈希失败、显式 throw、写 GITHUB_PATH 失败均应使该步骤失败。GitHub 对内置 pwsh 脚本前置 `$ErrorActionPreference = 'stop'`，这些命令均为 PowerShell cmdlet，没有靠后续打印覆盖 native 子进程退出码的接点。[GitHub shell 规则](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#exit-codes-and-error-action-preference)

Get-Item／Get-FileHash 使用 LiteralPath；Join-Path、Add-Content 的值是字符串参数，没有把带空格的路径重新拼成命令执行。GITHUB_PATH 的文件由 runner 在下一步骤使用、前置到 PATH；当前步骤没有再次调用 Go，不需要在本步骤内重新加载 PATH。非 junction 分支不写入 GITHUB_PATH。Collect 在下一步重新解析 go，并在 `go test` 前执行 `go version`。[GITHUB_PATH 规则](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-commands#adding-a-system-path)

相同 SHA-256 支持“本次分别读到的两个可执行文件字节相同”，**不单独证明同一文件对象、官方原产、整个 GOROOT 内容相同、两次读取间没有变化或以后必定执行同一字节**。代码仍读取 junction 自身 Target，哈希不是凭名字任选另一份安装的授权。对于现有受控 runner 安装前提，这是一项额外的一致性核对，不是新的可信安装认证。

工具模块未改；[guard.go](../../prototype/c9evidence/internal/collect/guard.go) 第 59～106、159～198 行仍从 LookPath 找到的候选做绝对化／EvalSymlinks、普通文件与 bin 布局、src/cmd/go、VERSION、哈希核对，分析者版本、VERSION 与 GOVERSION 都必须是 go1.27.3，且实际 GOROOT 必须等于该 root。新 PATH 步骤没有让错误版本、嵌套异常路径或其他安装布局绕过守卫。未来 runner 不符合已知布局时可能继续拒绝；本报告不把 F-1 扩为适用于任意 Windows 安装。

### 4.2 N-157-1：IsPathRooted 不是完整的绝对路径谓词

位置：workflow 第 105 行，以及第 92～94 行、工具记录 §2.8 对“绝对目标”的解释。

`.NET Path.IsPathRooted` 会接受 `C:relative`、`\relative` 这类仍依赖当前目录／当前盘的字符串。因此该表达式本身不能证明 fully qualified。[微软 API 说明](https://learn.microsoft.com/dotnet/api/system.io.path.ispathrooted)

建议使用与要求一致的 `IsPathFullyQualified`，并显式排除空值；不要先依据当前目录补成绝对路径再称为原目标已通过检查。本轮未取得能使真实、可访问的安装 junction 返回上述异常字符串并通过后续两次哈希的具体路径；正常 junction 的已观测目标是完整绝对路径，未发现当前 runner 安装的可达误用反例。因此将其列为**非阻塞的谓词／说明精化**，不将一般字符串反例伪装成已证明的 runner 漏洞。若后续要求该条件独立担保任意异常 Target 的拒绝，则应先修正该谓词。

### 4.3 N-157-2：echo 内子命令失败不自动使步骤失败

位置：workflow 第 62～63、129 行；同形的旧第 61、75～76 行仅作直接关联说明。

`echo "... $(git rev-parse ...)"` 的步骤状态由 echo 决定；子命令失败不因 `set -e` 自动变为该 echo 的非零退出。新增的工具子树／workflow 身份打印可能留下错误诊断或不完整值而继续。第 68 行独立赋值的 `command -v go`、第 67／130 行独立 `go version`、第 70 行哈希命令则会消费实际状态，不能混为一谈。

建议先独立赋值并检查退出状态／非空，再打印；Collect 的路径打印可同样整理。当前文档明确这些是打印而非身份授权门，FR-4 还要求结果审查核对固定 C/T、子树和 workflow 关联，缺项不能成为有效取证；没有发现此打印路径能独自授予运行或有效结论。因此列非阻塞，不因新增诊断的健壮性建议推翻未变的工具守卫。

### 4.4 未改的执行设置

逐项读 diff 与完整 workflow，确认触发仍只有 `evidence/c9-run-1` 的 push，无 PR／手动／定时触发；contents:read，按 ref 的 concurrency 且不取消旧运行；三个 runner、fail-fast:false、20 分钟外层期限；GOTOOLCHAIN=local、GOFLAGS 为空、setup-go 1.27.3/cache:false；LE-4 本体、artifact 名称／目录／90 天／warn 规则均未改。

这不是纯文档 PR：它修改了正式 workflow 的准备路径；但 prep 分支和 master 不匹配其触发条件，合并本身不会启动该 workflow。以后新建／推送匹配分支仍是需要明确授权的独立动作。

## 5. FR-1～FR-8 的可决定范围

| 项 | 审查结论与运行前条件 |
|---|---|
| FR-1 | 根 go.mod 明确 require x/mod v0.42.0、x/sys v0.49.0；工具 go.mod 只有这两项，工具 go.sum 四行均原样存在于根 go.sum。固定 Go 的 `modcmd/download.go` 第 33～36 行及实现支持：go≥1.17 主模块无参数 download 下载明确 require 的模块，并取得 zip／解压目录。因此“一步也准备这两项”有静态依据，尚非本次 runner 的动态证明。有效环境、下载成功、同一缓存未被改动等仍是前提；Collect 出现下载须按表登记，不能据此自动补跑。 |
| FR-2 | 适当沿用 §2.5.3／2.5.5／7.3：名字／字节与实际消费者的关联，不是原子快照，也不认证 cmd/go 自己读取了同一批字节。标准库、语言版本／标签、缓存来源与资源上界未补证；无 setup-go 缓存恢复也不等于机器上没有任何既有缓存。不能用本轮 GO 替负责人接受这些声明边界。 |
| FR-3 | 已明确要求为正式 R 单独确认三种调用的限定前提与清理未知代价。源码依据不是完整无后代证明；第二阶段接受不延伸。darwin 不终止、Linux pidfd 不可用、请求／Wait 未返回的责任与未知保持，runner 销毁不是回收证据。它不接受 T-2／L-b1b 或依赖运行风险。 |
| FR-4 | C 为将来经授权的实际提交，推荐审查后合并提交；T=tree(C)。C、T、工具子树、workflow blob 是不同种类的值，不要求字面相等。HEAD=push GITHUB_SHA、tree(HEAD)=T 及 WORKFLOW_REF/SHA 的关联均须核对；本轮 PR 的合并测试快照不能充当正式 C。应在授权后才建立运行分支并推送一次。 |
| FR-5 | 保持每作业两个分析配置、全部对照与记录／收尾共享 120 秒，W=D−10s；编译、安装、LE-4/F-1 不在内部预算内。外层 20 分钟不是资源收尾证明或预算自动延长。R 上耗时、内存尚未测量，不用夹具秒数推导上界。 |
| FR-6 | 依 §7.1 的优先级和完整记录判 1～4；3/4 都退出 0，但只有 4 表示与候选一致。三个作业均有效才谈六配置完整取证；仍只是源码级分析，不是六种原生执行。已有无效不能因后续成功改善，缺失／到期结果不追认。全部结果都停下报告，3 不扩充例外，4 不批准清单或 G 项。 |
| FR-7 | 原 ZIP 的字节数／哈希与解包成员分别记录，按 §6.5 核对终结行、manifest 哈希、列出的文件、字节数／哈希及身份；目录集合核对须按原协议包含 manifest 本身的角色解释。入库重打包不得冒称原 ZIP。单独有日志、成功上传或哈希相等不足以补齐终判、关闭及完整性证据。 |
| FR-8 | 接线／准备失败没有对应正式采集证据；守卫、清理未知、预算、artifact 不完整按既有层级记录。保留且停止，不重跑、不清缓存、不换版本、不缩范围／加时／放宽守卫。另一次运行与必要 workflow 分支调整需重新审查和授权。 |

FR-1 固定源码依据：[Go 1.27.3 download.go](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/modcmd/download.go)。这里没有运行 download 来验证，也没有把根模块 CI 的缓存命中当作这一步的演练。

运行前还应由负责人明确：N2 的两项选择；正式 E-10(ii) 接受或替代方案；完整 C/T 与随 C 固定的工具／workflow 身份；运行场所／三 runner 六配置、一次分支推送的执行人；是否采用 LE-8 条件授权；FR-5～8 的处置规则。选择不同方案若改变已审命令、控制或 workflow，应先回到相应审查节点。该决定表没有使任何 G 项或精确条目自动满足。

## 6. 本机材料与未取得的证据

本轮读取了现存 `scratchpad/s207`（会话目录 `08d567b6-49ef-4441-9d49-02a3f663c9d7`）中的 YAML 比较源码、前后 workflow 副本、抽出的 bash／PowerShell 脚本、junction 演练脚本及两个 GITHUB_PATH 文件，没有执行它们。

- `yamlchk/new.yml` 为 5820 字节，SHA-256 `4dc921e5281013a19506fb6badbb4c82a7d79fb6d578885a050248d8542381e0`，与固定 Git blob 的原始字节哈希一致。
- 抽出的 PowerShell 正文与固定 workflow 对应脚本文本相同；演练脚本包含同一正文，并前置 stop／追加 Actions 风格退出处理。
- 演练的 junction PATH 文件为 132 字节，SHA-256 `48d3ffad6a0dd578c3a75acac98d53b5a8304d63694e94af4c98eec3c3b8ce3c`，内容为临时 real/x64/bin；plain 文件为 2 字节空行，SHA-256 `7eb70257593da06f682a3ddda54a9d260d4fc514f645237f5ca74b08f8da61a6`。
- 两份现存演练 go.exe 文件均为 98304 字节，当前 SHA-256 同为 `23240ef9f8b0a9a324110b1c2331de31dc1b0e08f5359cb707e51a939af56cd3`。这只是演练文件的当前一致性，不认证其为固定 Go 工具链，也不表示执行过它们。

该目录没有取得带完整调用环境、stdout/stderr、退出状态的独立演练日志或语法检查原始回执。因此“两个分支当时确实按该命令执行成功”的部分仍是作者登记；现存脚本和输出文件支持关联，不等于本轮复现或历史执行认证。两个失败分支未构造，新步骤未在 runner 运行；这些限制保持。审查者没有为了补证运行解析器、脚本、Go 或新实验。

## 7. 当前常规 CI

独立取得 [run 46073216782](https://github.com/XKHoshizora/specmetry/actions/runs/46073216782) 的一次当前状态查询及八份作业日志。pull_request，attempt 1，正式 success，8/8。所有日志的实际 checkout 都是：

`1e05905ba28f77ced29187a031e3cd94d7f6130a`

Git commit API 的父提交依次为固定 base、固定 HEAD，tree 为 `e079337f7b4aa0725d7a085007dcaa5d993583e2`，与受审 HEAD 相同。不是仅采用 run.head_sha。

| 作业 ID | 作业 | 核验结果 |
|---|---|---|
| 140754818863 | Windows Go tests | success，27 个 ok 包，10 条 SKIP，无 FAIL |
| 140754819284 | Ubuntu Go tests | success，27 个 ok 包，主测试 15 条 SKIP；随后独立 root CrossUser 8 项 PASS |
| 140754819347 | macOS Go tests | success，27 个 ok 包，7 条 SKIP，无 FAIL |
| 140754819335 | Go lint | success，0 issues |
| 140754819217 | Web | success；没有安装／报告 Go 版本 |
| 140754973361 | macOS dashboard build | success |
| 140754973385 | Ubuntu dashboard build | success |
| 140754973446 | Windows dashboard build | success |

七个显示 Go 版本的作业均为 **Go1.27.4**。Linux 主测试的 race 与 root CrossUser 分开；SKIP 是平台／环境条件测试，不能把主步骤跳过的八项 CrossUser 说成从未执行。根模块命令与独立模块边界没有变化：这些日志没有运行 `prototype/c9evidence`，也没有执行 `c9-evidence.yml`。它们不证明固定 Go1.27.3 下的新 PATH 步骤、正式输入兼容性或 120 秒预算成立。

旧运行失败与重跑按第 2 节范围保留；不因本次绿色推定旧 SQLite／CLI 问题已解决、同根因或资源上界。没有查询等待循环，没有 CI 操作。

## 8. 结束状态与后续边界

开始工作区干净，报告文件不存在；结束本地及远程 prep 分支、远程 master、PR head/base、HEAD tree 均与第 2 节相同，没有漂移。唯一新增文件为本报告，保持未提交。

本轮只读源码、文档、Git 对象、哈希、现存材料和线上已有日志／官方依据；没有运行 Go、测试、构建、模板、helper、伪工具、采集器或实验，没有修改受审文件、PR、CI、分支或设置。

最小下一步是只修 F-2 的错误推论及对应摘要，再作定向复核；两项 N 可在明确范围内精化。随后由负责人分别决定准备变更合并、N2／正式 E-10(ii)、C/T 和一次正式执行。H-1、G 项、精确例外、依赖整体运行信任、第三批生产实施及 WP2.5／M2 验收均不因本审查得到批准。报告完成后暂停。
