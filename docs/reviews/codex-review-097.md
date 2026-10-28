# Codex Review 097 — PR #38，Review 096 B1-R 定向复审

## 1．分层结论

**技术 GO，建议合并本次固定对象的原型与文档修订。** Review 096 **B1-R 可以关闭**；结合 Review 096 对原点名路径的关闭证据，**Review 095 B1 在本次受审原型范围内整体收口**。没有发现新的 BLOCKER、confirmed 假通过或直接回归。

这不是规范冻结或生产验收：共享退出错误现在有持续保存及返回两个消费者；证据支持的是受控原型的错误传播，不是任意宿主、原生不可终止故障或生产事务。最终常规 CI 正式 8/8 成功，修复代码诊断三平台正式成功，当前技术及 CI 合并检查均满足本次原型 PR 的审查要求。

| 事项 | 本轮裁定 |
|---|---|
| B1-R | 关闭：区间自身失败内部 flag，清除错误同时返回；调用者丢弃结果、回调 Goexit 均不能漏判 |
| Review 095 B1 | 结合已有审查证据整体关闭，限定为已审原型及其受控前提 |
| 共享调用链、测试工具变化 | 成立；13 处直接调用已核对；补充自述与外层核验分开 |
| Review 095 N1/N2 | 维持 Review 096 的关闭结论，没有直接回退 |
| N-096-1 | 措辞修正成立，可以关闭这项文档意见；PID 2668 的历史未知没有关闭 |
| N-096-2 | 仍未修，维持非阻塞；不得把相关分类边界宣传为完整失败分类保证 |
| 合并建议 | 对固定 HEAD 建议合并；本报告不执行或授权合并 |
| 冻结 | G3/G6/G7 仍是局部支持；G5 未满足、P-3 未执行；不能宣布规范冻结或生产开工 |

**BLOCKER：无新增。NON-BLOCKER：保留 N-096-2。PRODUCT DECISION：无新增。ACCEPTED RISK：无新增。** Q-1/Q-2(a)、C-4、D21 已批准，不重新列为待选。

## 2．身份、范围与历史

开始核验：**2026-10-28 10:16:48 UTC（19:16:48 JST）**。收尾身份核验：**10:28:11 UTC（19:28:11 JST）**。

