# Codex Review 070：PR #30 文件引用完整性及整合结果定向复审

## 一、结论

**GO。Review 068 的 B3、N1 可以关闭；未发现原 Review 067 B1/B2 被重构破坏或其他实质性直接回归。固定目标的最终 CI 已正式 completed/success、attempt 1、8/8，PR #30 可以建议合并。**

结合 Review 067、068 已成立且仍适用的审查证据，本轮足以支持负责人对 **WP2.4 本工作包范围**作出验收决定。本报告不替负责人宣布验收，不覆盖后续工作包、全里程碑的 CLI/E2E 集成或 M2 整体验收。

| 项目 | 裁定 |
|---|---|
| Review 068 B3：文件引用对象完整性 | **关闭**。固定提交、必要的树、目标普通文件 blob 均完整验证；解析路径与计数使用同一次验证流，失败不发布部分结果 |
| Review 068 N1：对象头长度与分块 | **关闭**。长度检查在解析／清空头部缓冲区之前，合法头与正文同块不会被误拒；上轮独立反例重放通过 |
| Review 067 B1：全部提交候选验证 | 保持关闭；共享读取后，所有候选的类型、长度、ID 重算及整体失败语义仍成立 |
| Review 067 B2：所有输出模式提交前校验 | 保持关闭；文本／JSON、manual 预演与最终事务、首次 worktree 回滚及提交后输出故障回归通过 |
| 新 BLOCKER／产品待决事项 | 无 |

有一次本机完整 gitx 包的累计 600 秒超时，**没有记为通过或据此认定耗时根因**；同样默认超时下的必要定向补跑随后通过。该边界、WSL 初次启动失败及 CI 的条件跳过见第七、八节。M-14/M-15/M-16 均保持未解决。

## 二、固定身份、证据继承与只读边界

