# Codex Review 111：PR #44 P-5 实现与首次诊断证据独立审查

## 1．分层结论

**固定实现技术 NO-GO，当前不建议合并。** 除已正确保留的符号检查 T／诊断 failure 外，新增两项实现问题：**B1，正常 EOF 路径没有显式关闭门禁的父端读句柄；B2，新清理辅助链仍可遗失一次必要观察失败。** 这些是当前 test-only 原型的资源和证据契约问题，不是要求提前完成生产 Runner。

| 事项 | 裁定 |
|---|---|
| 实施授权是否生效 | **是**；PR #43 合并、同树核验与 master CI 已先于实施登记完成 |
| E2 接线、ABI 与分类主路径 | 源码方向符合已审计划；两个架构的局部原生、接缝和模型结果可保留，不能据此给整个实现 GO |
| 新 BLOCKER | **B1、B2**，见 §4、§5；均有实际源码路径及允许的事件序列 |
| “唯一 T 为 P5.static.symbols” | **在本次已打印的检查结果中属实**；不等于源码没有其他缺陷或所有资源已经核实释放 |
| 首次原型诊断 | **正式 failure，P-5 按计划为 invalid（T）**；不改判，不以其他通过项抵消 |
| 最终 HEAD 常规 CI | **正式 success，attempt 1，8/8**；不运行原型模块 |
| 作者方案 (a) | 可以作为请求下一步授权的方向；**只改 grep 并增加 otool -hv／-L 不足以补齐全部差额** |
| 当前合并建议 | **暂不合并**；先修正实现问题、固定等强度符号判据，再由负责人决定补证授权 |
| G5、冻结、生产与验收 | 本轮均不宣布满足或授权；Q-3(a)、Q-4(a)、H4／H6 和支持承诺不变 |

## 2．固定身份、范围与授权链

开始核验：**2026-10-30 22:16:06 JST**。本地工作区及暂存区干净，没有需要覆盖的其他任务改动。