| 对象 | 独立核验 |
|---|---|
| PR | [XKHoshizora/specmetry#38](https://github.com/XKHoshizora/specmetry/pull/38)，开始/收尾均 open、未合并，auto_merge=null |
| base/master | `2c5cf23a8c5a025be49b895f353af77b70a6248f` |
| HEAD | `800b5e82d449478d60315ede19e532f368d883d2` |
| HEAD 树 | `256dca23819757bb1111944cd37855482908b008` |
| 上轮对象 | `b80982c2bbf4f7db1c6af179ad6cd58139b1cb8b` |
| 固定代码点 | `48ff0cde615f36b31e4046d19f3dd83e6f6556e3` |
| 最终合并快照 | `c4ee9bb195b135d3521e4f03008f09f0be1addcb` |

远程与本地身份一致，没有更换对象。开始及写报告前工作区/暂存区干净，Review 097 不存在；保留其他会话内容，最终仅新增本未提交报告。

已读取实际适用的根 AGENTS.md、Review 095/096 全文、本轮完整增量、原型文档 §14.15～§14.17、§116 及必要调用链。相对上轮 **12 文件，+771/-59**，为 6 份文档及 6 份原型测试文件；未重审全部 T1～T6 或其他已验收模块。

提交顺序：

1. `73c8eb3a6276d610d2068173feab70f8ec09414a`：原样归档 Review 096。
2. `b641e174ddec2f34dd209ddf760995b159ba5a90`：回归先行，5 个原型文件。增加接缝、8 项故障与3个对照、label、alsoBecause、注入位置和内层读数；新增屏障后的属性读取。旧的清除/基线/设置错误消费没有提前修正。
3. `be25df5db6a3a18e29161665044682bd3b19c5f6`：仅 gate_test 的补充清理字段和赋值；被测错误处理未变。
4. `48ff0cde615f36b31e4046d19f3dd83e6f6556e3`：共享区间修复及调用者签名同步。
5. `b34e914d2397dd83bfbd6d4761aa8f2ed9fbcc93`：文档、状态。
6. `800b5e82d449478d60315ede19e532f368d883d2`：原型文档的引用与措辞收尾。

代码点到 HEAD **仅五份文档**：AGENTS、implementation-plan、review-resolution、wp2.5-check-runner、wp2.5-prototype；`prototype/`、`.github/` 无差异。相对上轮，生产代码、根依赖、Schema、迁移、根 ci.yml、诊断 workflow 均未变。规范正文的语义未改，仅状态和证据指针同步。

Review 096 归档与 HEAD 的 blob 均为 `ee0192184a8eaa1764a2ae84282c6bfcecfd5d35`；文件 SHA-256 为 `1d53f0233b95b192c80bf4a84229983e5478cfecc4494384f29aa8f9ba813d93`，与 §116 一致，末尾字节仍为 **0A 0A**，保留原文最后空行。`git diff --check` 的唯一该类提示就是归档的 new blank line at EOF，不是应规范化的本轮缺陷。没有宣称重新取得归档前临时文件。

review-resolution 原有 **7035 行前缀逐行相同**，本轮只追加 §116 共115行。旧审查身份、§115 与旧结果保留；原型文档在新说明中收窄旧概括和证据范围，没有把旧绿色改写为曾经完整检验清除结果。

## 3．B1-R：共享退出链已闭合

以下行号指固定 HEAD，与代码点的原型内容相同。

### 3.1 错误保存与返回

`prototype/wp25/exclusive_linux_test.go:112～163`：

| 路径 | 当前处理 | 消费与结论 |
|---|---|---|
| 基线读取失败 | 内部 `s.flag`；返回含 errBaseline、errBaselineUnread 的错误 | 无论调用者是否误把它归为拒绝，scen.done 的 invalid 优先；不能再 confirmed |
| 可靠非空基线 | 只返回 errBaseline，不 flag，不 SET 1、不清除 | 保留合法的计划内拒绝 |
| SET 1 失败 | 内部 flag 并返回；不注册清除 defer | 不假定区间已建立；没有补做清除或新重试 |
| 回调返回 nil | 显式 return fn() 将结果交给命名 err，再执行 defer | 清除失败可加入结果，不被 nil 覆盖 |
| 回调返回 error | 命名 err 保存回调结果，清除失败用 errors.Join 合并 | 两个错误均可保留，不替换回调原因 |
| 回调 invalidf/Goexit | 先由 invalidf 保存回调失败，栈展开仍执行清除与释放 defer | 无人接收返回值也有内部 flag；两个失败均进入最终 invalid |
| SET 0 失败 | 内部 flag，同时 Join 进返回错误，带 errSubreaperClear | 调用者检查、计数或丢弃返回值均不能漏判；后续属性为0/进程清理成功不能豁免 |

关键是 **159行 flag 与160行 Join 并存**，不是新增末次属性读取来替代错误保存。P-1 屏障的 `subreaper(s)` 是另一个读数；即使它为0，先前 flag 仍保留。

当前13处回调都不正常返回非 nil；因此“回调 error 与清除 error 的 Join”按命名返回及 Go defer 流程核对，没有冒称有原生动态用例。Goexit 也不产生一个可供 Join 的回调返回 error：回调失败与清除失败通过场景的两条 invalid 原因共同保存；本轮组合回归实际覆盖了它。

### 3.2 清除、解锁与租约次序

先注册 **122～128行租约释放 defer**，成功 SET 1 后再注册 **153～162行清除 defer**，LIFO 顺序为：

清除 → 保存/合并清除错误 → 取得闸门锁、撤销 excl、登记区间、Broadcast、解锁。

g.mu 在执行回调、清除、flag 时没有持有；flag 只短暂取得 s.mu。已核对调用点没有持 s.mu 反向等待该区间的新增锁顺序。Goexit 时 s.invalidf 的 s.mu 已释放，两个 defer 仍执行。未发现新增死锁、错误覆盖或把 flag 变成无期限等待。

原有 cond.Wait 的等待/共享持有者生命周期未被改成统一有界，也没有增加超时政策；本轮仅确认新增错误保存没有引入新的等待依赖。不能从并发原型成功推出生产拓扑或系统调用硬实时保证。

### 3.3 13处直接调用均传入实际场景

| 调用点（固定 HEAD） | 对返回值的用法 | 内部保存为何必要 |
|---|---|---|
| p1_linux_test.go:19，p1Inner 经 p1Exclusive | 检查后 invalidf | 正常返回时保留返回原因；Goexit 时内部 flag 仍可消费 |
| p1_linux_test.go:35，p1Barrier | 分类错误/entered | 原反例：屏障 clear 失败不能再按 entered=true 放行 |
| exclusive_linux_test.go:537，TestExclusiveGate | 计数成功 | 即使只少一次 good，也应 invalid，而非 refuted |
| :593，TestExclusiveConcurrentChecks | 计数/拒绝 | 与并发调用者共用场景，flag 持续保存 |
| :640，TestExclusiveBypass | 丢弃 | 不依赖返回值 |
| :673，TestOtherReaper | 丢弃 | 不依赖返回值 |
| :700，TestExclusiveBaselineLive | 期望拒绝 | unknown 与可靠非空仍区分 |
| :724，TestExclusiveBaselineZombie | 期望拒绝 | 同上，包含僵尸基线 |
| :739，TestAdoptionChain | 检查后 invalidf | 返回错误与内部 flag 都有消费者 |
| :765，TestSweepIdentityMismatch | 丢弃 | 不依赖返回值 |
| :792，TestSweepPidfdUnsupported | 丢弃 | 不依赖返回值 |
| :828，TestF1Resources 第一次 | 丢弃 | F1 既有预测不能豁免清除失败 |
| :841，TestF1Resources 第二次 | 期望拒绝 | 基线错误不可冒充已观察到残留 |

五处丢弃返回值的路径均由共享层 flag 覆盖。macOS p1Exclusive 的签名同步仅多一个未使用的场景参数，仍直接执行回调；Windows 无此 Linux 区间。没有把 Linux 修复扩大成跨平台原生机制保证。

## 4．回归先行、修复及有效性

已读取回归先行与修复诊断中相关完整日志。旧实现没有预先修复判定，日志保留旧错误结论；新实现不是一律 invalid，健康与计划拒绝仍 confirmed。

| 新增故障用例 | 旧实现内层 / exit | 修复后 / exit | 命中 |
|---|---|---|---:|
| p1.barrier-clear-fail-once | confirmed / 0 | invalid / 1 | 1 |
| p1.inner-clear-fail-once | confirmed / 0 | invalid / 1 | 1 |
| p1.observer-and-clear-fail-once | invalid / 1，仅观察者原因 | invalid / 1，观察者与清除两项原因 | 2 |
| t2.clear-fail-once-counted | confirmed / 0 | invalid / 1 | 1 |
| t2.clear-fail-once-checked | confirmed / 0 | invalid / 1 | 1 |
| t2.clear-fail-once-dropped | confirmed / 0 | invalid / 1 | 1 |
| t2.baseline-unread-once | confirmed / 0 | invalid / 1 | 1 |
| t2.set1-fail-once | refuted / 1 | invalid / 1 | 1 |

原反例的日志注入位置是 **exclusive.clear(p1.barrier; the real clear returned <nil>)**，明确命中内层区间之后的屏障退出；并非仅让前面的内层失败。新理由同时包含共享层清除失败和屏障自身的 interval 错误。屏障之后读到0，运行仍 invalid，证明新属性读取没有替代持续保存。

组合用例两个位置依次为 **p1Inner.observer**、**exclusive.clear(p1.inner; the real clear returned <nil>)**。真实观察者创建后关闭，再用既有接缝令回调 invalidf；clear defer 随 Goexit 执行。新日志两项原因俱在；旧日志只有观察者原因，所以旧用例虽然已 invalid，也不能靠该后果碰巧通过。

计数、检查、丢弃三类清除回归分别真正运行了相应 T2 场景；SET 1 失败的命中记录明确写 **the real call was not made**。可靠非空基线、healthy、alive/exited、正常 T2 gate/bypass/adoption 等对照继续 confirmed/0。新的 t2.baseline-unread 不能再与正常的 control.refusal 混同。

### 证据的严格范围

- 清除接缝**先执行真实 SET 0，只替换返回值**。日志中的真实返回全部为 <nil>，读数为0。它验证“观察到失败必须保存”，没有验证真实清除失败导致属性保留1。
- 真实 SET 0 被 EPERM 拒绝、GET仍为1的证据，仍来自既有 P-2 prctl.set0 实验；实验进程退出不是内层清除成功。本轮没有新增这种原生失败实验。
- 修复日志中八项外层均 Pass=true、Final=ended、remedy=not-needed、insurance=nothing left，且没有外层工具错误。新字段另列内层 verified 清理；两层事实没有互相替代。
- 新回归是门禁正确拒绝的 PASS，不是新增机制 confirmed；旧内层六个错误 confirmed 也不等于旧外层门禁通过。
- 作者 WSL R1～R4 只作作者变异证据，本轮未独立撤回。R2 主要依赖 alsoBecause 的证明范围如实保留；编译失败不计检出。

## 5．测试工具变化

### injectOnceAt

`witness_test.go:308～323`：旧 injectOnce 委托给 injectOnceAt(name,name,site,pid)，保持原来的每故障一次语义。多接缝版本先要求当前注入名称匹配，再用同一互斥锁保护 onceDone[key]。

当前组合使用不同的 **…/observer** 与 **…/clear** key，允许同一故障在两个位置各一次；其他既有调用仍按故障名去重。key 与 label 在本轮调用拓扑中没有冲突；label 在闸门构造后、使用前赋值，不是在并发区间中变更。没有把“每 key 一次”说成任意未来 key 使用均不会冲突。

### because / alsoBecause

`gate_test.go:679～685` 在已有内层 verdict/退出码匹配之后检查两个理由子串；缺任一项只会令 verdictOK=false，不会把 false 改回 true。它们没有代替故障命中、负载、见证、归属、清理和退出码。

`judge:601～619` 仍优先处理外层工具失败，要求未允许问题为空、补救成功或不需要、最终 ended。保险清理与被测路径矛盾仍能在868～870行重新否决；找到预期理由不豁免并发工具错误、补救失败或未知结束。未发现本次扩展改变组合判定的失败传播。

### sites、inner_observed、inner_cleanup

- sites 来自该次真实内层进程 stdout 的 WP25INJECT JSON；inner_observed/inner_cleanup 来自指定 scenario ID 的 WP25RESULT。前者记录接缝位置，后二者是内层报告的读数与清理自述。
- RESULT JSON 解析或 Scanner 失败持续写入 r.err；缺目标结果也不能匹配通过。新增文本字段则是**可选诊断字段**，不是外层结束/归属证明，也不是写者认证。
- sites JSON 解析失败时，当前解析器不追加该 site；marker 仍可能计入 hits。Observed/Cleanup 缺失会保留空值。它们没有被用来放宽 judge 或设定 Final=ended，故不能把空值/畸形 site 当作“已核验位置/内层已清理”的正面证据。
- 本次相关实际日志具有完整 site 和清理文本，审查逐条核对；没有凭字段数量相同或文案含 verified 就判定外层清理成立。若以后把这些补充字段升级为硬性证据门禁，应同步严格校验，不能沿用当前展示字段的容错行为。

因此，工具变化在当前用途下成立；没有发现新增绿色假通过，也没有要求原型建立任意恶意输出的认证协议。

## 6．独立执行、失败与未验证边界

在仓库外以固定 Git 对象导出、未修改代码：

`C:/Users/hoshi/.codex/visualizations/2026/10/12/01a13137-991b-7b42-8ea2-3c24008343a0/review097/target/`

预先限定两组既有 Windows 测试各一次，每组 **120 s**，GOTOOLCHAIN=local、GOPROXY=off、独立 GOCACHE；不安装工具、不运行项目配置，不增加实验类型。Go **1.27.1，windows/amd64**。

| 独立运行 | 结果 |
|---|---|
| 14项目标门禁，同时包含根层 P1.healthy.1 | 首次 **PASS，25.667 s**；14/14 门禁，1根层场景 confirmed |
| TestP1、Q12Classify、GateJudge、WitnessJudge、StartPoints、PhaseClock | 首次 **PASS，10.761 s**；P1六个机制观察 confirmed；直接测试另计 |

第一组含新增 control.p1-exited、原有 P1/P4 正反对照和8项旧 B1 观察故障。八项故障均有一次命中、内层 invalid/1，相关 site/inner_cleanup 已输出；全部外层 Final=ended、无工具错误。根层场景从一开始就包含，未重复 Review 096 的“只选门禁导致 TestMain 无场景而拒绝”的选择错误。

本轮尝试读取 WSL 发行版列表，立即得到 **Wsl/E_ACCESSDENIED**；这是环境准备失败，未启动 Linux 实验，不是产品失败，也没有提权、安装或绕过拒绝。因此：

- **Linux 八项新增故障的旧/新动态对照来自独立读取的 CI 日志，不是本机复现。**
- 本机没有执行 Linux SET 0 故障、Goexit组合、作者 WSL 变异或 macOS 动态运行；对应结论由源码流程及 CI 支持。
- 没有本轮本机测试失败、压力补跑或远程重跑。历史回归先行失败与维护测试失败另见下节。
- 每个被运行门禁的外层状态均 ended，独立场景报告清理完成；结束时 Get-Process 未发现 wp25.test。没有终止未知 PID 或遗留自己创建的夹具。

## 7．CI：事件、对象与计数分别核验

独立读取四个 run 的正式状态、job 元数据及 **22份完整解码 job 日志**。Results 步骤重复打印按 ID 去重；机制场景、门禁、组合及直接测试没有混计。没有触发或重跑 Actions。

### 原型诊断

| run | 对象 / 事件 / attempt | 正式结论 | Ubuntu：场景/门禁/组合 | Windows | macOS |
|---|---|---|---|---|---|
| [40656037766](https://github.com/XKHoshizora/specmetry/actions/runs/40656037766) | 完整 b641e17 / push / 1 | failure | 53/53、95/103、13/13 | 53/53、92/92、13/13 | 33/33、77/77、13/13 |
| [40658194158](https://github.com/XKHoshizora/specmetry/actions/runs/40658194158) | 完整 48ff0cd / push / 1 | success | 53/53、103/103、13/13 | 53/53、92/92、13/13 | 33/33、77/77、13/13 |

回归先行 job：Windows **122707185566**、macOS **122707185970**、Ubuntu **122707186083**；修复 job：Windows **122714309251**、Ubuntu **122714309646**、macOS **122714309690**。每份日志实际 checkout 与相应完整 SHA 一致，META 同步。Ubuntu 回归先行失败列表恰为 §4 的八项，没有用其他失败凑数；Windows/macOS 只有新增的通用对照，不把 Linux 专属项算作它们通过。

修复 META 为 Go1.27.3；Linux/amd64 6.17.0-1022-azure，Windows/amd64 10.0.26100，macOS 26.6.2/Darwin25.6.0 arm64。Ubuntu 通过既有 RUNNER_OS=Linux 分支实际传入 **-race**。普通原型诊断没有 root 步骤。

场景 ID 并集仍为 **86**（原有53，加P1六个/P2九个/P4十八个），不是跨平台的139次观察都独立，也不把新增8项门禁加成机制场景。三个内层专用测试 TestInnerForeignPID、TestInnerLeak、TestInnerStopError 在根层设计 SKIP，内层调用另计，不把 SKIP 写成通过。Linux的11个 T2场景及P1六次、P2九次实际执行；并发检查、归属拒绝/零动作、F1资源、固定时钟等相关对照未见回退。

### 最终常规 CI

[40661900394](https://github.com/XKHoshizora/specmetry/actions/runs/40661900394)：pull_request，attempt1，**completed/success，8/8**，head_sha 为固定 HEAD。

| Job | ID | 正式状态 |
|---|---:|---|
| Go tests Ubuntu | 122726717365 | completed/success |
| Go lint | 122726717645 | completed/success |
| Web | 122726717674 | completed/success |
| Go tests macOS | 122726717718 | completed/success |
| Go tests Windows | 122726717926 | completed/success |
| Build Ubuntu | 122726856859 | completed/success |
| Build macOS | 122726856959 | completed/success |
| Build Windows | 122726857029 | completed/success |

八个作业实际 checkout 均为完整 `c4ee9bb195b135d3521e4f03008f09f0be1addcb`。Git commit API 核对父提交依次为固定 base、HEAD，树为 `256dca23819757bb1111944cd37855482908b008`，与目标树一致。

Ubuntu 普通 Test 带 -race；root Cross-user path tests 是另一个步骤，独立编译后 sudo 运行，二者均成功，不能混称同一个 race/root 运行。普通步骤的 root 专属跳过、其他平台权限/旧 Git 用例的跳过不计通过。根模块常规 CI **不运行嵌套原型模块**。

### 旧常规 CI 的 M-1 观察保留

[40656041186](https://github.com/XKHoshizora/specmetry/actions/runs/40656041186)：pull_request、b641e17、attempt1，**completed/failure，7/8**。八份日志实际 checkout 为 `c9ea84f02c45e740e565c99e9e323d257cb09534`；父提交为固定 base、b641e17，树 `d267cdc68932aa413cef6659f1aa582c4c29043f` 与该提交树相同。

失败 job **122707197257**（Windows）：`TestCheckWhileTheDatabaseIsCreated`，`openrace_test.go:155`，**round11、check9，Access is denied**。其余7项正式成功，没有被省略或当作取消。

与 Review035 N2/M-1 的**同一测试、相似错误签名**相符；本轮未修改根模块。但这些事实不能证明底层同根因或与增量绝对无关。最终运行该测试通过，也不关闭 M-1。本次没有重跑该旧 run，不把旧失败合并归入 M-14/M-15 或原型门禁。

## 8．非阻塞项、历史与冻结门槛

**N-096-1 修正成立。** 原型 §14.12、§14.17 及处理记录 §116 保留 PID2668 的身份/PPID/名称启发式线索，明确归属、EINVAL 根因及当次完整清理未知，Final=unknown、Pass=false。§115 历史文字不回改，由后续更正限定。没有用保险“nothing left”证明该未知对象已清理，没有用新绿色回填旧运行。

**N-096-2 继续非阻塞。** p2 源码未改：
- prctl.set0 的过滤器不支持时，先等PID的握手可能使原本 unsupported 变成 invalid；
- self procInfo unknown 仍可能变成 /proc=false，再判 refuted。

这些当前路径不构成 confirmed 假通过，本轮没有发现新的相反证据。§14.11前置说明及§14.17已明确排除过强分类概括。G6证据只能按实际安装过滤器、完整正常/拒绝序列的环境适用；冻结前应修正这两处并补最小不支持/自读失败对照，或将尚未统一的原型分类限制继续写清。它不是要求重选产品支持范围，也不是本轮必须实现的生产修复。

| 门槛 | 本轮支持 | 冻结前/生产阶段剩余边界 |
|---|---|---|
| G3 | 共享区间错误传播恢复可信；P1真实Cmd/句柄移交、F1返回、释放后持续禁用得到局部支持 | Windows alive 额外持有Job句柄的受控构造仍改变最后句柄关闭前提；不证明真实不可终止故障。宿主生产拓扑及资源接线仍须兑现 |
| G6 | P2九组过滤器/能力/登记/回收、SET0拒绝与属性保留证据继续适用，不经本次 exclusive | 受测内核/架构、seccomp前提；N-096-2分类边界；不把实验进程退出当作清除成功，生产探测接线未验证 |
| G7 | P4九类各两次、恢复知识状态、退出码、期限及移交仍有局部支持 | 真实API结果、改变前提和接缝分开；不推出任意Windows宿主/真实不可终止情形或生产事务 |
| G5 | 未满足 | P3未执行，不自动缩小macOS支持 |
| G1/G12及其余 | 既有决定和模型规则不变 | 模型零访问/记录不是生产准入、数据库、seq、事务、CLI或完整持久化组合证据 |

旧 `85ec2ca` 及更早 Linux P1/T2 清除链证据降为历史观察，范围正确；不经 exclusive 的 P2/P4、Windows/macOS 不因这个缺陷被一概抹掉。新的替代代码点是48ff0cd。所有旧失败、取消和作者变异的强度保留，不把原型绿色推导成全部冻结门槛满足。

本轮无需再修 B1-R，也没有新的产品选择或保证豁免需要批准。后续最小工作是按原有权限完成合并核验；冻结前仍处理已登记规格/必要验证（含G5及G6边界），获准实施后再完成生产Runner、CLI、数据库与seq/事务、配置固定、取消、输出及完整验收接线。本报告不自动授权这些动作。

M-1、M-14～M-17、N-087-1 保持未解决/原状态；Q-1/Q-2(a)、C-4、D21不重选，方案B与Log完整性、归属保护没有新增豁免。**技术GO及建议合并不等于规范冻结、产品批准、生产授权或WP2.5/M2验收。**