- 日期：2026-10-22，Asia/Tokyo。
- PR：[XKHoshizora/specmetry#30](https://github.com/XKHoshizora/specmetry/pull/30)。
- 本轮 HEAD：`b8606b5e51ee2ca7de5cf9bfe0b2f45f6d431827`。
- 新 base：`8b45ef598da1599ea672072338fef5ed9c6f77ee`。
- HEAD tree：`abaf20ded9e0eba63dbb664d8455228863d99dcb`。
- Review 068 对象：`e70c09bb404db8371ac268c2bf64c4379141f26f`；其 base 仍是当时的 `d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`，没有用新 base 重写旧身份。
- 修复完成、整合前 HEAD：`8c0db79518da6087969cf59e6e73d2e456c4685d`。
- 核心修复提交：`87fb135ba1829d41d19404e5be707af0272994c6`。
- 开始及收尾远程核对均与指定 HEAD/base 一致，PR 保持 Open，`auto_merge=null`。结论未延伸至其他提交。

已读取适用 [AGENTS.md](../../AGENTS.md)、完整 [Review 067](codex-review-067.md)、[Review 068](codex-review-068.md)，本轮处理记录 [§80～§82](../review-resolution.md) 及必要的冻结规范／调用链。本轮继承上轮未变模块的有效结论，不重新审查整个 M1、WP2.3 或 M2。

原工作区起始干净，报告文件起始不存在。按固定对象在仓库外导出隔离副本，新增临时反例、故障注入、编译和测试均在该副本或其临时夹具中完成；未切换、清理、修改原工作区实现。原仓库只新增本未提交报告。

## 三、增量与整合核查

### 1. 修复来源

`e70c09b → 8c0db79` 共 19 文件、1242 行新增／254 行删除。其中源码与测试是 **14 文件、950 行新增／250 行删除，全部来自 `87fb135`**：

- 新增 `internal/gitx/filelines.go`、`verify.go` 及 FileLines 测试。
- 重构 `resolve.go`、移除 `gitx.go` 中 TreeFiles/CountLines 和 `objects.go` 中已无调用的两个包装函数。
- 调整 gitx 相关回归，新增 N1 头长测试。
- 调整 `internal/proposal/coderef.go`，新增共享引用完整性测试。
- 新增 `internal/cli/coderef_integrity_test.go`。

其余为 Review 068 归档和开发／状态记录。已独立比较 `87fb135 → 8c0db79` 的源码、测试、Schema、依赖等相关树，差异为空，没有把归档或状态提交误算成额外生产修复。

### 2. 新 master 的整合

整合提交 `cc2d006251d9d272914fdc9e0f27902caa9119ad` 的父提交依次是：

1. `8c0db79518da6087969cf59e6e73d2e456c4685d`；
2. `8b45ef598da1599ea672072338fef5ed9c6f77ee`。

`8c0db79 → b8606b5` 仅六文件、251 行新增／5 行删除：`.github/workflows/ci.yml`、AGENTS、实施计划、M1 证据报告、review-resolution 和 Review 069 归档。独立对象比较确认：

- `internal`（含迁移及全部测试）、`cmd`、`schemas`、`web`、`scripts`、`go.mod`、`go.sum`、`.golangci.yml` 与 `8c0db79` 一致。
- `.github` 与新 base `8b45ef5` 一致。
- 相对 Review 068 对象的源码／测试增量仍只有上述核心修复，不包含其他任务。
- 实施计划保留了本分支的 WP2.4／§80～§81 状态及 master 的 M-16 缓解记录，没有因相邻行冲突丢弃任一侧含义。

因此本轮只核对已审 PR #31 的继承和实际运行适用性，没有重审其批准的预算取舍。

### 3. 归档与活跃记录

独立核对的归档 SHA-256：

| 报告 | SHA-256 |
|---|---|
| Review 067 | `3794cd450746b76d6f5f623d647ec762a0ed39d2e3b57d4889309746302be7b6` |
| Review 068 | `09c6b472d610162f9a47aab709e3fd2f3aacc6446b48b66d476ff299a95b31a7` |
| Review 069 | `eb257e9177ee514d40a288b9fa9325a0b083aeb6e87f3473ace06b0e3b1e41b7` |

与原报告、归档记录一致。`review-resolution.md` 在 `e70c09b` 时的完整内容，及 `8c0db79` 时的完整内容，均是最终文件的未改写前缀：§80 原发现、§81 的 B1/B2 关闭与 B3 阻塞、§82 的 Review 069／PR #31 合并与整合记录按时间追加。

旧报告中的 NO-GO、各自受审 SHA 和验证限制未被后续成功 CI 替换。§81、§82 仍将 B3/N1 保留为“已修复，待独立复审”，没有提前宣布关闭或验收。§82 记录的 PR #31 已合并，与 §81 当时尚未合并的历史状态不冲突。

## 四、B3：文件引用完整性可以关闭

### 1. 生产调用链及验证对象

[proposal/coderef.go](../../internal/proposal/coderef.go) 的 `CheckCodeRefs`（第 43 行）与 `CheckCodeRefsAt`（第 56 行）继续共用 `checkCodeRefs/fileChecker`；前者解析当前执行 worktree 的 HEAD，后者使用传入的已采集 HEAD。Git 路径在第 170 行统一调用 [gitx.FileLines](../../internal/gitx/filelines.go)。

[FileLines](../../internal/gitx/filelines.go) 第 34～139 行：

1. 只接受完整 SHA-1 提交 ID；受控调用明确设置 `objects=true`。
2. 完整读取固定提交，核对预期 ID、类型和内容，从该次读取的首行取得根树 ID。
3. 按各引用路径逐层读取必要树；每层完成全部校验之后，才从同次读取的 buffer 解析条目。
4. 对实际指向普通文件的 blob 去重、完整流式读取，行数由该流的 `lineCounter` 计算。
5. 任何一步失败均返回 nil 结果；最后才组合路径→行数 map。

类型不是仅凭正文“看起来可以解析”判断：commit、tree、blob 的实际类型都必须与所处层次相符。`parseTree` 从已验证的树正文解析 mode、名称和 20 字节 OID，并拒绝截断、无法解析或重复名称等不能可靠解释的结构；普通文件、目录、链接／gitlink 分开处理。这不是全仓库语义 fsck 的承诺。

[verify.go](../../internal/gitx/verify.go) 第 40～58、78～158 行的共用读取：

- 每批对象以一次 `cat-file --batch` 按请求顺序核对完整 ID、已知类型、非负长度、完整正文、分隔和最终 EOF；FileLines 会按上述提交／各层树／blob 阶段发起多批读取。
- 按 `<type> <size>\0` 与实际流过的正文重算对象 ID；不因 Git 返回 0 就相信内容完整。
- Git 失败、stderr、取消、协议或完整性异常均不能返回成功类型 map。
- 树／提交使用内存 buffer，blob 使用不保留内容的计数器。当前调用者的这两种 sink 都完整接受传入字节；派生结果只在读取及类型检查成功后使用。

这里没有“先验证，再调用 ls-tree/cat-file 做一次未验证判定”的第二条读取。树映射和计数只是本次调用内的可信派生缓存；混合路径中后续对象失败，不会返回已完成路径的部分结果。

### 2. 上轮独立反例原样重放

将 Review 068 的独立临时用例原样移入本轮隔离副本，未改成无关前置错误。Windows Git 2.56.0.windows.2 与 WSL Git 2.43.0 的结果一致：

| 场景，均对照固定 A，分别调用两个接口 | 当前结果 |
|---|---|
| 头部可读、声明 500 字节、正文截断的 blob | exit 6、nil 结果 |
| 原 blob ID 文件中装入另一个合法 blob 的完整字节 | exit 6、nil 结果 |
| A 子树 ID 文件中装入 B 子树，B 多出 ghost 路径 | exit 6、nil 结果，不再误报 A 中存在 ghost |
| 根树／固定提交对象被另一个对象替代 | exit 6、nil 结果 |
| 所需 blob／子树删除 | exit 6、nil 结果 |

每个场景有健康及恢复原对象后的对照，并比较读取前后 `.git` 内容和 mtime；没有让验证过程修复对象，也没有修改原仓库。前三种正是上轮在旧对象上独立检出的假成功；旧版红灯继承 Review 068 的有效记录，本轮当前版本的相同反例全部拒绝。

另实际运行并核查本 PR 的提交截断、深层／中间树、共享 blob、类型不符、健康与异常混合的回归：类型错误夹具使用正确的对象哈希构造，确实测试“类型不符”，不是先被错误 ID 挡住。Unix 所需 blob 的实际不可读权限另有本轮独立用例，普通用户下为 exit 6、nil；恢复权限后成功。

### 3. 证明判定使用同一次已验证读取

独立增加真实 Git 调度测试，分别针对提交、根树、中间树、blob：

- 等待该对象的真实 stdout 已被校验器完整接受、结束分隔和哈希检查通过后，才将隔离对象文件换成损坏字节。
- 本次 FileLines 仍使用刚验证的原始数据，两个共享 blob 路径得到正确行数；被观察的对象在本次只消费一次，没有第二次未验证读取。
- 下一次独立调用不沿用旧缓存，立即拒绝已损坏对象；恢复原字节后成功。

四个场景在 Windows 和 WSL 均通过。钩子包装真实 stdout，未伪造 Git 数据或改动验证／判定的生产顺序。这个测试证明“结果源自已验证流”，不声称对象库在返回或 SQL 提交时与读取时构成跨系统原子快照；已读到有效对象后再发生的外部修改，不被冒充为本次读取了损坏内容。

### 4. 真实 Claim/propose 路径、回滚与隐私

原 Review 068 的公开 Claim Execute 反例原样重放：损坏文件 blob、没有提交引用，防止借 ResolveCommits 意外挡住目标分支。当前在两种模式均得到 exit 6，seq 均为 `6→6`，新 worktree 登记数维持 0，完整业务转储不变；不再输出 `exists=true` 或成功文本。恢复对象后正常提交。

又独立为 propose 构造三类反例：截断 blob、替代 blob、替代子树。每个输入同时含健康引用及相关异常路径，在文本和 JSON 模式下执行真实 Git／SQLite：

- 全部为 exit 6；JSON 是单个完整、Schema 合法的 `error.v1/environment`。
- 动态读取全部业务表和字段及 meta，前后完全一致；不是只比较 Proposal／Item 行数。
- stdout/stderr 不出现对象 Canary 或原始 Git stderr；数据目录扫描未发现 Canary。
- 修复对象后的相同 Proposal 正常保存，健康引用存在、A 中本来没有的 ghost 仍不存在。

Windows 与 WSL 均通过。仓库的 Claim/propose 两项命令回归也通过，覆盖 Claim 首次 worktree、混合引用、文本／JSON、修复后成功。上述是公开命令执行入口的进程内测试，使用真实 Git 和 SQLite，不冒称正式发布二进制的完整 E2E。

### 5. 正常语义和范围

健康普通文件、带／不带行号、空文件、最后一行无换行、真实路径缺失和行号超界的预期值保持；目录、符号链接和 gitlink 仍不算普通文件。`TestFileLines` 在运行检查前用真实提交树断言模式 120000、160000 和可执行文件确实存在，没有回到早期无效夹具。

显式 HEAD 的对照在 HEAD 已移动、B 的新路径对象损坏时仍检查健康 A；自行解析 HEAD 的 propose 接口检查 B 则正确拒绝。Claim 的 `TestClaimChecksFilesAtTheCapturedHead` 也通过，`checked_head` 继续绑定捕获值。unborn 和非 Git 的既有语义、正常 propose 前置条件均保留。

**裁定：B3 关闭。** 保证仅覆盖本次引用判定实际需要的对象，不扩展到未引用 blob、无关树或历史，不要求 check.inputs 的 TreeEntry 因此读取 blob。

## 五、N1 与原 B1/B2 的直接回归

### N1：关闭

[verify.go](../../internal/gitx/verify.go) 第 83～100 行先找到本块的头部片段，再检查 `len(v.head)+len(line)>maxBatchHeader`，然后才追加、解析／清空缓冲区。因此同一完整头不会因换行恰在当前 Write 内就逃过限制，也不会把同块中的正文长度算进头长。

独立重放上轮“正确 ID／正文、超长填充头一次写入却被接受”的精确反例，当前拒绝。合法／非法字节流在整块、逐字节和跨块下的对照通过；合法头与正文同块、多个对象、空正文，以及错误 ID／类型／长度、截断、分隔、额外内容的独立解析回归仍通过。N1 的原严重程度未被扩大，没有发现新的 false-safe 路径。

### Review 067 B1：保持关闭

ResolveCommits 仍枚举、去重全部候选，改为调用同一 `readVerified`，sink 为 nil；不能在唯一非提交或歧义分支跳过验证。类型结果只在完整 batch 成功之后使用，存储检查及最终整体发布顺序未变。

全部候选、替代字节、截断、歧义、取消、枚举后对象消失回归，以及上轮独立的损坏非提交／歧义提交反例，均在当前版本通过；真实 batch 中途取消仍返回环境错误、无部分答案。健康 blob/tree/tag、真实歧义和可靠缺失保留原 false 语义。

### Review 067 B2：保持关闭

CLI Claim 的回执事务代码未改：`claimOutput` 仍在第 227 行先验证 `claim-receipt.v1`，再选择文本／JSON。事务内准备输出，manual 预演与确认后正式事务共用同一回调。

Windows／WSL 都通过：三种批次的两种正常输出；两模式非法回执回滚；manual 预演失败与第二次回调失败的精确区分；首次 worktree 回滚；提交后写入错误／短写事实保留且不追加第二份 JSON。原独立 `seq=0` 注入对照也通过，没有成功或注入值输出。

### 受控执行及其他接点

- FileLines 的每次读取设置 `objects=true`，经 `call.run → objectGit → runWithGit`；Code Ref 的 PD-26 和部分克隆前置拒绝仍在读取前。
- ResolveCommits 的读取仍经 `scanCall`，保留按二进制身份验证的安全能力、显式空 graft/shallow 覆盖和受控环境。
- 删除 `runObjects/runObjectsTo` 后，剩余 `runObjectsCode` 调用及 `call` 对象入口没有绕过门禁。环境过滤、禁传输、禁 replace、FSMonitor 限制、stderr 丢弃和取消等待未被改动。
- Clean、上下文读取、TreeEntry 和提交扫描维持各自契约；相关源码未被顺带重构。必要 Clean／TreeEntry 回归通过，不由本轮增加全仓库对象完整性承诺。
- Schema、迁移、推导、SQLite 重试／错误映射及方案 B 没有变化。

## 六、CI：固定树适用，8/8 正式成功

[run 39723458143](https://github.com/XKHoshizora/specmetry/actions/runs/39723458143)：`pull_request`，attempt 1，正式 **completed/success**，HEAD 为本轮目标。已读取八个 Job 的正式状态和实际日志，均 checkout：

`1041a859c46331675d53bff04aa887ca1b54e098`

独立核对父提交依次为固定新 base 与固定 HEAD，tree 为 `abaf20ded9e0eba63dbb664d8455228863d99dcb`，与目标 HEAD 完整树一致。

| Job | ID | 正式结果 |
|---|---|---|
| Web | 119641962869 | success |
| Go tests (ubuntu-latest) | 119641963060 | success |
| Go tests (windows-latest) | 119641963101 | success |
| Go tests (macos-latest) | 119641963119 | success |
| Go lint | 119641963137 | success |
| Build with dashboard (ubuntu-latest) | 119642078487 | success |
| Build with dashboard (windows-latest) | 119642078511 | success |
| Build with dashboard (macos-latest) | 119642078551 | success |

三平台 CI 均为 Go 1.27.3。本轮在固定副本真实 `go list` 得到 33 包，与三平台包结果集合逐项相同、无重复：26 个 `ok`、7 个无测试文件。33 是当前快照数量，不是长期硬编码要求。

- Windows 动态分成 32+1；第一组全部非 CLI 包完成之后，才单独执行 CLI，参数含 `-timeout 20m`。其他包没有增加显式超时，保留默认 10 分钟。
- Ubuntu/macOS 仍一次执行 `./...`；Ubuntu 普通步骤走 `-race` 分支。Linux root 步骤另行编译、实际通过 8 个跨账户测试，本身不带 `-race`。执行命令、运行器身份及步骤是证据，不以没有竞态告警代替是否启用 `-race`。
- 本次 B3/N1 回归三平台均实际执行：7 个顶层测试、含子测试共 23 条 PASS；Claim 为 48 条、Propose 为 59 条 PASS，均是含父／子测试的日志口径，不是各有这么多独立顶层测试。
- CLI／gitx／proposal 耗时：Windows 431.044／336.919／38.637 秒；Ubuntu 367.256／176.190／27.261 秒；macOS 236.447／141.715／20.924 秒。没有据此解释资源竞争、保证未来预算足够或关闭 M-16。

按设计跳过与通过分开：

- 真实 unsafe／旧 Git 条件用例：`TestClaimWithARealGitThatFetches`、`TestARealGitThatFetches`、`TestResolveCommitsWithAGitThatFetches`、`TestWP23ReadsWithAGitThatFetches` 及旧版本 init 用例在 CI 跳过，不是三平台真实 Git 2.32 通过。
- Windows 权限条件用例（含 ResolveCommits 不可读存储）跳过；不把 Unix 的通过写成 Windows 通过。
- Ubuntu 普通步骤中的 8 个需 root 用例跳过，独立 root 步骤实际通过；Windows/macOS root 步骤不适用。没有跳过整个 Job 不代表没有条件跳过测试。
- M-14 诊断默认未启用，其跳过不证明问题解决。

旧成功和失败保留在原报告及 PR 说明中：Review 067 的首轮 lint 失败、Review 068 所记 macOS 大写前缀夹具失败，没有被新成功改写。整合前 run 39704552634、作者 Windows／WSL 实验和变异记录是分 SHA 的作者证据；本轮没有冒称独立逐行比较其全部日志，也没有把作者局部 40 分钟超时的本地整仓运行当作本轮默认设置通过。本轮以最终固定树的正式日志及下节独立复验作判断。

## 七、本轮独立运行、失败与未验证边界

隔离目录为仓库外 `review070`，固定导出包 SHA-256：`d45532278219273879a2760fe2abda6ec7a98db7c8c81ec1ed874bf781a6259c`。原 Review 067/068 临时反例及本轮补充仅在副本中，未加入原仓库。

本机 Windows/amd64，Go 1.27.1、Git 2.56.0.windows.2；WSL Ubuntu/amd64 普通 UID 1000、Git 2.43.0，运行相同副本交叉编译的测试程序。依赖用缓存，`GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`；没有网络／远端实验或 CI 重跑。

| 实际验证 | 结果及适用边界 |
|---|---|
| Windows 完整 proposal 包及原独立 B3 库反例 | PASS，126.650 秒；两个 Code Ref 接口、七种原场景、健康／恢复对照 |
| Windows 全部 TestClaim*/TestPropose* 加原独立 CLI 反例 | PASS，372.887 秒；文本／JSON、固定 HEAD、前置条件、manual、回执、首次登记等直接回归 |
| Windows 新增同次读取调度 | PASS，gitx 13.738 秒；commit/root/subtree/blob 四场景 |
| Windows 新增 propose 三类损坏 × 两种输出 | PASS，CLI 37.899 秒；完整业务快照及 Canary 检查 |
| Windows 必要 gitx 定向补跑，含原被中断分类用例、Clean、TreeEntry | PASS，431.437 秒；未增大默认超时 |
| WSL 相关 FileLines／候选／N1／独立解析与取消／同次读取测试 | PASS；真实 Git 2.43，不是真实旧 Git 2.32 |
| WSL 两个 Code Ref 接口、原独立 B3 七场景、健康及固定 HEAD 对照 | PASS；独立所需 blob 权限拒绝用例实际通过，非 root |
| WSL 全部 TestClaim*/TestPropose*、原独立 CLI 反例及新 propose 三类反例 | PASS；绝对路径启动测试程序，真实 Git/SQLite；含通道回归但不把非 Git PTY 夹具当作 Git 上下文证明 |

必须保留的本机失败：

1. **首次完整 gitx 包未通过**：累计默认 10 分钟后超时，包记录 601.103 秒；当时 `TestClassifyChecksTheStorage` 运行约 10 秒、`K2_a_truncated_.idx` 约 1 秒。超时前所涉及 B3/N1 和候选验证已实际通过，但不能把整个包记为通过。随后以相同默认超时运行必要子集，包含该完整存储分类测试及此前中断的子例，结果通过。没有独立证明超时根因，不把它直接定性为死锁、CI M-16 或某项资源竞争；也没有修改超时、测试门槛或原仓库设置。
2. **首次 WSL 启动没有执行测试**：内联 shell 参数传递使目录变量未按预期传入，报 `/internal/gitx` 不存在；仅 UID／Git 版本打印成功。改为隔离副本中的脚本文件和绝对路径启动后完成上述验证。该初次启动失败不是产品检出，也没有算入通过。

没有本机 macOS，其结果来自正式 CI；没有本轮独立重跑作者真实 Git 2.32 实验、全部变异／桩阶段红灯、整套 M1、整个 WP2.3 安全审查或正式发布包 E2E。作者的首次未检出类型变异、补测试后检出的记录仍是辅助证据，不被冒充为三平台压力测试或本轮独立变异结果。

## 八、分类、合并与验收建议

**BLOCKER：无。** B3 的原可复现假成功及必要层次／类型失败已关闭，原 B1/B2 防线仍成立。

**NON-BLOCKER：没有新增确定的实现缺陷。** N1 可关闭；本机整包超时和启动错误作为验证观察／边界保留，不借此关闭或重分类既有维护项，不要求本 PR 修改无关超时。

**PRODUCT DECISION：无新增待决。** 本轮实现的是既有对象完整性和快照绑定保证；共用 propose 路径的直接修复属于已确认的 WP2.4 范围，没有改变 E4/E6、D1～D3、版本优先、存在性含义或完整盘点要求。

**ACCEPTED RISK：无新增。** 仅沿用已批准的 WP2.3 方案 B 局部边界，不豁免文件引用完整性、Log、正常合作并发、其他账户、数据库或隐私。PR #31 的 CLI 预算调整继续按已批准范围适用；M-14/M-15/M-16 不是本轮接受的风险，均未关闭。

最终回答：

1. **Review 068 B3、N1 均可以关闭。**
2. **原 Review 067 B1/B2 没有发现直接回归，保持关闭。**
3. **PR #30 为 GO，可以建议负责人合并固定目标。** 正式 CI 与合并快照核验已满足；后续提交或 base 变化须核对新增内容及证据适用性，不能自动继承到任意新树。
4. **结合既有独立审查，足以支持负责人验收 WP2.4 的本工作包范围**；正常合并结果与 master CI 核验仍由负责人按项目流程完成。本报告不宣布已验收，不覆盖后续检查执行、status/gaps 或 E2E 工作包，也不宣布 M2 整体验收。
5. **没有必须在本工作包合并前补齐的产品决定或技术证据。** 如实保留第七节未验证边界，不把它们写成已经完成的实验。

本轮只新增本未提交报告，采用仓库相对链接；未修改实现、其他文件或历史审查，未提交、推送、合并、改基、重跑 CI 或修改超时，未启动 WP2.5。全部源码位置及结论以本报告固定 HEAD 为准。
