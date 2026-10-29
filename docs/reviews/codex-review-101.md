# Codex Review 101 — PR #41，P-3 独立审查

## 1．结论

**固定 PR 技术 NO-GO，当前不建议合并。** Q-3(a) 的解码、独立规范化和两架构的正常数值观察有实质支持；但是新增原型仍有三项具体问题：计划故障会覆盖真实观察错误、工具/观察器关闭错误未进入终判，以及枚举的依赖内部重试不受声明期限控制。它们属于当前原型的有效性与收尾问题，不是要求提前开发生产 Runner。

| 问题 | 裁定 |
|---|---|
| A．技术与合并 | **NO-GO；先修 B1～B3，再定向复审**。最终常规 CI 和诊断均正式 success，此事实不抵消下述反例 |
| B．证据 | 可采纳固定代码、所列系统上的正常 R/N/M、布局和指定接缝响应；不能采纳“所有相关工具错误均能 invalid”“全部句柄释放已确认”“枚举已受剩余期限约束”的完整保证 |
| C．G5 | 增加同一代码点的 arm64/x86_64、SDK 布局及 E1 局部证据；**G5 尚未满足**，仍缺本次问题收口、生产入口裁定及支持差额审查 |
| D．产品决定 | Q-3(a)、Q-1/Q-2(a)、C-4、D21 等沿用已批准范围；本轮不新增产品决定或接受风险 |
| 历史 | Review 099/100 的规范层关闭结论不重开；当前问题是实施与所声明协议的差距 |

三项 BLOCKER 的位置、事件序列、证据强度和最小修正见 §4。exact 集合的显式检查、原始值记录方式和少量证据表述列为 NON-BLOCKER，见 §5；没有把未命中的猜想判成当前 P-3 的假通过。

## 2．身份、范围与历史核验

开始：**2026-10-29 11:59:01 JST（02:59:01 UTC）**。远程与本地一致，工作区/暂存区干净，没有目标报告；本地分支 `test/wp2.5-prototype-p3`。读取适用 AGENTS.md；仓库内没有另一个下级 AGENTS.md。