| 项目 | 核验结果 |
|---|---|
| PR | [XKHoshizora/specmetry#44](https://github.com/XKHoshizora/specmetry/pull/44)，open、未合并、`auto_merge=null` |
| HEAD | `067526760e07be3da1d054c3da1939b39bf331dc` |
| base／远程 master | `021c956c33a65cc17ad050bda34883df355adcd0` |
| HEAD 树 | `1ef9f0bb70269b9c501dcc1b37ed6443f94d1c77` |
| 取证代码点／树 | `0213fb706e29130110af7ad8971dd8bbd94dd1d4`／`b097f99421c58ae8c3eaba8ad82bdea8e1e8f624` |
| 本地 | `test/wp2.5-prototype-p5`，HEAD／树与固定对象一致 |
| 已审计划 | `35ecd7984f4d4624f43b32e2e618c14e0257ee80` 的原型文档 §16 第四稿 |

读取当前完整 AGENTS.md、完整 Review 110、§16 及本轮补充／结果、§136～138、全部 21 文件增量和必要共享调用链。适用的受版本控制 AGENTS.md 只有根文件。Review 110 保留原文，18663 字节，SHA-256 `d1260d3671a2d8fec9745a80cc52beee88cb3020570db834d2764811320d9be7`，与 §136 相同；自归档提交 `e220042` 之后未变。

### 2.1 条件授权已生效

- §136 明确授权的前提是 PR #43 合并核验全部成功；不是 Review 110 的 GO 自动授权执行。
- 独立读取 PR #43：最终 HEAD `13cd4b99f9ca6986c4e5a3c0b21247b0d72aabf6`，已合并为本轮 base；父提交依次为 `0442f14`、`13cd4b9`，树 `c6b26542bb095cc0e6be41d5227d334dcc79178d` 与最终 HEAD 相同。
- PR #43 最终常规 run 41048732960 正式 success；master push [run 41050553619](https://github.com/XKHoshizora/specmetry/actions/runs/41050553619) 为 attempt 1、8/8 success，八份日志均检出 `021c956`。正式汇总于 **11:57:50 UTC** 更新完成，早于实施登记提交的 **12:00:28 UTC**。
- `35ecd79` 到该合并的计划技术正文未改；原型与 workflow 也未改。收尾只改标题、授权状态和必要指向。

### 2.2 登记、代码和回填顺序

| 提交 | 时间（JST） | 实际范围 |
|---|---|---|
| `83854cb24e7547b8370625c2828035e0c536e80f` | 21:00:28 | 仅 §16.14 与 §137，登记授权、计划和实现起点 |
| `aeb0a4c0da2724fc3f1e5e5ebd608d3f893c10d6` | 21:29:06 | 仅 §16.15，补充代码落点、模型、记录及符号检查写法 |
| `0213fb7` | 21:29:28 | 原型与诊断 workflow 的固定实现提交 |
| 固定 HEAD `0675267` | 其后 | 五份文档回填；prototype／.github 完全相同 |

Git 历史可以确认两次文档登记先于代码提交；不能仅凭提交顺序独立证明所有本地编辑都发生在登记之后。§16.15 的模型提取、结果字段、P-5 独立布局前提及共享重构均在授权范围内，未另改预测、次数、预算、角色 C 或产品决定。但“没有偏离／共享行为不变”的声明仍须受实现核查约束，不能豁免 §4、§5 的问题。

base→HEAD 为 **21 文件、3064 行增加／76 行删除**；其中代码提交 16 文件。没有生产代码、根依赖、Schema、迁移、正式验收语义或根 ci.yml 的变化；处理记录只追加 §137～138，历史审查未改写。`git diff --check` 成功。

## 3．实现主链中获得支持的部分

### 3.1 E2、指针、返回值与接线

`e2waitid/waitid_darwin.go:15`～`30` 动态导入 `/usr/lib/libSystem.B.dylib` 的 `waitid`，linkname 到固定的 `syscall.syscall6`；两个架构跳板都只有 JMP 和只读地址槽。参数按 uint32／指针宽度传递，消费者取 `int32(r1)`，没有自制另一套 errno 过滤。

独立读取 Go **1.27.3，`2aae1000ea1d4b28923b96db28acf4e3b99d6aff`** 的 [`syscall_darwin.go:328`～`408`](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/syscall/syscall_darwin.go#L328)、runtime 的 [arm64 调用桩](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/runtime/sys_darwin_arm64.s#L484)及 [amd64 调用桩](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/runtime/sys_darwin_amd64.s#L510)，并对照固定 x/sys `b6b8557e8e2974925aec7a136ba0e323eedc5f43` 的声明与生成包装器。所用签名、寄存器落点和过滤形状与计划相符；固定函数仅在 int32(r1) 为 −1 时传回 errno。

`tree_darwin_test.go:350`～`374` 的 waitidCall 是 E2 的唯一实际调用者，局部数组先置哨兵，指针转换位于调用参数表，调用后 KeepAlive；记录保留返回值、errno、原始缓冲区及哨兵结果。readWaitid、旧观察器的 ESRCH 提前路径、waitidNoWait、P-3 读取及 P5.err 均经此接线；原来的 raw SYS_WAITID 调用已删除。源码扫描不能抵抗任意未来改写，但覆盖当前实际入口，没有发现 E1 残留。

这支持固定工具链下的源码设计与所测调用；**源文件写着 libSystem 不等于本次 Mach-O 符号归属已经取得证据**，见 §7。Q-3(a) 的低 8 位规范化仍在身份、结构、状态核对之后；信号和异常输入没有被掩码成合法退出。

### 3.2 原生、独立预测、分类与门禁

`p5_darwin_test.go:343`～`565` 区分健康调用、构造错误、恢复读取和接缝站点的真实结果；X1～X5 的分类与计划一致。X2 的原生健康失败是反证，X3～X5 是协议 T；注入前失败不被换成成功。P5.err 不论错误调用的形式，仍取得恢复读取，以保留观察；原有 T／mismatch 不被后来成功清除。

N 仍用 p3NormW／p3Pred，由 R 独立计算，不调用 M 解码器。原始元组先记录，接缝只在真实 X1 后替换一次为 EINTR；对应证据明确标记被替换。实际计数由唯一 wrapper 递增，规则次数由消费者分支递增，计划次数另定；当前 wrapper 内确实只有一次底层调用。不是从实际计数反算一个相同的期望。

共享 parseInner／verdictOf 的提取基本保持旧判定；P-5 exact 另外要求明确存在布局和目标的最终结果、对应 evidence、非缺失 calls，以及 0／1／2 的固定期望。布局、唯一理由、命中、退出状态与外层见证／归属／最终状态仍分别核对。没有借这次提取关闭 Review 101 N1／N2。

### 3.3 W／R、锁与截止的主要接线

`p5proto_test.go:164`～`278` 实现 W2 置标记并解锁，W4 才执行唯一 Wait；R2 在同一控制锁内核对标记并 kill，R3 解锁后发布结果。父端只取独立状态快照，不取控制锁；R0 先登记再启动，持有表按门禁→最终收尾顺序使用，同一内层最多一个 pending 请求。

真实 ops.state 仅在 Wait 返回后读取 ProcessState，没有把阻塞 Wait 放进状态锁。Wait 的必要观察错误会累积，后来成功不清除；请求原始 errno、状态与完成分开保存。持有者的 until 共用绝对截止，k／w／d_g 和 D+30 未重新放大。唯一回收及 SIGCHLD 前提有专用接线；普查对内层不走裸 PID 动作，请求成功或 ESRCH 不冒充结束。

这些主要顺序成立，**不等于资源完整收尾已成立**。正常读端关闭缺口和清理错误传播见下文；挂起请求、未确认退出与移交，本次只有模型证据，不能写成原生验证。

## 4．BLOCKER B1：正常 EOF 路径没有关闭父端读句柄

**位置**：`p5gate_darwin_test.go:89`、`:148`～`151`、`:217`～`249`、`:473`～`491`；`p5proto_test.go:297`～`307`；模型对照 `p5proto_check_test.go:260`～`266`。

**合法事件序列及源码证明**：

1. p5Launch 创建 rd／wr，Start 成功，父端关闭 wr，把 rd 交给自己的复制任务；Cmd 只接收写端，不能替父端关闭 rd。
2. 内层正常结束，复制任务的 ReadFrom 读到 EOF 后返回；该任务没有 Close 或 defer Close。
3. runP5Gate 取得 `cs.Done=true`，所以不会进入唯一的 `p5StartStop(rd.Close)` 分支。它读取稳定输出、完成判断。
4. Wait、复制、请求都完成、无遗留进程时，pending=false，没有移交项；函数返回前也没有其他 rd.Close。

因此两条正常 P-5 门禁可以打印 PASS／“nothing handed over”，却没有确定性的读句柄关闭及关闭错误消费路径。EOF 只表示没有写端，不表示读端已经关闭。GC／文件 finalizer 或测试进程最终退出不能代替协议中的释放核验；本轮没有声称动态观察到了某个尚存 FD。

**模型与接线脱节**：TestP5ProtoCopyStop 的正常 EOF 对照在 `:266` 手动关闭 r2；真实门禁没有对应动作。模型通过不能证明这项实际收尾存在。当前两架构门禁日志均是 output_done=true、stop=none，与上述未覆盖路径一致。

**依据与影响**：已审 §16.8／16.9 要求文件有明确所有者和最终释放，工具关闭失败进入 T；Review 110 特别区分“进程结束／复制完成”与“资源释放”。这是当前正常路径的问题，不是长驻生产宿主的增强。

**最小修正方向**：在实际持有结构中保留读端关闭责任，EOF、读错误、超时与提前出口都经唯一的关闭任务／所有权状态收口；不重复关闭，不把 Close 放入控制或状态临界区；共用既有截止。先消费关闭结果再发布最终门禁结论，pending／带错完成项保留并移交。补正常 EOF 的真实接线回归及关闭返回错误的拒绝对照，不只在模型里手工 Close。未在本轮实施或执行。

## 5．BLOCKER B2：新清理辅助链会遗失必要观察失败

**位置**：`p5_darwin_test.go:821`～`857`，尤其 `:824`～`827`、`:838`、`:843`～`855`；调用者 `p5gate_darwin_test.go:575`～`603`、`p5_darwin_test.go:787`～`811`。关联动作检查 `p5gate_darwin_test.go:547`～`568`。

这里审查的是新增加的 P-5 辅助链，不要求顺带重写旧原型所有保险清理。

### 5.1 一次错误可被下一轮成功覆盖

p5EndProven 每次轮询重新建立 rerr：

1. 一个已证明归属、身份已知的对象进入 alive；普查本身成功。
2. 第一轮 recordAlive 读取状态失败，返回 errUnknownState；rerr 被设为错误，循环继续。
3. 下一轮对象可靠结束，recordAlive 返回 false／nil；新一轮 rerr 已被重置为 nil。
4. 函数返回 `nil, found, nil`，调用者可写“ended n”；先前错误既没有返回，也没有场景级持续保存。

这是直接由循环可推出的事件序列；不需要假造 verdict，也不依赖原生 EINVAL。与 W1 中已经正确采用的持续错误策略相反。

### 5.2 初始 unknown 也未被保存

初始 known 遍历只接纳 present／同身份／非僵尸；procInfo unknown 被直接跳过。后续普查或状态恢复不能补写这一次必要读取是否成功。动作前 procInfo、tagged 的失败也被布尔条件吞掉；跳过动作是正确的，但观察失效没有相应错误记录。直接终止调用的错误亦被忽略，须区分允许的 ESRCH 竞争与其他工具失败。

**实际影响及限定**：本次日志没有发生这些错误，不能据此改写任何原生读数。现有两个 P-5 门禁若已发现遗留／超时，通常已有其他失败，**本轮没有证明该错误单独把这两条已失败门禁恢复成 PASS**。但函数自己的错误、保险清理说明和最终收尾 `final_T` 可漏掉实际发生过的工具失败，违反 §16.7 的 T 持续规则及 Review 110 的所有权／错误消费要求。不能将“后来确认结束”当作“先前观察没有失效”。

**最小修正方向**：在本次清理调用和其持有者中持续累积观察／动作错误，可靠 absent、身份变化、仍存活、unknown 分开；unknown 对象不能因未进入 alive 就丢失原有登记责任，也不能凭 unknown 增加信号权限。把累积错误送到保险判定、门禁不可豁免问题和最终汇总；结束知识与实验有效性分别保存。增加一次失败、随后恢复的定向接缝对照，并核对调用者实际消费错误。本轮仅作源码证明，未运行新故障。

## 6．诊断独立核验与可以保留的证据

[run 41055799521](https://github.com/XKHoshizora/specmetry/actions/runs/41055799521)：push、attempt 1，正式 **completed/failure**。四份完整日志均实际检出 `0213fb7`；工具链均为 go1.27.3，GOTOOLCHAIN=local；两个 macOS 作业静态与执行步骤均 CGO_ENABLED=0，实际运行二进制的构建元数据也相符。

| Job | ID | 正式结果 | 按 ID 去重的根层／门禁／组合 |
|---|---|---|---|
| macos-latest | 124062896987 | failure | 139 confirmed／101 pass／16 pass |
| macos-26-intel | 124062897203 | failure | 139 confirmed／101 pass／16 pass |
| ubuntu-latest | 124062897166 | success | 53 confirmed／111 pass／13 pass，普通测试实际带 -race |
| windows-latest | 124062897340 | success | 53 confirmed／92 pass／13 pass |

139 是旧 96 加 P-5 的 43 个根层结果（静态入口 1、布局 3、原生 39），不是 139 种新机制。门禁内层和 Results 步骤重印没有重复计数。四作业均有三个内层专用 SKIP：TestInnerForeignPID、TestInnerLeak、TestInnerStopError；这些不算通过，外层实际执行另按门禁／组合核对。

### 6.1 两架构 P-5 的局部结果

- **环境**：arm64 为 macOS 26.6.2／25G83、xnu-12377.161.14~5、VMAPPLE；Intel 为 26.6.1／25G76、xnu-12377.161.13~4。两者 Xcode 26.6（17F113）、SDK 26.5、Apple clang 21.0.0。公开源码、SDK 布局和实际运行内核是不同层，不能互相等同。
- **原生**：逐项核对 13 个场景族各 3 次，共 39/39；33 次 calls=2、6 次 calls=3。例：原始 si_status=263 的两读都被保存，N／M 为 exit 7；ECHILD=10、EINVAL=22 的真实错误后均有两次健康读取，哨兵与原始元组可见。这些不是过滤前非零 errno 已建立的证明。
- **布局**：3/3，calls=0；每次 55 个布局／常量键与所用 Go 定义核对。两个内层各自的布局也 confirmed，不借用根层布局。
- **接缝**：两个架构均恰好命中一次 EINTR 站点，真实 X1 先保存；目标 invalid、exit 1、仅一条预期理由、calls=1。健康对照 confirmed、exit 0、calls=2。布局和目标最终结果、evidence、清理声明、见证和 Final ended 都存在。
- **W／R 原生覆盖**：四次门禁内层都在 k 之前自行结束，没有请求任务、没有移交，输出读到 EOF；所以不能声称 R1 等锁、R2 挂起、晚到错误或最终移交获得原生验证。进程／观察器的既有清理声明不覆盖 §4 未关闭的父端读句柄。
- **模型**：10 项直接检查通过，包含 8 项新协议／分类检查和 TestP3Decode／TestP3Parse。它们支持具体纯函数和同步顺序；不等同于所有真实 darwin 接线、资源释放或不可终止故障。
- **旧回归**：P-3 的 63 个根层结果、22 个门禁及有关 W／C 对照通过，新代码点都经 E2。旧 E1 数值保留为历史，不能拿来补齐 E2 的符号归属。

### 6.2 “唯一 T”的准确范围

两个 macOS 作业的 Prototype 步骤成功；P5.static.toolchain／vet／build／settings／entry／load 成立；**符号项是所打印静态结论中唯一 T**。最终 P-5 static verdict 正确使两个 job 失败，run 也正式 failure。WP25SUMMARY 的 invalid=0 只统计场景，不包含 shell 静态项；不能把它读成整个 P-5 无 T。

因此日志统计陈述可保留，但“其余通过”须受本报告发现的收尾缺口限制，不能升级为实现整体通过或所有资源已释放。

## 7．符号检查：已知脚本问题与尚未取得的归属证据

### 7.1 pipefail 与 grep -q

位置：`.github/workflows/wp25-prototype.yml:113`、`:119`～`130`。

两份 macOS 日志均记录 `bash --noprofile --norc -e -o pipefail`，真实 nm 命令输出有跳板与 `U _waitid`，随后出现 **`printf: write error: Broken pipe`**，符号项记 T。grep -q 找到匹配后可停止读取，大段 printf 因读端提前退出而失败，pipefail 又使整个管道失败，足以解释这一检查链的假阴性。这与 [GNU grep 官方说明](https://www.gnu.org/software/grep/manual/html_node/Usage.html)一致。

不能推断本次一定是固定退出码 141：没有保存各管道成员的退出状态，日志写的是 printf 错误。也不能因此推断另一个符号条件已经成立。同型风险还存在于 settings 的小输出管道，以及第二、第三个符号管道；本次 settings 通过，不代表写法无此问题。

**最小方法修正**：先执行并完整保存每个工具的输出和独立退出状态，再用文件输入或明确耗尽输入的匹配器检查。区分工具失败、未匹配与匹配器自身失败；不能用 `|| true`、忽略任意 SIGPIPE 或去掉失败传播来使检查变绿。可以只为打印摘要容忍无匹配，裁定仍必须消费真实命令状态。补一个大输出匹配对照和一个工具非零退出对照即可检查这个缺陷，无需扩大机制实验。

### 7.2 没有 from libSystem，究竟证明什么

实际打印的是 `(undefined) external _waitid`，没有 `(from libSystem)`。已知的是该测试产物含未定义的外部 `_waitid` 和候选跳板；**既不能证明它来自错误库，也不能据此宣布已经证明来自 libSystem**。

独立读取 Apple cctools 固定提交 **`e0d56624eca2a76c2ace4c21850df9e666de4ca5`**：[`nm.c:1804`～`1821`](https://github.com/apple-oss-distributions/cctools/blob/e0d56624eca2a76c2ace4c21850df9e666de4ca5/misc/nm.c#L1804) 只有在 MH_TWOLEVEL、相应 undefined 状态及非零 library ordinal 等条件下打印库说明；[`nlist.h:210`～`237`](https://github.com/apple-oss-distributions/cctools/blob/e0d56624eca2a76c2ace4c21850df9e666de4ca5/include/mach-o/nlist.h#L210) 定义了 ordinal 与加载命令的关系。这是固定公开实现依据，不声称本次 Xcode nm 的构建恰好对应此提交。

固定 Go [`macho.go:326`～`333`](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/macho.go#L326) 的内部链接头生成没有设置 MH_TWOLEVEL；其 [`machoDyldInfo:1428`～`1484`](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/macho.go#L1428) 又可以按动态导入库生成绑定记录。因此“nm 没打印 from”与“产物没有每符号的绑定信息”不是同一命题。

本次没有取得实际 Mach-O 头、加载命令和对应绑定记录；源码提供了具体解释方向，**尚不能据此补写本次产物究竟走了什么命名空间／绑定表示**。作者把原因保留未知是适当的；把它仅归因于“nm 格式”或仅看到库依赖就关闭归属缺口则证据不足。

### 7.3 候选办法能证明什么

| 办法 | 可以证明 | 不能单独证明／使用前条件 |
|---|---|---|
| `go tool nm` | 候选跳板、地址槽、未定义 waitid 符号存在 | 哪个库承担该导入；实际 ABI 正确 |
| `nm -m` | 在工具所支持表示中给出符号类型；有有效 ordinal 时可显示关联库 | 没有 from 不等于没有正确绑定；须了解头标志和工具适用性 |
| `otool -hv` | 实际 Mach-O 的架构、文件类型与头标志，如命名空间标志 | `_waitid` 对应哪个库 |
| `otool -L` | 产物声明的共享库依赖和版本 | **某个符号绑定到某库**；libSystem 在列表中并非符号归属证据 |
| `otool -l` | 原始加载命令及其顺序，供 ordinal 映射 | 若没有该符号自己的 ordinal／绑定记录，仍不能完成归属证明 |
| 解码实际产物的每符号绑定记录 | 把 `_waitid` 的导入／绑定位置及库 ordinal，与加载命令对应起来 | 须固定工具／格式、唯一记录、错误出口与允许的 re-export 解释；不能把 flat／dynamic lookup 或解析未知当作成功 |

otool 选项范围依据固定 cctools 的 [`otool-classic.1:66`～`74`](https://github.com/apple-oss-distributions/cctools/blob/e0d56624eca2a76c2ace4c21850df9e666de4ca5/man/otool-classic.1#L66)。若现成工具无法读取该产物的绑定表示，应先固定补证方法并报告缺口；不能临场只接受 -L。若要新增运行时地址／镜像观察，那是额外方法，必须先明确所测的是同一调用槽及库／re-export 关系并另获授权，本轮没有批准或执行。

**方法还是保证调整**：换成能证明同一 `_waitid` 导入归属的等强度检查，是 §16.4 的方法修订，仍需先固定判据和工具失败处理；把“来自 libSystem”降成“存在 libSystem 依赖”是实质降低取证要求，不能作为机械修脚本被顺带接受。

## 8．静态产物与实际测试产物、非阻塞项

**N1：产物对应关系的证据限制。** 静态检查读取 `$RUNNER_TEMP/wp25.test`，实际执行由之后的 go test 另行构建／启动。两者固定源码、Go 版本、CGO_ENABLED、架构和实际构建元数据一致；没有字节哈希／Build ID 相等证明，不能说“检查过的那个文件就是正在执行的文件”。§16.15 在代码提交前明确登记了这条方式，当前可按同源码／同设置的限定采纳。后续宜保存二者 Build ID／哈希或固定使用同一产物，同时保持原测试选择、期限和回归覆盖，不能为统一文件而减少验证。

**N2：模型仍有局部命中和所有权盲点。** TestP5ProtoRequestHung 没有明确的“已经进入 fake kill／已经尝试 W2”握手；pending 查询与 20 ms Sleep 本身不能排除相关 goroutine 尚未获得调度的顺序。因而不能仅凭该测试通过认定它确定覆盖了持控制锁挂起时的 W2 等锁。TestP5ProtoRequestSerial 启动的 fake Wait 循环也没有在测试末尾释放到完成；若干模型失败出口没有统一释放闸门的 Cleanup。它们不是自有 OS 子进程，不提供原生遗留证据，但不能据“10/10”断言目标调度和模型任务回收都已核实。建议补确定性的命中握手、统一释放和完成核对；锁顺序另有本轮静态依据，§4 的真实读端缺口则已属于 BLOCKER，不能用模型手工关闭掩盖。

**N3：汇总解析不是第二套有效性门禁。** p5StaticFile 忽略 Scanner／Close 错误、容许覆盖与未知项，p5Summary 也明确只打印、不裁定。当前 workflow 另有实际命令状态和 failed 标记，因此本次静态 T 没有被它清除；仍应把汇总完整性作为信息限制，后续若用该文件决定有效性，须先收紧解析和错误传播。未据此推定本次又发生了第三项静态失败。

旧 N1／N2、维护事项与历史未知不因编号相近混同；这里的 N1～N3 是本报告的限定性意见，不重开无关已验收模块。

## 9．常规 CI、最小下一步与授权边界

[run 41057548900](https://github.com/XKHoshizora/specmetry/actions/runs/41057548900)：pull_request、attempt 1，正式 **success，8/8**。八份完整日志实际 checkout 都是 **`e56b841b6af1b3261c8cca962fc34c567781af78`**；父提交依次为固定 base／HEAD，树与 HEAD 相同。

| Job | ID | 结论 |
|---|---|---|
| Go tests (macos-latest) | 124068904585 | success |
| Go tests (ubuntu-latest) | 124068904787 | success |
| Web | 124068904805 | success |
| Go lint | 124068904830 | success |
| Go tests (windows-latest) | 124068904899 | success |
| Build with dashboard (ubuntu-latest) | 124069066840 | success |
| Build with dashboard (windows-latest) | 124069066854 | success |
| Build with dashboard (macos-latest) | 124069067022 | success |

普通 Ubuntu／macOS／Windows 分别 15／7／10 个 SKIP 不计通过；Ubuntu 普通 -race 与单独 root 的 8 项 CrossUser 通过分开。常规 CI 不运行原型；不抵消原型诊断 failure，也不验证 B1／B2。

**建议的最小下一步，须由负责人决定授权**：

1. 不改变本次 failure／invalid 与局部原始观察；先固定 B1、B2 的修复范围及直接回归，避免把方案 (a) 缩成只有 workflow 修补。
2. 在代码前写清 §16.4 的新符号判据：工具及版本、实际产物身份、每符号与库的对应关系、无法解释的表示如何失败关闭；header／-L 只能作为辅助，不放宽归属要求。
3. 经负责人授权后，定向修正资源和错误链、消除脚本提前退出，并按固定方法补证；新代码点的次数、期限、角色 C、exact、归属与 H4／H6 不变。是否取得新诊断须单独明确授权，本报告不触发、不代批、不建议盲目重跑旧 run。
4. 新固定代码、有效两架构结果及失败传播回归须再独立复审，再决定合并；P-5 全部成立之后仍只是冻结审查的输入。支持与已测组合差额、生产入口落地和其他 F/L/P/N 事项继续按原边界处理。

可交接给实施方的提示词：

> 请先等待负责人授权，不改实现或重跑诊断。Review 111 发现 P-5 正常 EOF 路径缺少读端关闭／关闭错误消费，以及新清理链丢失必要观察错误。若获准，请只定向修正两项并补对应真实接线／一次失败后恢复的回归；另先固定符号归属的等强度新检查方法，保留命令失败，不以库依赖替代符号绑定。保持计划预算、次数、归属、exact 与产品决定；新固定代码点取证须另获授权，保留首次失败，不自行合并、冻结或转入生产。

## 10．独立验证与未验证边界

本轮独立执行的是只读 Git／文档／源码检查、官方固定源码与工具资料读取、现有 CI 完整日志解析和按 ID 去重、合法事件序列与调用链推演。**没有构建、运行 E2／P-5、布局探针、模型测试或新机制实验；没有创建实验子进程，没有安装工具或触发／重跑 workflow。** B1／B2 是源码证明，不冒称原生故障复现。

作者本机交叉构建、Windows／WSL 测试及模型运行只作为作者证据；CI 的两架构结果另列。本轮的只读检索有一次通配路径准备失败，改用 rg 的文件过滤后完成；一次工具编排正则错误未执行任何实验。GNU Bash 单页资料读取超时未用于结论，相关判断使用已取得的 GNU grep 官方说明、实际 shell 配置及日志。均不计为产品失败或有效反例。

当前缺陷、首次失败、已有局部支持、非阻塞增强、冻结前门槛和生产阶段工作已分别列明。既有维护事项及历史未知保持原范围；没有新增产品批准、保证调整或风险接受。

## 11．结束核验与暂停

**2026-10-30 22:27:28 JST** 结束核验：远程 PR HEAD／base／master 与 §2 一致，PR 仍 open、未合并、无 Auto-merge；诊断仍为取证代码点的 attempt 1 正式 failure，最终常规 CI 仍为固定 HEAD 的 attempt 1 正式 success。本地分支、HEAD、树均未变；受版本控制文件与暂存区差异为空，仅新增未提交的 `docs/reviews/codex-review-111.md`。Review 110 的字节哈希保持不变。

审查完成，暂停。合并、修正方法、代码修复和新诊断均等待负责人分别决定；本报告没有发出执行、冻结、生产或验收授权。
