# Review 132：PR #50 链接方法静态审计及方法提案 M-1～M-5

## 1. 分层结论与固定对象

**审计报告：限定技术 GO，建议作为“记录原方法停止、保留未采纳提案”的纯文档合并。** 原方法停止是正确的审计结果，不构成报告本身的 NO-GO。本轮没有发现需要阻止这类审计文档入库的 BLOCKER；第 5 节列出两项非阻塞事实精化。它们在未来方法定稿前应更正，不能继续用作技术依据。

**原方法：继续停止，不可执行。** S-6 存在具体的输出采集路径错误；S-4 的强完备性前提未获证明。不能用当前审计、常规 CI 或提案获批代替这两个技术条件。

**调整后的方法：尚未成立，M-1～M-5 当前不能作为可执行方案整体采纳。** 尤其方法 M-2 得到的是更弱的表示／归属范围结论，尚不足以支持原 Q-B3-1(a) 所要求的“目标代码不在生产产物中”。负责人决定不能补成技术证明；若选择降低证据要求，须明确另作保证／风险决定，不能保留原强结论而只改 absent 的定义。

本报告中的“方法 M-1～M-5”专指审计文档 §9 的提案，与仓库既有同号维护事项不同。没有据编号重名增加实现阻塞。

**开始核验：2026-11-05 11:13:10 +09:00。** 本地、远程分支和 PR 身份一致，工作区及索引干净，PR open／未合并，origin 的 fetch/push URL 均为 `https://github.com/XKHoshizora/specmetry.git`。

| 项目 | 固定并独立核验的值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #50](https://github.com/XKHoshizora/specmetry/pull/50) |
| 当前分支 | `docs/wp2.5-link-method-audit` |
| HEAD | `6dce1a1a94d9633d1cdc1ae9cc2d1a757d476836` |
| tree | `2ec158016e12b7f5c6f021f18046060c62cef477` |
| 远程 master／PR base | `30bc8aa951cf16b8fef2614e56ff902b5d47c475` |
| 当前常规 CI | `41898301526`，pull_request，attempt 1，completed/success，8/8 |
| 实际合并快照 | `9841dac47407271e64bc56b0e7159b4d25db6dc3` |

本轮没有运行 Go、构建、链接、驱动、验证分析器或实验；仅用源码读取、文本检索、Git、只读哈希及 GitHub 只读接口。没有实施或批准方法提案。

## 2. PR #49 收尾、授权与本 PR 范围

已读取当前 AGENTS.md、完整 Review 131、§168／169、六文件完整增量、审计全文、链接计划及必要的冻结／实施规则。审查以固定 Go 源码和现有代码为准，没有因审计结果而重开已关闭的收尾协议。

PR #49 的合并及条件授权核验成立：

