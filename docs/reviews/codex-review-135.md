# Review 135：PR #52 H-1 方法候选稿第二稿定向复审

## 1. 分层结论

**固定文档技术 NO-GO，暂不建议合并。** B-134-1、B-134-2、B-134-4 在本次受审的文档规格范围内关闭；B-134-3 的来源域及依赖修订主体成立，但仍有一项残余 **B-134-3-R：ELF 变体与 R6 身份检查之间没有明确接缝**。另发现本次 R1 澄清引入的 **B-135-1：GOENV=off 的 JSON 预期与固定源码不符**。两项都有具体条文／源码反例，不是因为方法未实现、未采纳，或完整方案 (a) 尚未兑现而阻塞。

**N-134-1、N-134-2 的文档精化完成。** 无名 STT_FUNC 的政策已写定；相应 HC-6(iii) 的有效命中仍受 B-134-3-R 影响，不据文字预期认定对照已验证。运行时缓解与条件性风险的说明准确，没有接受新的 L 或证明 SQLite 实际触发相关能力。

H-1 保留继续定稿的有限技术基础，**未采纳、不可据此执行**；原链接方法继续停止。没有本轮实际产物 absent／present、归档解析或故障运行证据。(b) 仍为待负责人选择的保证弱化方案，不是技术必选项。

## 2. 固定身份、范围与归档

开始核验：**2026-11-07 13:59:50 +09:00**。本地 HEAD、分支、内容树及 origin 与固定对象相符；索引／工作区干净。随后独立查询远程分支、master 和 PR，均相符；PR open、未合并。

| 项目 | 核验值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #52](https://github.com/XKHoshizora/specmetry/pull/52) |
| 分支 | `docs/wp2.5-host-set-method-draft` |
| HEAD | `b10564bf44443f9ecaa661ef91c47be8bd0ee3d1` |
| tree | `08aa99a387387cffd0fb69d6f1a0d2a6b9714012` |
| master／PR base | `56d41b97e83abe85f59f4d7370f8effc617b8448` |
| 上轮对象 | `ca9884b616882ddcef1411de9d3f3a95a63d612e` |
| 常规 CI | `42225668432`，pull_request，attempt 1，completed/success，8/8 |
| 实际 checkout 快照 | `ae201df9423beffea6bd579b48d7dfc54376f23e` |

读取当前 AGENTS.md、完整 Review 134、候选稿第二稿全文、§174、本轮全部七文件增量及必要的证明准备／链接计划、冻结规则和固定源码。沿用 Review 133／134 的有限证明及既有收尾结论，没有重新审计无关机制。

相对 ca9884b，共 **674 行增加、143 行删除**：AGENTS.md 2/2、implementation-plan 1/1、review-resolution 76/0、Review 134 归档 283/0、候选稿 308/136、link-proof-plan 2/2、wp2.5-implementation-plan 2/2。候选稿之外是归档、追加记录和状态同步；链接计划技术正文没有采用 H-1。

- `9beb480284c108b170df483483b12cfed38e17e8` 只新增 Review 134；父提交为上轮对象。`b10564b` 在其后修订文档，没有夹带生产、测试、原型、workflow、工具链、依赖、Schema、迁移、冻结正文或验收项变化。
- Review 134 为 **32,650 字节**，SHA-256 **`8150bd116a317ea6e507fadebe1d75246754b0ec9ee748aee65d63eb6d11c8f5`**，与上轮报告及 §174 回执一致。现文件、HEAD blob 与归档提交 blob 均为 **`49532eda5ec2f1cb68e1eef70e3e2cb88147a6da`**；没有重做归档前暂存操作，也不冒称取得了该操作的新证据。
- 独立读取 Git blob 作字节前缀核对：旧处理记录 **1,124,832 字节**是现 **1,137,957 字节**的完整前缀，§174 只追加，旧 §173 未改。历史审查文件没有改写。
- `git diff --check ca9884b HEAD` 通过。§174 授权仅为定向文档修订与一次最终常规 CI；本报告不扩大该范围。

