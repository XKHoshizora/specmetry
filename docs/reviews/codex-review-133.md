# Review 133：PR #51 原 Q-B3-1(a) 强命题的限定证明准备

## 1. 裁定与固定对象

**当前文档：限定技术 GO，建议作为“有限目标的充分条件、未成立部分及未采纳方法提案”的纯文档合并。没有发现阻止这一用途的 BLOCKER。** 这不是对全文每一句推论的无条件认可：第 5 节登记两项非阻塞精化，本报告不采纳对 `Xsignal` 的过强反向蕴含，以及 L6 中依保留对象重新解释代码来源的论据。它们不得用于后续 present 判定、方法采纳或负责人选方案。

**逐组：**启动／回收一族 17 个目标具有继续定稿的限定技术基础；`Xsigaction`、`X__sigaction`、`___libc_sigaction` 另须实际文件／包选择前提。`Xsignal` 尚无充分的 absent 论证；也没有本轮的产物 present 证据。F-5 的能力完整性仍未解决。

**H-1：有继续文档定稿的技术基础，尚不是完整可执行的方法；没有采纳。** 原方法继续停止。第三批实施、链接验证、场所、workflow、资源与依赖例外仍未获本报告授权。

开始本地核验为 **2026-11-05 21:51:35 +09:00**；随后独立核对远程分支、master、PR 与运行元数据，均与固定对象一致。工作区和索引干净，PR open、未合并。

| 项目 | 固定并独立核验的值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #51](https://github.com/XKHoshizora/specmetry/pull/51) |
| 分支 | `docs/wp2.5-a-strong-claim-proof-prep` |
| HEAD | `ef36fb8cbd80e620a3a5af86b43595b923bb9c13` |
| tree | `302b832f58508b0dccafd62c3ce984d0db72f020` |
| 本地／远程 master、PR base | `9c5860676de2b714f42a7210dca887bc61c6ec41` |
| 常规 CI | `41964621276`，pull_request，attempt 1，completed/success，8/8 |
| 实际检出快照 | `db40735afec44c657a2ad8e116f57944b3a16bbd` |

本轮只读源码、Git、文本、已有日志及只读哈希；没有运行任何 Go 命令、构建、链接、验证分析器、驱动或实验，也没有安装工具、触发或重跑 CI。

## 2. 范围、历史与授权

读取当前 AGENTS.md、完整 Review 132、处理记录 §170～171、证明准备全文、链接审计与原计划、生产实施计划 §12，以及冻结 §19.4／§22 的相关规则。沿用已关闭的收尾协议，没有重新审计无关机制。

base→HEAD 只有提交 `ef36fb8`，六份文档，**512 行增加、6 行删除**：AGENTS.md 2/2、implementation-plan 1/1、review-resolution 75/0、link-proof-plan 5/1、新证明文档 427/0、wp2.5-implementation-plan 2/2。§171 在原文末追加；旧处理记录、历史审查没有修改。链接计划只增加状态与证明准备引用，技术正文没有偷偷采用 H-1。冻结正文、验收、生产代码、原型、测试、Schema、迁移、依赖和 workflow 均无差异。`git diff --check` 通过。

### 2.1 PR #50 收尾

