# Codex Review 061：PR #27 方案 B 的 B1 定向独立复审

审查日期：2026-10-19。审查者：Codex。本会话不是本轮方案 B 的实施会话；此前只参与实现前的设计分析及仓库外原型，没有编写或提交 `979f705` 的生产修复。

**NO-GO。扫描空覆盖、Windows 原子创建／同句柄删除及正常合作锁的主要修复成立；B1 在已批准的方案 B 契约下仍不能整体关闭。** 本轮独立复现两项残余：owner 已缺失时完整树检查仍通过、随后清理删掉其余内容；调用方崩溃后，下一次 Scratch 在真实 Git 子进程仍存活时回收其目录。前者缺少对协议必要对象的完整预检，后者把调用方进程退出误作整个读取生命周期结束。现有 CI 通过没有覆盖这两个调度。

接受负责人的局部保证调整，不重新选择方案 A／B。Unix mkdir/open、最终比较/unlink 的获批窗口，以及 Clean 不能排除 false-clean，**不再按旧绝对保证列为本轮阻塞**。Log 成功漏读未获豁免，其原临时防护路径问题已获得有效修复证据。

PR #27 暂不建议合并，WP2.3 本层验收依据尚未闭合。不代替负责人宣布验收，不启动后续工作包；B2、B3、已关闭 N1 保持关闭，M-14／M-15 等维护事项保持原状。

## 一、固定身份、工作区与适用范围

