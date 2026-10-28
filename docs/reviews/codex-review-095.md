# Codex Review 095：PR #38，Q-1／Q-2 同步与 P-1／P-2／P-4 隔离原型

## 1．分层结论

**PR #38 当前技术 NO-GO，不建议合并固定对象。** 新增原型提供了真实、可定位的局部证据，Q-1／Q-2(a) 的实质规则也忠实于负责人决定；但新增观察入口没有完整接入实验的失败关闭规则。P-4 的退出码读取错误可以成为合法的 NULL，外层的一次失败也可能被后续清理成功掩盖，最终仍为 confirmed／0；P-2 和 P-1 的新调用者有同类遗漏。具体源码反例及最小修正见 §4。

这不是因为生产 Runner 尚未实现、Intel macOS 没有运行，或因为已批准的产品决定还要重新选择。**阻塞的是本 PR 自己声明的实验有效性契约。** 本轮没有发现需要重开 Q-1／Q-2 的产品取舍，没有新增接受风险。

| 层次 | 裁定 |
|---|---|
| 产品决定落实 | 实质成立；Q-1／Q-2(a) 已批准，不重新列为待选。摘要同步的遗漏见 N2 |
| 当前原型技术结论／合并建议 | **NO-GO**；先修新增观察链及其终判，再定向复审 |
| 正式 CI | 原型诊断三作业 success；常规 CI attempt 1、8/8 success。不是失败的 CI，但绿色不能否定源码可证明的门禁缺口 |
| 冻结门槛 | G1／G12 的决定方向可以确认；G3／G6／G7 得到局部支持，不能整体标为满足；G5 仍未满足 |
| BLOCKER | B1：新增观察错误可漏过终判，见 §4 |
| NON-BLOCKER | N1：P-1 的“释放后”断言实际在释放前取样；N2：部分状态摘要仍是旧口径 |
| PRODUCT DECISION／ACCEPTED RISK | 本轮无新增产品待决或风险接受；后续若改变支持范围或保证，仍须单独决定 |

## 2．身份、范围与历史保留

