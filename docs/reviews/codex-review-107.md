# Codex Review 107：PR #43 P-5 定稿计划独立审查

## 1．分层结论

**固定文档技术 NO-GO，当前不建议合并；计划暂不具备供负责人授权实现、执行的条件。**

新增 **B1：门禁内层的可用期限与准入要求互相冲突**。按 §16.8 唯一写出的单次准入规则，内层最多剩约 30 秒，却要求至少剩余 60 秒；布局及目标均不能执行。既有 P-3 的 120 秒准入余量及一分钟观察余量也不能直接套入这个内层。该问题由条文和算术即可确定，不需要先实现或运行 E2。

| 问题 | 裁定 |
|---|---|
| 文档是否可建议合并 | **暂不建议**，先修正 B1 并复审期限接线方案 |
| 是否可供负责人授权执行 | **暂不可**；不能授权一份按其准入规则必然缺少门禁有效证据的计划 |
| 固定工具链及 E2 静态方向 | 有固定源码与 x/sys 先例支持，可以继续定稿；尚无候选构建或运行证据 |
| H4 混合证据路径 | 沿用负责人已批准的路径，不要求新增过滤前 errno 动态取样 |
| 新产品待决事项／接受风险 | 本轮没有批准或新增；Q-4(a)、Q-3(a) 等既有决定保持有效 |
| G5、规范冻结、生产实现 | G5 尚未满足；本报告不冻结、不授权实现或验收 |

另有两项非阻塞精化 N1、N2，见 §5。它们不被描述为已经存在的原型假通过。

## 2．固定身份、工作区与范围

审查开始：2026-10-30 10:21:42 JST。远程身份复核：2026-10-30 10:28:50 JST。交付前再次核对身份及工作区，结果见 §10。