- [PR #27](https://github.com/XKHoshizora/specmetry/pull/27)。
- 固定 HEAD：`8e089ab7ed5a12f250c30723d9fc1d1df75af6ea`。
- base：`e0a21b46b5ad7e05146f254aa7523d6054ce2e76`。
- 上轮受审生产对象：`ee144af61e00cb4d24bbae27feab10d8be60ecfc`。
- 本轮主要范围：上轮对象至固定 HEAD 的 B1 增量；必要时追溯 Clean、扫描、子进程及遗留清理调用链。不重做整个 WP2.3、迁移或事务设计审查。
- 起始工作区 `C:/Users/hoshi/Desktop/specmetry`，分支 `feat/m2-wp23-git-reader`；origin fetch／push 均为 `https://github.com/XKHoshizora/specmetry.git`；本地 HEAD 与目标一致，暂存区、未提交及未跟踪文件均为空。
- 已读取适用根 AGENTS.md；未发现上级或 gitx／docs 的额外适用指引。完整阅读 Review 060、review-resolution §70／§71、B1 设计报告、实施交付记录及本轮规范／验收增量。以用户本轮只读复审授权为准，不沿用实施会话的提交／推送权限。
- 开始与收尾通过 GitHub 连接器核对远程：Open、未合并，HEAD／base 均与固定身份一致。未自行改换目标或更新工作区。

27 个增量文件中包含 Review 060 归档、历史状态、批准／规范、设计报告、实现及交付记录。生产代码改动集中于 gitx；gitscan 生产代码只改 Scratch 注释，测试增加探测失败及清理异常的不写入断言。未变生产 SQLite、错误映射、迁移、推导和 CLI 的既有结论继续保留。

### 提交链核验

| 提交 | 独立核对 |
|---|---|
| `659b914901bfd4fc9a251fc102c69b2ebcfb509b` | 仅归档 Review 060；是目标祖先，原提交保留 |
| `d6f9153bbde5cc5cd496edde9c55e4af8b080ba7` | 接续 §70 与实施计划历史状态；是目标祖先，未重置或重写 |
| `9462a56168c9a27a0413992da88cdc7a00c0ec95` | 批准／规范先于代码；记录 §71、方案 B 边界及设计报告格式归档 |
| `979f705d38d4f42412de13223096a02ee3ddd428` | 代码与测试实现；树 `0c8c0790f8fcfbc55ab4b909e8f7c49794124d76` |
| `8e089ab7ed5a12f250c30723d9fc1d1df75af6ea` | 仅文档交付及状态细化；树 `99eb880dd3e38b8a04e549d0847072501b2b1202` |

目标与 `659b914` 的 Review 060 文件无差异；当前文件 SHA-256 仍为 `25c6f4e2ea6e0ebbf2e24fe261fd0f929ae3c138c30534ea91c9641854b4fab2`。历史审查没有改写。设计报告的三个绝对 Markdown 链接改为纯文本并补归档说明，§12 明确说明并非与原稿字节相同；没有冒称原样归档。

## 二、已批准契约与范围判断

[review-resolution §71](../review-resolution.md) 与 [WP2.3 §14.3](../wp2.3-git-reader.md) 明确区分：

- 正常遵守协议的 Specmetry 并发受支持；私有目录从建立、Git 使用到清理均处在租约生命周期内。
- 不合作同用户进程对 Git 私有临时命名空间的干预，包括普通误操作，属于获批限定范围；Unix 创建／最终删除窗口及 Clean 的 false-clean 可能性明确保留。
- Windows 能原子创建和同句柄删除的部分仍须实现。
- 发现未知、额外、缺失或身份不符时保留并失败；清理失败不发布成功结果。
- Log 不能因防护路径替换而成功漏读；跨账户、数据库、来源权限、隐私及交付证据不获放宽。

本轮同步的 architecture、验收项和 §14.3 没有把批准扩大到数据库、来源或其他临时空间，也没有宣称旧绝对保证已全部实现。没有发现新增 broker、系统服务、业务重试、迁移或后续工作包等功能范围越界。下述两个问题属于**未兑现仍有效的异常处理／生命周期要求**，不是要求负责人重新批准方案 B。

## 三、BLOCKER：B1 仍有两项残余

### B1-R1［P2］：完整树检查遗漏 owner 的“必须存在”条件，预先缺失后仍读取并删除内容

位置：[privdir.go:575](../../internal/gitx/privdir.go#L575)，尤其 `checkContents` 对 `ownerName` 的特殊分支与结束时的数量比较；清理入口为 [removeOnce:664](../../internal/gitx/privdir.go#L664)。

`entries` 不含 owner。遍历遇到 owner 时会比较身份和类型；但**没有要求遍历一定遇到它**，最后仅比较普通条目的 seen 数与 entries 数。owner 在检查开始前已消失时，普通条目仍齐全，整个检查返回 nil。

**独立运行：WSL Linux，普通用户，真实 Git 2.43.0。** 在固定对象副本新增 `TestR61MissingOwnerBeforeReadAndCleanup`，不改生产算法：

1. 正常 prepare 一个干净仓库，完成创建清单。
2. 将仍被打开／加锁的 owner 移至本夹具 namespace 下的另一个名字，使原 owner 路径在检查前已缺失；没有在最终 fstatat/unlink 窗口插入替换。
3. 调用 checkContents、正常 `p.read` 和 `p.close`；检查原有 config 字节与存在性。

实际结果：

```text
checkContents = nil
p.read = clean / nil
p.close = environment error
configRead = ENOENT
configPreserved = false
```

清理先删完普通清单内容，直到最后按名删除 owner 才因缺失失败，最终重试结束报错。**公开 Clean 的整体返回并非成功**：它会传播最后的清理错误；本轮没有把内部 read 阶段的 clean 扩大为公开 false-clean 或错误 ready。违规点是完整预检未拒绝已存在的协议异常，且失败时已经删除其余内容，不能满足批准后的“缺失时在删除前保留”语义。

虽然最初移动属于同用户干预，本例检验的是干预之后、明确进入完整预检时的缺失状态；不是要求代码阻止获批的未观测最终窗口。用例与现有 `TestScratchManifestAnomalyIsRetained/missing` 相同层次，后者只删 HEAD，尚未覆盖必要 owner 的缺失。

**最小修复要求**：对尚未进入 ownerGone 阶段的目录，在读取／清理的完整预检中要求 owner 名称存在、类型及身份与持有对象一致；缺失时停止并保留全部尚存内容。区分正常已删 owner 的最终目录清理阶段。补 owner 缺失的读取、清理及后续遗留处理断言；不改变 Unix 最终 unlink 的获批边界。

### B1-R2［P2］：调用方崩溃释放两把锁，但 Git 子进程未必结束，下一进程会清理活动目录

位置：[Scratch.acquire:234](../../internal/gitx/privdir.go#L234)、[reapLocked:288](../../internal/gitx/privdir.go#L288)、[adopt:319](../../internal/gitx/privdir.go#L319)，以及 [commandIO:577](../../internal/gitx/gitx.go#L577)。

namespace lease 和 owner flock 都只由调用方持有。执行 Git 未传递保护租约，也未建立“调用方异常退出后 Git 必先结束”的受控生命周期。调用方进程退出，OS 释放其锁；下一 Scratch 便可采用完整记录并删除目录，未确认该目录对应的 Git 子进程是否仍活跃。

**独立运行：WSL Linux，普通用户，真实 Git status。** 在另一个固定对象验证副本，仅把 `cmd.Run()` 展开为相同的 `Start(); Wait()`，两者之间加入测试同步点。本机 Go 源码的 Run 正是这一顺序；没有更改锁、身份、删除或返回算法。

- 测试子进程正常调用公开 Clean，持有真实 v2 lease 和 owner 锁。
- `exec` 已启动真实 Git status 后，测试暂停该 Git 以确定性控制调度，并向上层测试报告 PID 与私有目录。
- 测试令调用方 `os.Exit(99)`，不执行 defer；不对命名空间做 rename、mkdir、unlink 或文件注入。
- 在 Git 尚未退出时调用下一次 NewScratch，核对 Git 存活和原目录存在性。

实际结果：

```text
real Git status still live = true
Linux /proc State = T (stopped)，不是已退出的 zombie
NewScratch = nil
private directory = ENOENT
```

这是确定性暂停模拟“Git 已启动但尚未完成”的调度，不是声称自然负载频率已测出，也不声称已复现数据库或源码损坏。测试结束会终止自身启动的 Git。其失效条件只需要**正常读取进程意外退出，Git 的执行寿命比它长**；没有利用获批排除的同用户命名空间替换。不能以“父进程已崩溃，锁已释放”证明全部 Git 使用结束。

现有 `TestScratchProtocolProcesses/crash` 的 helper 在建立目录和写入合成内容后退出，**没有同时运行 Git**，所以它有效验证 OS 释放锁及完整遗留回收，但不足以证明此情形下目录已经不活跃。

另有独立正对照 `TestR61CancelWaitsForRealGit`：在相同的真实 status 已启动调度下取消 context，Clean 等 Git 结束后返回错误、清空自身目录并释放租约，随后可以重新取得租约。该对照通过。**普通取消链成立；异常进程退出链不成立，不能混为一谈。**

**最小修复要求**：使私有目录的活动保护覆盖所有实际消费它的 Git 子进程，直到其退出；例如由执行侧持有可证明有效的租约，或建立并验证可靠的子进程终止／回收协议。不同平台的继承／锁语义必须实际证明，不预先认可某个候选机制。回收不能只看父进程锁释放、PID／年龄或 owner 记录。新增“Git 活跃期间调用方崩溃”的真实多进程反例、正常完成与取消对照。无须改选方案 A，也不需要改生产 SQLite。

## 四、本轮确认成立的部分

### 1. 扫描显式空覆盖与严格读取

`scanCall` 先校验 Scratch，经过 `scanGit → objectGitLocked` 的 PD-26 门禁，再固定返回的可执行路径执行。Tips、Classify 的 batch-check／确认调用、IsAncestor、Log 均走该后端；配置／路径等非对象查询保留既有路径。

两个值在过滤继承环境之后追加，**空值确实进入子进程**，不是删除变量。真实读取不创建防护 missing 路径；未发现路径回退。保留 `--no-replace-objects`、禁 commit-graph／MIDX、no-show-signature、P3 检查与全量失败语义；不复用原仓库 shallow／grafts 截断历史。

独立运行 `TestScanEmptyOverrides`、真实三父图及增量排除、缺失／损坏父对象、严格 Log、签名程序禁止等用例通过。记录型 PassAsIs 代理运行真实 Git，未伪造提交输出。仓库外仅将 scanCall 变异回 v2 missing 路径并在 Git 返回后恢复输入，原回归明确失败：

```text
TestScanEmptyOverrides: roundtrip: [<tip>]
预期两个提交，实际一个；FAIL
```

这是运行得到的检出，不是编译失败。目标后端完整返回两个提交；本轮没有复现目标代码因原防护路径替换而成功漏读。保留 Review 060 的原始反例及其固定对象，不把旧结果改写为新机制仍失效。

### 2. 行为探测、文件身份与缓存

[scan_probe.go](../../internal/gitx/scan_probe.go) 在 v2 lease 内手工写入合成对象，无远端、alternates、过滤器或 Git 写命令。根、子、双父 tip 构成正反对照：

- 未设覆盖时，两个截断文件使结果只有 tip。
- 两个空值须返回完整的 tip、子、根序列。
- 排除子提交须只剩 tip。
- 所需父对象分别损坏和缺失时必须非零。

这个探测图的第二个父提交也是第一个父提交的祖先，不足以单独证明非祖先多父场景；实际公开 Log 三父非祖先图的回归补上了该边界，本轮和三平台 CI 都有运行证据。

findGit 从打开的文件取得身份、size、mtime，并把 PATH 找到的同一路径用于实际执行；身份／大小／mtime 变化重置 version、PD-26、sparse 与 scanEmpty。探测成功后再次核对二进制，才记录 scanEmpty；失败、取消、清理异常不缓存成功。没有发现新的 PATH 再发现替代探测对象的调用。

`TestScanCapabilityFollowsExecutableIdentity` 实际换入不同身份、相同路径／大小／mtime 的代理文件，得到两次探测控制；取消探测释放租约、失败不缓存、unsupported 不运行用户 Log 的断言均通过。**这些代理证明门禁／缓存逻辑，不是某个真实旧 Git 版本的运行证据**。本轮未运行真实 unsafe 旧 Git；其最低能力支持不从代理测试推导。

探测自己的合成目录遵守 v2 清单与租约；清理发现未知文件时，探测整体失败，不缓存成功、不写扫描事实。其一般崩溃生命周期仍受 B1-R2 限制。

### 3. 正常合作锁、版本隔离与清单

`tmp/git-reader-v2/namespace.lock` 排他创建；已存在时打开并要求为空、身份一致。witness 与实际租约是不同打开句柄，长期锁文件不 unlink，避免合作方各自锁住不同文件。竞争非阻塞失败，不增加业务重试。

每个 privateDir 的 lease 覆盖创建、填充、所有 Git 调用与清理；正常失败路径关闭自己的 handles 并释放。Clean 先关闭父私有目录，再递归子模块，未发现嵌套获取自身租约的死锁。scanGit 持 stateMu 时以已选定路径执行探测，不递归获取 Git 状态锁。正常多进程竞争、release 和无在途 Git 的 crash 回收测试通过。

v1 固定对象只枚举顶层 tmp，且只认 `read-<32hex>`，不会进入 `git-reader-v2`；v2 只在子树内采用 v2 owner／清单，既不接管顶层 v1，也拒绝子树中 v1 记录。这一隔离有旧代码选择规则和新测试支持，不依赖“未发现旧进程”。**作者及本轮均未实跑旧二进制双版本并发**；对此名称布局隔离的源码与直接测试证据充分，不单独构成阻塞，但不扩展成整个旧版共存已动态验证。

清单记录排他创建句柄取得的身份、相对名和类型；写内容前保存记录。目录建立后逐级登记。正常清理先预检整树，按后序逐项核对删除并更新记录；不再 RemoveAll 猜测未知树。额外、普通内容缺失、身份替换、复制 owner、未知格式等已有用例保留并失败，重试重新核对；完整可信遗留可回收。部分建立无完整记录、部分清理记录不一致或已无 owner 时保守保留并拒绝后续操作，可能需要人工处理，未把它们误称为自动可回收。owner 必须存在的遗漏单列 B1-R1。

### 4. Windows

[privid_windows.go](../../internal/gitx/privid_windows.go) 的 NtCreateFile 使用相对已持有父句柄、FILE_CREATE 与 DIRECTORY_FILE，已存在名称拒绝；创建返回的身份与后续打开身份比较。创建 guard 不共享删除，覆盖实际使用期；避免了为创建 guard 请求 DELETE 导致正常遍历共享冲突的早期问题。

最终目录删除在内容与 Git 使用结束后释放 guard，再打开带 DELETE 的句柄，比较**原创建身份**；随后在这个同一核对句柄上设置 disposition。没有先比较再另一次按名删除。owner／清单条目同样经该核对／disposition 链路。

本机 NTFS 的 `TestWindowsAtomicPrivateDirectory` 实际通过：排他冲突、创建身份、持有期间阻止 rename；取得删除句柄后移走原对象并建立同名目录，删除只作用于原对象、替代物保留。相关正常权限遍历、Clean 与子模块生命周期测试通过。CI Windows 也实际运行该用例。其他持有期间 rename 的拒绝是 **OS 阻止**，本机／最终 CI 不冒称完成替换后的代码检测；最终 CI 有 14 条 Windows refused the move 记录。

未新验证 ReFS、其他文件系统或 Windows 双账户动态操作，不作外推。

### 5. 错误、取消、业务状态与直接回归

Git／探测／清理／回调错误仍阻止整个扫描发布。gitscan 的候选先读后在单事务发布、部分读取丢弃的生产算法未变，未增加 SQLite 重试或错误映射。

新增两类失败不写测试分别为 unsupported 探测和探测目录未知内容。状态快照比较：

- seq；
- commits 的已有 project／SHA／seq／时间；
- 全部已有关联的 SHA／目标／来源；
- 全部游标字段；
- 每个事实表的行数。

这些断言超出仅看返回码或几张表行数；并非所有数据库列的逐字节比较，未扩大其证明边界。原部分读取丢弃、浅克隆生命周期、正文不落盘与并发跳过直接回归，本机两平台及最终 CI 均通过。并发跳过仍在内层完成、外层写入前取得状态快照，旧 N1 未回归。

B2 截断位图／损坏索引直接回归、B3 排除与稀疏门禁直接回归通过；没有重新打开 B2、B3、旧 N1。取消等待真实 Git 结束的独立正对照与调用方异常退出的反例分别记录，不用一个替代另一个。

## 五、NON-BLOCKER：活动描述仍有过时状态

**N-doc：PR 顶部描述仍停留在 Review 059 修复、等待 Review 060，未反映获批方案 B 和当前对象。** 起始／收尾读取 PR 描述均见：

```text
Fix for Codex review 059 — B1 fixed, awaiting independent re-review
Status: B1 fixed, awaiting independent re-review (Review 060)
```

正文仍以 `ddb908a／ee144af` 及旧目录机制为当前修复，容易使读者误解目标代码与批准边界。建议更新当前摘要、目标提交、Review 060 结论、§71 批准、新实现与最终 CI 的身份；历史记录可保留为明确的历史段落。该项是新文案问题，不是重开旧 N1。

活跃 §14.3、architecture、实施计划和交付记录已明确接续批准，没有扩大豁免。开发指南的目录地图仍概括为 tmp 句柄／旧 owner 身份机制，末尾虽补了新协议，建议同步主表避免与 v2 实际对象混读。设计报告中“未提交／尚未批准”由 §12 明确标作历史原稿，§71 是活跃授权；不要求改写历史分析或 Review 060 原文。

## 六、独立运行、作者证据与边界

所有本轮运行在固定 HEAD 导出的仓库外目录进行。原仓库未切换、不改生产代码或测试。

| 本轮独立验证 | 环境／结果 |
|---|---|
| 固定副本 gitx 定向组 | Windows/amd64，NTFS，Go 1.27.1，Git 2.56.0.windows.2；198.660s，PASS |
| 同组 Linux | WSL Ubuntu，普通 UID 1000，真实 Git 2.43.0，Linux 6.18.33.2-microsoft-standard-WSL2；PASS；Linux 临时目录新建夹具 |
| gitscan 失败不写、部分丢弃、并发跳过、浅克隆与正文隐私直接组 | Windows 62.533s，PASS；同一组 WSL PASS |
| 新增独立 owner 缺失用例 | WSL；生产算法未变；按保证断言 FAIL，结果见 B1-R1 |
| 新增真实 Git 活跃时调用方崩溃 | WSL；仅 Start／Wait 间调度；按保证断言 FAIL，结果见 B1-R2 |
| 新增真实 Git 活跃时取消正对照 | WSL；PASS，Git 已退出才清理／释放，可重新取租约 |
| scanCall 退回 missing 的变异 | WSL；编译成功，真实执行，原 TestScanEmptyOverrides 因 1/2 提交 FAIL |

固定副本定向组覆盖空覆盖、unsupported、二进制身份、探测取消、三父图、Scratch API、合作双进程、v1/v2、清单异常、原私有目录替换／重试／生命周期、Windows 创建删除，以及 B2／B3 的直接项。Windows 不编译 Unix 最终窗口用例，不以未运行当通过。原 Unix mkdir/open 与最终 unlink 用例通过表示**获批限制被准确重现**；不是代码阻止这些域外调度。

实验材料保留于：

`C:/Users/hoshi/.codex/visualizations/2026/10/19/01a153a5-24ad-7d82-b219-2787d737ce0a/review061`

其中 `target.tar` SHA-256 为 `3a66f66a23f5ab5d6c7daa13dac9e183b2e354808992853319ff096e5e15d399`；manifest.json 记录验证源码、变异源码及日志散列。target 是未改动目标副本，validation 仅含新增审查用例和子进程同步点，missing-mutant 只改 scanCall。新增 Linux 测试二进制由 Windows Go 工具链交叉编译；随后在 WSL **实际运行**，不称为 Linux 原生编译。

本轮无编译失败冒充反例检出的情况。前次已完成的设计原型与作者实施实验不计作本轮独立运行。本轮未独立重做作者的全包本地耗时、旧版 Git、跨用户 root 基座、macOS 本机运行、ReFS、压力／模糊或本机 Race Detector，也没有全仓／CLI 本地重跑。macOS 与跨账户证据来自下一节独立读取的 CI 日志，标明来源。

作者交付记录如实保留了早期沙箱权限失败、Windows DELETE guard 遍历冲突、测试／编译修正、错误的 Args 断言与 lint 失败历史；本轮不把这些纠正前运行当成功。记录型 Git 代理未被描述为真实旧版 Git。M-14／M-15 和 M-9 仍未关闭。

## 七、最终 CI 身份与实际日志

独立读取 [最终 run 39253367700](https://github.com/XKHoshizora/specmetry/actions/runs/39253367700) 的原始元数据：pull_request、HEAD 为固定 `8e089ab`、**attempt 1**、completed／success，8/8 作业成功。独立读取全部八个 job 日志，实际 checkout 全部相同：

```text
merge a7f15f34488f8448d94b0389380367b4e140a2e7
parent e0a21b46b5ad7e05146f254aa7523d6054ce2e76
parent 8e089ab7ed5a12f250c30723d9fc1d1df75af6ea
tree   99eb880dd3e38b8a04e549d0847072501b2b1202
```

父提交及树经 GitHub Git Data 对象独立核对；树与本地固定 HEAD 相同。没有只用 run 的 head_sha 代替 checkout 身份。

| Job | ID | 实际结果 |
|---|---|---|
| Go lint | `118062415612` | success |
| Web | `118062415846` | success |
| Go tests macOS | `118062416050` | success；Go 1.27.2 darwin/arm64，Git 2.56.0 |
| Go tests Ubuntu | `118062416115` | success；Go 1.27.2 linux/amd64，Git 2.55.0 |
| Go tests Windows | `118062416178` | success；Go 1.27.2 windows/amd64，Git 2.56.0.windows.2 |
| Build with dashboard Ubuntu | `118062556190` | success |
| Build with dashboard Windows | `118062556304` | success |
| Build with dashboard macOS | `118062556574` | success |

三平台日志中新增空覆盖、unsupported 代理、身份重探测、探测取消、多父图、合作双进程、版本隔离、清单异常实际 PASS。macOS／Ubuntu 的获批 Unix 窗口实际运行并记录残余；Windows 的原子创建／同句柄删除实际 PASS。B2／B3、并发快照 N1、扫描失败不写及部分丢弃的直接项实际 PASS。

最终 CI 的 gitx／gitscan 包耗时分别为：macOS 83.526s／17.264s；Ubuntu 161.079s／39.265s；Windows 256.744s／60.567s。这是最终文档 HEAD 的 CI，不能与作者本机耗时混用。

Ubuntu 普通 Test 的脚本与实际 Linux 环境选择 `-race`；普通步骤中四项 CrossUserScratch 因需要 root 而 SKIP。随后专门步骤另行 `go test -c` 编译、sudo 执行，四项 CrossUserScratch **实际 PASS**，该二进制没有 `-race`。Windows／macOS 的 root 步骤按设计跳过，也不编译 Linux 专属双账户测试。没有把这些结果称为 Windows 双账户验证。

三个平台 `TestWP23ReadsWithAGitThatFetches` 均因未配置真实 unsafe Git 而 SKIP；Windows 的不可读对象存储测试以及平台相关不可读来源项 SKIP。跳过不计为通过。新增两项独立反例不在该 CI 的已提交测试中，故 8/8 成功不能覆盖它们。

### 与代码 CI 的区分

[代码 run 39251919896](https://github.com/XKHoshizora/specmetry/actions/runs/39251919896) 独立核对元数据为 HEAD `979f705`、attempt 1、success、8 个 job 成功。作者提供的合并对象 `44c0b61595f47112afb1902f8092ffb1211ef2e8` 经 Git Data 核对，父提交为 base／`979f705`，树为 `0c8c0790f8fcfbc55ab4b909e8f7c49794124d76`，与代码 HEAD 相同。本轮未重读代码 CI 的全部 job 日志；上面完整日志结论来自**最终 run**。两轮身份和内容树分开保留。

保留此前 CI lint 失败与 Review 060 的旧 CI 历史，不将本轮两个首轮成功解释为整个 PR 从未失败。

## 八、裁定与下一步

- **BLOCKER**：B1-R1（owner 缺失预检／保留）和 B1-R2（调用方崩溃时 Git 活动保护）仍需修复。它们不由本次局部保证调整豁免。
- **NON-BLOCKER**：N-doc，更新 PR 当前描述及开发指南主表；不改写历史审查。
- **实现范围**：未发现新的功能范围越界或文档把同用户豁免扩大到数据库／来源／跨用户／隐私等保证；已有契约的两处实现缺口如上。
- **B1 能否关闭**：在获批方案 B 下仍不能整体关闭。扫描原 Log 防护路径问题及 Windows 子机制可保留本轮成立结论。
- **PR #27**：**NO-GO，暂不合并**。
- **WP2.3 本层验收**：现有依据不足以支持本层交付验收；不否定此前未变模块结论，也不代替负责人宣布验收。
- **产品决定／接受风险**：沿用 §71 已批准的限定边界，无需重新选择 A／B；不新增豁免或关闭维护事项。

最小接手范围为 B1-R1 的完整预检与 B1-R2 的 Git 子进程／租约寿命，补精确反例、正常结束和取消对照，保持已通过的空覆盖、Windows 原子性和业务失败不写契约。实施方完成后仅定向复核这些修正及直接回归；不重开 B2／B3／旧 N1，不改生产 SQLite 重试／错误映射，不启动后续工作包。合并与本层验收仍由负责人决定。

本轮仅新增本未提交报告；仓库外保存最小验证材料。没有修改原仓库其他文件，没有提交、推送、合并或改基。收尾远程身份仍与固定目标一致。