## 3. B-134-1：集合与归档见证——关闭

位置：[候选稿](../wp2.5-c9-host-set-method-draft.md) **69～130、243～244、260、371 行**。

第二稿把三层明确分开：H_search 仍是固定的保守检索集合，A 必须检索其全部成员；H_K 要求正确归档中的定义；排除成员要求实际文件／包选择与工具输入的正面证据。排除不再由 nm 未命中推得。排除名字异常出现在归档或 A 中，触发 M.excluded，不能静默忽略。

本轮独立读取 libc v1.77.1 的相关声明与构建条件：

- `ccgo_linux_amd64.go:3` 为 linux && amd64；386、arm 文件第 3 行分别为 linux && 386、linux && arm。
- `X__wait3_time32`、`X__wait4_time32` 分别在 386 文件 **153188／153215**、arm 文件 **152687／152714** 行定义；在 K 上不编入。这两个名字只作为宿主，没有进入 21 个目标的 T0。
- `libc_musl.go`、`builtin.go` 的 linux 条件包含 amd64；`libc_all.go` 无平台构建限制。`abi0_linux_amd64.s` 的 Ywait3／Ywait4 分别在 27634／27651 行定义，与对应的 K 成员见证一致。

四类静态推演成立：

| 情况 | 应有结果 |
|---|---|
| K 文件选择及归档定义正常，time32 正常排除 | 没有由正常排除产生的 T；继续按 A 分类 |
| K 应编入的 Xwait3 定义见证缺失 | M.witness，T(mapping) |
| 实际 GoFiles／工具输入与 K 清单不符 | P.R4.*，T(premise)；依赖此事实的结果不能有效 |
| 定义文件正面排除，归档与 A 都没有 time32 | 不要求该成员的归档定义；仍保留完整 H_search 检索 |

若排除成员出现，M.excluded 优先阻止正常目标结论。§3.4 的同步表也已撤回把 time32 加入 T0 的写法。**健康 K 不再因正常排除必然得到 T。** 这是文档闭合，不是本轮取得了实际 go list／归档见证。

## 4. B-134-2：nm 语法和成员身份——关闭

位置：候选稿 **152～167、342～396 行**。本轮重新读取官方固定源码；go1.27.3 tag 仍解析到 **`2aae1000ea1d4b28923b96db28acf4e3b99d6aff`**。

### 4.1 固定源码与受限范围

