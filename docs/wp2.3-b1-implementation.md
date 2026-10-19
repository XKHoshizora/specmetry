# WP2.3 B1 方案 B：实施交付记录

2026-10-19，实施方 Codex。依据负责人的 B1-D1～D4 明确批准（[活跃决定 §71](review-resolution.md#71b1-方案-b-的批准与实施范围)），实施、测试、提交并推送现有 [PR #27](https://github.com/XKHoshizora/specmetry/pull/27)。本记录不作独立审查 GO、B1 关闭或工作包验收结论。

## 1. 身份与既有工作

- 原生产对象：`ee144af61e00cb4d24bbae27feab10d8be60ecfc`。
- 原／推送时远程 base：`e0a21b46b5ad7e05146f254aa7523d6054ce2e76`，分支 `feat/m2-wp23-git-reader`。
- 原未推送提交完整保留并推送：`659b914901bfd4fc9a251fc102c69b2ebcfb509b`（归档 Review 060）；`d6f9153bbde5cc5cd496edde9c55e4af8b080ba7`（§70 及状态）。没有重置或重复归档。
- 决定／规范先行提交：`9462a56168c9a27a0413992da88cdc7a00c0ec95`。
- 代码与测试提交：`979f705d38d4f42412de13223096a02ee3ddd428`，内容树 `0c8c0790f8fcfbc55ab4b909e8f7c49794124d76`。本报告随其后的纯文档收尾提交保存；最终 HEAD 及对应 CI 以交付回执为准。
- [设计报告](wp2.3-b1-design.md)入库前只整理三个绝对路径 Markdown 链接为代码格式纯文本并添加归档说明；原稿 SHA-256 `266d5fbaa130da323a0558806e47eb62531343d760cd6ff336c703601b92c2a4`，不宣称整理后字节相同。原始分析对象及证据边界不改。

## 2. 批准范围与实现对应

| 批准 | 实现 | 保证边界 |
|---|---|---|
| D1 扫描去路径 | `scanCall` 固定两个显式空值；先经 PD-26，再按 Git 文件身份、大小、mtime 的缓存进行行为探测；执行固定探测过的可执行路径 | 不支持／无法确认／取消／清理失败均 exit 6，无 missing 路径回退，不接受部分结果 |
| D1 探测 | 在 v2 租约下手工写合成对象；无远端、无过滤器，不执行 Git 写命令；根、子、双父图，graft/shallow 截断正对照，完整读取、增量排除及损坏／缺失负对照 | 新增探测自身使用同一协议，不把新探测放到未保护的系统临时目录；既有 PD-26 探测未重写 |
| D2 合作并发 | `tmp/git-reader-v2/namespace.lock` 永久保留、内容为空；每个私有目录从建立到全部 Git 调用及清理持有租约；竞争安全失败 | 串行化私有读取租约，不给业务事务增加自动重试；扫描实际读取不使用临时防护目录 |
| D2 版本隔离 | v2 为 tmp 下独立子树，v1 只认顶层 `read-*`；新代码不接管 v1 目录或 v1 owner | 依据命名空间／旧代码实际选择规则隔离，不以“没看到旧进程”推断安全；未做旧二进制双版本并发实跑 |
| D2 Windows | NtCreateFile FILE_CREATE 原子排他创建并读取创建句柄身份；创建 guard 不共享删除，覆盖实际 Git 使用；最终释放 guard 后打开删除句柄，比较原创建身份并在该同一核对句柄上设置 disposition | 交接时若被替换则拒绝；核对后删除不重新解析名字。不是声称创建句柄本身一直拥有 DELETE 权限；该访问权会妨碍正常目录遍历 |
| D3 清理 | owner 中保存创建条目相对名、类型和身份；清理前检查整个树，按清单逐项、后序删除；每次重试重新核对；可信崩溃遗留同样经过清单与锁校验 | 未知、额外、缺失、身份不符保留并 exit 6；不 RemoveAll 猜测内容、不靠前缀／年龄／PID 认领。owner 已丢失等不可证明遗留保留报错，需人工处理 |
| D4 Scratch | 公共参数保持；nil、零值、关闭仍内部错误；NewScratch 仅做 datadir 保护、协议及遗留准备，不打开数据库 | 取消须待 Git 子进程结束后清理；失败释放自己的 handles／锁，不能经变化后的路径猜测删除 |

## 3. 直接测试与平台

- **Windows/amd64，本机**：Go 1.27.1，Git 2.56.0.windows.2。已运行原 Scratch／移除调度、正常 Clean／Log、所有新增方案 B 测试、Git Reader 与扫描器相关回归；最后定向运行 gitx `62.417s`、gitscan `70.852s` 均通过。Windows 原子创建冲突／创建身份／保护重命名／删除核对后换名的同句柄删除测试实际通过。
- **WSL Linux/amd64，普通用户**：真实 Git 2.43.0，Windows 交叉编译 Go 测试二进制，非 Linux 原生 Go 工具链。实施过程中 gitx 整包 PASS；gitscan 整包 PASS。最终提交身份另由下述 CI 实际核对，不混用未提交版本证据。包括 B2、B3、N1 直接回归、正常重试、身份替换拒绝、创建清单、真实多进程竞争与释放／崩溃回收、来源及对象内容／mtime 既有断言。
- **WSL Linux root 跨账户基座**：四项 `TestCrossUserScratch*` 全部 PASS。root 仅用于启动两个非特权测试账户；可写祖先拒绝、安全祖先下攻击者不能替换／种植、另一属主祖先拒绝、sticky 祖先正常。
- **macOS**：由下述 CI 实际运行，不冒称本机执行；已核对日志中新测试 PASS，特别是 Unix 精确窗口、真实多进程、空覆盖、二进制身份重新探测、取消及三父合并。
- **静态检查**：相关包 golangci-lint v2.14.0 在 Windows 和 GOOS=linux 分析均为 `0 issues`；CI 另行执行 tidy、vet、格式及 lint。没有修改生产 SQLite 行为、迁移、推导、CLI 或 Schema。

新增关键断言：

1. `TestScanEmptyOverrides` 在公开 Log 的真实执行前后插入往返调度；记录证明两个空值实际进入子进程，完整返回两提交。
2. `TestScanEmptyMultiParentAndFailures` 检查真实三父合并、全部提交集合及增量排除，随后缺失／损坏所需父对象必须环境错误。
3. `TestScanRefusesUnsupportedEmptyOverrides` 用记录型真实 Git 代理隐藏空覆盖，探测必须失败、不能运行用户 Log、不能缓存成功；`TestScanCapabilityFollowsExecutableIdentity` 在路径、大小、mtime 相同但文件身份改变时验证重新探测。代理不冒称真实旧版 Git。
4. `TestScanProbeCancellationReleasesLease` 在新探测已建立目录之后取消，检查空覆盖能力未缓存、自己的临时内容清理且租约可重新取得。
5. `TestScratchProtocolProcesses` 是真实父子进程：活跃租约竞争拒绝且内容不变；正常释放后可重用；崩溃退出不运行 defer，OS 释放锁后下一进程按完整清单回收。
6. `TestScratchManifestAnomalyIsRetained` 分别注入额外文件、身份替换和缺失文件；读取及清理均拒绝，已存在内容保留。`TestScratchVersionNamespacesDoNotMix` 检查 v1 遗留不被接管、v2 内的 v1 记录被拒绝。
7. gitscan 的 `TestScanFailuresWriteNothing` 新增不支持空覆盖及探测发现未知内容两类失败；检查完整已有字段、提交、关联、游标和 seq，不只看行数。其余部分读取丢弃、并发跳过、浅克隆生命周期和共享 seq 直接回归保持。故障修复是测试显式移除自己注入的见证文件，不是生产清理未知文件。

### 3.1 回归检出力

在仓库外导出 `979f705` 的隔离副本，仅将 scanCall 改回 v2 私有目录里的 missing 路径，保留真实 Git 及精确往返 hook。`TestScanEmptyOverrides` 实际失败：两个提交只回调一个，测试报告 `roundtrip: [<tip>]`。原仓库没有使用这个变异，也没有把编译失败计作检出。

### 3.2 获批边界的原始反例

- `TestCreationWindowWithApprovedBoundary`：Unix 精确 mkdir/open 空目录替换仍可接纳并删除替代物；Windows 原子创建 guard 在该点阻止实际替换。分别记录，不能都称为“替换后拒绝”。
- `TestApprovedUnixFinalWindow`：在最后身份核对通过后、unlink/rmdir 前，分别替换 owner 文件和空目录；Unix 可删除替代条目并返回成功。这是负责人明确接受的局部后果，不是新增代码阻止了它。
- 同用户往返干预 Clean 输入仍可能影响返回值，不能保证排除 false-clean；没有在本轮新复现 false-clean，也未把这个未验证场景冒称安全。Log 实际读取没有这条临时防护路径。

## 4. CI 固定身份

代码 CI：[run 39251919896](https://github.com/XKHoshizora/specmetry/actions/runs/39251919896)，attempt 1，HEAD `979f705d38d4f42412de13223096a02ee3ddd428`。

实际 checkout 为合并快照 `44c0b61595f47112afb1902f8092ffb1211ef2e8`，父提交依次为 `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`、`979f705d38d4f42412de13223096a02ee3ddd428`；树 `0c8c0790f8fcfbc55ab4b909e8f7c49794124d76` 与代码 HEAD 相同。

首轮 **8/8 成功**，没有重跑／取消。三平台 Go 测试、Go lint、Web、三平台 dashboard build 全部通过。已读取三个 Go job 的实际日志：新空覆盖、真实多进程、身份缓存及取消／三父图测试实际 PASS；Windows 原子创建／同句柄删除实际 PASS；macOS、Ubuntu 的获批 Unix 窗口测试实际 PASS。Ubuntu 普通 Test 带 `-race`，随后 root 跨用户步骤不带 `-race`，四项 Scratch 跨用户测试实际 PASS。

按设计未运行：三个平台的 `TestWP23ReadsWithAGitThatFetches`（未配置真实 unsafe Git）；Windows 的 POSIX 不可读对象存储测试；Windows／macOS 不运行 Linux 双账户基座。没有把跳过计为已验证。未把仓库外变异或 WSL 本机证据当作 CI 三平台实跑。

## 5. 失败、取消及重跑历史

- 首轮本机沙箱内 Git init 因临时目录 ACL Permission denied 失败；改在获授权的正常用户权限下运行隔离夹具，没有放宽产品 datadir 检查。
- 早期 Windows 原子创建 handle 请求 DELETE 访问，导致正常 datadir／ReadDir 遍历共享冲突；测试检出后改为创建 guard 无 DELETE，最终删除采用核对原身份的同一删除 handle。保留了完整权限检查断言。
- 初期新增测试有编译／未使用项及命名空间路径断言错误，Linux 首轮还发现复制 owner 后 inode 复用的夹具问题；改为保留原 owner 对象以保证替代身份不同，不将无效前提作为通过证据。
- 一次 Windows gitx 整包 `416.919s` 失败仅在新测试“实际 Log 提交号在 Args 中”的断言；生产 Log 经 stdin 传提交号，修正为检查实际 `--stdin` 调用。gitscan 同次 `63.252s` 通过。后续指定包／定向与 CI 结果分别记录，不把首次写为通过。
- lint 最初检出 7 项编译外问题，修正后为 0。缓存 linter 首次走 mise shim 触发离线版本解析，已停止本任务的该进程树并固定 Go bin 路径重跑；这不是产品错误，未修改全局工具配置。
- 仓库外 missing 路径变异的失败是预期检出；没有对生产代码作该回退。未反复运行压力直到通过。
- 原 CI 39163839528、39164780269、39202572684 的 lint 失败，审查 060 的旧 CI 39218204960 等记录保留，不改写历史。

## 6. 文件清单

本轮新提交／收尾涉及：

- 规范与状态：`AGENTS.md`；`docs/acceptance-criteria.md`；`docs/architecture.md`；`docs/development.md`；`docs/implementation-plan.md`；`docs/review-resolution.md`；`docs/wp2.3-git-reader.md`；`docs/wp2.3-b1-design.md`；本文件。
- gitx：`internal/gitx/call.go`、`clean.go`、`gitx.go`、`gitx_test.go`、`privdir.go`、`privid_unix.go`、`privid_windows.go`、`scan.go`、新增 `scan_probe.go`、`schemeb_test.go`、`schemeb_unix_test.go`、`schemeb_windows_test.go`，以及 `scratch_test.go`。
- 直接接点／测试：`internal/gitscan/scan.go`（仅 Scratch 注释）、`internal/gitscan/scan_test.go`、`internal/testutil/fakegit.go`、`internal/store/sqlite/crossuser_linux_test.go`（断言成功后只有空协议锁，仍保留跨用户权限检查）。
- 原已有未推送提交还包含 `docs/reviews/codex-review-060.md`，原样保留。历史审查 058～060 没有改写。

## 7. 剩余窗口与交接状态

获批边界仅限不合作同用户进程（包括误操作）干预 Git 私有临时命名空间。Unix 创建／最终删除窗口、Clean 可读替代输入且不能排除 false-clean，按 §71 接受；发现异常后仍保留、返回环境错误。跨用户保护、正常合作并发、数据库／来源权限、隐私与交付证据未获放宽，Log 漏读成功未获豁免。

Windows 证据限本机 NTFS 与 CI 实际平台，没有双账户动态运行，也不外推到所有文件系统。真实旧版 unsafe Git、本机 macOS 未新运行；CI 的 unsafe Git 用例按设计跳过。本机没有 Race Detector 结果，Ubuntu CI 的普通 -race 与 root 无 -race 步骤分开。没有做穷尽压力或完整 CLI 本地重跑，不改变 M-14／M-15、M-9 等遗留状态。

PR #27 保持未合并，未启用 Auto-fix／Auto-merge。独立会话继续定向复核；合并与验收仍由负责人决定。本实施会话不宣布 B1 关闭，不启动后续工作包。

## 8．Review 061 后续：两项阻塞的定向修复

本节接续上文历史交付，不把原 CI 通过改写成反例已被覆盖。负责人明确授权实施方修改、测试、提交并推送现有 PR #27（[决定 §72](review-resolution.md#72review-061-与两项定向修复授权)）；方案 B 批准范围不变。本节不是审查结论，不宣布 GO、B1 关闭或 WP2.3 验收。

### 8.1 固定起点与原样归档

- 起点 `8e089ab7ed5a12f250c30723d9fc1d1df75af6ea`，base `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`；本地与远程均为 `XKHoshizora/specmetry` 的 `feat/m2-wp23-git-reader` / PR #27。开始时仅有未跟踪 Review 061；既有提交、文件及报告全部保留，没有 reset、清理、覆盖或改基。
- 原样归档提交 `d4a42e614568a0a608d7efd55d7cd5655ea4a5b0`；源文件、仓库归档、暂存 blob 和提交 blob 逐字节一致，SHA-256 均为 `622940c44df6a80f6918676d5e6254c342dcb60e4291e1b2238186bb60d0388c`。Review 061 审查对象仍为 `8e089ab`，完整原文不改写。
- 原最终文档 HEAD CI [39253367700](https://github.com/XKHoshizora/specmetry/actions/runs/39253367700)，attempt 1，目标 `8e089ab`，8/8 成功；实际 merge checkout `a7f15f34488f8448d94b0389380367b4e140a2e7`，树 `99eb880dd3e38b8a04e549d0847072501b2b1202`。它与上文代码提交 `979f705` 的 CI 39251919896 分开，均不能证明 Review 061 的两个新调度安全。原审查的独立反例和本轮修复前失败分别保留。

### 8.2 根因、机制与代价

**B1-R1**：旧 checkContents 仅校验遍历到的 owner，没有要求它存在；最终 owner 删除之前，其他清单内容已经删掉。现在 checkOwner 先验证根链，重新打开路径上的 owner，检查普通文件、原身份、无链接身份和完整规范记录；完整遍历还必须实际看到 owner。创建填充、公开读取前后、清理预检及遗留认领共用校验。异常预检返回环境错误，现存目录、内容、替代物保留。只有本次可信删除成功后才设置 ownerGone，允许随后目录删除重试；不能把旧句柄有效或无 owner 遗留当作可信阶段。

**B1-R2**：旧租约只由调用方持有；其崩溃释放锁不代表真实 Git 结束。现在所有消费私有目录的 call 在 Start 前同步写入 `consumer_pending=true`；实际 Run/Wait 返回（或 Start 失败）、完整输入验证通过后才同步清除。它覆盖 Clean 的 status、ls-files、diff-index、check-attr、diff-files，以及空覆盖探测的全部正反对照。取消也先等待实际 Git。同步写入失败不启动／不发布结果；无效记录与未清除屏障禁止认领，锁释放不构成结束证明。没有生产 fd／handle 继承、进程监控／终止服务或仓库配置程序，也不以 PID、目录年龄或固定等待猜测结束。

协议升级为独立 `tmp/git-reader-v3/`；旧 v2 无消费者完成证明，不能在原子树中升级认领，v1／v2 遗留不接管。正常完成、取消及无消费者的完整可信遗留仍能回收。调用方在屏障期间死亡时，即使 Git 后来退出，也因没有持久完成记录而继续保留并 exit 6；同一 v3 命名空间后续私有读取会被阻塞，需另行确认安全后人工处理。本轮不新增自动恢复命令。这是明确的安全失败分支，不把两项阻塞排除为不合作同用户行为。

Windows 原子 FILE_CREATE、创建原身份、保护句柄及核对原身份的同句柄删除代码未变。扫描实际读取仍用两个显式空值，不依赖临时 missing 路径；PD-26、P3、禁 replace／commit-graph／MIDX、no-show-signature、严格失败与部分结果丢弃均保持。

### 8.3 先失败、后修复的本机证据

仓库外证据目录：`C:\Users\hoshi\.codex\visualizations\2026\10\19\01a153a5-24ad-7d82-b219-2787d737ce0a\fix061-1792416345689`。只运行新建临时仓库／home；不作用于真实项目数据。旧实现仅增加等价 Start/Wait 测试同步点和回归夹具，先编译并运行，再修改生产机制。后续收窄同步接口及改用结构化管道传参，不改变旧失败调度的含义。

| 证据 | 旧机制实际结果 | 新机制／对照断言 |
|---|---|---|
| `TestOwnerRequiredBeforeCleanup`，空／非空清单 × 缺失／损坏 owner | Windows、Linux 均 FAIL：预检通过；非空清单内容被删除，损坏记录也被接纳 | 环境错误，删除前完整快照不变；下一 Scratch 仍拒绝并保留 |
| `TestCleanMissingOwnerPreservesInputs`，读取前／读取后／清理预检 | 两平台均 FAIL；读取前缺失仍运行 Git，随后内容改变／删除 | 公开 Clean exit 6；读取前缺失不运行 status；现存内容完整保留 |
| `TestGitConsumerLifetimeProcesses`，Clean／真实空覆盖探测 × 调用方崩溃 | 两平台均 FAIL：调用方 exit 99、真实 Git 已确认暂停且未恢复时，NewScratch 成功并删除目录 | 同时检查错误和目录内容快照；Git 存活时不回收；恢复并原生确认 Git 退出后，未持久确认的屏障仍保留报错 |
| 同一多进程测试的 release／cancel | 旧机制四个正常对照 PASS | 新机制正常等待／取消等待结束，内容清空、租约可重用 |
| `TestConfirmedConsumerCompletionAllowsReaping` | 修复后新增正对照，不冒称旧实现执行了新字段断言 | 正常 Wait／取消前未启动／Start 失败均记录完成；故意只关句柄留下目录，下一 Scratch 安全回收 |
| 原重试、异常清单、v1／v2 隔离、扫描事实原子性 | 保留旧测试，不重新展开 B2／B3／N1 | owner 正常存在可清理；本次已删除 owner 后重试正常；额外／缺失／替代条目保留；扫描失败时提交、关联、seq、游标和既有字段不变 |

原始失败日志 `before-windows.log`、`before-linux.log`，第一轮绿色日志 `after-windows.log`、`after-linux.log`；最终定向结果另存 `windows-final-regressions.log`，Linux 最终结果另列。Windows 的 owner 移动调度实际执行成功，没有把“OS 阻止”算作该用例通过。生命周期测试用独立调用方进程、真实 Git、原生暂停确认与退出通知同步，不依赖睡眠；超时仅表示失败。测试结束受控终止／释放自己建立的进程及夹具。

最后补齐 owner 在**打开之前**的普通文件检查，覆盖检查与遗留认领两处，避免 FIFO 类型使预检阻塞。目录型 owner、复制完整记录但身份不同的替代 owner 在空／非空清单上均保留报错（Windows、Linux 实跑）；Unix 新增 `TestOwnerFIFORejectedBeforeOpen`（Linux 实跑，macOS 以 CI 为准）。此补充不改变 Unix 未观测名称替换的获批边界。

三平台故障夹具进一步使用明确输入闸门：真实 Clean 的 check-attr 消费原有路径输入后等待 EOF；真实探测 log 仅在测试调度中加 `--stdin` 并等待空输入 EOF。先确认原生暂停，再让调用方 exit 99；OS 关闭调用方管道不解除 Git 的暂停，下一 Scratch 的保留检查发生于恢复之前。正常释放原输入 EOF，取消则等待 Git 被终止。该管道不承载保护租约，不是生产 fd／handle 继承方案，也不改生产 Git 参数；测试用的过滤器程序名不能由 Clean 执行，正常对照会检出执行错误。`windows-gated-lifetime.log` 六项 PASS，`linux-gated-lifetime.log` 六项及 FIFO PASS。Darwin 停止回复按原生 [wait.h](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/sys/wait.h) 的状态与 SIGSTOP 确认，不能把共享 BSD helper 的判读当作动态成功。

**最终夹具的检出力**：仓库外导出 `f935fb9`，只将 call 的 beginConsumer／finishConsumer 调用退回 checkContents，保持最终输入闸门、真实 Git、清单和平台保护。Linux 实际运行两个崩溃用例均 FAIL：调用方 exit 99、Git 暂停未恢复时，NewScratch 返回 nil、原目录 ENOENT（`gated-mutant-linux.log`）。不是编译失败，也没有把该变异用于生产仓库。

| 平台／证据层 | 实际执行与限制 |
|---|---|
| Windows 本机 | Go 1.27.1、Git 2.56.0.windows.2、NTFS；NtSuspendProcess 成功确认测试 Git 暂停，进程 handle 等待实际退出。修复前失败与修复后 owner／崩溃／正常／取消对照均已运行；不是双账户或所有 Windows 文件系统验证 |
| WSL Linux 本机 | UID 1000、Git 2.43.0；交叉编译的 Linux 测试二进制，非 Linux 原生工具链。SIGSTOP + WUNTRACED 确认暂停；pidfd 的 POLLIN 确认退出。gitx 整包、修正预期后的 gitscan 整包 PASS；root 仅用于启动两个普通账户的 8 项跨用户测试，全部 PASS。没有本机 -race 结果 |
| macOS | 本机仅交叉编译，不算运行；最终代码 CI 的 darwin/arm64、Git 2.56.0 实际通过 owner 八组合、FIFO、公开 Clean 检查、完成回收及六项真实 Git 对照。原生退出观察使用预先注册的 kqueue NOTE_EXIT；不是以本机交叉编译补充运行证据 |
| 静态检查 | golangci-lint v2.14.0，包含测试的相关包检查 0 issues；最终 CI 另做全仓 tidy、vet、格式及 lint |

保留开发失败记录：新增 v2 隔离夹具最初漏建父目录，首轮 Windows 整包 gitx 543.062s 仅该项失败，其余通过；首轮 gitscan 118.739s／Linux gitscan 因旧的“移除额外文件即可恢复”预期失败，改为持久屏障继续拒绝，并检查剩余内容和业务事实不变后通过。sqlite 本机整包 36.044s 通过，没有修改生产 SQLite。初期测试辅助进程直接从环境传路径导致三条 G703 告警，定位到测试入口后改为自己的结构化管道协议；最终没有关闭规则或修改生产 Clean 路径。已有临时 linter 被外部清理后在本隔离目录重建固定版本，未改全局配置。

### 8.4 本轮 CI、文件与剩余边界

本轮提交按顺序为：

| 提交 | 内容 |
|---|---|
| `d4a42e614568a0a608d7efd55d7cd5655ea4a5b0` | 原样归档 Review 061 |
| `941d4e227f5e83d0aaa5f417cd088319ea21d3dd` | 两项生产机制、先失败回归、§72 及规范同步 |
| `c05cb4e67af523068d99647b339f672a3408c8a1` | owner 打开前类型拒绝与目录／替代／FIFO 对照 |
| `f935fb9300373ba44790737055d365866a3221a4` | 三平台明确输入闸门、Darwin 原生暂停判读与故障夹具收尾；本轮最终代码对象 |

首轮修复 CI [39264392152](https://github.com/XKHoshizora/specmetry/actions/runs/39264392152)，attempt 1，目标 `941d4e2`，**7/8**：仅 macOS Go tests 失败，不能记为三平台通过。实际 checkout `7836e040a407f46eff4e858521caadbf584c5cff`，父提交为 base／`941d4e2`，树 `fbcd7b11a3d1c805c49c0ffce3aed01a313e5dad` 与该代码对象相同。macOS 六个生存期子测试均未完成有效调度：5 次收到原生 SIGSTOP 回复，但 x/sys v0.49.0 共享 BSD Stopped 排除了 SIGSTOP；另 1 次 Git 在信号之前已经退出。修正为原生状态确认和上述明确输入等待，不重跑旧 CI 来覆盖失败，也不跳过 macOS。

最终代码 CI [39266213035](https://github.com/XKHoshizora/specmetry/actions/runs/39266213035)，attempt 1，目标 `f935fb9300373ba44790737055d365866a3221a4`，**8/8 成功**。8 个 job 的实际 checkout 均核对为 `50f832302443fcbfb45d38c2a2fb0905a0d07f12`，父提交依次为 `e0a21b46b5ad7e05146f254aa7523d6054ce2e76`、`f935fb9300373ba44790737055d365866a3221a4`；合并快照树 `ea7732a39bc12bb23155572ea5ea915a343b0f6e` 与代码对象相同。未重跑任何 attempt，没有把首轮失败覆盖成通过。

| job | ID | 实际日志核对 |
|---|---|---|
| Go tests (ubuntu-latest) | `118106341354` | Go 1.27.2 linux/amd64，Git 2.55.0；普通 Test 带 `-race`，修复／直接回归 PASS；随后单独 root 跨账户步骤无 `-race`，8 项 CrossUser PASS |
| Go tests (macos-latest) | `118106341345` | Go 1.27.2 darwin/arm64，Git 2.56.0；owner／FIFO、六项真实 Git 生存期、正常回收及获批 Unix 窗口实际 PASS |
| Go tests (windows-latest) | `118106341388` | Go 1.27.2 windows/amd64，Git 2.56.0.windows.2；owner 八组合实际执行，无 rename skip；六项真实 Git 对照、Windows 原子创建／同句柄删除 PASS |
| Go lint | `118106340871` | tidy、vet、仓库相对 Markdown 链接、全仓格式与 lint PASS |
| Web | `118106341391` | 格式、lint、类型、测试与构建 PASS |
| Build with dashboard (ubuntu-latest) | `118106564945` | 构建、嵌入 dashboard 测试、smoke PASS |
| Build with dashboard (macos-latest) | `118106564836` | 构建、嵌入 dashboard 测试、smoke PASS |
| Build with dashboard (windows-latest) | `118106564752` | 构建、嵌入 dashboard 测试、smoke PASS |

三平台日志均确认 owner 缺失／损坏／目录型／替代身份、读取前后及清理预检、正常完成／取消／启动失败、真实调用方 exit 99 后 Git 暂停、下一 Scratch 的完整保留和退出后仍拒绝猜测回收；没有把存在性与错误码单独当作内容保留证据。已核对原 manifest、重试及方案 B 扫描正反对照（空覆盖、实际二进制变化、不支持与取消、多父图、增量、缺失／损坏），并沿用 B2 bitmap、B3 排除目录与 N1 并发跳过的直接回归。生产 SQLite 或其他模块不因整仓测试通过而重新验收。

**按设计跳过／不适用**（全部不计为运行通过）：

- 三平台：`TestInitRefusesRealOldGit`、`TestARealGitThatFetches`、`TestWP23ReadsWithAGitThatFetches` 未配置真实 unsafe Git；记录型代理只验证相应分支，不能替代旧 Git 实跑。
- 三平台：M-14 诊断 `TestM14ReadOnlyProbe`、`TestM14OpenPath` 未开启；不是 M-14／M-15 已解决。
- Windows：`TestSourceListUnreadableSource`、`TestClassifyWithUnreadableStorage` 的 POSIX 权限调度跳过；Unix FIFO 文件不参与 Windows 构建，不算 Windows 阻止了 FIFO。Windows 双账户与其他文件系统没有动态证据。
- Ubuntu 普通非 root 步骤：8 项 CrossUser 跳过，随后 root 步骤实际运行并 PASS。Windows／macOS 不编译 Linux 双账户基座；不能把它们计为跨用户动态通过。

本文件的证据收尾是后续纯文档提交，代码仍为上述对象。**最终文档 HEAD CI 单独按该提交 SHA 核验，其完整 HEAD、run／attempt／8 job 与 checkout 回执写在 PR 当前描述及本轮交付回执**；不把本节的代码 CI 8/8 当作最终文档 HEAD CI，也不以旧 CI、变异检验或交叉编译代替它。

完整文件清单（相对固定起点）：

- 文档：`docs/reviews/codex-review-061.md`、`docs/review-resolution.md`、`docs/wp2.3-b1-implementation.md`、`docs/wp2.3-git-reader.md`、`docs/architecture.md`、`docs/acceptance-criteria.md`、`docs/development.md`、`docs/implementation-plan.md`。
- gitx：`internal/gitx/call.go`、`gitx.go`、`gitx_test.go`、`privdir.go`、`scan_probe.go`、`review061_test.go`、`owner_unix_test.go`、`consumer_unix_test.go`、`consumer_linux_test.go`、`consumer_darwin_test.go`、`consumer_windows_test.go`、`schemeb_test.go`、`scratch_test.go`。
- 直接接点：`internal/gitscan/scan.go`（仅 Scratch 注释）、`internal/gitscan/scan_test.go`（异常探测／未确认消费者的事实不变）、`internal/store/sqlite/crossuser_linux_test.go`（仅命名空间路径断言）。

共 24 个文件。没有修改迁移、推导、CLI、Schema、生产 SQLite 重试／错误映射或 CI 配置。后续证据收尾只追加／接续本报告，不改变代码对象。

获批的 Unix mkdir/open、最终 unlink/rmdir 窗口原始测试继续保留，不能写为新代码阻止；Clean 不能排除 false-clean，Log 完整性没有豁免。记录型 Git 代理仍不代表真实旧版 Git。unsafe Git、Windows 双账户／其他文件系统、压力与长时间故障运行等未验证边界分列，不扩大为安全结论。B2、B3、已关闭 N1 保持关闭；M-14／M-15 等遗留不变。Auto-fix／Auto-merge 保持关闭，PR 不合并，不启动后续工作包，不生成下一轮复审提示词。