| 项目 | 核验值 |
|---|---|
| PR | [XKHoshizora/specmetry#43](https://github.com/XKHoshizora/specmetry/pull/43) |
| HEAD | `f7cd0a54a9a0452a68603f3ef7bb3e90b6db48e7` |
| base／远程 master | `0442f14ebfc985d2dfadd9684efeb44d821763e4` |
| 完整内容树 | `6baf55f0559dd0d7fa6887edb16c03f095a26135` |
| 本地分支 | `docs/wp2.5-p5-plan`，本地 HEAD 与固定对象相同 |
| 提交关系 | HEAD 的唯一父提交为上述 base；本轮计划是一个纯文档提交 |
| PR 状态 | open、未合并、`auto_merge=null` |

开始工作区干净，暂存区无改动。读取了当前完整 AGENTS.md；仓库内没有更下层的受版本控制 AGENTS.md。未重置、切换或清理其他任务的文件。

完整差异为五文件、341 行增加／3 行删除：

| 文件 | 实际变化与裁定 |
|---|---|
| `AGENTS.md` | 状态同步，2 行增加／2 行删除 |
| `docs/implementation-plan.md` | 状态同步，1 行增加／1 行删除 |
| `docs/review-resolution.md` | 只追加 §132，71 行；原有处理记录未改写 |
| `docs/wp2.5-check-runner.md` | 增加 4 行状态／§16 指向，规定计划不一致时以 §16 为准、但不改变规范规则 |
| `docs/wp2.5-prototype.md` | 增加 263 行，头部与完整 §16 |

`prototype/`、生产代码、测试、Schema、迁移、根模块及原型依赖、两套 workflow 均无变化。完整读取 Review 106、Q-4(a) 的决定（处理记录 §131）、§132、§16，并追踪现有 P-3、runInner、最终结果与解析辅助函数。Review 106 在本轮与 base 字节一致；当前文件 SHA-256 为 `e4c540a87f792e33afcb5a94b081c9c28f99635ffdc9acbd6a8051a234d86be3`。

## 3．既有决定与独立静态依据

### 3.1 授权边界

处理记录 §131（`docs/review-resolution.md:8379` 起）明确：Q-4 选 (a)，认可“固定工具链过滤源码＋真实错误＋错误后健康恢复”的限定路径；知悉非零残留屏蔽没有动态取样、真实 EINTR 没有穿过跳板的原生验证。错误结果不获豁免，支持范围与 Q-3(a) 不变。PR #42 合并核验后只授权另开纯文档 PR 定稿计划；P-5 执行仍须计划独立复审后另行明确授权。

独立核对 PR #42 已合并为固定 base，父提交为 `03a6f6e1e9dd7e64d58de321b5d8f7cf532b636d` 和 `3d0c2f1fd21cefecc220c052aef4e5cdbd7db964`；对应 master push CI `40916689386` 正式 success。本轮只作必要基线核验，不重新验收已关闭规格。

### 3.2 固定来源与过滤逻辑

通过官方仓库独立解析 `go1.27.3`，得到提交 **`2aae1000ea1d4b28923b96db28acf4e3b99d6aff`**，与 §16.1 相同。重新取得 §16.2 的全部 11 个文件并逐份计算 SHA-256，**11/11 与计划登记值相同**。此处确认的是 Go 1.27.3 的固定内容；没有把作者本机 Go 或未来 runner 当作已经匹配。

| 固定 Go 源码 | 本轮核对结果 |
|---|---|
| `syscall/syscall_darwin.go` | `errno(r,e)` 在 `int32(r)==-1` 时返回 e，否则返回 0；`syscall6` 经 `syscalln` 再过滤；linkname／nosplit／uintptrkeepalive 与计划一致 |
| `runtime/sys_darwin.go` | 进入／退出 syscall、libcCall 与通用调用桩的链一致 |
| `runtime/sys_darwin_arm64.s`、`runtime/sys_darwin_amd64.s` | 参数传递与目标调用后的线程 errno 读取位置一致；不是原生执行证据 |
| `runtime/os_darwin.go`、`runtime/signal_darwin.go`、`runtime/proc.go` | SA_RESTART、SIGSYS、线程绑定的引用相符；线程绑定仍不证明 errno 残留 |
| `cmd/compile/internal/noder/noder.go` | 带库名的动态导入限制与计划引用相符 |
| `cmd/link/internal/ld/macho.go` | 最低系统版本相关静态引用相符，不证明所有受支持版本已运行 |
| `syscall/zsyscall_darwin_amd64.go`、`syscall/zsyscall_darwin_arm64.go` | 现有 wait4 经 libSystem 的先例存在，不构成 waitid 候选入口已构建成功的证据 |

关键来源：[固定 syscall 过滤器](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/syscall/syscall_darwin.go#L330)、[runtime 调用链](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/runtime/sys_darwin.go#L23)、[arm64 调用桩](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/runtime/sys_darwin_arm64.s#L484)、[amd64 调用桩](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/runtime/sys_darwin_amd64.s#L510)。

依赖 `golang.org/x/sys v0.49.0` 解引用为 **`b6b8557e8e2974925aec7a136ba0e323eedc5f43`**，当前 go.mod／go.sum 的版本、校验和值与计划一致。独立读取其 [libSystem linkname 声明](https://github.com/golang/sys/blob/b6b8557e8e2974925aec7a136ba0e323eedc5f43/unix/syscall_darwin_libSystem.go#L13) 和 [arm64 kevent 跳板／地址常量](https://github.com/golang/sys/blob/b6b8557e8e2974925aec7a136ba0e323eedc5f43/unix/zsyscall_darwin_arm64.s#L96)，存在所引用的静态先例。

### 3.3 指针、返回宽度与实际绑定

计划把 C 的 32 位参数／返回值与 Go 的寄存器宽度分开，消费者按 `int32(r1)` 判断；指针在无函数体声明的实参位置直接转换为 uintptr，调用者之后 KeepAlive。固定编译器的 [`escape.go:433` 起](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/escape/escape.go#L433) 对无函数体声明施加 uintptr keepalive，[调用参数重写](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/escape/call.go#L274) 与 [unsafe 第 4 条规则](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/unsafe/unsafe.go#L113) 支持这一静态方案。没有据此断言新包已通过 vet、链接、ABI 或运行检查。

唯一低层调用者 waitidCall、readWaitid 的三个入口、ESRCH 提前回退及删除 E1 的覆盖方向与现有调用链相符。实际实现仍须由源代码检查、两架构构建和运行一起核验，不能仅检查一处新函数。

### 3.4 构建与符号检查的可用性

`go vet ./...`、`go test -c`、`go version -m` 有相应检查作用。固定工具链的 [测试包构造](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/load/test.go#L403) 会在未禁止时生成测试二进制 build info；[构建设置](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/load/pkg.go#L2501) 包括 CGO_ENABLED。因此没有“测试二进制必然不能用 version -m”的静态反例。

Go nm 可检查 Mach-O 符号及 U／T 等分类，不能单独证明导入库来源或调用 ABI。Apple 固定公开 [cctools `nm.1`](https://github.com/apple-oss-distributions/cctools/blob/e0d56624eca2a76c2ace4c21850df9e666de4ca5/man/nm.1) 说明 `-m` 可显示 undefined／external 及预期提供符号的库，支持计划组合使用两工具的方向。这里核对的是固定公开说明，不是未来实际安装的 Xcode nm 输出。

实现时应分别保存跳板 TEXT、地址常量及 `_waitid` 导入，不能仅凭名字包含 trampoline 就证明执行跳板。受检查二进制与实际诊断二进制须对应同一固定源码、工具链及设置；这属于实现时核验责任，不要求现在先构建。构建／检查工具失败与原生语义反证分开记录，方向可采纳。

## 4．BLOCKER B1：短内层不能通过自身准入，复用 P-3 的余量也不兼容

**位置**：`docs/wp2.5-prototype.md:4246`～`4255`（§16.8），直接关联 `:4199`～`4200`、处理记录 §132 的期限摘要。

### 4.1 确定的算术冲突

计划规定：

1. 单次准入要求 `D−s ≥ 60s`。
2. 外层在 s 建立 `d_g=min(s+60s,D)`。
3. 内层收到 `d_i=d_g−30s`，将其作为自己的 D。
4. 内层仍运行 TestP5 中的布局及目标，单次截止仍是 `min(开始+60s,d_i)`，没有另写内层准入例外或较短阶段预算。

设内层启动／进入首个场景的延迟为 δ≥0，则：

`D_inner − s_inner = d_i − (s+δ) ≤ 30s−δ < 60s`。

即使零启动延迟、机器无限快，按本节单次准入规则也必须“未执行”。`layout.1` 和 `w.exit7.1` 都不能提供要求的最终结果，EINTR 命中不能到达，健康对照也不能 confirmed。把外层的 D 之后 30 秒收尾借给内层，不符合已写出的传递规则。

“根层”如果意在排除门禁内的 TestP5 子测试，也不能自动解决：计划没有定义内层替代准入及阶段分配；既有复用路径还有下一项冲突。必须把这一意图写为可实现的规则，不能由实施方临场选择不同预算。

### 4.2 不能只删除一个 60 秒检查

实际既有辅助函数另有约束：

| 现有入口 | 与新短内层的冲突 |
|---|---|
| `prototype/wp25/p3_darwin_test.go:188`～`195` 的 admit | 要求距离子进程自身 t.Deadline 至少 120 秒 |
| 同文件 `:216`～`220` 的 obsEnd | 观察截止不晚于 t.Deadline 减一分钟 |
| `prototype/wp25/gate_test.go:537`～`559` 的 runInner | 当前子进程 timeout 固定 150 秒、父端等待 180 秒，并有 WaitDelay；需要为 P-5 传入截止，不能照搬 |

新计划传给内层的测试 timeout 最多约 35～36 秒。原 admit 因而拒绝；若只绕开 admit，原 obsEnd 仍可能落在场景开始之前。反过来保留 runInner 原 150／180 秒，则没有落实新计划的 60 秒外层与 D。这里指出的是**计划对现有复用入口的接线冲突**，不是要求修改所有旧 P-3 回归的预算。

### 4.3 影响与最小修正

这是有效验证的确定性不可达，不是新机制尚未实现，也不是要求 39 次运行都耗满上限后仍能全部完成。诚实判未执行可以防止假通过，但无法让该计划达到其授权目标。

最小修正为纯文档定稿：

- 分开根层原生运行与门禁内层批次的准入、观察及收尾预算；内层必须有至少一条能实际运行布局＋目标＋自身收尾的合法时间线。
- 明确 P-5 如何给 admit、obsEnd、runInner 和外层见证／普查／补救／保险清理传递同一截止；旧 P-3 的原预算可以继续保留在旧回归路径。
- 保留子进程到 `d_i+5s` 的等待时，注明外层此后最多剩约 25 秒至 d_g，所有外层步骤必须消费剩余量，不能每步再获完整预算。
- 明确超期、未执行、未命中及清理未知继续失败关闭；不以删布局、减次数、放宽门禁、自动重跑或暗增预算替代修正。

无需本轮写代码或新实验。修订后先用事件与算术推演复审，才有足够条件提交负责人授权实现。

## 5．NON-BLOCKER：代码前应精化的两项协议说明

### N1：异常返回组合与原生反证的边界应显式落表

**位置**：§16.3 `docs/wp2.5-prototype.md:4132`～`4135` 与 §16.7 `:4231`。

前者将 `ret=-1,err=0` 或 ret=2 定为 T，后者将工具有效时原生错误调用的返回／errno 不符、健康调用不是 0／0 定为 refuted。存在文字重叠：例如 ECHILD 构造的真实 E2 返回 −1／0。按前者是 T；宽读后者又是候选反证。

§16.7 已写 T 优先，因此并没有由此证明绿色假通过；也可以把 §16.3 理解为明确的 T 特例。建议代码前把它写清：哪些是候选真实返回不符、哪些是报告／调用协议失效；同一真实元组只有一个终判，保留原始值与具体原因。计划内 EINTR 的注入前真实失败仍按已有明确规则持续记 T，不受这项精化影响。不据此重新打开 Review 099 或要求扩大故障模型。

### N2：`kind` 已存在，calls 必须贯穿最终结果及解析

**位置**：§16.11 `docs/wp2.5-prototype.md:4284`；直接关联 `prototype/wp25/main_test.go:83`～`121`、`gate_test.go:578`～`588`。

现有结果已经有 `kind`，取值为 normal／counterexample／fault-injection／refusal；新计划又称“增加 kind（static／native）”。两个维度应分开命名，或明确对原字段的扩展、适用范围及兼容策略，避免悄悄覆盖实验种类。推荐保留原 kind，另加证据种类字段。

calls 应加入共享的**最终**结果类型，经收尾后的 WP25RESULT 输出、runInner 的 typed JSON 解码、inner/all 和 exact 检查保存及消费。仅在 WP25P、Observed 文本或未进入共享类型的额外 JSON 字段中打印，并不能让外层检查它。必需目标的 calls 缺失不能当作合法零次；布局允许 0，目标故障 1、健康 2、错误后恢复 3，应分别核对。

尚无实现，不能断言作者已经采用丢字段方案。此项是代码前协议精化，不是当前共享门禁存在新假通过的证据。P5.c 要求两次 W 读取，现有 p3C 只有一次，实施时也须明确增加对应读取和证据，不能把“复用”当成已自动满足次数。

## 6．其余计划的限定性评价

| 审查项 | 结论与边界 |
|---|---|
| 39 次原生＋3 次布局 | 7 个 W 场景×3=21；3 个 C×3=9；ESRCH×3=3；两种错误恢复×3=6，合计 39；布局另列 3，不混成 42 次原生 |
| 布局 | 现有 p3ProbeKeys 对应 55 项；复用其严格解析和错误终判的方向成立，布局不能证明内核宏分支 |
| 受控对象与顺序 | 直接子进程、启动身份、未回收僵尸、两次读取、先确认组结束再 reap、独立普查的计划可采纳；只有实现后才能证明逐项执行 |
| ESRCH 回退 | exitObserver 提前路径、waitidNoWait 和 P-3 readW 全部列入入口替换；不能仅改新 P5 主路径 |
| ECHILD／EINVAL | 自身 PID 不是自身子进程、无退出选项的错误构造及错误后读取受控僵尸的方向符合已审固定 XNU 依据；本轮未做原生调用 |
| N 与 M | 计划独立使用 p3NormW／p3Pred，不以被测解码器生成期望；旧 N2 的改写前原始值限制诚实保留 |
| EINTR | 故障分支只调用一次，健康分支两次并比较 W5；它证明消费者拒绝接缝错误，不证明真实 EINTR 穿过 libSystem／跳板 |
| 注入前失败 | 真实结果先保存、非成功不改写并持续记 T、无命中不能通过；预期理由不能豁免其他错误，规则方向成立 |
| exact 门禁 | 必须实际有布局及目标最终结果；非目标 confirmed、命中一次、目标唯一理由、退出码及内外清理同时成立。期限修正不能取消这些要求 |
| 旧回归 | 7 个 W／C 门禁、2 个健康对照及完整旧 P-3 另列，经 E2 重跑后只能作为 E2 回归，不能继续冒称新代码点是 E1 证据 |
| 全局期限 | D 对 TestP5 静态入口检查、布局、原生、新门禁及对照的范围已列明；D 后 30 秒有所有者、不越过套件截止；问题在内层接线，见 B1 |
| workflow 外静态步骤 | 工具链／vet／build／settings／symbols 不属于 3 分钟 TestP5，而属整个 20 分钟作业；旧回归与 TestP5 共享 15 分钟 go test 截止，不能每项另获 15 分钟 |
| 收尾 | 被测状态先保存、补救不恢复结论、未知归属零动作；应继续以收尾后的最终结果裁定，不把前置 WP25P 快照当作最终清理证明 |
| 停止条件 | 反证、工具失效、未执行、无命中或无法有效取证时保留结果并交负责人；不自动重跑、换入口、增预算或降标准，成立 |

以上是计划与既有源码的静态评价，不是场景、门禁或工具已运行通过。

## 7．常规 CI 独立核验

[run 40919163851](https://github.com/XKHoshizora/specmetry/actions/runs/40919163851)：pull_request，attempt 1，正式 **completed/success，8/8**；run 的 head_sha 为固定 HEAD。

实际合并快照 **`b170da39936e0ad55a18603735fffedd78308808`**；父提交顺序为固定 base、固定 HEAD；树 **`6baf55f0559dd0d7fa6887edb16c03f095a26135`**，与 HEAD 相同。独立读取了八份 job 日志，各自 checkout 均为该完整合并快照：

| Job | ID | 正式结果 |
|---|---|---|
| Go lint | 123607388506 | completed/success |
| Go tests (ubuntu-latest) | 123607388913 | completed/success |
| Web | 123607389001 | completed/success |
| Go tests (macos-latest) | 123607389122 | completed/success |
| Go tests (windows-latest) | 123607389185 | completed/success |
| Build with dashboard (macos-latest) | 123607598556 | completed/success |
| Build with dashboard (ubuntu-latest) | 123607598641 | completed/success |
| Build with dashboard (windows-latest) | 123607598680 | completed/success |

Ubuntu 普通 Go 测试走 `-race`；单独的 root 跨账户步骤编译测试二进制后执行 CrossUser，不是普通 race 步骤的替代或同一证据。八个跨账户用例在普通步骤跳过、在单独 root 步骤通过。三平台普通步骤的 SKIP 分别为 15／7／10，涉及缺少特定真实 Git、M-14 诊断、Windows 权限用例等；SKIP 不计通过，无 job 跳过。

常规 CI 只检查已提交的根仓库内容，**没有运行独立 prototype/wp25 模块，更没有运行尚不存在的 E2／P-5**。8/8 不消除 B1，也不证明符号、ABI、H4 或新的期限接线成立。本轮没有触发／重跑任何 CI 或诊断。

## 8．实际验证、失败与未验证边界

- 独立执行：完整五文件 diff／diff-check、身份与树／父提交核对、Review 106 与决定读取、固定官方源码及哈希核对、现有辅助调用链静态追踪、期限事件／算术推演、八 job 完整日志核验。
- `git diff --check base HEAD` 无输出且成功。未执行本地 Go 测试、构建、vet、布局探针或新机制实验；没有以作者旧观察替代本轮运行。
- 本轮在 Windows／PowerShell 下审查，没有原生 macOS 运行，未申请资源或安装工具。实际 runner 的 Go、Xcode、SDK、系统／内核、架构及 Rosetta 情况仍须未来授权运行采集。
- 官方旧 nm 网页读取未获得可用内容，改为读取上述固定 cctools 官方源码；不把未能读取的网页当作已验证。查询／脚本准备中的失败不计产品或机制检出。
- 原型历史的失败、取消、未知归属／清理、未检出变异及 Review 101 N1／N2 均按原范围保留；本轮没有重跑、改写或据新 CI 回填历史。
- 静态过滤可证明固定实现对成功返回如何过滤，不能证明未来 E2 实际绑定、过滤前非零 errno、真实 EINTR 或任意版本 libSystem 行为。后两项的限定证据路径已经批准，不在本轮重新要求动态取样。

## 9．最小后续、可授权范围及交接提示词

**当前必要下一步只有计划修订与定向复审，不是先写代码试运行。**

1. 修订 B1：写出可准入的根层／内层角色预算、统一截止传递和完整收尾时间线；覆盖既有 admit／obsEnd／runInner 及外层清理接点。
2. 代码前明确 N1 的异常返回分类和 N2 的共享结果字段／调用次数消费；保持所有错误累积、exact、归属、清理及旧回归要求。
3. 固定修订后的计划提交，进行定向独立复审。通过后再由负责人明确授权 test-only E2／P-5；本报告不发出执行授权。

若上述条件满足，可提交负责人选择的授权范围仍是：固定 Go 1.27.3、CGO_ENABLED=0、固定 x/sys、两种原生 macOS 架构同代码点的 test-only E2 包／跳板／接线、列明的静态检查、39 次原生与 3 次布局、新门禁／对照和旧回归。一次固定代码点诊断，保持既有权限、平台和作业预算；异常停下保留证据，不自行改入口或重跑。H4 原始残留、真实 EINTR、旧系统差额等依原批准边界披露。

冻结前仍须：可执行的固定计划、明确执行授权、实际构建／原生／门禁有效结果、独立审查及 G5 其他差额的相应裁定。生产接入、生产事务与完整 Runner 验收留在授权后的生产工作，不因尚未实现而新增成本轮阻塞。

交接提示词：

> 请仅修订 PR #43 的 P-5 计划，处理 Review 107 B1：区分根层与门禁内层的准入及阶段预算，消除“内层最多 30 秒却要求 60 秒／复用 120 秒余量”的冲突；明确 admit、obsEnd、runInner、外层核验和保险清理如何消费同一截止，并给出布局＋目标确实可执行的事件时间线。不要增预算、减次数或放宽 exact／失败关闭。代码前一并精化 N1 的真实异常返回分类，以及 N2 的 kind／calls 最终结果与内外层解析。保持 Q-4(a) 已批准的限定 H4 路径、所有历史记录及支持承诺。只做文档修订，固定提交后交定向独立复审；不构建或执行 E2／P-5，不触发诊断，不冻结或授权生产。

## 10．结束核验与暂停

2026-10-30 10:30:48 JST 的交付核验：远程 PR HEAD、base、master 与 §2 完全相同，PR open／未合并／无 Auto-merge。本地 HEAD／树相同，跟踪文件与暂存区无差异，唯一新增项为 `?? docs/reviews/codex-review-107.md`。常规 CI 在 10:28:50 JST 的末次查询仍为同一 attempt 1 success。

交付仅新增本报告，保持未提交。没有修改被审内容、提交、推送、合并、改基、构建、实验或触发 workflow。Q-4(a) 不重开；本轮不批准新风险或保证调整。M-1、M-14～M-17、N-087-1 保持原范围，WP2.5／M2 未宣布验收。完成后暂停，等待负责人决定。
