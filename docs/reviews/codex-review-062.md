# Codex Review 062：PR #27 的 owner 预检与真实 Git 消费者生存期定向复审

审查日期：2026-10-19。审查者：Codex。按负责人本次纠正后的角色指令执行只读审查；此前本会话参与过方案 B 的上一轮实施，该事实不隐去。本报告只以本轮重新读取的固定对象、源码、CI 日志及仓库外复验为证据，不把此前实施自测转换为本轮独立验证。

**最终结论：NO-GO。** Review 061 的 B1-R1（owner 预检）可以关闭；B1-R2 的原始“调用方崩溃、真实 Git 尚存活却被回收”反例已修复，正常完成、取消及启动失败对照也成立。但完成记录的 `Sync` 失败后，内存状态已经解除屏障，公开 Clean 随后的清理会删除全部临时内容，违反本轮明确要求的“写入、同步、验证失败时保守保留”。该直接残余使 B1-R2 尚不能完整关闭，B1 整体仍未关闭。

这不是声称目标版本仍误删活动 Git 的目录，也不是成功漏读、错误 ready 或数据库污染的复现。唯一剩余 BLOCKER 是下述完成持久化失败分支；不重新展开整个 WP2.3，也不要求改变已批准的方案 B。

## 一、固定身份与范围