- [PR #49](https://github.com/XKHoshizora/specmetry/pull/49) 已合并，最终 HEAD 为 `8672144fc4d3473abe61a0b0e4d843c7cb7b7240`；merge commit 是本轮 base `30bc8aa951cf16b8fef2614e56ff902b5d47c475`。
- merge 的父提交依次为 `53fe5195fe0a4efe0e7b974f43d990e6e9c09ad8`、`8672144fc4d3473abe61a0b0e4d843c7cb7b7240`，tree 为 `363ca392b0f7d695b66f76e8a1def37811ff4143`，与该 PR 最终 HEAD 相同。
- Review 131 受审对象 `b26b15d` 包含在合并结果中；归档与收尾提交也在其历史中。归档文件为 19,348 字节，SHA-256 `3f5fc57fb86a7b2e5b5207946bc29c1fa29d5f7fddba58ee67b1e4b3c8285a0b`，工作文件与 `f76d5e8` 中 blob 同为 `7defab4fc654b9699ad30c0318c7ffc087d04748`。
- [master run 41889899481](https://github.com/XKHoshizora/specmetry/actions/runs/41889899481) 为上述合并提交的 push、attempt 1、success，8/8；读取八份日志确认实际检出合并提交本身，详见第 9 节。§168 所列只读审计准备的条件已生效；不是第三批实现、链接验证执行或依赖例外授权。

本 PR 只有一个提交 `6dce1a1`，共六份文档，**471 行增加、13 行删除**：AGENTS.md 2／2、implementation-plan 1／1、review-resolution 56／0、新审计文档 400／0、link-proof-plan 10／8、wp2.5-implementation-plan 2／2。§169 仅在旧文末追加；原审查文件、旧处理记录未改。链接计划只加结果引用／状态，已审的 §8.3／8.4 协议未改。

没有代码、测试、原型、依赖、Schema、迁移、workflow、冻结正文或验收项变化。`git diff --check` 通过。PR #49 收尾及本 PR 都没有把提案登记为已采纳。

## 3. 固定源码与模块身份

### 3.1 Go 源码取证

独立读取官方 Git 数据，`go1.27.3` 指向提交 **`2aae1000ea1d4b28923b96db28acf4e3b99d6aff`**，其 tree 为 **`09275a3482ffe831661a72c202c6261e0eb6ae7f`**。后面的源码判断都锁定该提交，不依赖可变 master 或本机 Go 可执行文件。

为核对 S-2，本轮按固定树读取 `src/cmd/link/internal` 各目录的未截断文件清单，并读取 **89 个非测试 Go 文件**；独立文本盘点三类调用、核对 loader 的实际置位入口，再检查 K 相关的条件与阶段。另读取了编译器的闭包／ABI／DWARF／SSA 路径，以及 cmd/go 的工具输出、缓存与归档路径。没有运行这些源码，也没有编写或执行验证分析器。

关键官方来源包括：

- [loader 的可达位与文本顺序](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/loader/loader.go)、[SymbolBuilder](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/loader/symbolbuilder.go)、[deadcode](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/deadcode.go)。
- [ELF 符号写出与 ABI 名字](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/symtab.go)、[FIPS 标记](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/fips140.go)。
- [内联 PC 范围的收集](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/dwarfgen/dwinl.go)、[空内联 DIE 的省略](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/dwarf/dwarf.go)、[SSA CSE](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/ssa/cse.go)。
- [cmd/go shell 输出处理](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/work/shell.go)、[buildid／缓存重放](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/work/buildid.go)、[link 动作](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/work/exec.go)。

### 3.2 独立 dirhash

本轮没有运行 `go env` 或 `go mod verify`。直接读取已存在的三个模块目录，以 PowerShell／.NET 的文件读取和 SHA-256 独立重算：

1. 枚举目录内文件，名字统一为 `module@version/相对路径`，使用 `/`。
2. 按名字的 ordinal 顺序排列；读取原始字节，不做换行转换。
3. 每文件形成 `小写十六进制 SHA256 + 两个空格 + 名字 + LF`；拒绝含 LF 的文件名。
4. 对整个 UTF-8 摘要再次 SHA-256、Base64，加 `h1:`。

算法与固定 Go 中 vendored x/mod 的 [dirhash.Hash1／HashDir](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/vendor/golang.org/x/mod/sumdb/dirhash/hash.go) 相符，未调用其 Go 实现。

| 模块 | 本轮读取文件数 | 独立重算结果；与 go.sum 及审计记录相符 |
|---|---:|---|
| modernc.org/libc v1.77.1 | 4199 | `h1:Ct8j47QtiZ1Enj2DtFXQtUqrPCAjdCmPjtCuvrYQ0Hs=` |
| modernc.org/sqlite v1.60.1 | 1899 | `h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw=` |
| golang.org/x/sys v0.49.0 | 555 | `h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=` |

这确认本轮所读目录的文件名／内容与记录一致，不认证上游原产，不涵盖文件权限／时间、运行时环境或其他模块。它也不等于实际生产闭包或 go list 的证据。

独立核对了 T0 用到的 `libc_musl.go`、`ccgo_linux_amd64.go`、`syscall_musl.go`、ABI0 汇编及相应声明、sqlite 的 Open／newConn 定义。它们支持所列目标的能力及包装来源，不证明 SQLite 已经调用这些目标。§1.2 的全部文件总数仍按作者的静态枚举理解；本轮未用实际 go list 验证任何选入集合。构建标签／文件名对 T0 的选入与排除可以静态核查，不能称为已观测的构建结果。

## 4. S 项逐项裁定

### 4.1 S-2：盘点与 K 上的例外

**可采纳，限固定源码、K 和列明前提。** 独立盘点结果与报告一致：

| 入口 | 非测试调用数，不含 loader 自身定义 | 说明 |
|---|---:|---|
| SetAttrReachable(..., true) | 29 | 其中一处是 deadcode.go:362 的 mark |
| SymbolBuilder.SetReachable(true) | 20 | 最终调用 loader 的置位方法 |
| CreateSymForUpdate | 77 | symbolbuilder.go:57～60 查找／创建并置位 |
| 合计 | 126 | 扣除 mark 本身后 **125 处，19 文件** |

附录 A 的文件／行号／种类与盘点相符。loader.go:900～905 的位写入与两个辅助入口相连；“27 处”不是完整盘点，这一更正成立。

K 上的分类也有明确依据：其他架构／格式、动态链接／宿主对象、非 ELF build ID、分段与跳板分支不能无条件带入 K；DWARF、pclntab、moduledata、TLS 和数据标记不因“被置为可达”变成用户函数文本。amd64 的该构建不走所列跳板路径。

不经 mark 的相关文本清单为 `runtime.text`、`runtime.etext`、`go:textfipsstart`、`go:textfipsend`，以及条件性的 `runtime.mapinitnoop`。最后一项是 runtime 包的 Go 函数，不是合成的 libc 函数；mapinitcleanup:338～353 会将弱 map.init 调用改向它并直接置位。FIPS 两个文本标记在 loadfips 中直接加入 Textp；不能把它们当作未知包代码。

对 libc 的**独立文本符号**，限定 K 下的 mark／to 关系可采；不能扩展成“每份内联进调用者的代码也必有一个 F 的 to”。输出写入错误不由 fmt.Printf 的返回值自动使链接失败，交叉核对仅能支持其实际覆盖的文本范围，不能宣称发现任意数据行截断。

### 4.2 S-3：T0、派生与 ABI

**主要映射可采，限 T0；第 5 节的 ABI 内联说法需精化。** 固定源码支持：闭包 `.funcN`、嵌套闭包数字后缀、`-rangeN`、`.gowrapN`、`.deferwrapN`；内联产生闭包时取原函数名／包，临时 hash 在对象写出时去掉，内容寻址文本可同名。

ABI wrapper 与原函数使用同一名字、不同 ABI；ELF 由 mangleABIName 添加 `.abi0`，dumpdep 只打印名字而不打印版本。审计把两者区分是必要的。F／Y 的 ABI0 汇编调用和 Go 声明经独立读取相符。

T0 是这些具体非泛型函数及包装；它们没有自己的类型参数，故可以不为本次映射审计泛型目标实例。**这不表示整个产物的名字没有泛型语法**，也不表示解析全量 dumpdep／ELF 时可据此忽略泛型名字；方法 M-3 的全量语法仍要明确。

`Xexecv`／`Xexecve` 等前缀必须有分隔符；未知后缀、缺失主定义、无法确定的包装或映射仍为 T。不能因为在 libc 自身归档没找到闭包，就忽略其他包对象中由内联形成的同名／内容寻址闭包。

### 4.3 S-4：限定形式成立，强形式未证明

**停止所需的理由成立；没有证明 T0 已发生遗漏。** dwinl.go 依指令的内联索引登记调用和祖先，建立非零 PC 范围；internal/dwarf 的 isEmptyInlinedCall 会省略自身与全部后代都无范围的实例。这支持报告的限定形式。

强形式还需要证明：与 F 有关的残留机器指令必保有 F／其子树的位置信息。现有收集／写出代码本身不提供这个不变量。另读固定 SSA CSE:269～303 可见等价值替换只保留代表值，不能从 DWARF 收集器倒推出所有来源均被保留。

必须区分三件事：

- 没有指令残留的空实例，是源码明确允许省略的情况。
- 归属经优化变化，是强完备性尚未排除的条件；本轮没有新建／运行具体反例。
- T0 各函数在固定 K 上是否会经历这种变化，尚未逐目标证明；不能从一般优化机制推出 T0 必然漏记。

同样，“函数包含不可消除的调用”只讨论效果是否保留，不自动证明调用指令仍带 F 的内联索引。报告 §4.3 将这件事留作提案而没有宣称目标已经通过，是正确的；不能在后续采纳时跳过这项证明。

### 4.4 S-6：具体采集错误成立，缓存与归档结论可采

固定源码可沿完整链核对：

1. deadcode.mark 将图写到**链接器 stdout**。
2. cmd/go 的 shell.runOut:630～631 将工具 stdout／stderr 指向同一缓冲；两来源不再保持独立边界。
3. reportCmd:477～525 补末尾 LF、替换 WORK 前缀、加导入路径标题，保存到 action.output。
4. link 动作 defer flushOutput；默认 TextPrinter 指向 **go build 的 stderr**。因此原计划采集 go build stdout 的文件不能作为 dumpdep 证据，预期为空。这是固定调用链的具体反例，不只是“尚未证明”。

缓存方面：现有二进制 build ID 可使链接直接跳过；主包的特殊缓存路径可重放编译／链接输出；showStdout 只在有内容时打印读缓存文件的 cat 痕迹。是否有链接命令必须单独核对，非空输出不能证明本次链接发生。新缓存／新输出是防跳过的输入条件，也要取得实际执行记录，不能只看目录名。

归档方面：编译命令的 `-p`／`-o`、成功缓存时的 cp 行及缓存文件内容哈希可以关联该次归档。缓存写入失败可能没有 cp 行，不一定使 go build 失败；必需包缺少定位证据应为 T。目录和 hash 是关联／追溯，不认证来源；A、B 分别使用的缓存与归档不能混配。

“整段缓存在内存中”只说明这段 cmd/go 代码没有主动截断，不证明所有外层输出已完整落盘。读取、写入、关闭、采集和身份错误仍要持续进入 T。

### 4.5 S-7：可达文本条目限定成立

**可采，限 K。** AssignTextSymbolOrder 从可达外部文本与 Go 对象中的可达文本建立 Textp；dupok 分配去重。genelfsym:195～223 为 runtime.text、Textp 每项、runtime.etext 调用 putelfsym；文本路径不使用数据路径的 NotInSymbolTable 过滤。putelfsym 在 LOCAL／GLOBAL 两轮中选择一个绑定写出；缺段是链接错误。

K 下不适用的文本插入／外部对象／动态链接分支不能混入结论；FIPS 标记包含在限定清单中。可采的是每个链接器符号对应的 STT_FUNC 条目，不是名字的一一对应。不同条目可能同名，符号表存在也不能直接证明代码会执行。

### 4.6 S-1／S-5

S-5 可保持：flagDumpDep 在固定链接器非测试源码中只在定义和 mark 中使用，改变打印，不改变可达性算法；不把这一点扩大成 A/B 的字节或时序相同。

S-1 的更正成立：类型符号清除后可以再次 mark；同名 ABI／内容寻址符号不等于同一个符号。UsedInIface 和 ReflectMethod 后缀可能组合出现，并可能在 from、to 两端出现，须保留原始记录再解析。三类根可从入口、unreachableMethod、mainInittasks 读出；它们是预期根，不是“总输出恰好三行”的断言。

## 5. 非阻塞事实精化

### N-132-1：ABI wrapper 内联被包装体的说法

审计文档 **133** 行称“包装可能把被包装函数内联进去”，将 F 代码在 `F.abi0` 中作为例子。对本轮锁定工具链，不能仅凭 ABI wrapper 的生成与命名源码作此推断。

固定 [gc/main.go:257～281](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/gc/main.go) 先运行去虚化／内联，之后才 GenABIWrappers；其注释也明确该阶段不再受益于内联。makeABIWrapper:335～347 生成对 F 的调用／尾调用，没有在该路径再次调用内联器。应撤回该例子，或另给固定源码中实际再次变换的依据，不能当成已证事实。

这不否定同名不同 ABI、ELF 名字改写及 T0 包装映射；也不影响当前必须停止的结论。**按非阻塞事实精化登记**，在后续方法定稿／使用该例子之前更正。没有据此声称已经运行了 C6 或观察到 F 被内联到包装中。

### N-132-2：C5 的效果与归属

审计文档 **301** 行的“调用 noinline 函数，使至少一条指令归属于内联实例”应写成拟达到的构造条件，而非仅由有副作用调用就保证的结论。独立的 inlining 诊断加 **A 本身的内联 DIE／非空范围命中**仍是所需证据；缺任一项应 T(control)。

当前表已有 A 的实际 DIE 命中条件，未命中不得计覆盖，因此这项措辞没有形成当前可执行结果的假通过。宜在定稿时把 S 依据、构造预测及 R 命中分开，并避免反过来把几个成功对照当成全部目标的归属完备性证明。

两项精化均不自动升级为审计文档合并 BLOCKER；它们也没有被当作已接受的风险或 L。方法若要继续使用相关推断，必须先完成修订与复审。

## 6. 方法提案 M-1～M-5 的技术评价

### 6.1 方法 M-1：方向成立，提取协议还需定稿

采集 go build stderr 的方向有源码依据，保留 A 为主证据也合理。现稿不足以直接实现一个唯一、完整的段落提取器，至少须固定：

- 链接命令、buildid 命令、唯一的正确导入路径标题及后续安装／清理命令的识别规则；路径的引用、WORK 还原与 A/B 关联。
- 载荷状态中，只有严格符合图行及已知标志语法的内容才能进入段落；警告、第二个标题、未知行、畸形箭头或无法区分名字边界，一律 T，不吞掉或当普通结束。
- “第一条非图行”只有在它是该次动作预期的、已识别的后续命令时才可结束；不能泛化为“像 shell 命令就结束”。结束后若再次出现图载荷／重复标题或未知内容，仍须拒绝。
- `$WORK` 被替换进名字、标题重复、错误顺序、缺失／截断／非法末尾等各有固定预期。保留完整原始 stderr，提取段落只是派生证据，不替代原始字节与命令状态。

合并的工具输出没有可恢复的 stdout／stderr 来源标记。M-1 不能宣称恢复链接器的原始 stdout；也不能凭语法排除所有可伪装成图行的任意输出。受信任固定工具的限定、未知内容为 T，以及 A/B 交叉核对的实际范围必须保持。

**裁定：技术方向可继续文档定稿；当前不是已完成／已验证的取证协议。** toolexec 另引工具和控制责任，不是本轮自动获批的替代。

### 6.2 方法 M-2：不能直接支持原方案 (a)

需要严格区分三个层次：

| 层次 | 现有方法可能支持的结论 |
|---|---|
| 符号／包装／内联表示 | 完整解析及映射成立时，没有发现相应表示；在 S-4 的限定形式下，可进一步约束带 F 归属的非空 PC 范围 |
| F 的相关代码不存在 | 还需要“所有残留代码必保留这些表示／归属”的完备性；当前未证明 |
| 启动、回收、信号能力不可达 | 还涉及目标集合完整性、变量号原始入口、其他位置及其他到达方式；当前方法明确不证明 |

原 Q-B3-1(a) 的产物例外依赖第二层，不能用第一层替代。第三层也不能从第二层的有限 T0 结果自动推出。

方法 M-2 所写“含不可消除／不可合并的调用”只是待证明的技术条件。必须逐目标证明在固定编译流水线中，该效果保留**且**至少有残留指令保有 F 子树的位置，并覆盖包装、跨包内联、常量分支和其他变换。单看调用函数名、函数有副作用或某次健康对照成功，不足以建立这个不变量。当前没有这份逐目标证明。

一个尚未排除的反模型是：F 的独立符号不可达，相关操作已进入调用者，而记录没有 F 的归属；表示检查给弱 absent，但相关操作仍在。**这是当前前提未排除的逻辑情况，不是已动态复现或已证明发生在 T0 的反例。** 它足以说明弱结论不能直接恢复原强保证，不足以声称 T0 必然失败。

`X__syscall0～6`、`___syscall_cp`、`Xsyscall` 接收系统调用号；有限 T0 absent 不排除从其他位置传入同样常量或变量。ABI Y 包装／回调／函数表也须纳入映射。F-5 目标完整性未解决，不能靠更改 absent 名字绕过。

**裁定：当前方法 M-2 只提供更弱的观察／归属结论，不能单独作为原 Q-B3-1(a) 的证据路径。** 文档 289、343、363 行涉及“等待 M-2 的决定”时，后续应同时要求技术证明或替代方法复审；不是负责人选择该提案后原技术条件就成立。

若继续原 (a)，应先补足上述强完备性，或另提能证明同等命题的方法。若负责人确实希望接受更弱的证据，它是新的保证／风险取舍，需要明确修改例外条件及规范／验收，并记录未覆盖部分；不能仍声称满足原 (a)。本报告不批准这一变化。

### 6.3 方法 M-3：必须保留多重性与原始身份

规范化方向必要，但单纯比较名字集合不充分。静态例子：A 有 F 与 `F.abi0` 两条，B 有两个同名 F 的 to；删掉一个 to 后，规范化名字集合仍都是 `{F}`。现 C9 删掉该名字的全部行能检出整组缺失，不证明部分同名记录丢失能检出。

执行前至少要固定：

- 保留原始 ELF 名字、条目索引、绑定／段／地址／大小，以及原始图行序号、from／to／标志和重复次数；规范化不是覆盖原始记录。
- `.abi0` 只在确认来自 mangleABIName 的规则下还原，不是任意字符串截后缀；源码判断涉及 loader 中同名 ABIInternal 文本，不应未经说明等同于“当前 A 中必有同名条目”。
- 两个标志及组合只按完整末尾语法解析；from 与 to 都处理，不做全局随意替换。
- 对 ABI 同名对、内容寻址闭包和类型重新 mark，规定多重集合／允许次数及可确定的分组。无法唯一关联时给出保守 T 或明确只作组级核对，不能冒称逐条一一对应或完整调用路径。
- A/B 对应和 C9 的删除规则按同一粒度固定；保留“最后一行变体未命中不算覆盖”的原限定。

**裁定：需补齐解析／身份和多重性规则，当前不能直接采纳为完整性算法。** 不要求本轮实现通用 Mach-O／ELF 工具或修改 Go 链接器输出。

### 6.4 方法 M-4：固定 K 上可支持

五项清单与 S-2 相符，runtime.mapinitnoop 是允许出现但非必然出现的条件性包函数；“无条件列入允许清单”不等于要求它实际存在。清单不能按前缀扩成 `runtime.*` 或任意合成文本，也不能忽略其他未作为 to 出现的包代码文本。

**裁定：固定 K 下有技术依据，可供负责人决定；不是其他配置的通用例外。** K、链接模式、工具版本或源码变化须重新核对。

### 6.5 方法 M-5：可支持有限的重放拒绝

showStdout:635～641 的 `cat <缓存文件> # internal` 确实是非空缓存输出重放的痕迹，可作为 T(identity) 的充分拒绝条件。其反面不成立：没有 cat 行可能因为没有输出，不证明发生真实链接。

普通 `cat > importcfg.link << EOF # internal` 是写入配置的展示，不是读取缓存重放；该配置正文还会包含 GOCACHE 路径。匹配器须解析命令方向／参数和源文件，不能只查 `cat`、`# internal` 与 GOCACHE 子串。

仍要分别检查新缓存／新输出身份、实际 link 工具调用及完整参数、产物和归档 hash、命令状态。C10 只有真正跳过链接才算命中，实际又链接则 T(control)，不能改变期望使其通过。

**裁定：收紧方向有依据，精确匹配与控制前提须先写定；没有单凭 M-5 得到“本次真实链接已证明”。**

## 7. 输入与 C1～C10

T0 的定义位置和 Y 包装可以由已核对的模块源码追溯；`Xfork` 返回 ENOSYS、`Xpopen` 是未实现占位，与实际有能力的入口分开列是合理的。通用系统调用入口必须继续单列，不借文本枚举改成完备例外清单。

| 对照 | 本轮评价及证据边界 |
|---|---|
| C1／C2 | noinline 的直接／初始化路径可形成健康正对照；实际归档、to 及来源命中仍需运行 |
| C3 | 函数值／回调构造须实际保留并被调用；声明有全局表本身不保证任意优化下同样形态 |
| C4 | 非常量 MethodByName、ReflectMethod 与实际到达是命中前提；不是所有同名反射写法都等价 |
| C5 | 保留第 5 节精化；内联诊断不代替同次 A 的 DIE／非空范围，未命中为 T |
| C6 | ABI0／ABIInternal 对照方向成立；四个实际条目和汇编 CALL 必须核验，不能只由名字规则自算预期 |
| C7／C8 | 构造中“无其他引用”及归档确有函数须成立；它们只验证该健康构造的 absent，不修复全部 T0 的内联完备性 |
| C9 | 作用于 M-1 的派生段落依赖尚未批准的方法修订；整组删除与部分同名丢失不同，不能扩大覆盖 |
| C10 | 必须实际命中跳过／重放前提；无命中仍 T(control)，不计为缓存拒绝已验证 |

四个真实正对照的定义和仓库使用路径可追溯；主符号在归档、最终以何种表示存在、是否真正 present 均仍是 R 证据。几个正对照不能替代全目标／全条目完备性论证。

表中引用 M-1／M-3／M-5 的对照应在方法决定之后同步回正式计划；当前只是提案依赖，不是已批准的新预期。原生构建、接缝改写、解析模型、静态预测分别计证据。未命中、未知映射、工具失败或清理未知继续为 T，预期拒绝不能豁免其他错误。

## 8. 一次 go env 越界的影响

审计 §0.3 和处理记录 §169:10792 明确承认运行过一次 `go env GOMODCACHE`，违反“不运行 Go”的边界。**这是已登记的越界，不因查询用途或“未采证”而补成授权。**

现有仓库记录提供了命令、声称的版本、用途及后续停止说明，但没有该次原始命令回执、实际可执行文件路径／哈希、时间、退出状态、原始 stdout／stderr 或启动器记录。本轮不能独立确认“只一次”、真实版本／shim 行为、没有下载／其他副作用等全部细节；“没有构建、链接或下载”仍是作者陈述。本轮没有为复核再运行 Go。

其输出虽然没有进入 S 结论，**用于发现输入目录**，因此不能说对过程完全没有影响。独立读取默认位置后，本轮按固定模块前缀重算三份 dirhash，与记录及 go.sum 一致，使已读模块内容的身份不依赖信任那次查询结果；固定 Go 官方提交的源码也单独核验。这些可独立支持的证据不必一并作废。

影响边界：保留违规事实、不能将过程描述成零 Go 运行；不采纳那次查询作为工具链／运行环境或实际构建证据。若负责人要求流程收尾，可补充已有可取得的原始回执或明确记录缺失，不能事后重造或通过再运行来填补。本轮不为这个未影响已独立核验源码／哈希的发现步骤新增实验或永久合并门槛。

## 9. 两套 CI 的独立证据

### 9.1 PR #50

[run 41898301526](https://github.com/XKHoshizora/specmetry/actions/runs/41898301526)：pull_request，attempt 1，completed/success，8/8。八份日志都实际 fetch／checkout 并以 git log 显示 `9841dac47407271e64bc56b0e7159b4d25db6dc3`。

该快照父提交依次为固定 base、固定 HEAD；tree 为 `2ec158016e12b7f5c6f021f18046060c62cef477`，与 HEAD 同树，没有只用 job head_sha 推定检出。

| 作业 | job ID | 正式结果／日志范围 |
|---|---|---|
| Go lint | 126813624011 | success；tidy、vet、文档链接与 lint |
| Go tests Windows | 126813624348 | success；27 个 ok 包，10 SKIP |
| Web | 126813624399 | success；格式、lint、类型、测试、构建 |
| Go tests macOS | 126813624489 | success；27 个 ok 包，7 SKIP |
| Go tests Ubuntu | 126813624620 | success；27 个 ok 包，15 SKIP；普通步骤 race；独立 root CrossUser 8 PASS |
| Build Windows | 126814221542 | success；默认嵌入构建、测试与 smoke |
| Build macOS | 126814221558 | success；同上 |
| Build Ubuntu | 126814221675 | success；同上 |

### 9.2 PR #49 合并后的 master

[run 41889899481](https://github.com/XKHoshizora/specmetry/actions/runs/41889899481)：push，attempt 1，completed/success，8/8。八份日志实际检出合并提交 `30bc8aa951cf16b8fef2614e56ff902b5d47c475` 本身。

| 作业 | job ID | 结果 |
|---|---|---|
| Go tests Ubuntu | 126784198531 | success；27 包、15 SKIP；普通 race，独立 root CrossUser 8 PASS |
| Web | 126784198877 | success |
| Go tests Windows | 126784198989 | success；27 包、10 SKIP |
| Go lint | 126784199238 | success |
| Go tests macOS | 126784199250 | success；27 包、7 SKIP |
| Build Ubuntu | 126784760117 | success |
| Build macOS | 126784760159 | success |
| Build Windows | 126784760242 | success |

两套日志没有测试 FAIL／job error；Go 为 go1.27.3，Ubuntu／Windows amd64、macOS arm64。SKIP 集合逐名比较相同：共同的五项真实 fetch／旧 Git 对照和两项 M14；Ubuntu 普通步骤另跳过八项 CrossUser，随后 root 步骤独立通过；Windows 另跳过三个 unreadable-storage/source 用例。macOS／Windows 的 Linux root 专用步骤正式为 skipped，不计入上述测试 SKIP 条数或通过数。SKIP 不记为通过，也不把 root 证据计成普通 race 断言。

PR #50 的 CLI 耗时为 Ubuntu 563.093s、macOS 124.537s、Windows 426.930s；master 为 447.579s、287.023s、405.086s。仅为观测，不判根因、不提供上界、不关闭 M-16 或其他维护事项、不调整预算。

常规 CI 只验证已存在的实现和文档，不验证本审计推演、M 提案、链接存在／不存在、驱动或新对照。历史运行、取消、失败和此前审查结论保留，没有本轮新机制诊断或重跑。

## 10. 最小下一步、待决与交接

| 层次 | 本轮裁定／下一步 |
|---|---|
| 审计文档入库 | 限定技术 GO，建议纯文档合并；保留 N-132-1／2 事实精化与 M 提案未采纳状态，交负责人决定 |
| 原执行路径 | 停止正确，仍不可执行；不能因报告通过而重启 |
| 继续原 Q-B3-1(a) | 先保持“目标代码不存在”的原命题，补逐目标表示／归属完备性或另提同等强度方法，再独立复审；F-5 完整性仍是独立条件 |
| 接受弱观察路径 | 若负责人希望选择，应另列保证／风险变化及规范／验收影响；不能作为原 (a) 自动兑现，本报告不批准 |
| 方法解析定稿 | 固定 M-1 边界和失败规则、M-3 身份／多重性、M-4 的 K 清单、M-5 的真实链接与重放区分；修正两项非阻塞源码／对照措辞 |
| 执行前 | 固定目标／包装／正对照、命令、SHA／树、前端／工具／模块／产物身份；取得实际环境前提；驱动实现和自检经授权／复审；场所、workflow 与资源分别决定 |

若现有强完备性无法给出，最小安全下一步是**继续停下执行、只做方法文档修订／证明准备**，而不是增加几次健康构建来代替证明。保持原生产构建形态；不能用禁用内联等改变构建的结果替代原产物 A。可以在另行授权后取得最小动态命中证据，但它应检验已固定的假设，不能把未命中或工具失败改写成方法通过。

可以独立交负责人处理的仍包括 Q-B3-1(b)/(c)、生产 CGO、X/E/U、F-7／F-8 等；其是否采纳及是否改冻结规则须明确。场所、资源、实现和执行均未由本报告批准。已关闭收尾协议没有因本轮结果重开；第三批及后续批次、依赖例外、新 L、WP2.5／M2 验收保持原边界。

交接提示词：

> Review 132 认可 PR #50 作为记录停止及未采纳提案的静态审计文档；原方法继续停止。先修正 ABI wrapper 内联例子与 C5 的证据措辞，固定 M-1 的完整边界、M-3 的多重性／身份及 M-5 的重放识别。M-2 当前只给更弱的表示／归属结果，不能因批准提案就视为满足原 Q-B3-1(a)；若继续原保证，须给出逐目标完备性或另提同等强度方法；若希望接受弱保证，另列负责人决定和冻结／验收影响。保持现有收尾、预算与授权；只做另获授权的文档／源码证明准备，不自行运行 Go、构建、链接、驱动、试验或 CI。

## 11. 证据限制与结束核验

本轮独立证据：固定官方源码读取及文本盘点、模块原始字节的只读哈希、Git 祖先／归档／差异核验、PR／提交／run／job 元数据及十六份已有原始日志。没有运行本机 Go 或任何构建／链接／验证程序，没有原生、接缝或新模型执行证据。

未独立恢复作者那次 go env 的原始回执，未验证实际 go list、产物 ELF／DWARF、归档／缓存重放、对照命中和 runner 前提。本轮只采纳它们所对应的静态结论和明确边界，不把作者陈述或未来计划升级成已验证事实。

**结束核验：2026-11-05 11:24:20 +09:00。** 本地 HEAD、tree、分支与 origin 未变；远程分支、远程 master、PR HEAD／base 均与固定对象及开始核验一致。PR 仍 open／未合并，合并快照仍为 `9841dac47407271e64bc56b0e7159b4d25db6dc3`。跟踪文件与索引干净，工作区仅有 `?? docs/reviews/codex-review-132.md`。本轮原仓库仅新增这份未提交报告，没有覆盖其他会话内容，报告后暂停。