| 对象 | 完整身份 |
|---|---|
| PR | [XKHoshizora/specmetry#41](https://github.com/XKHoshizora/specmetry/pull/41)，open、未合并、auto_merge=null |
| base / master | `dbcd5d4597e295b31d6a3e0fb3d295b135e6386b` |
| HEAD | `c776f6a5f0e9a1fd09d4a1959e18122fe99b8307` |
| HEAD 树 | `6d976e3262a9ebe0acfebbe9f3aaebf8656911d1` |
| 原型代码点 | `4e690d813b88af3005da6e6ca25d9cc851ee6f92` |
| 受审规格 | `344cbcae2f2ac0903c1a1052a0266cbb68cd03ef`；实施依赖 PR #40 最终 `b359b0cb02848434e3c38475b42f157dff1b9014` |
| 常规 CI 合并快照 | `15f70b48f77dc919c36fc7d8b60fe91de64e2fe4` |

独立读取全部 14 文件差异（+2153/−56）、Review 100、原型文档 §15、规范 §19.5/§19.12 及必要关联条文、§124。PR 不是生产实现，没有根依赖、Schema、迁移或根 ci.yml 改动；诊断 workflow 的差异是精确分支、标准 Intel 矩阵和 macOS 元数据，权限、路径过滤、失败传播、作业20分钟及测试15分钟未变。

提交顺序及实际范围：

| 提交 | 作者时间（JST） | 实际内容 |
|---|---|---|
| `acb25bc94b3a6fae7fb5d6557f8059eaaeff5807` | 03:40:12 | 仅原型文档，预先登记 §15.1～15.5 |
| `692f2ad43caecb13bf4bbac5a4f2f9e6da086154` | 03:58:39 | 仅原型文档，补充/偏离1～5 |
| `34342df366c558e53a9f0e77c8875cc762ab4e79` | 04:00:01 | 仅原型文档，登记门禁选择器 |
| `4e690d813b88af3005da6e6ca25d9cc851ee6f92` | 04:05:30 | 本轮唯一原型代码与 workflow 提交 |
| `c776f6a5f0e9a1fd09d4a1959e18122fe99b8307` | 04:21:28 | 仅五份文档的结果和状态 |

Git 对象证明计划提交在代码提交之前、三个计划提交没有原型代码；不能仅凭作者时间证明工作站上第一次编辑代码的时刻。后两次登记也明确是“写代码时发现”，不冒称全部细节在第一次编码之前已写定。

`4e690d8 → HEAD` 的 prototype/.github 差异为空。base→HEAD 的历史审查目录无变化；review-resolution 的 base 7708 行前缀逐行完全一致，只追加 §124 的92行。固定差异 `git diff --check` 通过。没有修改原有产品决定、维护事项或已关闭审查原文。

## 3．规格实施与可采纳的证据

### 3.1 W/K/C 与独立 N

位置：`tree_darwin_test.go:165`～`:372`，`p3_darwin_test.go:60`～`:99`、`:327`～`:400`、`:595`～`:779`。

- W 以六个 uint32 读取；W1、W3a 与 W4 先检查，只有可靠正常退出才取 `&0xff`。信号仍是独立种类；没有把未知或 API 失败掩码成0。
- K 保存有符号 data，核对真实返回的 fflags、0～0xffff 范围及退出/信号/core/停止模式；C1 比较规范化的种类和值。
- N 不调用 decodeW/decodeK，也不使用它们的 String 来构造预期；W 用 `%256`，K 用商/余数解释。固定预测表直接登记256、263、−1等的预期，不从 M 反算。代码仍可能存在共同规格假设，但本轮没有发现二者共享同一个错误函数而自证通过。
- waitid 两次读取在同一直接子进程确认为未回收僵尸之后；C 路径先读 K，再确认僵尸、读 W，最后才 reap。不会因提前 Cmd.Wait 改变 W。
- 原生值被候选规则拒绝时记入 M/预测比较；API 错误进入工具失效。`k.esrch` 若成功注册、未走提前分支，注册结果本身与预测不同而 refuted，不伪装成已验证回退。注册的非预期 API 错误仍 invalid。

两个 macOS 作业逐个 ID、每次重复核对到以下原生读数（不是本机复现）：

| 场景 | W 原始状态 | K 原始 data | N/M |
|---|---|---|---|
| exit7 | 7 | 0x0700 | exit7 |
| exit256 | 256 | 0 | exit0 |
| exit263 | 263 | 0x0700 | exit7 |
| exit−1 | 0x00ffffff | 0xff00 | exit255 |
| exit0x7f000007 | 7 | 0x0700 | exit7 |
| exit INT_MIN | 0 | 0 | exit0 |
| SIGKILL | code2/status9 | data9 | signal9 |

重复 W 相同；C 的 `263 / 0x0700` 与 `0x00ffffff / 0xff00` 原始值不同、规范化相同；k.esrch 确实记录 ESRCH 与两次状态7。它们支持受控前提、所测内核下的推演，不证明固定公开 XNU 与运行内核相同，不重开 Q-3(a)。

### 3.2 布局与环境

位置：`p3_darwin_test.go:225`～`:310`、`:879`～`:1059`，`p3_check_darwin_test.go:124`。

实际是 **55个键 + 一项128字节容量比较 = 56项比较**，不是56个独立键。每架构三次探针均记录55键。解析器检查完整键集、重复（包括同值）、未知/缺少/多余行、最后换行、规范十进制、符号和宽度；64KiB上限代替 Scanner，不存在被忽略的 Scanner 错误。C 程序检查 fflush/ferror；Go 另核对工具退出状态和输出读取。

SDK26.5 的 siginfo_t=104、kevent=32、kinfo_proc=648以及字段偏移、SYS_waitid=173、NOTE_EXITSTATUS等与 Go 侧相符；只证明用户态 SDK/Go 布局，不证明内核宏分支。部分类型采用“偏移/大小是uint64、常量按16/32位符号”解释；当前55键的合法域不会因 int64 容器而截断。

同进程必须先有布局 confirmed；没有探针或存在 refuted/invalid 会阻止依赖读取，全部 unsupported 则依赖场景 unsupported。元数据读取错误先 flag，之后 SkipNow 不能恢复有效性。

uname.machine 与 hw.machine 同源的偏离如实登记。本轮独立阅读 x/sys v0.49.0 的 Uname：Machine 确实来自 CTL_HW/HW_MACHINE。这两项不能当作独立交叉验证；当前架构规则另结合 GOARCH、Rosetta状态和作业镜像，满足受审条件，不因此改变支持承诺。

### 3.3 已有观察器的接入

`waitidNoWait` 和 exitObserver.wait 实际接入新解码器，startTree 调用 startGroup 后再建观察器；P-3 则可自己注册或不注册。新代码点上33个既有 macOS结果确实重跑，旧 `18bc6a4` 的 macOS观察降为历史是正确的。但新代码点的正常绿色不能补上本报告指出的关闭错误与重试边界。

## 4．BLOCKER

### B1：计划内枚举故障覆盖了同一次真实读取错误，能够 confirmed

**位置**：`prototype/wp25/p3_darwin_test.go:842`～`:856`，调用者`:814`～`:824`；终判 `main_test.go:354`～`:371`。

`confirmEnd(tr,true)` 首次取得 `groupLive` 的 `(m,err)` 后，switch 先匹配 `fail && first`，只把真实err打印进F，随即用注入错误覆盖它；后面的 `case err != nil: s.flag(...)` 不再执行。F里出现错误文本不是 T 中的持续保存。

合法反例序列：

1. 负载、组长和成员的前置核验成功。
2. 目标第一次真实 groupLive 返回 EIO/EPERM等计划外错误。
3. 计划枚举故障分支记录hit=1，将此err覆盖；候选返回“不确认：成员列表未知”。
4. 调用者的预期正好匹配；随后实际观察恢复，既有 confirm/reap/cleanup 全部成功。
5. s.invalid 仍为空，结果可为 **confirmed/0**。这违反 §19.12“工具失效先于机制故障预测”，以及函数自身“真实读取失败是工具失效”的说明。

**独立验证**：在仓库外，将固定源码的 confirmEnd 函数体直接抽出，只改接收者/依赖的名称供 Windows 编译；以受控返回值代替 Darwin API，沿现有 scen.done 终判。不是原生 EIO，也不是完整 P-3 场景：

| 输入 | 实际结果 |
|---|---|
| 第一次读取正常，计划故障打开 | hit1、confirmed/0（健康计划故障对照） |
| 第一次读取 EIO，计划故障关闭 | s.invalid 保存错误，invalid/1 |
| 第一次读取 EIO，计划故障打开 | F含真实EIO，但invalid为空，**confirmed/0** |

这验证具体分支，而不是直接伪造 verdict。真实 P-3 后续 cleanup 不会补回已丢失的错误；没有把这个纯控制流结果冒称 macOS 原生复现。

**影响与最小修正**：在计划覆盖前独立保存真实调用的非预期错误；真实调用失败不得成为有效单故障实验。计划故障只改变 M 的观察输入，不豁免 T。加一项“目标真实错误 + 计划故障 + 后续恢复”的定向回归，保留健康故障仍 confirmed。无需改变产品决定或生产代码。

本次 CI 的六次 group.enum-unknown 的 F 都明确显示真实读取 `error <nil>`；因此不能声称这六行已经触发 B1。它们可以支持“健康读取被计划替换后拒绝确认”的局部响应，不能证明门禁会保留并发真实错误。

### B2：工具文件和观察器的 Close 错误被丢弃，完整终判与资源声明不成立

**位置**：`p3_darwin_test.go:1101`～`:1115`，`:513`～`:521`；`tree_darwin_test.go:108`～`:120`；`main_test.go:295`～`:298`、`:350`～`:371`。

新增 toolRaw 忽略 stdout.Close/stderr.Close 的返回；P-3登记的 obs.close 也返回 void 并忽略 unix.Close。关闭错误不交给s.flag或其他保存者。仅观察直接进程退出、解析成功和普查为空，不能补证父端文件/kqueue的关闭成功。

明确控制流反例：工具真实启动并退出0、输出完整，父端任一次 Close 返回错误；或者观察器拿到合法事件后关闭失败。其余预测、进程清理均成功时，没有分支消费该错误，s.done 仍可给出 **confirmed**。这不是缺少压力实验，而是错误返回没有到终判的路径。

另有确定的早退资源缺口：stdout 创建成功、stderr 创建失败（反向亦然）时，invalidf 发生在成功句柄的 Close/所有者登记之前；场景清理只管理进程，不能释放这个文件句柄。该路径会失败，不称为绿色假通过，但不能写成资源已清理。

**证据范围**：源码证明；本轮没有在 macOS 注入原生 Close 错误，也未声称 CI 实际发生了该错误。Close 的失败究竟表示已关闭还是状态不确定，须按相应平台处理，不能笼统说所有错误都泄漏，也不能把“忽略/不重试”当作已释放。

**最小修正**：成功取得每个句柄后立即建立退出所有权；关闭结果由共享层持续保存并进入 invalid，包括部分创建、Start失败和s.done的关闭。观察器需把关闭结果交给拥有它的场景。用真实关闭后返回注入错误等有边界接缝验证“关闭失败→本项/诊断失败”，并验证正常释放。不要求实现生产资源管理器或盲目重试未知句柄。

### B3：成员枚举存在外层无法控制的 ENOMEM 重试，所声明预算没有覆盖实际调用链

**位置**：`p3_darwin_test.go:842`～`:864`、`:1145`；`tree_darwin_test.go:55`～`:58`；`census_darwin_test.go` 的procList；规范 §19.5 枚举预算及 §19.12 期限。

新 P-3 的确认循环虽然建立deadline，但 groupLive 直接调用 x/sys.SysctlKinfoProcSlice。固定依赖 **v0.49.0 → `b6b8557e8e2974925aec7a136ba0e323eedc5f43`** 的实现，在第二次sysctl因表增长返回ENOMEM时，于函数内部直接continue，**没有截止或次数检查**。工具普查的kern.proc.all也经这个实现。

独立核对[官方固定源码](https://github.com/golang/sys/blob/b6b8557e8e2974925aec7a136ba0e323eedc5f43/unix/syscall_darwin.go#L520)：526行开始循环，541～544行在ENOMEM时重来，直到成功或其他错误才回到调用者。

反例是连续多次“查大小成功、实际读取ENOMEM”的合法API序列：即使已越过枚举、观察/清理阶段的截止，P-3仍留在该内部循环；外层的时间比较、r.limits和cleanLimit都只能在它返回以后运行。go test最终panic不是该项按预算结束并核实清理，§19.12自己也明确不把这种终止当作清理确认。

同类较小问题：kqWaitRaw在EINTR后无条件continue，到期后仍可能不断进行零等待调用；需要在重试边界消费同一个截止。toolRaw把事先计算的d在Start后完整交给reap，r.limits/cleanLimit属于事后有效性检查，不能单独证明整个阶段准时收尾。

**最小修正**：至少让P-3所用枚举的可重试步骤受同一绝对截止控制，或在当前原型明确、可靠地拒绝其无法支持的路径；到期保存错误并保留对象的收尾责任。不要用另一个遗留goroutine包住不可取消枚举就称为已回收，也不要提高现有预算。本项不要求系统调用硬实时返回；问题是可以控制的用户态重试循环没有检查期限。

**证据范围**：固定依赖及调用链的静态证明，没有制造真实ENOMEM风暴、挂起系统调用或不可回收进程。未测原生ENOMEM本可作为证据限制；但当前实际循环与无条件“在剩余期限内重试”的规格不一致，不能仅以“以后生产再测”处理。

## 5．NON-BLOCKER 与门禁的具体裁定

### N1：exact没有显式校验预期集合；本轮不能把未命中的缺失反例说成P-3已假通过

位置：`gate_test.go:540`～`:550`、`:731`～`:749`、`:921`～`:935`。

exact只遍历已出现的非目标结果，检查它们confirmed；没有预期ID集合/数量。parser按ID覆盖，也不拒绝同进程的重复结果。建议声明本用例完整集合 `{P3.layout.1, 目标ID}`，逐项存在、恰好一次，拒绝其他ID；不要仅从“其余遍历都成功”推出集合完整。

独立 Windows 共享链反例/对照（每项一次、每次总期限120秒，均在隔离副本）：

1. 首次选两个真实进程场景 normal-drain/silent-holder，丢弃已实际执行的非目标结果后，**门禁正确拒绝**：6个创建对象没有Owned登记，Final仍ended、保险清理nothing left。这个首次未命中必须保留，不能算发现了P-3绿色反例。
2. 之后换为真实 normal-drain 与零进程的 doc-under-load，除parser丢弃非目标结果外，不改inner verdict/退出或judge；健康与丢结果两组均得到 exact PASS。证明显式集合检查确实缺失，但不是macOS布局场景的原生复现。

对当前P-3做进一步条件核对：layout会真实启动5个工具，目标另外1个；当前13个外层结果均created6、unowned为空、shortfall0。若仅丢掉实际layout的结果，这5个创建记录会被登记检查抓到；若layout完全没有运行，layoutPremise会拒绝目标。因此不能仅凭“forall空集”就断言现有P-3少跑layout仍绿色。这里保留为非阻塞的显式完整性/可维护性增强，而非用它另造当前CI假通过。

### N2：R的标签不严格符合“改写前保存”

`readWaitid:250`和`kqWaitRaw:319`先执行改写钩子，再构造wRaw/kRaw；之后记录的R已是改写结果。F/site保留被改字段的前值，结合未改字段可以重建本次单字段改写的原值，但R本身不是说明声称的完整“原始读取”。

建议明确分开R_before与被测输入，或至少在现有字段中显式保存两者；这也便于证明注入没有借原本已异常的读数凑出同规则拒绝。当前原生运行没有启用钩子，R数值不受此问题影响；本轮没有仅凭标签问题否定全部原生读数。

### N3：证据措辞与支持范围

- §15.7“本轮所有运行都不需要补救”等应明确指新增P-3子集。整个套件包含故意遗留和补救的旧门禁/组合，不应扩大成全套件都无补救。
- 元数据的hw.machine/ uname.machine同源已经披露，不能将它们写成两份独立架构证据。metadata命令步骤只打印错误，不决定原型成功；真正准入由进程内检查承担。
- 当前“已验证组合”旧表有意仍保留待审状态，不要求本PR提前宣布G5成立。生产入口E1/E2尚未选择；如果未来选型涉及支持、信息或风险变化，须另行决定。

## 6．exact、登记、归属与清理的正面证据及限制

当前10个接缝确实各命中1次、内层invalid/退出1、目标理由仅一条、目标清理verified；三个健康对照内层confirmed/0。完整日志逐项核到返回字段/改写值和目标规则：

| 接缝组 | 本轮命中与拒绝 |
|---|---|
| W PID/signo/code | 第一次真实读后改相应字段，W1 |
| W highest/negative | 7→0x01000007、7→0xff0000ff，W3a |
| W repeated | 第二次7→8，W5 |
| K status/range/stopped | fflags去NOTE_EXITSTATUS；data→0x10000/0x117f，K1/K2 |
| C mismatch | K的0x700→0x800，W仍7，C1 |

外层仍分别检查工具错误、未允许证据问题、remedy、Final和insurance；没有因because命中恢复PASS。新Owned合并用于两场景的同一创建记录集合，已登记对象的身份不因此变成动作授权；动作入口仍经普查或ownedNow核实本轮归属。

Shortfall方面：目标exact要求清理verified；当前s.done有shortfall就写NOT verified并invalid，而非目标必须confirmed。因此当前P-3不能合法借另一个shortfall>0的结果掩盖登记缺口。N1建议仍是逐结果检查数值/结构并建立显式集合，不要把全局总数说成逐对象匹配。

工具直接进程由Cmd/childProc持有；Clang后代只凭本轮随机标记与启动身份的普查处理，不假装它们有Go helper回执。标记或状态读不到会flag，未确认归属不发信号。发现存活后代先保存失效，再补救；补救不会抹掉失效。B2的句柄处理、B3的普查重试仍是这条链的实际缺口，故不声称全链完成。

## 7．CI 身份、逐项计数与适用性

### 7.1 原型诊断

[run40751152882](https://github.com/XKHoshizora/specmetry/actions/runs/40751152882)：push、attempt1、正式completed/success，四作业均success。独立读取四份完整日志；实际checkout和WP25META均为完整代码点 `4e690d813b88af3005da6e6ca25d9cc851ee6f92`。

| 作业 / ID | 系统与工具 | 机制结果ID（去重） | 门禁 / 组合 |
|---|---|---|---|
| macos-latest /123037855426 | Go1.27.3 arm64；macOS26.6.2/25G83；xnu-12377.161.14~5/VMAPPLE；Rosetta0；Xcode26.6，SDK26.5，Clang21 | 96：旧33+P-3的63，日志均confirmed | 90：旧77+新增13；组合13 |
| macos-26-intel /123037855377 | Go1.27.3 amd64；macOS26.6.1/25G76；xnu-12377.161.13~4；Rosetta oid ENOENT；工具版本同上 | 同上 | 同上 |
| ubuntu-latest /123037855444 | Linux7.0.0-1017-azure，Go1.27.3；实际带-race | 53，日志均confirmed | 111 /13 |
| windows-latest /123037854710 | Windows10.0.26100，Go1.27.3 | 53，日志均confirmed | 92 /13 |

逐ID核对P-3的21种场景各三次，集合完整，没有缺项；这63个运行ID不是63种独立机制，更不能与另一架构相加成126种场景。布局55键/56比较、K24次、W21次、C9次、group6次。四作业的Results步骤会重复打印WP25RESULT，已去重；WP25P也有P-1行，按ID前缀过滤后P-3恰为63行。

两个macOS作业均有TestP3Decode、TestP3Parse实际PASS。三个内层专用TestInnerForeignPID/TestInnerLeak/TestInnerStopError在主运行设计SKIP；它们相应内层证据通过各门禁报告与原代码链判断，不把主运行SKIP算成机制通过。

P-3各项的T没有日志记录、结果cleanup均以verified开头，最大单次记录359ms/308ms；这些是实际读数，不是最坏时限保证。B1～B3解释为何“没有记录工具错误”不能被扩大为“所有相关错误皆可检出、资源和期限全部证明”。本轮发现的是代码反例，不改写run的正式success，也不声称该次实际发生了未记录Close错误或ENOMEM循环。

### 7.2 最终常规 CI

[run40753715499](https://github.com/XKHoshizora/specmetry/actions/runs/40753715499)：pull_request、attempt1、正式completed/success，8/8。八份完整日志的实际checkout均为 `15f70b48f77dc919c36fc7d8b60fe91de64e2fe4`：

| 作业 | ID |
|---|---|
| Web | 123046882977 |
| Go tests Ubuntu | 123046883209 |
| Go tests macOS | 123046883330 |
| Go tests Windows | 123046883335 |
| Go lint | 123046883409 |
| Build Ubuntu | 123047136669 |
| Build Windows | 123047136735 |
| Build macOS | 123047136738 |

GitHub Git commit API核实父提交依次为固定base、HEAD，树 `6d976e3262a9ebe0acfebbe9f3aaebf8656911d1` 与HEAD相同。Ubuntu普通测试带-race；root跨用户步骤是另行编译并运行，不能写成root race。真实旧Git、unsafe Git、平台权限及默认关闭的M14诊断等设计跳过保留。

常规CI不进入嵌套原型模块；8/8不替代诊断，更不验证本轮发现的故障组合。没有触发、重跑或取消任何远程作业。

## 8．本轮独立执行、首次失败与边界

环境：Windows10.0.26300/amd64，已安装Go1.27.1；GOTOOLCHAIN=local、GOPROXY=off。固定HEAD由git archive导出到仓库外 `C:\Users\hoshi\.codex\visualizations\2026\10\12\01a13137-991b-7b42-8ea2-3c24008343a0\review101`；未安装工具，未运行用户项目检查。

| 执行 | 次数/期限 | 实际结果与证据类型 |
|---|---|---|
| 原样副本定向测试 | 一次，90s | TestPipeNormalDrain、TestGateJudge、TestWitnessJudge、TestStartPoints、TestRecordAlive全部PASS；包6.435s；有根层机制场景，未触发零场景无效汇总 |
| exact的两个真实进程场景 | 健康/丢非目标各一次，共用120s | 健康PASS；丢结果被unaccounted拒绝，包预期之外FAIL/1，7.304s；已结束/无保险残留。**不是**有效绿色反例 |
| exact含零进程非目标 | 健康/丢非目标各一次，共用120s | 两组均PASS，7.300s；只支持§5 N1的显式集合不足，不冒称原生P-3布局遗漏通过 |
| confirmEnd原函数体控制流 | 重叠错误、健康、非计划错误各一次，分别30s | 重叠错误confirmed/0，健康confirmed/0，非计划错误invalid/1；后者预期失败保留。是API返回值接缝模型，不是Darwin原生EIO |

日志分别为 `pristine-targeted.log`、`gate-exact.log`、`gate-exact-no-process.log`、`enum-overlap.log`、`enum-healthy.log`、`enum-unplanned.log`。用于反例的改变只在隔离副本；真实exact实验没有修改inner verdict或judge，进程均由本轮go测试创建和现有Job/见证协议管理，所有外层Final为ended、insurance为nothing left，测试父进程已返回。纯控制流检查不启动helper。

本机不能原生运行Darwin。B2/B3为完整调用链/固定依赖的静态证据；没有原生Close错误、ENOMEM/EINTR、core或不可终止故障的独立动态证据。没有把抽出的纯函数/控制流、交叉平台测试或作者临时变异称为原生验证。

作者Windows/WSL测试早于最后两处Darwin修改，不能作为固定代码点全部Darwin路径的本机证据；本轮以固定点的两架构CI和源码判断补充。作者纯函数9处变异、首次有符号规范写法未检出后补例的历史仍保留，仅作作者辅助证据，没有复述为本轮全部独立检出。

## 9．G5 与后续事项的分层

| 类别 | 最小后续 |
|---|---|
| 当前PR必修 | B1真实错误在计划覆盖前保存；B2句柄/关闭结果进入终判并闭合早退所有权；B3枚举/重试使用真实可检查的剩余截止。修复后补目标故障+恢复及正常对照，定向独立复审；本报告不授权修改/重跑 |
| 冻结前必需 | 采纳修复后原型的适用证据；裁定生产E1/E2入口及兼容性；明确支持承诺与已测组合差额；对其余F类门槛及技术候选作冻结审查 |
| 证据限制 | 仅macOS26.6.2 arm64虚拟机和26.6.1 x86_64；SDK26.5与无公开源码的xnu-12377.161.x并非规范固定源码；只测信号9、每种三次；没有原生core/ENOMEM/EINTR。未测15.x Intel不能自动移除支持 |
| 授权后生产工作 | Runner/CLI、迁移0009、事务/取消/配置/空批次与输出叠加、C-9生产覆盖、真实平台接线与故障测试；不作为本PR提前实现条件 |
| 非阻塞增强 | N1显式集合/重复结果检查、N2改写前R、N3表述精化；不机械重开已关闭T1～T6或其他工作包 |

本轮能认可的G5增量，是两架构同一代码点的数值与布局局部支持、ESRCH回退及所列接缝响应；不认可由此推导生产ABI稳定、全部工具故障关闭、任意宿主/版本、进程树保证或G5整体完成。当前没有新的产品待决，若生产选型需要改变支持/风险，仍交负责人。

维护事项M-1、M-14～M-17、N-087-1保持原范围。Q-3(a)和其余已批准决定不重开；不冻结规范、不授权生产实施、不宣布WP2.5/M2验收。

## 10．结束核验

结束核验：**2026-10-29 12:20:28 JST（03:20:28 UTC）**。再次读取远程PR、master和两组CI：HEAD/base仍分别为 `c776f6a5f0e9a1fd09d4a1959e18122fe99b8307`、`dbcd5d4597e295b31d6a3e0fb3d295b135e6386b`，PR仍open、未合并、auto_merge=null；诊断/常规CI均仍attempt1正式success。本地HEAD和树仍为固定身份，工作区只新增本报告，跟踪文件和暂存区差异为空。没有混入其他对象。

原仓库仅新增本报告，保持未提交；未修改其他文件、提交、推送、合并、触发或重跑CI，未扩展P-3。
