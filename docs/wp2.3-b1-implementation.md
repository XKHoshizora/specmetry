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