- [work/gc.go:491～536](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/work/gc.go#L491) 的 packInternal 取对象文件名的前 **16 字节**、不足补空格；[archive.go:282～319](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/archive/archive.go#L282) 去右侧空格。libc 三个可读对象成员为 `_go_.o`、`abi0_linux_amd64`、`tls_linux_amd64.`；hc 为 `_go_.o`、`hc_amd64.o`。截断后重名拒绝，不能按重复成员名猜归属。
- [gc/obj.go:63～72](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/compile/internal/gc/obj.go#L63) 写编译归档的 PKGDEF／Go 对象；[objfile/goobj.go:29～74](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/objfile/goobj.go#L29) 跳过 PKGDEF，保留对象成员身份。因此这里只把可读对象成员数用于 nm 前缀规则，不把 PKGDEF 也算作一份输出成员。
- [nm.go:139～162](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/nm/nm.go#L139)：多 Entry 总有 archive(member) 前缀；单 Entry、单文件参数没有前缀；U 地址栏为空白；非 U 使用 %8x；名字是余下文本，允许空格。
- %8x 本来是**最小宽度**。第二稿限定恰好八字符在这里有额外依据：K 的 Go 对象地址来自 [goobjFile.symbols:183](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/objfile/goobj.go#L181) 的 uint32 DataOff（[Reader.DataOff:822](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/goobj/objfile.go#L822)），不超过八个十六进制数字。不能把该限制推广到任意原生对象的地址。
- goobj.go:154～219 支持 T/R/D/B/?、静态的小写及 `_`、U；静态名字带 `<1>`。归档层 ABI0／ABIInternal 可同名，不能套用 ELF 的 `.abi0` 规范化。见证限定为**正确成员、T、逐字节基名**，U／数据／静态及其他成员不能替代。

### 4.2 十四个行模型的逐项裁定

| 模型 | 静态裁定 |
|---|---|
| 1 | 带正确 `_go_.o` 前缀的 T 与 Xexecve 基名相等，可作见证 |
| 2 | 同名第二份 T 合法，保留两份原始条目与多重性 |
| 3 | 八空格地址栏加分隔空格的 U 合法，不作见证 |
| 4 | 正确 ABI0 汇编成员中的 Yexecve T 合法，可作见证 |
| 5 | 小写 t 和 `<1>` 合法，但不当作该导出基名的定义见证 |
| 6 | 同名 R 行语法合法，不作函数定义见证 |
| 7 | 名字含空格，余下文本整体保留，不能按三个 token 截断 |
| 8 | 预期多成员却缺前缀，D.nm.line |
| 9 | 不在预期集合的成员，D.nm.member |
| 10 | 地址宽度／U 空白／非 U 数字／类型异常，D.nm.line |
| 11 | 部分 stdout 加退出 1，X.exit；输出非空不豁免 |
| 12 | 退出 0 但末行没有换行，D.nm.trunc |
| 13 | 行合法但缺少 H_K 的必要 T 定义，M.witness |
| 14 | time32 的 T 出现在 K 归档，与正面排除矛盾，M.excluded |

单成员无前缀、正确原始路径／成员与名字的区分也已写在语法及正对照中。上述是**受限行模型的人工源码推演**，没有运行 nm 或解析程序。

nm 的成员错误会设置非零而继续输出，Close／Flush 返回又未全部消费；第二稿没有再把退出 0 等同全部 I/O 成功。外层的写入／读取／关闭、末行与必要见证检查仍在。行边界截断且仅丢无关行的限制已披露；它不变成完整输出认证。没有据此要求修改 Go 工具。

## 5. B-134-3：来源域修订成立，ELF 变体仍有残余

### 5.1 可以采纳的主体修订

候选稿 **194～209、261～263、301～318 行**分开 X（工具与采集）、I（身份）、P（事实前提）、D（内存解析／结构）、M（映射）、C（对照）；R5a 在解析成功之后检查形态，R5b 检查结构。上游失败后的依赖检查为“未评估”，不伪造一次失败或命中；主类别取实际失败的最高优先域，全部原因和未评估项保存。

这已修正 Review 134 的“整个 R5 抢先落入 T(premise)”问题。有效基线、固定修改位置、恰好等于预期失败集合、额外真实错误使对照无效的要求合理，也不能被放宽为找到一个理由即通过。

HC-5 的固定意图可静态成立：加入编译 profile 参数命中 P.R1.compile；构建信息出现 pgo 命中 P.R1.buildinfo；删掉 GoFiles 中 amd64 文件时，静态清单和 x/list 对照分别失败。R1／R4 只依赖完整采集记录，第二个检查不应因为第一个失败便被错误跳过。健康基线首先受第 6 节的 GOENV 问题影响，不能把这些意图当作已运行命中。

### 5.2 B-134-3-R：变体改变身份，却要求只失败于读取检查

**位置：候选稿 199、206、263、299～315 行，尤其 303、310～312 行。**

允许的事件顺序如下，无需任何新实验：

1. 原生产输出 A 已核验有效，构建后及读取前哈希同为 h；基线通过。
2. HC-6(i) 复制 A，把 symtab 的 sh_size 加 1。按 303 行须独立确认修改，且副本哈希不同于 h。
3. 文中要求“然后评估”，仍使用身份 → 解析 → 形态 → 自检的依赖。R6①②没有针对**解析输入副本**规定独立接缝。
4. 若用原构建身份约束该副本，副本不是原 -o 产物，或它的读取哈希与构建哈希不同，先失败于 I.*。D.R5b.size 随之未评估，不能称命中；即使继续评估到 D，额外 I 也会使“失败集合恰好为 D.size”不成立。
5. HC-6(ii)、(iii) 改 st_name 同样改变副本字节身份。它们也不能按现文直接保证只失败于 D.name／D.unnamed。

另一种合理实现是**先对真实 A 完成身份核验，再只向解析器提供带注入标签的内存副本**，但现文未明确把 R6 的对象与副本分开，也没有限定哪些检查重跑、哪些真实结果持续带入。仅写“已记录的证据副本”“基线已通过”“注入位置执行前写定”，尚不足以决定这个跨域接点。实现者不能运行时自行跳过 I，也不能用副本的新哈希替代原 A 的构建身份。

因此这是**指定检查可达性／唯一分类的文档残余**，不是已观察到的 absent 假通过；按严格实现会保守拒绝，却达不到所宣称的对照命中。不能用“都是 T”关闭它。

**最小修正：**在文档写定以下边界，不需本轮实现或运行：

- 真实工具、原 A、归档及采集先做完整有效性和身份核验，保留不可变的原始证据；真实错误照常持续使对照无效。
- 明确 HC-6(i)～(iii) 是**身份已核验后的解析输入接缝**，而非另一个冒充生产 A 的产物。记录原身份、副本身份、精确修改与消费者；副本不能重新取得生产关联。
- 固定每个变体所重跑的检查子图与依赖、预期失败／未评估项；保留所有额外真实错误。其他名字／nm 记录变体也应标明消费层，不借同一规则豁免真实记录的身份或采集错误。
- 基线的有效性阶段与“全部对照完成后的 C／R6④ 汇总”分开，避免把尚待执行的 HC-5／HC-6 自身当作基线先决条件。
- 静态列出基线健康、计划读取变体、真实身份失败叠加、额外工具／清理失败及上游未评估的结果。只有指定检查确实评估并命中才算有效对照。

**B-134-3 尚不能关闭，按一项残余 B-134-3-R 计数。** 原优先级问题的主体修复不因此作废。

## 6. 新增 B-135-1：GOENV=off 的固定 JSON 预期错误

**位置：候选稿 257 行，R1 实际证据③。** 新增文字要求：设置 GOENV=off，且 go env -json 中 GOENV 为 off。

固定 Go 1.27.3 实际源码链是：

1. [cfg.EnvFile:344～349](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/cfg/cfg.go#L344) 看到进程环境 GOENV=off，返回 **空字符串、false、error**，并不返回 off。
2. [envcmd.MkEnv:79～89](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/envcmd/env.go#L79) 丢弃该 error，把空字符串保存为 GOENV 的 Value。
3. [main.go:304～309](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/main.go#L304) 保存 MkEnv 到 cfg.CmdEnv；[runEnv:336](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/envcmd/env.go#L336) 使用它；[printEnvAsJSON:605～618](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/envcmd/env.go#L605) 直接编码 Value，没有把空值改为 off。

所以正常的推荐配置会给出 **`"GOENV": ""`**。按 257 行检查它必须为 off，R1 会拒绝健康配置；301 行的有效基线也就不能成立。该反例来自本轮新要求与固定源码，属于直接回退，不是对无关旧问题的重开，也没有运行 go env 补证。

**最小修正：**区分驱动实际传入的环境与工具输出。记录／核对实际传入 GOENV=off；在固定工具链上要求 JSON **存在 GOENV 键且值为空串**，不能把缺键、非零退出、解析失败当作健康。空输出值本身不能反推传入过 off。非 off 分支继续读取配置文件的实际路径、内容与哈希。

同时把“cmd/go 不读配置文件”精确为“不读该用户 GOENV 配置文件”：[cfg.initEnvCache:361～369](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/cfg/cfg.go#L361) 在该分支跳过用户文件，仍读取 GOROOT/go.env。不能据 GOENV=off 推出整个工具链不读任何配置。

完整 build 命令白名单、构建 -trimpath 与自动 compile／asm -trimpath 的区分、-x 在响应文件改写前打印及 GO_BUILDER_NAME 的限定说明，其余方向成立。固定 [shell.go:591～632](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/work/shell.go#L591) 与 [exec.go:3646～3704](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/go/internal/work/exec.go#L3646) 支持这些判断；没有把正常自动参数当作禁用构建选项。目录哈希仍只提供时点身份／路径关联，没有升级成实际读入字节的认证。

## 7. B-134-4 与两项精化

### 7.1 B-134-4——文档规格范围关闭

位置：候选稿 **50～65、152～167、243～247、269～320、330～338 行**。

生产与对照现在共用包路径、函数／方法基名、种类、证明状态、H_search 和独立预期。固定 [PathToPrefix](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/internal/objabi/path.go#L18) 中只有点号的转义限于末段；其他所列特殊字节在整个路径转义。本轮三个固定包路径均不需要转义。汇编中间点到点号、再加包前缀的依据与源码一致；归档与 ELF 名字不混用。

HC-7 的 sqlite Driver.Open／newConn、libc Xmalloc 的声明分别在当前固定模块缓存的 driver.go:287、conn.go:56、mem_musl.go:20，可对应描述。方法仅作正面表示检查，不扩展方法 absent 证明。

唯一清单已经闭合：

| 范围 | 固定内容／要求 |
|---|---|
| 7 项构造对照 | HC-1～4、HC-8～10；HC-8/9 明确接替旧 C2/C3 |
| HC-2 | P2.func1、P3.abi0、W.abi0 必须命中派生分支；基名不顶替；正确汇编成员中的 W 定义与 Go 包装分开保存 |
| HC-3 | 固定调用点内联诊断，加 A 中 Q 不在而 G 在的双条件；副作用不是内联保证；未命中失败关闭 |
| HC-4 | R、U 的归档定义都在，A 中两者均不在，才能检验健康 absent |
| HC-7 | 4 个正面描述；不提供方法的 absent 定理 |
| HC-5／HC-6 | 分别 3／6 个输入变体；构造与指定失败出口已列，HC-6 的可达性须修 B-134-3-R |
| HC-10 | 第二次确实没有链接才命中 I.link；真的又链接时为未命中，不改预期 |

这相当于 7 项构造、4 个正面描述及 9 个输入变体的 **20 个细分检查项**，不是 20 次已运行，更不是沿用“旧 C1～C10 全覆盖”的计数。§3.4、§5、§6.1 及 §174 的清单一致。

删除 B／dumpdep／DWARF 和旧 C4/C5/C9 的取舍保持 Review 134 的限定：失去的独立交叉核对和图截断覆盖明确；HC-3 仅替代所述分类控制，HC-6 不认证任意 ELF。对照成功仍不能替代宿主集合完备性。B-134-4 的描述／清单关闭不等于 HC-6 已取得有效证据。

### 7.2 N-134-1——政策精化完成

R5b 在 **262 行**明确：名字偏移有效，但 STT_FUNC 名字为空，D.R5b.unnamed 失败；越界名字的 unnamed 检查则未评估。固定 [debug/elf/file.go:710～772](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/debug/elf/file.go#L710) 确实丢弃 getString 的成功标志；原始 st_name 与 NUL 检查有必要。

固定 [ld/symtab.go:186～222、270～281](https://github.com/golang/go/blob/2aae1000ea1d4b28923b96db28acf4e3b99d6aff/src/cmd/link/internal/ld/symtab.go#L186) 支持按 K 的文本写出规则拒绝无名 STT_FUNC。严格说 ELF 还包含保留的零号空符号，属于 STT_NOTYPE；这不影响所规定的 STT_FUNC 政策，不把“无名只有 SECTION”的概括扩成所有条目的一般断言。

HC-6(iii) 把非目标、非标记条目的 st_name 设 0，局部预期就是 unnamed；e 的选择避开目标及标记。**精化本身关闭，变体有效性依然受第 5 节残余制约**，不重复新增阻塞。

### 7.3 N-134-2——完成

候选稿 **409～431 行**把基线的时点扫描、排他区间内 subreaper 的领养和规定阶段清扫、C-10 的取得租约时检查分别说明；没有连续监控或区间不变的推论。

本轮读取 libc_musl.go:1034～1048，Xsignal 的 Reset／Ignore／Notify 能力与文字相符。条件性风险是：一次检查之后若闸门外活动改变处置／回收状态，可能破坏协议前提；不是已证明 SQLite 实际执行，也不是当前未知自动获准 PASS。原始入口未覆盖仍未被接受为 L。

## 8. CI 独立核验

[run 42225668432](https://github.com/XKHoshizora/specmetry/actions/runs/42225668432)：正式为 pull_request、attempt 1、completed/success，**8/8**。独立取得八份完整作业日志，逐份核对 fetch／checkout 及 git log 的实际结果，全部为 **ae201df9423beffea6bd579b48d7dfc54376f23e**，没有只用 job head_sha。

独立读取远程 Git commit：父提交依次为固定 base **56d41b97e83abe85f59f4d7370f8effc617b8448**、固定 HEAD **b10564bf44443f9ecaa661ef91c47be8bd0ee3d1**；tree **08aa99a387387cffd0fb69d6f1a0d2a6b9714012** 与 HEAD 相同。

| 作业 | job ID | 结果及适用范围 |
|---|---|---|
| Go tests Ubuntu | 127918400097 | success；27 个 ok 包；15 个测试 SKIP；普通测试带 -race；独立 root CrossUser 8 PASS |
| Go tests macOS | 127918400357 | success；27 个 ok 包；7 个测试 SKIP；Linux root 步骤 skipped |
| Go tests Windows | 127918400448 | success；27 个 ok 包；10 个测试 SKIP；Linux root 步骤 skipped |
| Go lint | 127918400549 | success；tidy、vet、文档链接、lint，0 issues |
| Web | 127918400551 | success；前端常规检查与构建 |
| Build macOS | 127918704824 | success；嵌入构建、web 测试及 smoke |
| Build Ubuntu | 127918704909 | success；同上 |
| Build Windows | 127918705008 | success；同上 |

三个测试作业实际版本均 go1.27.3，平台依次为 linux/amd64、darwin/arm64、windows/amd64。日志无测试 FAIL／作业 error；不把 root 步骤 skipped 当作通过。

共同 7 项测试 SKIP：TestClaimWithARealGitThatFetches、TestInitRefusesRealOldGit、TestARealGitThatFetches、TestResolveCommitsWithAGitThatFetches、TestWP23ReadsWithAGitThatFetches、TestM14ReadOnlyProbe、TestM14OpenPath。Ubuntu 普通步骤另跳过 8 项 CrossUser，随后独立 root 步骤逐项通过；Windows 另跳过 TestResolveCommitsWithUnreadableStorage、TestClassifyWithUnreadableStorage、TestSourceListUnreadableSource。集合与 Review 134 已归档的上一运行记录吻合；本轮没有重取旧运行的全部日志并宣称重新验证历史。

CLI 耗时为 Ubuntu **510.459s**、macOS **153.050s**、Windows **375.765s**，仅为本次观测，不是上界、根因或维护事项关闭证据。旧运行和失败回执保留。

**常规 CI 没有运行 H-1、nm 语法模型、ELF 变体或宿主证明验证。** 它的成功不能抵消本报告两项文档阻塞。

## 9. H-1、依赖策略及最小下一步

H-1 的有限基础和已关闭的驱动／pidfd／资源／截止协议按原范围保持；本轮未改其状态机、预算、次数或 exact 规则。修复两个文档问题即可再次复审，不要求先做未授权实现或实验。

采纳／执行之前仍须分别完成：

1. 修订 B-134-3-R 的输入层与身份边界、B-135-1 的固定工具预期，保持真实错误持续消费；同步 §174 之后的追加记录和必要状态，旧原文不回改。
2. 独立复审后，负责人决定是否采纳 H-1；采纳才同步链接计划 §3.4 列明的分类、输入、对照和总结论，不让候选稿暗中替代现行停止的方法。
3. 词法重算、R1～R6 实际证据、精确目标／宿主／包装、驱动／解析／对照实现与审查、场所／workflow／资源及执行授权仍未完成。当前目录和时点哈希不代替实际输入关联。
4. Xsignal 仍没有 absent 证明，也没有本轮 present 证据；完整清单不能得到“全部 absent”是已披露的方法边界。它不证明原 (a) 永远不可能，有限 17 个具名函数的证明也不证明能力整体不可达；F-5 尚未解决。
5. (b) 的精确清单、版本／哈希／平台／调用位置／用途、升级与 replace 漂移失效、陈旧项检查、原始入口和运行时信任边界、冻结／验收同步仍须负责人决定。保留源码能力与允许其在关键区间活动是不同的风险范围。已观察错误、未知归属／清理、宿主禁用等硬规则不能一并豁免。CGO、Q/X/E/U、F-8、第三批及后续授权保持原状态。

建议交接提示词：

> Review 135 对 b10564b 为文档技术 NO-GO。B-134-1、B-134-2、B-134-4 在文档范围关闭，N-134-1／2 的精化完成。仅修 B-134-3-R：真实 A／采集身份先核验，输入变体通过明确的、身份核验之后的解析输入接缝消费，保留所有额外真实错误及未评估状态；不能以“都为 T”算命中。另修 B-135-1：固定 Go 1.27.3 在传入 GOENV=off 时 JSON 的 GOENV 为空串，必须与实际传入环境分开核对，缺键／工具失败不能通过。只在另获授权范围内修订文档和追加记录，不先运行 Go、nm、构建、链接或实验，不改冻结／实现，不采纳 H-1、(b) 或依赖例外，不合并或授权第三批。

## 10. 验证限制与结束身份

独立验证包括：固定本地／远程／PR 身份、Git 增量与 blob／归档哈希、旧处理记录字节前缀、固定官方源码、必要模块声明／构建条件、当前正式 CI 与八份原始日志。正文中的反例和行模型均为静态推演，**不是实际运行的 Go／nm／ELF 故障结果**。未重算模块词法全集或全部目录哈希；沿用历史有限证明时明确保留原范围。

没有运行 Go、nm、构建、链接、词法重算程序、分析器、驱动或实验；没有安装工具、触发／重跑 CI、修改受审文档／代码／测试／依赖／workflow／冻结规范或验收项，没有提交、推送、合并。原仓库只新增本报告。已删除 worktree 的待决问题、factgen/legality、维护事项与冻结限制保持。

**结束核验：2026-11-07 14:11:09 +09:00。** 本地 HEAD／tree／分支／origin 与开始一致；远程分支及 PR HEAD 仍为 `b10564bf44443f9ecaa661ef91c47be8bd0ee3d1`，master／PR base 仍为 `56d41b97e83abe85f59f4d7370f8effc617b8448`，PR open／未合并，快照仍为 `ae201df9423beffea6bd579b48d7dfc54376f23e`。跟踪文件及索引干净，工作区仅有 `?? docs/reviews/codex-review-135.md`，没有身份漂移或他人修改。

本报告不采纳 H-1，不选择 (b)，不批准例外／新 L，不授权第三批或链接验证，不验收 WP2.5／M2。报告后暂停，等待负责人决定。