- [PR #50](https://github.com/XKHoshizora/specmetry/pull/50) 已于 2026-11-05 02:54:08 UTC 合并，最终 HEAD `62b6d2f9e3b46779c8ed1e6ceeb86e814bd26e77`，merge 为本轮 base `9c5860676de2b714f42a7210dca887bc61c6ec41`。
- merge 父提交依次为 `30bc8aa951cf16b8fef2614e56ff902b5d47c475`、`62b6d2f9e3b46779c8ed1e6ceeb86e814bd26e77`，tree `95ad173b4724975707b483fc8033601cfdf2dc06` 与其最终 HEAD 相同。Review 132 受审提交 `6dce1a1`、归档 `a912acf`、收尾 `62b6d2f` 均是祖先。
- 归档现文件 34,136 字节，SHA-256 `47e37d30f737f5056e022af98d1a6f90f7d7050f40d1577a82f549fec9b0b009`；工作文件与 `a912acf` 的 blob 均为 `cf329849d62b2aee4faaa568b4ba6ba044e2f0b6`，与 §171 回执一致。没有另一个归档前临时源可重新做三方比较，不能把记录中的暂存步骤说成本轮重新执行。
- [最终 PR CI 41956623694](https://github.com/XKHoshizora/specmetry/actions/runs/41956623694) 正式 success、attempt 1、8/8；八份日志实际检出 `8f2d0ece55e2f7036f64b25c57fdab1a7bf75263`。独立读取该快照，父提交为 `30bc8aa…`、`62b6d2f…`，tree 为上述 `95ad173…`。
- [master CI 41958169544](https://github.com/XKHoshizora/specmetry/actions/runs/41958169544) 为 merge 本身的 push、attempt 1、success、8/8；八份日志实际检出 `9c58606…`。§170 所列限定证明准备的条件授权已生效；不等于实施或验证执行授权。

### 2.2 N-132-1／2 与越界回执

**N-132-1、N-132-2 按原范围关闭。** 审计 §3.2 已撤回 ABI wrapper 内联被包装体的例子，保留正确的命名／ABI 映射；固定流水线确实先全局内联、后生成 ABI wrapper，后者只生成调用／尾调用。C5 改为拟达到的构造，同时要求独立内联诊断与 A 本身的 DIE／非空范围命中，缺任一项为 T(control)，副作用不再被当作归属保证。

本轮可读到原 Claude Code 会话文件中的指定工具调用与结果。`toolu_018g5euKvcN2Qkd7picJVX7V` 的时间为 2026-11-04T18:59:17.154Z，结果为 18:59:25.270Z，命令、目录输出、最后的 `C:\Users\hoshi\go\pkg\mod` 与审计 §0.3 所引一致；结果没有标为错误。没有重新执行命令。

准确的证据分层是：

- 已有回执证明该工具调用文本和所保存输出，且 `go env` 的 stderr 被 `2>/dev/null` 丢弃。
- 缺失实际 Go 可执行文件路径／哈希、该次版本、独立退出状态、stderr、启动器／进程审计。此前 Go 1.27.1 的观测不能补成该次版本。
- “本轮 137 次 Bash 只有一次 Go”是作者对会话范围的盘点；本轮只独立核对指定调用对，没有重做整个会话的完整性审计。“没有构建、链接或下载”仍不能由这份回执证明。
- 查询参与了目录的定位／确认，是已登记越界，未事后补成授权。独立模块哈希和固定 Go 源码不依赖它；不采纳它作为实际工具链或构建身份的证据。

§170 的修订准确保留上述层次，历史 §169 未回改。原方法停止、方法 M-1～M-5 未采纳的状态没有变化。

## 3. 独立静态依据与证据身份

重新读取官方 tag，`go1.27.3` 指向 **`2aae1000ea1d4b28923b96db28acf4e3b99d6aff`**。以下判断锁定该提交，不以本机 Go 或可变分支替代。

关键固定来源：

- [内联入口与 defer 拒绝](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/inline/inl.go)、[StaticValue](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/ir/expr.go)。
- [去虚化／内联阶段](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/inline/interleaved/interleaved.go)、[静态去虚化](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/devirtualize/devirtualize.go)、[gc 主流水线](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/gc/main.go)。
- [方法包装与 HaveInlineBody](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/noder/reader.go)、[常量分支的 IR 写出](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/noder/writer.go)、[闭包命名／内容寻址](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/ir/func.go)。
- [逐函数编译](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/ssagen/pgen.go)、[指令按 ATEXT 分配](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/obj/plist.go)、[CodeblkPad](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/data.go)。
- [ELF 文本条目](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/symtab.go)、[内容寻址合并](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/loader/loader.go)、[SSA 的条件证明与分支删除](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/ssa/prove.go)。

另读取 ABI wrapper、intrinsic 表、对象写出、cmd/go 的 profile 选择及编译参数路径，核对相关前提。模块源码来自本机只读缓存；**独立重算全部 23 个 go.mod 模块的 dirhash，均与 go.sum 相同**。方法为逐文件 SHA-256，按模块前缀／相对路径的 Ordinal 顺序组成 `hash␠␠path\n`，再对 UTF-8 清单求 SHA-256／Base64；没有调用 Go。

关键三份结果：libc v1.77.1，4,199 文件，`h1:Ct8j47QtiZ1Enj2DtFXQtUqrPCAjdCmPjtCuvrYQ0Hs=`；sqlite v1.60.1，1,899 文件，`h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw=`；x/sys v0.49.0，555 文件，`h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=`。

这些哈希只识别所读目录，不认证上游原产、不证明构建实际使用了它们，也不证明作者临时脚本正确。没有运行实际文件选择、内联诊断、ELF 判定或正反对照。

## 4. L1～L6、宿主集合与逐组充分条件

### 4.1 引理与定理的采纳范围

| 项 | 本轮裁定 |
|---|---|
| L1 | 固定 K 的文本载体论证可采。CodeblkPad 遍历 Textp；Compile、Flushplist 按函数／ATEXT 生成文本；S-7 的限定范围沿用并抽核 genelfsym:214～222。不是“任何等价操作都只来自一个源码函数”的证明 |
| L2 | 对本轮普通函数宿主成立。TryInlineCall:914～915 消费 inlCallee；StaticValue:879～946 只追溯所列表达式和未重赋的 PAUTO。跨包内联、返回函数值的已内联结果会沿原函数体引入名字，因此须继续闭包扩展，不能只找直接调用 |
| 方法／去虚化 | 静态接口去虚化保持方法选择；生成的方法 wrapper 在 reader:4190～4193 自行内联，不能因为全局内联已经结束就忽略。当前小集合没有方法；Xsignal 的集合有方法，不能套用这个名字定理 |
| PGO／泛型 | 排除实际 profile 是必要前提。无仓库 *.pgo 只是一项静态检查；R1 仍须实际编译证据。目标自身非泛型还不够，需宿主链也不经过未审泛型／方法实例；当前小集合满足其静态范围 |
| L3 | 可采。实际进入 IR 的 ODEFER 使 hairyVisitor 拒绝设置 fn.Inl；HaveInlineBody 先检查 fn.Inl。__ccgo_strace=false 中的跟踪 defer 已被常量分支处理排除，不能拿来凑不可内联条件 |
| L4 | 固定普通函数的命名与分隔符规则可采；ABI、闭包、deferwrap、range 等不能只按裸前缀。每条原始 ELF 条目的身份／多重性仍须保留；方法、泛型和未知名字不能猜测映射 |
| L5 | 固定输入上的模块范围有依据。只读检索 23 模块及仓库，libc 标识符的模块引用范围确实集中在 libc 与 sqlite；没有发现使其他模块取得这些可内联函数体的路径。linkname 的无函数体别名与汇编 CALL 是引用／重定位，不是把 F 的主体复制进别处的理由。不能把此事实扩大为语言上永远禁止标准库导入外部代码 |
| L6 | 合并机制本身可采；“选中其他对象便不再算 F 的代码”不作为证明依据，见 N-133-2。必须保持原强命题，不能靠重命名来源获得 absent |

本轮认可的充分条件是**对已经穷尽的本轮小宿主集合**的逆否证明：若 F 的主体仍有指令，固定流水线要么保留 F 的普通文本，要么把主体内联进集合中的宿主文本；本轮相关复制路径未进入方法／泛型／初始化或会丢失名字的目标承载闭包。这样的可达文本在未剥离 A 的符号表中留下相应名字。它不要求 DWARF 保留每一份来源归属。

这不是对任意 Go 程序的通用 L2/L6 定理背书；也不是“只要看不到 F 本名就可以 absent”。执行时身份、实际编译条件和完整符号读取仍是前提。

### 4.2 集合算法的独立核对

独立检索了模块的声明、引用、相关函数字面量和 ABI0 CALL，并核对如下关键闭包；没有只接受作者的总数：

- `Xexecve`：Go 名字为自身、`Xexecv`、`Xexecl`、`Xexecle`、`Xexeclp`、`Xexecvp`、`Xexecvpe`、`X__execvpe`、`Xfexecve`，再加对应 9 个 Y 包装，共 18。K 中引用包括 `ccgo_linux_amd64.go:126013/126057/126101/126110/126140/126167/126199/126207/126225`。其他平台的同名定义没有把该闭包引入 sqlite 或其他声明。
- `X__execvpe` 是自身、`Xexecvp`、`Xexecvpe`、`Xexeclp` 及 4 个 Y；`Xexecv` 是自身、`Xexecl` 及 Y；其他 exec 叶子为自身及 Y。ABI0 块的 CALL 确实对应这些函数。
- wait 链为 `Xwait→Xwaitpid`、`Xwait3→Xwait4` 及 Y。全集另有 386／arm 的 `X__wait3_time32`／`X__wait4_time32`，所以不能把 K 的 2／4 当作全集的 3／6。
- `Xsystem`、`Xvfork`、`Xwaitid`、`Xpclose` 的相关小集合与表格相符。sqlite 的非测试源码与汇编中没有找到这组 T0 标识符；这里的“没有”是对固定文件的文本检查，不是运行可达性结论。
- 对 `___libc_sigaction`，独立读到 `Xabort`、`X__sigaction`、`Xsigaction`、断言／builtin 检查函数及 Y 的扩展。sqlite 的 vec/vfs 确有 Xabort 或 builtin 引用，不能混入 K 实际包集合后还声称只有 26 个。sqlite/lib 的 `sqlite_g_0000000000000003.go` 也有 builtin 引用，但其约束只选择 darwin；本轮读到约束，不把它当成 linux 已选文件。
- `Xsignal` 通过 `deliverSignal`、`checkSignal`、`Free` 和 `raiseSignal` 进入方法／通用入口，足以证明本定理不能直接适用；本轮没有重新生成 5,523、5,080、4,899 等大集合，未将其精确总数当作独立证据。

两处算法边界须保持：

1. §3.1:154 排除 init 不能成为通用规则：init 无法被其他代码按名引用，不妨碍它自己内联 F。当前认可的小集合没有相关 init 或包级值引用，故此缺口没有形成本轮目标的反例。未来出现这种命中，必须纳入初始化／闭包或停止给该目标 absent；H-1 目前也明确排除包级声明情形。
2. “`//` 之后忽略”及行首顶层声明定位不是 Go 词法分析的一般替代。例如字符串中的 `//` 不必是注释，原始字符串中的行首 func 不必是声明。对本轮关键引用逐处读取后，没有发现因此漏掉小集合的实际引用；字符串、局部同名和其他平台的命中可以扩大集合，不能据此证明算法对任意将来的源码绝不漏项。临时脚本未提交本身不构成 BLOCKER，执行前仍须固定逐目标名单与引用见证。

### 4.3 各组裁定

| 组 | 目标与结论 | 具体前提／断点 |
|---|---|---|
| 甲 | `Xfexecve`、`Xwait4`、`Xpclose`、`Xexecle`、`X__execvpe`、`Xexecl`、`Xexeclp`、`X__sigaction`：限定充分条件成立 | 独立核对 K 定义中真实 defer：126216、30021、137723、126027、126132、125983、126071、133603；不是跟踪分支。该依据要求实际选择这些定义。前 7 个另有全集小宿主论证；X__sigaction 不可把 K 结论扩大到所有平台 |
| 乙 | `Xsystem`、`Xexecve`、`Xexecv`、`Xexecvp`、`Xexecvpe`、`Xvfork`、`Xwaitid`、`Xwaitpid`、`Xwait`、`Xwait3`：小宿主集合的充分条件成立 | 普通函数／Y 包装；无相关方法、泛型或包级引用；排除实际 PGO；完整符号读取与固定模块输入。无需预言究竟会不会内联，也没有据函数太大就自行断言绝不内联 |
| 丙 | `Xsigaction`、`___libc_sigaction` 及与甲重叠的 `X__sigaction`：条件成立，附加前提必要 | K 的文件选择及六个实际包必须核对。darwin/freebsd Xabort 引用、vec/vfs 不能静默删掉；R4 不成立就不能继续使用 2／4／26 的集合。这里只认可静态条件，未取得实际 go list 证据 |
| 丁 | `Xsignal`：absent 论证未成立 | 方法包装名映射未写定、保守集合包含通用入口；尚未证明 absent 与已经证明 present 是两回事。§5.2 的强反向推断不采纳，见下节 |

甲组的简单 H={F} 形式只可在其实际定义及命名／合并边界成立时使用。甲／乙／丙的现有宿主闭包只有所列清理调用，没有把获认可目标主体搬入另一个无关名字闭包的相关路径；不可内联宿主的普通文本也不能仅因其 defer 闭包合并而消失。因此上述限定采纳不需要改变代码来源的含义。

**没有任何一组取得“实际 A 已满足这些条件”的证据，更没有由这些充分条件证明能力整体不可达。**

## 5. 非阻塞精化及不得采用的推论

### N-133-1：Xsignal 的反向蕴含过强

位置：[证明准备](../wp2.5-c9-strong-claim-proof-prep.md) **21、263～278、308、337 行**；处理记录 §171 **10923 行**的对应摘要，以及 AGENTS 的“很可能在产物中”状态说明。

源码确实有潜在调用边：`Free:470→checkSignal:476→deliverSignal:498→Xsignal`；通用入口另经 raiseSignal 到 deliverSignal。但是 **274 行将“Free 自身或任一被内联副本仍有代码”都升格为整条分支保留**，缺少充分依据。

Free 的代码不只有该调用：`:465` 的栈指针递减可以保留，而依赖 `checkSignals` 的分支未必保留。局部内联副本可以取得调用点事实；字段来自运行时数据，不等于编译器不能在某个调用点证明它为假。固定编译器的 StaticValue／内联和 SSA prove 的分支删除也没有提供“残留一部分 Free 就保留全部后续调用”的不变量。文中对 syscall 常量号内联副本已作限制，对 Free 却没有同样区分。

这是一项**静态蕴含尚未证明的缺口，不是本轮已经构建出原产物反例**。对自身独立符号，参数／字段未特化的调用边有更强的源码支持；仍应把“调用路径存在”与“该路径的目标指令在实际 A 中保留”分开证明，不能从保守宿主集合反推存在。当前无产物，因此 `Xsignal 很可能在 A` 至多是作者预测，不能作为 present 证据或排除方案的既成事实。

最小修正：保留“未证明 absent”及其独立断点；将摘要与 §5.2 改成潜在调用链／待验证预测，撤回对任意内联副本的必然蕴含。若希望继续主张自身符号的充分存在条件，另列逐环代码保留的依据与实际命中条件。无需为此次文字更正新增实验。

**非阻塞理由：**该推断没有被本轮认可的甲／乙／丙 absent 证明使用，Xsignal 仍明确未成立，H-1 也排除它。撤回该推断不改变“原 (a) 整体仍不能执行”的裁定。它必须在后续方法定稿／负责人据其选择范围之前更正，不能自动归入已接受的 L。

### N-133-2：L6 不得以合并后的代表对象重新定义来源

位置：证明准备 **114～116、126、141 行**。

loader:399～438 证明内容寻址符号可以共用索引、保留某份代表对象；ir/func:539～542 也确实把闭包标为内容寻址。**它不证明合并掉的目标来源已经从语义上被排除。** 115 行“保留别处那份，所以不是 F 的代码”不能成为满足原强命题的捷径。相同字节与重定位合并，不能仅依最后显示的名字否认它同时代表目标来源。

最小修正：把 L6 写成须排除或单独处理的来源歧义，不以代表对象定义 absent；将本轮适用性明确收束为当前有限宿主闭包的逐处论证。若未来目标承载代码可以经此合并失去所需名字，就不得给 absent，直至有额外必要表示的证明。

**非阻塞理由：**当前获认可目标不是任意内容寻址闭包；相关宿主的清理闭包没有承载这组目标主体，非内联宿主的普通文本仍提供必要表示。没有发现本轮目标可由此假 absent 的具体路径。本报告不采纳 L6 的来源重解释，也不由它扩大定理。今后扩目标不能只复用该句。

两项均为事实／证明范围精化，不是批准风险；没有据它们重开 Review 132 的已关闭结论或要求任意程序的通用证明。

## 6. F-5、C-9／C-10 与原方案的能力边界

新发现有固定源码支持，但操作类型必须区分：

| 位置 | 直接能力 | 与当前规则／方案的关系 |
|---|---|---|
| Xabort，libc_musl.go:669～675 | 调用 ___libc_sigaction；Tgkill／Kill 向自身发 SIGABRT；调用 resetSigDfl | 同时涉及处置与发信号。不是启动／回收函数；不能仅以 T0 的启动／回收结果覆盖它 |
| resetSigDfl，abort_linux.go:37～42 | 原始 RT_SIGACTION 与 RT_SIGPROCMASK，恢复指定信号默认处置并解除屏蔽 | 原始入口与直接 signal.Notify/Ignore/Reset 是不同路径。冻结 C-9 第二类只约束自有生产包，不能假称已经扫描第三方这些入口 |
| TLS.raiseSignal:524～537 | 执行模拟 C 处理函数，或 unix.Kill 向本进程发信号 | 发信号／运行处理函数，不等于设置 SIGCHLD 自动回收；需按目标、参数与前提分别评价 |
| TLS.killSignal:351～368 | 排队给模拟线程，或 unix.Kill 向本进程发信号 | 不是由名称即可认定为启动／回收能力，也不能无视其真实发信号效果 |
| X__syscall0～6、___syscall_cp、Xsyscall | 按参数系统调用号进入内核；2／3 另有模拟线程信号分支 | 有限命名函数 absent 不排除别处传入 exec、wait 或信号调用号。通用入口、函数表／unsafe 等仍是 F-5 的独立问题 |

冻结 §19.4 第一类包括对 SIGCHLD 的 signal.Ignore／Notify／Reset；C-10 检查 SIGCHLD 前提。不能把所有信号发送、模拟处置、内核处置修改等同成“已经破坏 C-10”，也不能因为某条路径只向自己发 SIGABRT 就说它没有 C-9／例外范围影响。需要按能力、信号值、调用位置与保证分别列清。

本轮充分支持的结论是：原 T0 不完整地代表了这些能力位置；有限 T0 的强 absent 也不等于能力整体不可达。没有证明 SQLite 的生产路径已经执行这些操作，也没有批准依赖例外。信号一族的完整方案需要技术补充或负责人另作范围／保证决定，不能把新发现顺手记为 L。

## 7. H-1 与 R1～R6

### 7.1 H-1 的技术基础及必要定稿

H-1 的三种目标状态方向合理，保留原强命题：

1. 工具与前提有效后，F 本身／有效派生表示存在，才能按固定规则 present。
2. 工具与前提有效，且已证明完整的 H(F) 全部无相应文本，才能 absent；它是充分条件，不是必要条件。
3. F 无直接表示、但宿主存在，只能“未得到 absent”。宿主代码可以不含 F 的残留主体，不能反向判 present；也不自动意味着工具失效。

T 仍优先于所有正常类别。H-1 是拟改原 §7 第 6、7 条，不能取消第 1～5 条的工具、身份、完整性与清理门禁。“未得到 absent”也不能算作“目标全 absent”或方法成功兑现原 (a)。

继续定稿至少须同步：

- 原计划 §1 的二分目标／三种总结论、§4.2 映射、§5／§7 读取和分类，以及 C1～C10。增加不确定目标时，原“无 T 即二分可判定”的总结论必须相应改写。
- 固定每个目标的 H、声明／文件／包来源、名字规则、原始 ELF 条目身份和未知映射出口；不能运行后按方便程度缩集合。
- noinline 目标与内联目标采用哪条证明、如何处理初始化、泛型／方法和内容合并。没有充分论证的目标继续未成立，不靠改 absent 含义恢复通过。
- 为新充分条件固定独立对照：目标本名消失但宿主仍在时不得 absent；全部必要表示缺失的健康 absent；目标／派生真实 present；前提或读取失败时 T。仅 Xmalloc 一个格式正对照不能验证整个完备性。
- 决定 B、dumpdep、DWARF、C5／C9 等是否仍作旁证；若保留，S-6/M-1 采集错误与身份／多重性问题仍须修；若删除，须明确取证要求与控制替代，不把旧对照自动计作新方法覆盖。

这些是未授权方法定稿的下一步，不是要求本 PR 提前实现工具。当前 H-1 具有继续准备的基础，尚未成立为完整可执行协议。

### 7.2 前提与实际证据

| 前提 | 已有静态支持 | 仍须取得的实际证据／边界 |
|---|---|---|
| R1 PGO／K | gc/main:240～246、interleaved:30～47、cmd/go profile 选择路径；仓库没有 *.pgo | 实际工具命令、参数、环境与构建元数据；没有仓库 profile 不能替代没有启用实际 profile。新缓存／实际编译记录与 profile 条件须关联 |
| R2 工具链 | 官方 tag 指向固定源码提交 | 实际执行的 go／compile／asm／link 与受控安装来源的身份和关联，不以 VERSION 或哈希记录独自认证官方原产 |
| R3 模块 | 本轮 23 份目录哈希与 go.sum 相符；导入范围与关键引用可读 | 实际模块图、replace／workspace／vendor 等输入条件与实际归档关联；目录哈希不是构建输入身份证明。go mod verify／模块清单各有作用，不能单独替代文件与编译记录 |
| R4 文件／包 | 构建约束、导入关系及关键反面对照支持静态选择 | 实际编入的 GoFiles／SFiles、包闭包与静态预期逐项核对；甲的 defer 路径及丙的小集合不能跳过这步 |
| R5 不剥离／ELF | 固定 K 的 S-7 路径与命名依据 | 实际 A 的 ELF、完整 .symtab、相关格式及条目读取成功；存在节名不等于读取完整 |
| R6 判定 | 有限集合与必要表示的静态论证 | 实际 A 的逐条匹配、独立对照与完整错误／清理消费；本轮没有 A，未给任何目标产物 absent/present |

原 (a) 整体仍缺：信号目标的充分证明／替代处理、F-5 的完整范围、H-1 的决定与方法同步、固定输入与对照、实际 R 证据，以及工具／收尾实现和执行的单独授权。17 个启动／回收目标的方法技术基础不自动成为 17 个事实 absent。

负责人可另行决定限定 (a) 范围、采用较弱的 (b)、换驱动 (c) 或改变依赖本身；每项都有相应规范／验收／维护影响。本报告不推荐用一个未经证明的“很可能 present”作选择的事实前提，也不代负责人选择。Q/X/E/U、F-8、生产 CGO、场所与资源维持待决。

## 8. 当前 CI 与历史证据

[run 41964621276](https://github.com/XKHoshizora/specmetry/actions/runs/41964621276) 正式为 pull_request、attempt 1、completed/success、8/8。读取八份完整日志并抽取 checkout／git log 的实际值；不是只看 job head_sha。

实际快照 `db40735afec44c657a2ad8e116f57944b3a16bbd` 的父提交依次为 `9c5860676de2b714f42a7210dca887bc61c6ec41`、`ef36fb8cbd80e620a3a5af86b43595b923bb9c13`，tree `302b832f58508b0dccafd62c3ce984d0db72f020` 与固定 HEAD 相同。

| 作业 | job ID | 实际结果与范围 |
|---|---|---|
| Web | 127038639399 | success；实际检出 db40735…，前端常规检查 |
| Go tests Windows | 127038639657 | success；同一快照；27 个 ok 包，10 个测试 SKIP |
| Go lint | 127038639662 | success；同一快照；tidy、vet、文档链接与 lint，0 issues |
| Go tests Ubuntu | 127038639760 | success；同一快照；27 个 ok 包，普通步骤 -race、15 个测试 SKIP；独立 root CrossUser 8 PASS |
| Go tests macOS | 127038639775 | success；同一快照；27 个 ok 包，7 个测试 SKIP |
| Build Ubuntu | 127038937353 | success；同一快照；默认嵌入构建、web 测试与 smoke |
| Build macOS | 127038937374 | success；同一快照；同上 |
| Build Windows | 127038937543 | success；同一快照；同上 |

三个 Go 测试作业为 go1.27.3：Ubuntu linux/amd64、Windows windows/amd64、macOS darwin/arm64。没有测试 FAIL 或 job error。共同跳过五项真实 fetch／旧 Git 用例与两项 M14；Ubuntu 普通步骤另跳过八项 CrossUser，随后 root 单独执行；Windows 另跳过三个 unreadable-storage/source 用例。macOS／Windows 的 Linux root 步骤本身正式 skipped，不与测试 SKIP 条数混算。

PR #50 最终运行与 master 运行也读取八份日志，实际检出和正式 success 与 §171 一致；其三个测试作业同为 27 包、15／7／10 SKIP，无 FAIL。保留此前运行、取消与失败历史，没有把本轮成功称作旧运行的重跑或旧问题已解决。

当前 CLI 耗时为 Ubuntu **578.061s**、macOS **172.598s**、Windows **405.302s**。只是耗时观测，不是时间上界，不推定并行争用根因，不关闭 M-16 或其他维护事项。

常规 CI 不运行本轮证明、宿主算法、链接判定或新的对照，不证明产物目标 absent、能力不可达或方案可以执行。

## 9. 下一步与交接

| 层次 | 裁定／最小下一步 |
|---|---|
| 本文档入库 | 限定技术 GO，建议纯文档合并；保留第 5 节两项精化及所有未成立结论，合并由负责人决定 |
| H-1 | 可以继续受控文档定稿；先收窄 Xsignal 推理，纠正 L6 的来源论据，固定有限集合及总结论／对照；不代表 H-1 已采纳 |
| 原 (a) | 整体仍无执行条件。技术补充与产品／保证范围选择分开，不以更弱 absent 定义替代原命题 |
| 第三批实现 | 仍须负责人决定依赖例外与 X/E/U/CGO 等，并完成相应规范同步和实施授权；设计／审计文档 GO 不代替这些决定 |
| 验证执行 | 输入、工具、对照、驱动、场所与预算另行固定、审查和授权；本轮不建议为了填一个证明空洞直接增加构建次数 |

交接提示词：

> Review 133 对 PR #51 的有限证明准备记录限定技术 GO；没有产物 absent/present 证据。按报告保留启动／回收 17 个目标与丙组的条件性基础，Xsignal 仍未证明 absent。先修正 N-133-1 的反向蕴含和 N-133-2 的内容合并来源论据，不能据它们判定目标存在或不存在。H-1 若继续，只做另获授权的文档定稿：固定逐目标集合／见证、R1～R6、T 优先的分类与新增不确定总结论、对照和 A/B 取证取舍；F-5 与信号能力处理交负责人单独决定。保持原生产构建及原强命题，原方法继续停止，不自行运行 Go、构建、链接、分析器、驱动、实验或 CI；不批准例外、新 L、第三批实现、资源或验收。

## 10. 验证限制与结束身份

独立执行范围为：Git／工作区／归档／差异核验，固定官方源码读取与文本检索，23 份模块目录哈希，指定历史会话工具回执读取，三个已有 CI 运行的正式元数据与二十四份原始 job 日志核对。没有提交或执行作者的临时宿主脚本，没有实现验证分析器，没有任何本轮原生、接缝或模型运行证据。

本轮没有实际生产 A、实际 go list、内联诊断、链接或 ELF／DWARF 命中、profile 条件、runner 清理或控制程序取证。大集合精确计数、会话整体完整性及未来构建环境不作独立已验证声明。首次诊断与旧历史未知、factgen/legality、已删除 worktree 待决、维护事项与冻结限制保持原范围。

**结束复核：2026-11-05 22:07 +09:00。** 本地 HEAD、tree、分支及 origin 与开始一致；远程分支为 `ef36fb8…`，master 与 PR base 为 `9c58606…`，PR HEAD 为固定 `ef36fb8…`，仍 open／未合并，合并快照仍为 `db40735…`。跟踪文件与索引干净，工作区仅有 `?? docs/reviews/codex-review-133.md`。没有其他会话改动或身份漂移。

本轮原仓库只新增本报告，未修改受审文档或实现，不提交、推送、合并，不授权第三批／验证执行，不宣布 WP2.5／M2 验收。报告后暂停。