| 项目 | 固定对象 |
|---|---|
| PR | [XKHoshizora/specmetry#38](https://github.com/XKHoshizora/specmetry/pull/38) |
| base | `2c5cf23a8c5a025be49b895f353af77b70a6248f` |
| HEAD | `1be48c328baea7b7c1664849dfedcc9f9d6d5107` |
| HEAD 树 | `60473bc3391be35962f20eb20a1edb0a1439b44c` |
| 事先固定计划 | `dcdb2b26067d8325cb4ef6ac28840f44e4da7704` |
| 原型代码点 | `d45a319dd3f61b5e98ff7bb95249c082fb2541dd` |
| 原型诊断 | [40439516254](https://github.com/XKHoshizora/specmetry/actions/runs/40439516254)，push，attempt 1 |
| 常规 CI | [40440832011](https://github.com/XKHoshizora/specmetry/actions/runs/40440832011)，pull_request，attempt 1 |
| 常规 CI checkout | `8086775a5fe0cb288992dedd48a4f3bbb5d70561` |

**开始核验：2026-10-28 04:30:28 UTC（13:30:28 JST）。** 远程 HEAD／base 与指定对象一致，PR open、未合并、Auto-merge 为 null。本地 HEAD／树一致，工作区与暂存区干净，Review 095 不存在。没有切换、重置或清理工作区。

读取适用 [AGENTS.md](../../AGENTS.md)、全部20文件增量、[§114](../review-resolution.md#L6855)、第7版草案及原型文档§14，追踪新增调用者所用的报告、进程、闸门、终判与清理辅助函数。没有重新进行已验收模块或全部 T1～T6 审计。

四个提交的顺序及范围独立核对如下：

1. `69151703b30c61cc0ba59c5935000a4d50201421`：Q-1／Q-2 与第7版草案同步。
2. `dcdb2b26067d8325cb4ef6ac28840f44e4da7704`：仅追加原型计划；没有原型代码。
3. `d45a319dd3f61b5e98ff7bb95249c082fb2541dd`：直接以计划提交为父提交，增加原型代码及分支触发。
4. `1be48c328baea7b7c1664849dfedcc9f9d6d5107`：仅 AGENTS、implementation-plan、review-resolution、wp2.5-prototype 四份文档。

base→HEAD **20文件，+2761／−60**：11个原型测试文件、诊断workflow、AGENTS及7份docs。生产代码、根模块依赖、Schema／示例、迁移和根 `ci.yml` 无差异；原型仍是独立Go模块。诊断workflow只增授权分支和对应注释，路径过滤、只读权限、并发不取消、三平台、20分钟作业预算、15分钟测试预算、Linux `-race`、失败传播及步骤均未变。

计划的§14.1～§14.6从 `dcdb2b2` 到固定HEAD保持相同，结果另加在§14.7之后。review-resolution原有 **6853行前缀完整保留**，§114为追加，历史报告没有改写。§114明确把结果及门槛状态列为待独立审查，没有自行宣布冻结或验收。

以下仓库行号均指固定HEAD。原型代码点与HEAD代码相同，相关位置也适用于 `d45a319`。

## 3．Q-1／Q-2(a) 的落实

### 3.1 宿主禁用与产品边界

[规范规则6:724～735](../wp2.5-check-runner.md#L724)、[G1:1553～1558](../wp2.5-check-runner.md#L1553)、A5／A6／E5、architecture §6.4、data-model §7.4和EV-28／EV-31，实质共同落实：

- F1、恢复前终止未确认的A5、恢复异常终止未确认的A6，使同一宿主持续禁用至进程结束；消费者后来结束、释放或清单清空不解除。
- 没有创建进程及已确认终止的管理失败A4，不仅因为该次失败而禁用；它们仍可能同时受到独立的O3／D21条件影响。
- O3和D21维持原触发规则，O1不被改称输出不完整，禁用不改变当前项的qualifying。
- 只限制后续检查准入，不扩大到Git／doctor等其他操作；禁用不是清理，不授权对未知对象动作，也不保证新进程重跑安全。

原型 [host:52～112](../../prototype/wp25/q12_test.go#L52) 的准入、移交与置位在同一互斥规则下完成，释放不清除disabled。直接表覆盖F1／A5／A6与不触发项，保留O3／D21；这支持**模型规则**，不证明生产单例、并发调用拓扑或数据库／Git零访问已经接线。

### 3.2 A12、字段、取消与持久化

[§8.6:676～679](../wp2.5-check-runner.md#L676)、[A12及说明:752～760](../wp2.5-check-runner.md#L752)、[§19.10:1942～1943](../wp2.5-check-runner.md#L1942)、[data-model:1119](../data-model.md#L1119)与EV-31一致：

| 事项 | 本轮裁定 |
|---|---|
| A12 的含义 | 树已确认的异常尝试；可能执行过，不证明实际执行，也不套用确定未运行的A4；是Q-2新增语义，不是D19原分支 |
| 记录 | 未被D8截断时尝试 `error/context_changed`，不读取结束上下文，`head_end=NULL`、`tree_state_end=unknown`，qualifying=false |
| 时间 | `started_at` 为异常观察的挂钟；duration为 `t_S3−t_anom` 的定义区间；finished_at在写事务前。不是实际指令执行时间 |
| 退出码 | 可靠取得的数值保留，不因读取晚、恢复异常或因果未知丢弃；未取得为NULL。没有重引入S3截点 |
| 取消 | 树已确认且写事务前取消，先按D8不记录；事务开始后按实际提交／回滚／未知处理。F1与取消叠加仍按D19 |
| seq／持久化 | 执行记录的既有seq与事务规则未改，失败与提交未知继续叠加，不覆盖U/T/C/O/K；不能凭无记录或退出6安全重跑 |

Q-2没有新增字段／枚举或自动修订历史配置，没有改变逐条推导、observed、M2-Q2、TD-23、方案B及隐私保证。模型 `decideAnomaly` 和固定判定表支持已选分支的逻辑；没有生产事务、迁移0009或真实seq验证。本轮不把它们当作缺失的生产验收来阻塞原型。

## 4．BLOCKER B1：新增观察入口没有完整失败关闭

**依据：** [事先固定的共同规则:2303～2316](../wp2.5-prototype.md#L2303)明确要求观察失败不可撤销、外层工具失败不能confirmed、清理成功不能恢复实验有效性。原有 `scen.flag` 与最终判定已能保持invalid；问题是新增调用者没有把所有相关错误交给它。

以下是固定代码中可以逐步推出结果的错误分支，**不是宣称本机或CI已自然发生这些原生API故障**。本轮未为这些入口增加新接缝或新实验类型；动态证据和源码证明分开列出。

### 4.1 P-4：读取失败被合法NULL吸收，独立对照又被跳过

**位置：** [p4Exit:333～338](../../prototype/wp25/p4_windows_test.go#L333)、[调用及记录:250～258](../../prototype/wp25/p4_windows_test.go#L250)、[外层读取:516～521](../../prototype/wp25/p4_windows_test.go#L516)、[断言:598～607](../../prototype/wp25/p4_windows_test.go#L598)。

具体序列，以正常命中 `resume.gt1` 的A12为例：

1. 真实构造返回2，Job结束确认成功，Runner的进程句柄可靠发出信号。
2. `GetExitCodeProcess` 返回错误。`p4Exit`把它变成空字符串，没有保存错误或调用 `s.flag`。
3. `p4Code("")`得到“未取得”；模型合法地产生 `exit_code=NULL`。其余A12字段、关闭和Wait均正常。
4. 外层自己的句柄成功读到 `exit 1`。但598～602行**仅在 `in.Exit != ""` 时对照**；此时跳过。期望和实际的模型退出码都为NULL，其余断言也可以全部满足。
5. 清理成功，`s.invalid`、mismatch均为空，`scen.done`给出 **confirmed，测试退出0**。

**预期：** 这次测量已发生工具读取错误，应invalid／非零，不能继续用“合法未取得”证明实验有效。生产记录允许NULL，不豁免实验工具自身的失败。

还有一个更直接的外层变体：`pre.assign`内层按A4正确结束，但516行外层零等待一次返回错误。代码将alive设为false、outerExit留空，未flag；这一行又不需要退出码对照。614行后续等待若成功，清理及其余断言即可使其confirmed。**后一次成功没有恢复前一次独立观测。** 异常路径250行的 `werr` 也缺少216行正常启动路径已有的错误处理。

[Microsoft GetExitCodeProcess契约](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-getexitcodeprocess)把API失败与成功返回的终态数值分开；[WaitForSingleObject契约](https://learn.microsoft.com/en-us/windows/win32/api/synchapi/nf-synchapi-waitforsingleobject)也把WAIT_FAILED与正常WAIT_TIMEOUT分开。这里不应把正常尚未结束与工具无法读取合成一个空值。

**最小修正：** 读取辅助函数返回明确的值／是否取得／错误；内层与外层的非预期读取错误持续进入invalid，后续成功不清除。对照不能通过双方或一方空字符串自动豁免。保留合法WAIT_TIMEOUT和明确计划的query故障构造：后者是在测试机制如何处理失败，不是要把所有负向实验一律invalid。增加现有P-4链路的单次失败后恢复回归，断言实际错误命中、最终invalid／非零及安全清理。

### 4.2 P-2：正常组的步骤错误可以只被打印，不参与终判

**位置：** [p2Check:228～246](../../prototype/wp25/p2_linux_test.go#L228)、[r.Steps的判断:455～499](../../prototype/wp25/p2_linux_test.go#L455)。

检查进程的poll把 `perr` 写入Steps，但无论该调用失败与否，仍继续 `waitid`、Wait并填写CheckStatus／CheckReaped。外层对步骤错误的扫描只存在于 `w.denied != ""` 的组，且只扫描到被拒绝调用之前；**normal没有这道检查**。因此，“检查阶段poll已经出现未处理的观察错误，之后waitid取得exit 0、回收成功、最终属性与描述符读数正常”的报告可以满足全部正常组断言，最终confirmed。

这不是要求把已按协议正常重试的EINTR、预期ECHILD或本组指定的seccomp errno算作无效；是要求区分它们与**没有按重试协议处理、只是记录后继续的错误**。不能让 `s.worked(len(Steps))` 或最终exit 0替代这项检查。拒绝组在目标调用之后的额外错误也不能一概忽略。

**最小修正：** 对每组声明允许的返回及步骤序列，保留明确的重试语义，检查完整步骤而非仅目标之前的前缀；非预期的工具错误持续invalid。失败后不得落入没有剩余期限保护的阻塞 `waitid`／Wait。增加正常组出现一次观察失败而后续回收成功的定向门禁对照，不改生产能力政策。

### 4.3 P-1：未知基线被重新包装成预期的非空基线拒绝

**位置：** 新增 [p1Barrier:21～25](../../prototype/wp25/p1_linux_test.go#L21)调用既有 [exclusive:102～107](../../prototype/wp25/exclusive_linux_test.go#L102)，由 [runP1:121、164～165](../../prototype/wp25/p1_test.go#L121)消费。

`childrenOf`读取失败时，exclusive返回包装 `errBaseline` 的错误。新调用者只看 `errors.Is(err,errBaseline) && !called`，没有向场景传播读取失败。在alive／exited中该布尔值正好是期望的true；如果之后清理与末次枚举恢复成功，实验仍可confirmed，输出却写成“因本进程有子进程，基线屏障拒绝”。

保守拒绝启动本身是正确的，**但“未知而拒绝”不能冒充本组要求的“可靠观察到非空基线而拒绝”证据**。若想专门验证未知基线的正确拒绝，应有明确的目标与期望，不能在普通对照中消掉工具错误。

**最小修正：** 保留错误的类别／原因，让新增调用者对非预期枚举错误flag；正常非空基线仍按原机制拒绝。健康对照还应确认实际进入函数，而不只确认“不是errBaseline拒绝”，避免其他错误也被当作正常准入。

### 4.4 影响与修复范围

这一项阻塞**本原型PR的有效性门禁**，不是确定的生产误杀、生产错误ready或Q-1／Q-2政策缺陷。当前CI和本轮健康运行里的已见原生成功、构造返回值及清理事实仍保留；不能把它们扩张成“新增观察链已经失败关闭”。

修正应限定于新增观察／报告结果进入终判的链路及相关调用者，审计同类等待、退出码、步骤与释放返回值；用明确期望区分计划内故障和外层工具失败。保留健康confirmed、可靠反证refuted、工具失效invalid与环境不具备unsupported，不把所有用例改成invalid，也不要求提前开发生产Runner。

## 5．P-1／P-2／P-4 的证据与边界

### 5.1 P-1：真实对象与模型准入可以支持什么

- 三平台alive／exited／healthy各两次在CI实际执行。Unix在补救前核对启动时间、父进程及僵尸状态；Windows用外层另开进程句柄，并在alive中核对被测Job成员资格。
- 清单检查比较实际启动的 `*exec.Cmd` 指针；Windows另比较Runner持有的进程句柄并零等待；F1分支没有先启动阻塞Wait。Unix观察描述符的关闭得到外层核对。
- goroutine证据是栈采样及已列帧，另有本轮创建的原型goroutine列表，不是goroutine总数。它不证明采样之间从未发生过瞬时Wait，也不覆盖未来依赖库所有等待实现；原型文档已限制范围，源码中的F1分支本身没有Wait。
- 外层先输出内层／分支／清单／准入快照，再补救。补救或释放失败会flag，普通 `scen.done` 仍核对自有进程结束与Linux子进程；补救成功没有清除mismatch／invalid。
- 数据库／Git／启动计数是模型body接缝，不是生产零访问。当前模型没有证明生产全局实例、其他已准入调用或宿主退出拓扑。

**Windows alive单列：** [55～71行](../../prototype/wp25/p1_windows_test.go#L55)复制了被测Job句柄，由外层继续持有。官方[Job生命周期说明](https://learn.microsoft.com/en-us/windows/win32/procthread/job-objects)明确KILL_ON_JOB_CLOSE作用于最后一个句柄的关闭。因此内层关闭自己的句柄时，本夹具刻意阻止了这一次终止触发。

它支持“在该受控前提下主进程仍存活、F1返回不阻塞、Cmd／有效句柄被保留、宿主禁用”的局部结论；**不支持**真实F1只剩最后句柄时仍存活、不证明真正不可终止进程、终止系统调用原生失败或全部G3前提。披露限制是必要且正确的，但不能据此宣布G3完整满足，也不能因此抹掉已取得的局部证据。

### 5.2 P-2：Linux能力与限制

过滤器只装在再次启动的实验进程及其后代；TSYNC返回、架构、系统调用号与prctl低32位参数匹配均可从源码定位。正常组无过滤器、三次；六个拒绝／恢复失败组各一次。自读与外读是同一内核状态的两侧读取，不是两套独立机制。

CI每组都保留了过滤器状态、原生步骤、被拒绝errno及前置调用；normal的own-pidfd waitid为ECHILD，子检查实际Start、观察exit 0、回收。SET1被拒绝时没有检查PID；SET0被拒绝时SET1／GET1、检查回收、GET仍为1和之后探测拒绝都有记录。外层在释放实验进程之前检查没有子进程与pidfd，之后核对退出码。

**SET0失败后实验进程退出不算内层清除成功**，代码／文档明确保持属性1直到退出。原生拒绝响应是有价值的局部证据；B1的错误漏判仍须修正。这里只支持被测x86_64、当前内核／过滤器组合，不能推出arm64、旧内核、任意容器或所有组合故障通过。父端的最终空集合不能单独证明某个检查曾运行，实际Start／状态／回收的链和读数须同时保留。

### 5.3 P-4：API、前提构造、模型分别计证据

| 场景 | 证据及结论边界 |
|---|---|
| healthy／返回1 | 真实ResumeThread与正常结束；是协议启动，不是精确实际执行起点的证明 |
| 返回0 | 夹具先以真实API恢复并记录返回1，Runner随后真实返回0。构造在Runner快照／打开线程之后；不拿当前计数或没有标记文件证明过去执行 |
| 返回2 | 夹具真实SuspendThread返回1，Runner真实返回2；是改变前提的构造，不是自然出现的事故 |
| 返回-1 | 既有接缝改为没有恢复权限的线程句柄，真实API给ERROR_ACCESS_DENIED；与直接伪造返回值不同 |
| pre.assign／snapshot／openthread | 接缝没有调用相应原生API；支持这些失败分支的响应与所有权，不证明自然故障已发生 |
| pre.unconfirmed／A5 | 挂起进程实际存在，外层在补救前确认未结束；模型禁用、不记录，Runner句柄移交，补救后仍禁用 |
| anomaly.unconfirmed／A6 | 真实返回2、查询失败接缝；不能确认树结束，即使主进程后来已退出仍走A6、保留可靠数值、禁用。不是不可终止进程实验 |
| A12／Q-2 | 模型给不合格context_changed、NULL／unknown结束上下文、可靠数值保留和定义区间；不接入SQLite、seq或生产输出 |

实际夹具事件、接缝命中、资源句柄和记录判定都有日志；周期时间只作辅助读数，不能证明历史执行。每次两遍、共18次是重复观察，不是18种独特机制。实际异常duration均为0ms也不能单独证明端点公式；非零固定输入判定表和源码补足的是**模型公式**。

外层在内层返回后保存状态，随后以自己持有的进程句柄补救并等待，未嵌套外层Job；只适用于helper不创建后代的已声明前提。归属来自本轮创建的Cmd及打开的句柄，没有拿未经验证的报告PID动作。P-4在清单释放后确实再次准入检查；它不替代P-1同名断言的实际顺序，见N1。

## 6．NON-BLOCKER 与文字同步

**N1：P-1“释放后仍拒绝”的取样发生在释放前。** [p1_test.go:171～178](../../prototype/wp25/p1_test.go#L171)先remedy，再取 `acc2/aerr2`，然后才 `h.release` 和逐项release；[192～193行](../../prototype/wp25/p1_test.go#L192)却把旧值标成after the end and the release。建议把第二次request移到释放及其错误处理之后，证据行也按实际顺序更新。

这是具体的覆盖／表述缺口，但当前release没有复位代码，[TestQ12Classify的host对照](../../prototype/wp25/q12_test.go#L234)与P-4又独立执行了释放后拒绝，所以不单独据此宣布模型已经可恢复或新增BLOCKER。不能把这些别处的证据写成P-1已经按原声称的顺序执行。

**N2：活跃摘要的旧口径尚未全部同步。** 例如规范 [H维度:670](../wp2.5-check-runner.md#L670)仍只列O3／D21，§19.1引言只列C-4／D21；[1987行](../wp2.5-check-runner.md#L1987)仍无条件称四项最小原型没有实施；data-model §4.11的failure_kind说明仍概括“三种原因”，而同文后面已加入A12。建议分别补Q-1／进程项、改成历史状态或指向§14当前状态、补A12。

具体规则6、A12与§14都很明确，未发现摘要形成另一个可实施的相反结果，故按非阻塞同步处理。原型文档§9～§12的旧产品选择已由头部明确标为当时记录，不需要重写历史，也不把Q-1/Q-2重新列待选。

另建议冻结时把§19.1中“在T_kill到期”精化为确认协议结束／以观察错误提前结束的移交点，与已有C-19及P-1 exited、P-4 query-failure路径对齐；不能让早返回的F1在置位前留下准入空档。Q-1已明确覆盖这些F1，这项说明不需要另增一个禁用选择。

## 7．独立验证、作者证据与未验证项

固定HEAD导出到仓库外隔离副本，使用现有 **Go1.27.1、Windows/amd64、Windows 10.0.26300**，GOTOOLCHAIN=local、GOPROXY=off、本轮独立GOCACHE；没有安装或升级。没有运行用户项目配置，没有修改隔离副本的原型源码或增加实验类型。

| 独立执行 | 结果及范围 |
|---|---|
| `go test -count=1 -v -timeout=90s -run '^(TestP1\|TestP4\|TestQ12Classify)$' ./...`（原型模块） | 首次PASS，27.232s；P-1六次、P-4十八次全部confirmed；Q-1十行／host、Q-2五行、栈识别检查通过。不是新增门禁回归或组合通过数 |
| 既有 `WP25_INJECT=report-malformed`，只运行 `TestP1/alive.1`，timeout60s | 首次如预期FAIL／非零：真实helper报告含 `limit not-a-pid`，内层invalid，launches1、work0、cleanup verified，0.095s。没有直接伪造verdict；证明共用畸形报告门禁确实执行 |
| `go test -tags nowebui -count=1 -timeout=120s -v ./internal/contract` | 首次PASS，6.299s，45合法示例／190非法反例，无FAIL／SKIP；不证明新记录语义或平台机制 |
| Git／文档静态检查 | 20文件范围、计划先行与不回改、历史前缀、代码点后四文档、workflow权限／预算、`git diff --check`完成 |

本机日志中25个本轮创建PID在结束盘点时均已不存在，场景自身也确认结束／回收；没有向未知PID发信号。B1中未处理的读取错误是**源码证明的条件反例**，没有冒称本机复现其原生错误；没有为此扩大到未授权的新故障机制或全量变异。Linux／macOS本轮没有本机动态运行，相关结论来自固定代码、完整CI日志与明确的API依据。

作者本地Windows／WSL运行、两项变异只作为作者证据：新增Wait后被检出，支持所列等待帧／原型栈的检测；过滤器不安装被检出，支持该状态／errno对照。它们不覆盖本轮指出的所有新增错误入口。编译时类型名冲突、首次完整运行TestStartPoints失败，以及P-4初稿构造时点在首次运行前改正的历史均保留，不算本轮独立复验或有效变异检出；WSL缺pwsh的unsupported也不算通过。只读检索中两次无效命令已改正，没有把它们算成产品失败或有效测试。

## 8．CI 独立核验与计数

独立读取两个run的正式元数据、全部11份作业日志、实际checkout及合并Git对象；没有依赖作者回执推定汇总，没有重跑／触发workflow。

### 8.1 原型诊断

run **40439516254**，push，attempt1，正式completed/success；三作业都实际检出 **`d45a319dd3f61b5e98ff7bb95249c082fb2541dd`**，WP25META也声明相同SHA。

| Job／ID | 环境 | 唯一场景结果 | 门禁回归 | 组合 |
|---|---|---|---|---|
| macOS／121976202174 | macOS26.6.2／Darwin25.6.0，arm64，Go1.27.3 | 33 confirmed（旧27＋P-1六次） | 73/73 | 13/13 |
| Ubuntu／121976202347 | Linux6.17.0-1022-azure，amd64，Go1.27.3 | 53 confirmed（旧38＋P-1六次＋P-2九次） | 82/82 | 13/13 |
| Windows／121976202551 | Windows10.0.26100，amd64，Go1.27.3 | 53 confirmed（旧29＋P-1六次＋P-4十八次） | 78/78 | 13/13 |

Ubuntu步骤按Linux分支实际带 `-race`，不是仅因为没有竞态告警就推定。TestQ12Classify、TestStartPoints、TestPhaseClock、TestStopRead、TestToolState、TestWitnessJudge、TestRecordAlive等直接测试实际PASS。三个内层专用测试TestInnerForeignPID／TestInnerLeak／TestInnerStopError在普通入口按设计SKIP，其既有外层门禁调用单独计数，不把SKIP记为通过。

Results步骤再次打印结果行；按每作业ID去重后才是上表计数。三平台共有139条执行观察，**场景ID并集86**（原有53＋新增33个带重复编号的ID）；新增P-1／P-2／P-4合计45次跨平台执行观察，分别18／9／18，不能说成45种独立机制。新增配置种类为P-1三种、P-2七组、P-4九种。门禁PASS、直接测试、组合PASS均未加入机制confirmed数。

日志没有本轮refuted／invalid／unsupported或FAIL。这说明已执行的这些观察得到绿色，不是B1错误路径已经被检验。代码点到HEAD的prototype／.github差异为空，诊断对当前代码适用；文档收尾没有被冒充成另一轮诊断。

### 8.2 最终常规CI

run **40440832011**，pull_request，attempt1，正式completed/success、8/8。

| Job | ID | 正式结果 | 实际checkout |
|---|---|---|---|
| Go lint | 121980512555 | success | `8086775a5fe0cb288992dedd48a4f3bbb5d70561` |
| Go tests (ubuntu-latest) | 121980512788 | success | 同上 |
| Go tests (macos-latest) | 121980512795 | success | 同上 |
| Go tests (windows-latest) | 121980512820 | success | 同上 |
| Web | 121980513266 | success | 同上 |
| Build with dashboard (windows-latest) | 121980620527 | success | 同上 |
| Build with dashboard (ubuntu-latest) | 121980620561 | success | 同上 |
| Build with dashboard (macos-latest) | 121980620599 | success | 同上 |

合并快照父提交依次为固定base与固定HEAD，树 **`60473bc3391be35962f20eb20a1edb0a1439b44c`** 与HEAD相同；八个作业checkout一致。三平台根模块契约检查45／190实际运行。Ubuntu普通测试带-race；另行编译并以root运行的八项跨用户测试PASS，不把root步骤当成race步骤。macOS／Windows的root步骤按平台跳过，真实旧Git、unsafe-fetch Git、显式M-14诊断和适用权限测试的SKIP仍是未执行。

常规CI不运行原型模块，不能替代原型诊断；诊断success也不能修复本报告的静态门禁问题或关闭M-14～M-17。旧失败、取消、macOS历史未知及N-087-1保持原身份，不回填成已解决。§114所述此前PR #37合并／master运行属于保留的历史登记，本轮没有把它们算作新原型证据。

## 9．门槛状态与最小下一步

| 门槛 | 可确认的规格／支持 | 尚缺什么 |
|---|---|---|
| G1 | Q-1的永久禁用、原子准入／置位与不因释放恢复的规则成立；模型有直接表和真实进程驱动 | B1及N1修正；生产单例、并发准入、已准入调用的下一项、完整资源生命周期仍待生产接线测试 |
| G3 | Unix活跃／僵尸F1、Cmd保留及无所列阻塞Wait；Windows额外Job句柄前提下的局部所有权／准入支持 | 新观察链失败关闭；Windows构造不能证明真实最后句柄关闭仍存活。G3不能仅凭18/18整体标满足；冻结时明确哪些证据差额仍需最小验证、哪些确实是生产阶段测试 |
| G6 | 当前x86_64环境的正常／单项seccomp拒绝、SET1与SET0失败路径已有原生返回与两侧读数 | B1中的完整步骤判定和错误传播；不能外推未测架构／旧内核／任意组合。生产探测与全局闸门集成另验 |
| G7 | 返回1／0／2／-1及A4／A5／A6／A12的知识状态和局部所有权方向得到支持 | B1；未构造的多线程、恢复后关闭失败等按既有门槛分类，不能用18/18替全部原生失败验证。模型不是生产父端／事务保证 |
| G12／D14 | Q-2记录语义、可靠数值保留、定义区间、取消优先级及不合格公式的规格可以确认 | 生产记录／seq／事务／输出、迁移0009和正式Schema约束及业务一致性测试未实现；当前45／190不覆盖它们 |
| G5／P-3 | 支持矩阵及缺口整理正确，承认Go侧两架构定义相同不是SDK／内核行为证明 | **未满足**：具体SDK／XNU版本、siginfo_t依据及支持声明／Intel适用验证仍缺。没有runner不算通过；不自动缩小已批准范围 |

**当前PR的最小后续工作：** 修复B1，把新入口的真实错误与正常尚未满足、计划内故障分开并持续进入终判；用现有P场景补单次失败后恢复的回归及健康对照，先保留内层状态再补救，确认非零与最终清理。修正N1的取样顺序、同步N2。更新证据后定向复审，不必为此实现生产Runner或重审所有T1～T6。

**冻结之前：** 在有效证据门禁下重新裁定G3／G6／G7的差额，补足G5及其他F类规格／支持声明；技术候选的采纳须与数据模型／架构／验收一致。Q-1／Q-2已经决定，不再作为缺少产品选择阻塞；本轮没有新增保证豁免。

**获得生产授权之后、验收之前：** 生产Runner、全局闸门与C-9、真实取消／配置固定／空批次、记录事务与seq、Schema／CLI输出、平台故障矩阵及E2E。它们尚未授权，本报告不要求提前完成，也不因原型绿色宣布它们已验收。

## 10．结束核验

**结束核验：2026-10-28 04:52:00 UTC（13:52:00 JST）。** 远程HEAD、base与合并快照均未变化；PR仍open、未合并、Auto-merge为null。诊断40439516254仍为代码点的push、attempt1、completed/success；常规40440832011仍为固定HEAD的pull_request、attempt1、completed/success。

本地HEAD与树不变，暂存区及所有已跟踪文件无差异；唯一新增是未提交的本报告，报告相对链接无缺失目标。没有修改其他会话内容，没有提交、推送、合并或触发／重跑CI。

**最终裁定：Q-1／Q-2(a)实质落实，原型有局部支持，但B1未满足本PR自己的有效性门禁；技术NO-GO，当前不建议合并。** 先修门禁及定向回归，不以常规8/8或模型confirmed宣布冻结、生产正确或WP2.5验收。本审查不批准保证调整、接受风险或生产实施。