| 对象 | 本轮核对身份 |
|---|---|
| PR | [XKHoshizora/specmetry#27](https://github.com/XKHoshizora/specmetry/pull/27) |
| HEAD | `d121bfb42fdbdb922fc136265a3792a86875709b` |
| base | `e0a21b46b5ad7e05146f254aa7523d6054ce2e76` |
| Review 061 对象／主要增量起点 | `8e089ab7ed5a12f250c30723d9fc1d1df75af6ea` |
| 最终代码对象 | `f935fb9300373ba44790737055d365866a3221a4` |

开始和收尾均经 GitHub 连接器核对 HEAD/base，未发生变化；PR Open、未合并，`auto_merge=null`。本地分支为 `feat/m2-wp23-git-reader`，HEAD 与目标一致。开始时工作区干净，报告文件不存在。审查中未切换、改基、提交、推送或修改原仓库实现。

已读取适用根 AGENTS.md、完整 [Review 061](codex-review-061.md)、[review-resolution §71／§72](../review-resolution.md)、[实施记录 §8](../wp2.3-b1-implementation.md)，以及规范、验收项、开发说明和实施计划的本轮增量。使用 `codex-security:verify-fix` 的原反例、控制、调用链与反证核验方法；按用户要求保存本 Markdown 报告。

`8e089ab → d121bfb` 为 24 个文件，提交依次为：

- `d4a42e614568a0a608d7efd55d7cd5655ea4a5b0`：归档 Review 061。
- `941d4e227f5e83d0aaa5f417cd088319ea21d3dd`：两项生产修复、回归及规范同步。
- `c05cb4e67af523068d99647b339f672a3408c8a1`：打开 owner 前的类型检查及对应测试。
- `f935fb9300373ba44790737055d365866a3221a4`：跨平台真实进程调度夹具修正。
- `d121bfb42fdbdb922fc136265a3792a86875709b`：仅更新实施交付记录；代码与 `f935fb9` 相同。

生产改动集中在 `gitx/privdir.go`、`call.go`、`gitx.go`、`scan_probe.go`；gitscan 生产文件仅改协议注释，其余为测试和文档。平台创建／删除实现、生产 SQLite、迁移、推导、CLI、Schema 未变。没有发现新增业务范围或扩大方案 B 的批准边界。

Review 061 与归档提交 `d4a42e6` 无差异，当前文件 SHA-256 为 `622940c44df6a80f6918676d5e6254c342dcb60e4291e1b2238186bb60d0388c`，对象仍指向 `8e089ab`，NO-GO 原文未改写。§72 记录的是实施授权和待复审状态，没有代替审查关闭。

## 二、BLOCKER：B1-R2-S［P2］完成记录同步失败后仍清理，而非保留

### 位置与逻辑证据

- [privdir.go:680](../../internal/gitx/privdir.go#L680)：`finishConsumer`。
- [privdir.go:478](../../internal/gitx/privdir.go#L478)：`saveRecord` 的 Truncate、WriteAt、Sync 顺序。
- [privdir.go:717](../../internal/gitx/privdir.go#L717)：`remove` 仅以 `consumerPending` 阻止清理。
- [call.go:76](../../internal/gitx/call.go#L76)：实际命令结束后调用 `finishConsumer`，失败传播。
- [clean.go:78](../../internal/gitx/clean.go#L78)：读取失败后仍执行 `p.close()` 清理。

`finishConsumer` 在 `saveRecord` 成功之前先执行 `d.consumerPending = false`。如果 owner 的完整 `false` 记录已经写入，但最后的 `Sync()` 返回错误，磁盘当前可读字节与内存记录仍一致。返回的错误声称 `its directory was retained`，但没有留下本次调用不可继续清理的失败状态。随后 `remove()` 看到 pending=false，`checkOwner` 又能读取到完全匹配的记录，因而删除清单内容、owner 和目录。

这不是输入身份变化的竞争窗口，也没有涉及获批排除的不合作同用户干预；只需要文件系统在完成记录同步时返回一次错误。它直接违反本轮要求“写入、同步、验证失败时保守保留，不发布成功结果”。**不发布成功结果这一半成立，保留这一半不成立。** 不能把它作为已经获批的局部风险。

### 独立复验

在固定 HEAD 的仓库外验证副本，仅在 `saveRecord` 最后一处 `d.lock.Sync()` 调用边界增加故障注入点：保持原有 Truncate/WriteAt、记录字节、身份、Start/Wait、返回和清理代码不变；指定的一次 Sync 调用模拟返回 I/O 错误，后续调用恢复正常。没有制造物理磁盘故障，也未把该测试代码写入原仓库。

Windows/NTFS 与 WSL Linux 均实际编译运行：

| 独立用例 | 实际结果 |
|---|---|
| `TestR62SyncFailureRetainsTheDirectory/begin` | PASS：启动前同步失败，真实 Git 启动数为 0，pending 保持 true，清理拒绝，全部现存内容保留 |
| 同用例 `/finish` | FAIL：真实 Git 已启动并结束，读取返回 environment，随后 remove 返回 nil，目录及 config/owner 消失 |
| `TestR62PublicCleanCompletionSyncFailure` | FAIL：公开 Clean 的真实 status 已 Start、已 Wait；仅完成记录的那次 Sync 失败，Clean 返回 environment／exit 6，但其清理删除整个私有目录及全部清单内容 |

公开路径的核心观察在两平台一致：

```text
status started = true
status waited = true
one completion Sync failure injected = true
Clean error = cannot persist the private Git consumer completion;
              its directory was retained; ... no result is accepted
private directory after Clean = ENOENT / file not found
```

删除前后比较的是完整临时树快照，并非只看返回码。Clean 的 State 零值打印为 `clean`，但伴随非 nil 错误；本报告**不将其算作成功 clean**。该实验未证明活动消费者被删除、正式事实被错误发布或外部文件受损。其确定影响是同步失败的保留契约失效，以及错误说明与实际清理结果相反。

### 最小后续修复与验收

完成记录只有在写入、同步成功之后，才可在本次调用内转为允许清理／复用的状态。任一失败须保留一个不可被普通 cleanup 覆盖的失败状态，只关闭自身资源并返回环境错误；不能仅因当前可读字节已经是 `false` 就忽略刚收到的同步错误。可以采用独立的完成持久化失败状态，或调整状态提交顺序，具体工程选择由实施方完成。

补“完整写入成功、仅完成 Sync 失败一次”的直接和公开 Clean 回归，断言所有现存内容保留；同时保持启动前失败零启动、正常完成／取消／Start 失败的成功清理。检查失败状态不会在同一次错误收尾中被重新覆盖。无需新增恢复命令、broker、迁移或产品豁免；不以扩大风险接受替代该修复。

## 三、Review 061 两项的逐项裁定

### B1-R1：可以关闭

[checkOwner](../../internal/gitx/privdir.go#L582) 在完整树遍历之前验证根链、owner 路径存在及普通文件类型，再打开名称上的对象，比较原文件身份、无链接路径身份和完整规范化记录。旧的打开句柄仍有效不能代替路径上的 owner；`checkContents` 还要求遍历实际看到 owner。

`adopt` 也在打开 owner 之前检查普通文件类型，随后执行记录解析、锁、目录身份和完整预检。Unix FIFO 被类型预检拒绝，不会先阻塞于只读打开。创建内容、消费者开始／结束和删除预检均接入此控制。

本机 Windows 与 WSL 实跑：

- 空／非空清单 × owner 缺失、记录损坏、目录型、复制记录但身份不同的替代物，全部 environment，读取／清理／下一 Scratch 拒绝，完整现存内容不变。
- 公开 Clean 的读取前、实际 Wait 后、删除预检三处移动 owner，全部拒绝并保留；读取前异常不启动 status。
- Unix FIFO 对照通过；Windows 本轮 owner rename 调度实际完成，没有把 OS 阻止 rename 计作该用例通过。
- 正常清理、正常重试以及本次可信删除 owner 后的目录重试通过。`ownerGone` 仅在可信 `removeEntry` 成功后设置，进程重启不能凭无 owner 遗留猜测进入该阶段。

另行用原始 JSON 字节构造缺少 `consumer_pending`、null、true、截断、未知字段、第二份 JSON、重复矛盾字段七类记录；关闭原持有句柄后尝试 NewScratch，七类均拒绝且快照不变。虽然 JSON bool 的缺项会先解为 false，后续 `checkOwner` 与完整规范记录的字节比较使缺项不能成为可回收证明；没有把 `loadRecord` 单个条件当作完整防线。

对照 `8e089ab` 的隔离副本，仅加入同等 Start/Wait 调度钩子及相关回归夹具，不改变旧身份／清理算法：owner 预检用例和公开 Clean 用例实际 FAIL，原因确为预检误通过及内容删除。目标版本相应用例 PASS，不是利用无关失败关闭问题。

### B1-R2：原崩溃反例关闭；完成同步失败残余阻止整体关闭

正常路径的顺序成立：`beginConsumer → checkContents → consumer_pending=true → WriteAt 完整检查 → Sync → commandIO Start/Run → Wait → checkContents → 完成记录`。不能建立启动屏障时不会进入命令执行。独立只读 owner 句柄故障对照确认真实 Git 启动数为零；实际 Start 钩子重新读取 owner，看到完整 true 记录。

所有消费本轮私有目录的 Git 调用已接入 `call.private`：Clean 的 status、ls-files、diff-index、check-attr、diff-files，以及空覆盖探测的全部正反对照。实际扫描仍使用空 graft/shallow 覆盖，不恢复可替换的 missing 路径。未发现本次遗漏的私有目录消费者。

真实消费者调度在本机 Windows 和 WSL 分别运行两类消费者 × release/cancel/crash 六项：

1. Clean 使用真实 check-attr；空覆盖探测使用真实 log。测试仅以输入闸门延后 EOF，先完成原生暂停确认，再让调用方 `os.Exit(99)`，不运行 defer。
2. 另一进程建立 Scratch 前，本轮额外核验操作系统状态：Linux `/proc` 为 `T (stopped)` 且 executable 为 `/usr/bin/git`，不是 zombie；Windows 进程句柄未 signalled，退出码 STILL_ACTIVE。没有用 PID 存在代替实际存活。
3. NewScratch 在 Git 仍存活时环境错误，目录与全部内容快照不变。
4. 测试恢复 Git，使用 pidfd／Windows 进程句柄等待其实际退出；缺少原调用方持久完成证明时，再次 NewScratch 仍拒绝且内容不变。
5. release、取消、取消前未 Start 和实际 Start 失败的正常对照均通过；已完整确认结束后留下的可信遗留可以回收。

`commandIO` 的生产 `cmd.Run()` 与测试分支 `Start(); hook; Wait()` 顺序等价；begin 在调用 commandIO 之前，finish 在其返回之后。输入闸门没有传递租约，不能代替生产屏障。probe 的测试 `--stdin` 仅使真实 Git 等待空输入 EOF，不改生产 Log 参数或输出。正常 release/cancel 证明该夹具能够完成真正的读取／等待，而非永远暂停后只测错误码。

旧对象同一崩溃夹具在 Clean/probe 两条路径都实际失败：Git 暂停未恢复，下一 NewScratch 却成功并删除目录；四个 release/cancel 正对照通过。另在目标隔离副本仅把 begin/finish 接点改回 checkContents，撤屏障变异成功编译后，两条 crash 回归同样 FAIL。这是本轮实际检出，不是沿用作者变异记录或把编译失败算作检出。

因此原始 B1-R2 路径得到有效修复证据；但第二节的完成 Sync 失败分支仍在同一个持久化控制内，不能宣称整项已关闭。

## 四、v3 协议、批准边界与直接回归

v3 位于独立 `tmp/git-reader-v3/`。旧 v1 顶层 read-* 和 v2 子树不认领；旧代码的枚举范围不会进入 v3。版本隔离测试实际保留旧目录及内容，v3 内旧协议 owner 拒绝。不把命名隔离的源码／夹具证据扩大为已实跑全部旧二进制共存。

调用方在消费屏障期间崩溃，可能持续阻塞同一 v3 命名空间后续私有读取，即使 Git 后来已经退出。这符合本轮“不确定即保留并失败”的授权：规范 §14.3、architecture、开发说明和实施记录明确披露持续阻塞、临时数据保留及需另行确认后人工处理。没有授权自动恢复命令，本报告也不以缺少该命令为门槛。

正常完成、取消、启动失败和无在途消费者的可信遗留没有发现错误残留屏障。异常输入导致结束检查失败时，移除注入的额外文件也不能代替完成证明，gitscan 的持久屏障回归正确拒绝。这一保留分支与本报告发现的“Sync 错误后反而删除”须分开，不能互相代替。

方案 B 的获批同用户限定、Unix mkdir/open 与最终 unlink/rmdir 窗口，以及 Clean 不能排除 false-clean 仍按 §71 适用；未声称代码已经阻止这些域外干预。不扩大到其他账户、合作进程崩溃、数据库、来源权限、隐私或交付证据，不豁免 Log 完整性。没有新增产品取舍或风险接受。

Windows 原子 FILE_CREATE、创建身份／保护句柄及同句柄删除的生产代码未变；本机 NTFS 的原子创建／删除直接回归通过。其他替换测试中 OS 阻止操作的情况，仅记录为阻止，不称为替换后动态拒绝。没有 Windows 双账户或全部文件系统动态证明。

两平台直接回归保留空覆盖真实传递、两提交往返、多父合并、增量排除、缺失／损坏、实际二进制身份变化重探测、unsupported／取消不缓存、清理重试及合作锁。B2 索引边界、B3 排除／稀疏门禁、N1 并发跳过按直接组复验，没有重开。

gitscan 的失败不写、部分读取丢弃、浅克隆生命周期、正文隐私和并发跳过组在 Windows 与 WSL 均 PASS；测试比较 seq、已有提交字段、全部关联、游标和事实计数，未只用行数代替更新检查。本轮同步故障注入验证到公开 Clean 和 call，不冒称已在该注入条件下独立执行所有数据库调用；未变扫描发布路径遇到错误不会接受结果。

## 五、CI 身份、真实运行与失败历史

本轮分别读取两组最终 run 元数据、全部 16 个 job 的日志及 Git Data 合并对象；各 job 的 checkout 后 `git log -1 --format=%H` 全部与下表相符。不是仅用 Actions 的 head_sha 推断检出对象。

| 证据 | 最终代码 CI | 最终文档 HEAD CI |
|---|---|---|
| run | [39266213035](https://github.com/XKHoshizora/specmetry/actions/runs/39266213035) | [39267810108](https://github.com/XKHoshizora/specmetry/actions/runs/39267810108) |
| HEAD | `f935fb9300373ba44790737055d365866a3221a4` | `d121bfb42fdbdb922fc136265a3792a86875709b` |
| attempt／结果 | 1；completed/success，8/8 | 1；completed/success，8/8 |
| 实际 checkout | `50f832302443fcbfb45d38c2a2fb0905a0d07f12` | `f841faef7a159af478a6ccfdcdc3e18f4d20db1f` |
| 第一父提交 | `e0a21b46b5ad7e05146f254aa7523d6054ce2e76` | 同左 |
| 第二父提交 | 该列 HEAD | 该列 HEAD |
| tree | `ea7732a39bc12bb23155572ea5ea915a343b0f6e` | `29d03231f642127f506b8c68a66466c72ce46dd5` |
| 与对应 HEAD 树比较 | 相同 | 相同 |

| Job（均 success） | 代码 run job ID | 文档 run job ID |
|---|---|---|
| Go lint | `118106340871` | `118111887651` |
| Go tests Ubuntu | `118106341354` | `118111888039` |
| Go tests macOS | `118106341345` | `118111888062` |
| Go tests Windows | `118106341388` | `118111888150` |
| Web | `118106341391` | `118111888246` |
| Build with dashboard Ubuntu | `118106564945` | `118112056579` |
| Build with dashboard macOS | `118106564836` | `118112056516` |
| Build with dashboard Windows | `118106564752` | `118112056578` |

两组的三平台 Go 日志均实际 PASS owner 八组合、公开 Clean 三阶段、完成回收三种正常路径及真实消费者六项；未跳过新核心调度。macOS 为 Go 1.27.2 darwin/arm64、Git 2.56.0，FIFO 和真实暂停／调用方退出／退出后继续保留均实际运行。Windows 为 Go 1.27.2、Git 2.56.0.windows.2，owner rename 八组合实际执行，没有相应 SKIP。Ubuntu 为 Go 1.27.2、Git 2.55.0。

Ubuntu 普通 Test 按实际脚本开启 `-race`；其中八项 CrossUser 因非 root 跳过。随后单独 `go test -c`、sudo 启动的跨账户步骤八项实际 PASS，该步骤**没有 -race**。Windows/macOS 的该步骤按设计 skipped，也不编译 Linux 专属双账户测试。没有把 root 和 race 两种证据混写。

**首轮失败保留**：[run 39264392152](https://github.com/XKHoshizora/specmetry/actions/runs/39264392152)，HEAD `941d4e227f5e83d0aaa5f417cd088319ea21d3dd`，attempt 1，7/8；macOS Go job `118100057968` 失败。该 job checkout 为 `7836e040a407f46eff4e858521caadbf584c5cff`，父为 base 与 `941d4e2`，树 `fbcd7b11a3d1c805c49c0ffce3aed01a313e5dad` 与该 HEAD 相同。

独立读取失败日志：六项生命周期调度均失败，五次状态 4479（0x117f），一次 0。旧夹具调用 x/sys v0.49.0 共享 BSD `WaitStatus.Stopped()`；本机模块源码确实排除了 SIGSTOP，而 Darwin SIGSTOP 常量为 17，故前五次被错误判为退出；另一次 Git 确已先退出。最终夹具增加输入闸门，并检查原生 stop 回复和信号值。归因来自失败日志与旧／新夹具及依赖源码核查，**本轮未独立重跑 macOS 的旧失败**。最终 macOS 的六项实际 PASS，不能回写成首轮有效通过或仅一次重跑消除了产品缺陷。

按设计不计为通过的边界：三平台真实 unsafe-old-Git 用例未配置相应版本而 SKIP；M-14 两项诊断未启用而 SKIP；Windows POSIX 不可读权限用例跳过、FIFO 不参与 Windows 构建。它们不证明旧 Git、Windows 双账户、其他文件系统或 M-14／M-15 已解决。

CI 没有本轮新增的完成 Sync 故障注入，因此两组 8/8 不反驳第二节。未变模块不因整仓 CI 通过而重新验收。

## 六、本轮实际验证及未验证边界

所有测试运行于固定对象导出的仓库外目录，未改原仓库代码／测试。Windows 为 Go 1.27.1、Git 2.56.0.windows.2、NTFS；WSL 为普通 UID 1000、Git 2.43.0、Linux `6.18.33.2-microsoft-standard-WSL2`。Linux 测试二进制由 Windows Go 交叉编译，随后在 WSL **实际执行**，不称为 Linux 原生构建。

| 本轮独立运行 | 结果及证据范围 |
|---|---|
| 未改目标 gitx：owner、Clean 三阶段、完成回收、生存期组 | Windows PASS，25.876s；WSL 加 FIFO，5 项顶层测试 PASS |
| 未改目标直接回归 | Windows gitx 232.163s、gitscan 72.354s，均 PASS；35 项顶层测试合计 |
| 未改目标 WSL 直接回归 | gitx 29 项、gitscan 5 项顶层测试，均 PASS |
| 增强操作系统存活检查的真实进程夹具 | 两平台六项均 PASS；额外确认父进程退出后 Git 仍真实存活，保留与退出后拒绝仍成立 |
| 原始 JSON 记录的七类独立破坏 | 两平台均拒绝并保留；不以默认 bool 值作为完成证明 |
| 实际 Start 读取屏障、同身份只读 owner 句柄 | 两平台正常对照 PASS；写屏障失败零启动、保留 PASS |
| 单次完成 Sync 故障，call 与公开 Clean | 两平台保留断言 FAIL；具体结果见第二节；不是物理磁盘故障实测 |
| 旧 `8e089ab` 加等价 Start/Wait 调度与新反例夹具 | WSL 实际 FAIL：owner 和公开 Clean 三组失败；两个 crash 子项失败，四个 release/cancel 子项 PASS |
| 目标仅撤除 begin/finish 的变异 | WSL 成功编译、实际运行，两条 crash 子项均检出删除，FAIL |

旧副本移植夹具时去掉引用新增字段的完成记录正对照，只加入 TestMain helper 注册及等价 Start/Wait 钩子；旧版身份、清理和消费算法未变。目标验证副本的 Sync 故障只替换指定调用的返回，其余生产状态转换未改；新增 `/proc`／进程句柄查询只观察存活，不参与生产决策。

**运行失败如实保留**：首次 Windows 沙箱运行被夹具私有 DACL 与沙箱权限冲突挡住，Git init／tmp 建立失败，未到目标断言，不计为产品缺陷或通过。随后经权限审批在相同仓库外副本运行通过。新增 Windows 存活观察辅助代码最初存在 WAIT_TIMEOUT 类型比较编译错误，修正后才执行；那次编译失败不计为漏洞检出。一次变异构建因工作目录尚未创建而未启动，准备目录后才编译运行；不计为有效实验。最终两平台 Sync 反例均为成功编译后的实际断言失败。

原作者的历史本地失败、重跑和变异仅作为实施记录核对；本报告没有把它们冒称本轮独立运行。未独立执行 macOS 本机运行、真实旧版 Git、Windows 双账户／ReFS、Linux root 跨账户、本机 Race Detector、物理断电／磁盘故障、完整 CLI/E2E 或全仓压力。macOS、Ubuntu root 及 -race 证据来自已核对的 CI，来源与本机分开。

本轮仓库外材料位于代码格式路径：

`C:/Users/hoshi/.codex/visualizations/2026/10/12/01a13137-991b-7b42-8ea2-3c24008343a0/review062`

目标导出 `target.tar` SHA-256：`016361d0f7ecd2dced821685677889d506b8782a4f6d3bda270a9e66f2dc70c4`。目录保留旧对象副本、目标副本、验证／变异源码、实际日志及 CI 身份回执；这些临时材料不移入原仓库。报告链接使用仓库相对链接，未改写历史审查原文。

## 七、NON-BLOCKER、边界与交付结论

**NON-BLOCKER**：本轮无新增独立非阻塞问题。Review 061 N-doc 的 PR 现行描述与开发指南主表已接续当前对象和 v3 协议，可关闭该文案项；不是重开旧 N1。缺少完成 Sync 故障回归属于唯一 BLOCKER 的证据缺口，不重复计项。

**批准边界**：沿用 §71 的方案 B 局部接受范围和既有决定，不新增产品决定或风险豁免。未确认消费者可能持续阻塞私有读取已准确披露，符合本轮授权；不要求新增自动恢复功能。M-14／M-15、M-9 等维护事项保持原状。

| 最终问题 | 裁定 |
|---|---|
| Review 061 B1-R1 能否关闭？ | **可以**。owner 路径、类型、原身份、完整记录及可信删除阶段的控制与反例闭合 |
| Review 061 B1-R2 能否关闭？ | **尚不能完整关闭**。原崩溃／真实 Git 存活反例已关闭，但完成 Sync 失败后保留的分支仍失效 |
| B1 能否在已批准方案 B 下整体关闭？ | **不能**，仅剩 B1-R2-S；不否定其余成立的修复 |
| 是否超出批准范围？ | 未发现功能／产品范围扩张；发现的是仍有效保留要求的一处实现违约，不是新风险接受 |
| PR #27 | **NO-GO，暂不建议合并** |
| WP2.3 本层验收依据是否充分？ | **尚未闭合**。修复上述单一失败分支后定向核验即可，无须全面重审 |

最小接手任务只有完成记录持久化失败时的保留状态及公开 Clean 回归，保留已有正常、取消、启动失败和真实崩溃对照。B2、B3、已关闭 N1 不重开，不改变扫描事务、迁移、推导、CLI、Schema、SQLite 重试／错误映射。

本轮仅新增本未提交报告。未提交、推送、合并或改基；不代替负责人宣布 WP2.3 验收，不授权或启动后续工作包，不关闭维护事项。
