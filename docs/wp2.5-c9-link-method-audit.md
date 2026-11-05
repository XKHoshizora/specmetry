# WP2.5 第三批：链接方法的只读审计（S-2、S-3、S-4、S-6、S-7）与配置 K 的方法输入

> 状态：**只读审计报告（纯文档）；Review 132 限定技术 GO，作为“原方法停止、提案未采纳”的审计记录**（2026-11-05；review-resolution §168 ~ §170；[链接验证计划](wp2.5-c9-link-proof-plan.md) §6）。
>
> - **Review 132 的裁定**（§170）：
>   - 原方法停止的结论正确，原方法仍不可执行；
>   - 提案 M-1 ~ M-5 没有作为完整可执行的方案采纳；
>   - M-2 只给出表示与归属层次的、更弱的结论，不足以支持原 Q-B3-1 (a) 要求的“目标代码不在产物中”，本次也不批准用它替代原强命题。所以 §0.1、§8.1、§9、§10 中“待（审查与）决定提案 M-2”的说法，都还须有技术证明或替代方法的独立复审；负责人选择该提案，并不使原技术条件成立。
>   - 两项非阻塞的事实精化已按审查修订：N-132-1（§3.2 的 ABI 包装例子）与 N-132-2（§8.2 的对照 C5）。
>   - §0.3 按审查补充了 `go env` 越界的证据边界。
>   - 其余结论没有改写。
> - 本文只回答链接验证计划 §6 的五项执行之前的静态审计，并固定配置 K 下的方法输入清单。它不批准依赖例外，不接受新的 L，不决定生产的 CGO、X / E / U、F-8、场所或资源，也不授权执行。
> - 编写本文时，除 §0.3 登记的一次越界（`go env GOMODCACHE`）之外，没有运行 Go、构建、链接、验证分析器、驱动程序或实验，没有安装工具链；证据都是固定源码的只读文本与 Git 核对。
> - 结论只适用于下面固定的源码与配置 K。凡写“须运行才能取得”的，都不是本文的证据。

## 0. 结论摘要

### 0.1 逐项结论

| # | 审计的问题 | 结论 | 对执行路径的影响 |
|---|---|---|---|
| S-2 | `mark` 之外设置可达的位置，是否可能使包代码的 Go 函数的文本可达；产生哪些合成的文本符号 | **在 K 与 §1.3 的前提下成立（限定）**。原清单的“27 处”不完整：设置可达的入口有三种，独立盘点为 125 处（§2.1）。其中只有 `runtime.mapinitnoop` 是包代码的 Go 函数，可以不经 `mark` 进入文本；合成的文本符号是 `runtime.text`、`runtime.etext`、`go:textfipsstart`、`go:textfipsend`（§2.3） | 给出 §7 第 5 条所需的清单；不阻断 |
| S-3 | 目标函数与派生、ABI、泛型、闭包及包装符号的映射，以及未知的情况 | **对 T0 所需的形式成立（限定）**：闭包、`-range`、`gowrap`、`deferwrap`、内联产生的闭包、ABI 包装的命名都由固定源码确定（§3）。泛型实例的语法没有审计，T0 中没有泛型函数，不适用。每个目标实际有哪些派生符号，须运行之后从归档取得 | 不阻断；名字的比较须按 §3.4 的规范化（提案 M-3） |
| S-4 | 相关内联代码与 DWARF 记录的对应、遗漏的条件与适用范围 | **强形式未成立**：源码只保证“仍有指令归属于某个内联实例（或其后代）”时写出 `DW_TAG_inlined_subroutine`；归属被合并到调用者位置的指令不会产生记录（§4.3）。所以计划 §3.2 的前提“内联的记录完整”与 §7 第 7 条“F 的代码不在产物 A 中”不能由源码推出 | **按计划，这条执行路径在执行之前停止**；须先审查与决定提案 M-2 |
| S-6 | 构建 / 链接缓存、输出的转发与完整采集、可取得的归档的身份与路径 | **缓存与归档两部分成立**：跳过链接的条件与归档的路径、身份都能由 `-x` 的记录确定（§5.1、§5.3）。**输出的转发与计划的假设不符**：`-dumpdep` 不出现在 `go build` 的标准输出上，而是与链接器的标准错误合并、经 cmd/go 改写之后写在 `go build` 的标准错误中（§5.2）；按计划 §5.1 第 5 步“标准输出单独写入文件”，采集到的文件预期为空 | **同样停止**；须先审查与决定提案 M-1 |
| S-7 | 相关可达文本与 ELF 符号条目的对应及遗漏的条件 | **在 K 与 §1.3 的前提下成立**：`linksetup` 结束时可达的、来自 Go 对象文件的每个文本符号恰好写出一个 `STT_FUNC` 条目，另加 `runtime.text`、`runtime.etext` 两个标记；没有剥离时没有遗漏（§6）。名字按 `mangleABIName` 改写（`.abi0`），与 `-dumpdep` 中的名字不同 | 不阻断；名字的规范化见提案 M-3 |
| S-1 / S-5 | （沿用已审范围）只补本次结论直接依赖的部分 | S-5 再次确认。S-1 须更正两点：类型符号可以两次作为 `to`；同一个名字可以属于不同的符号；另补根行等三点细节（§7） | 解析器与对照 C9 的规则须随之调整（提案 M-3） |

### 0.2 执行路径

- 按计划 §6，“任一项不能得出确定的结论 → 方法在执行之前就判为不可判定，不执行”。S-4 的强形式与 S-6 的输出采集都不能按现在的写法成立，所以**按现在的方法，这条执行路径停止**；可行性验证仍不可执行。
- 本文没有为了取得“通过”而增加实验、更换版本或静默改动主判据。需要调整的地方都写成 §9 的提案（M-1 ~ M-5），附影响说明，交独立审查与负责人决定。即使提案全部获采纳，执行仍须另行授权，§10 列出的执行之前的条件也都还没有满足。
- 已关闭的收尾协议（链接计划 §8.3、§8.4）没有改写。

### 0.3 本轮的一次越界（`go env`）

- **事实**：编写本文时运行过一次 `go env GOMODCACHE`，违反了本轮“不运行 Go”的要求。这是已登记的越界，不因查询的用途而变成授权，本文的过程也不能说成“没有运行过 Go”。
- **已有的原始回执**：来自 Claude Code 的会话记录（本机文件，不在仓库中）。
  - 工具调用 `toolu_018g5euKvcN2Qkd7picJVX7V`，调用时间 2026-11-04T18:59:17.154Z，结果时间 18:59:25.270Z，结果没有标为错误。
  - 命令原文：

    ```text
    git checkout -q master && git pull -q --ff-only origin master && git rev-parse HEAD HEAD^{tree} && git checkout -q -b docs/wp2.5-link-method-audit && git status --porcelain && git branch --show-current; ls ~/go/pkg/mod/modernc.org 2>/dev/null | head; go env GOMODCACHE 2>/dev/null
    ```

  - 输出（省略前面 `git checkout` 的进度行）：

    ```text
    30bc8aa951cf16b8fef2614e56ff902b5d47c475
    363ca392b0f7d695b66f76e8a1def37811ff4143
    docs/wp2.5-link-method-audit
    cc
    ccgo
    fileutil@v1.4.0
    gc
    goabi0@v0.2.0
    libc@v1.77.1
    mathutil@v1.7.1
    memory@v1.12.1
    opt@v0.2.0
    sortutil@v1.2.1
    C:\Users\hoshi\go\pkg\mod
    ```

  - 最后一行是 `go env` 的标准输出；它的标准错误被丢弃（`2>/dev/null`）。
- **回执中没有的**：以下各项记为缺失，不重造，也不为补齐而重新运行。
  - 缺失的项：实际执行的 `go` 可执行文件的路径与哈希、它的版本、`go env` 单独的退出状态与标准错误、启动器或 shim 的记录。
  - 本节上一版与 §169 写的“本机的 Go 1.27.1”，来自此前轮次（2026-10-29 ~ 10-31）对本机的观测，不是这次调用的回执。
  - “只读查询，没有构建、链接或下载”同样不能由回执证实，仍是作者的陈述。仓库 `go.mod` 只有 `go 1.27`、没有 `toolchain` 行，本机模块缓存中也没有 `golang.org/toolchain` 的目录；这些与“没有下载工具链”相容，但不构成证明。
- **“只一次”**：本轮从 Review 131 的决定到交付回执，会话记录中有 137 次 Bash 调用，没有 PowerShell 调用。逐条检索命令文本，只有这一次调用了 Go。这是工具调用层面的记录，不是操作系统层面的进程审计。
- **对过程的影响**：同一条命令中，`go env` 之前的 `ls ~/go/pkg/mod/modernc.org` 已列出该目录；`go env` 的输出与它一致，之后的读取都使用这个目录。所以这次查询参与了输入目录的定位（确认），不能说它对过程没有任何影响。
- **证据的边界**：
  - 模块源码的身份不依赖这次查询。§1.1 的三份 dirhash 是对所读目录的原始字节独立重算的，与 go.sum 一致；Review 132 也独立重算，结果相同。Go 源码取自固定的官方提交，另行核对。
  - 这次查询**不**作为工具链、运行环境或实际构建的任何证据。

## 1. 固定的对象与读取的方法

### 1.1 源码

| 对象 | 固定值 | 取得与核对 |
|---|---|---|
| Go | `go1.27.3`，提交 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff`，树 `09275a3482ffe831661a72c202c6261e0eb6ae7f` | GitHub API 读到 tag `go1.27.3` 指向这个提交；以 `git fetch --depth 1` 只读取得，`VERSION` 的第一行为 `go1.27.3`。下文的 Go 源码位置都相对这个提交的 `src/` |
| `modernc.org/libc` | v1.77.1 | 本机模块缓存的解压目录；按 go.sum 的方法独立重算 dirhash，得到 `h1:Ct8j47QtiZ1Enj2DtFXQtUqrPCAjdCmPjtCuvrYQ0Hs=`（4199 个文件），与仓库的 go.sum 相同 |
| `modernc.org/sqlite` | v1.60.1 | 同上，`h1:/blz53O951KWFOso4QQvEs/Fq6cDBKLtMVrYNSeJVKw=`（1899 个文件） |
| `golang.org/x/sys` | v0.49.0 | 同上，`h1:XbzkgYJHdqh/8m2Uu0W/dQv8nktxx4BFHp1M0gROTXA=`（555 个文件） |

- 重算的 dirhash 只证明这些目录与 go.sum 记录的内容一致，不认证上游原产。仓库其他依赖的模块缓存没有重算（§1.3 的 `.syso` 前提因此只是限定的检查）。
- 仓库对象：master `30bc8aa951cf16b8fef2614e56ff902b5d47c475`（树 `363ca392b0f7d695b66f76e8a1def37811ff4143`）。

### 1.2 配置 K 与“实际编入的来源”

配置 K 按计划 §2：`GOOS=linux`、`GOARCH=amd64`、`CGO_ENABLED=0`、`GOEXPERIMENT` 为空、`GOAMD64=v1`，生产构建形态（`go build -o <out> ./cmd/specmetry`，没有 `-trimpath`、`-ldflags` 或构建标签）。

- 默认的构建模式是 `exe`：`cmd/go/internal/work/init.go:261` ~ `262`，`internal/platform/supported.go:231` ~ `246` 的 `DefaultPIE` 对 linux 返回 false。
- 内部链接；没有动态导入时，链接器自动置 `-d`（`cmd/link/internal/ld/lib.go:842` ~ `846`），产物是静态的 ELF。
- 文件的选择按 go/build 的规则（文件名的 GOOS / GOARCH 后缀、`//go:build`、`import "C"` 在 `CGO_ENABLED=0` 时排除）静态求值，标签集合为 `linux amd64 unix gc amd64.v1 go1.1 … go1.27`。这是阅读的辅助，**不是** `go list` 的结果；执行时须以 `go list -deps -json` 核对（§10）。
  - `modernc.org/libc`（根包）：34 个文件，其中汇编 2 个（`abi0_linux_amd64.s`、`tls_linux_amd64.s`）；目标所在的 `ccgo_linux_amd64.go`（`//go:build linux && amd64`）与 `libc_musl.go`（`//go:build linux && (amd64 || …)`）都在内。`libc_linux_amd64.go` 的约束是 `!(linux && amd64)`，不编入（与 Review 127 的 N-127-4 一致）。
  - 根包导入的 libc 子包只有 `modernc.org/libc/uuid/uuid`；`modernc.org/sqlite` 另导入 `modernc.org/libc/sys/types`。两者在 K 上各两个文件，只有类型、常量与变量的声明，没有函数。
  - `modernc.org/sqlite`：根包 24 个文件，`lib` 126 个文件，`vtab` 2 个文件。
  - 这些文件中都没有 `//go:linkname` 或 `//go:cgo_` 指令（libc 根包逐个文件检索）。

### 1.3 贯穿全文的前提

下列前提来自上面的固定源码与配置；其中标“须运行”的，执行时须取得证据，不成立即 T：

1. 构建模式 `exe`、内部链接、没有 `-linkshared` 与插件：`ctxt.DynlinkingGo()` 为 false。
2. 没有宿主对象：`CGO_ENABLED=0`，而且所用模块与仓库中都没有 `.syso` 文件（模块缓存逐个检索；标准库中只有 `crypto/internal/boring/syso`（`//go:build boringcrypto`）与 `runtime/race`（只在 `-race` 时），都不编入）。**须运行**：以 `-x` 的记录确认链接命令没有宿主对象、产物没有 `PT_INTERP` 与 `.dynamic`。
3. 链接器没有 `-s`、`-w`：cmd/go 只在 `OmitDebug` 时传 `-s -w`（`cmd/go/internal/work/gc.go:605` ~ `607`），`go build` 不设置它（`go run`、`go test`、`go tool` 与 pkgsite 才设置）。**须运行**：以 `-x` 的记录确认。
4. 编译没有 `-dwarf=false`：cmd/go 同样只在 `OmitDebug`、plan9 或 wasm 时传它（`gc.go:109` ~ `111`）。**须运行**：同上。
5. FIPS 的链接器支持在 K 上开启：`cmd/internal/obj/fips140.go:178` ~ `209` 只对 wasm、windows/386 与 aix 返回 false。cmd/go 的 `-fipso` 只在 `GOFIPS140` 不为 `off` 时传（`gc.go:611` ~ `613`），两者不同，不要混淆。

## 2. S-2：`mark` 之外的可达

### 2.1 设置可达的全部入口（独立盘点）

可达标记只有一处真正的写入：`cmd/link/internal/loader/loader.go:900` ~ `905` 的 `SetAttrReachable`（第 902 行置位）；loader 包内没有其他写入。调用它的入口有三种：

| 入口 | 位置 | 非测试代码中的调用点（不含 loader 自身的定义） |
|---|---|---|
| 直接调用 `SetAttrReachable(x, true)` | — | 29 处、12 个文件，其中 1 处是 `deadcode.go:362` 的 `mark` 本身；`mark` 之外 28 处 |
| `SymbolBuilder.SetReachable(true)` | `loader/symbolbuilder.go:174` ~ `176` | 20 处 |
| `Loader.CreateSymForUpdate`：查找或创建符号，**并置为可达**，再返回更新器 | `loader/symbolbuilder.go:57` ~ `60` | 77 处 |

- 合计 `mark` 之外 125 处，分布在 19 个文件中。盘点的方法：在固定提交的 `src/cmd/link/internal` 中按这三种调用的文本检索（排除 `_test.go` 与 loader 包自身），再逐处读所在的函数与调用条件。逐处的清单（文件、行号、种类、所在函数）见附录 A。
- 原计划的“27 处、12 个文件”只对应第一种入口（实际为 28 处），没有包括后两种入口；不能把它当作完整性的结论。
- 另有三处把可达**清除**之后重新 `mark`：`deadcode.go:196` ~ `201`、`224` ~ `229`、`262` ~ `276`。它们只作用于类型符号（`R_METHODOFF` 的方法类型、`IsGoType` 的类型或 itab 的类型），重新标记仍经 `mark`，所以会打印第二次（§7）。

### 2.2 逐类判定（配置 K）

按调用点的执行条件分类。“不适用”指在 K（ELF、linux/amd64、内部链接、`exe`、静态）上不会执行；依据是调用点所在函数或调用者的条件。

| 类别 | 位置（所在函数） | K 上 | 能否使包代码的 Go 函数的文本可达 |
|---|---|---|---|
| 其他平台与格式 | `arm/asm.go`、`arm64/asm.go`、`loong64/asm.go`、`ppc64/asm.go`、`riscv64/asm.go`、`x86/asm.go`（386）、`macho.go`、`pe.go`、`xcoff.go`、`seh.go`（`collectSEH` 只在 `ctxt.IsWindows()` 时调用，`data.go:2847` ~ `2850`）、`data.go:952`（`windynrelocsyms`）、`lib.go:750`、`753`（`loadWindowsHostArchives`）；`elf.go:1598`（`IsShared` 的分支）、`elf.go:1630`、`1632`（MIPS） | 不适用 | — |
| 跳板 | `data.go:3352`（`AddTramp`）、各架构的 `trampoline` | 不适用：amd64 没有 `Trampoline`（只有 arm、arm64、loong64、ppc64、riscv64 设置，见各自的 `obj.go`），`data.go:100` ~ `103` 直接返回 | — |
| 动态链接 | `ld.go:229`、`235`、`248`（`PrepareAddmoduledata`，`ld.go:220` ~ `222` 在 `!DynlinkingGo()` 时返回；amd64 的 `gentext`（`amd64/asm.go:43`）只调用它）；`elf.go:1023` ~ `1149`（`elfdynhash`）与 `doelf` 中 `!*FlagD` 的分支（`elf.go:1466` 起）；`lib.go:938`（386） | 不适用（前提 1、2：`-d` 自动置位） | — |
| 非 ELF 的 build ID | `data.go:2608`（`textbuildid`，`data.go:2603` ~ `2605` 在 ELF 上返回） | 不适用 | — |
| 分割的文本段 | `data.go:2921`（`assignAddress` 的 `runtime.text.N`）、`data.go:3210` ~ `3221` | 不适用：`splitTextSections`（`data.go:2986` ~ `2988`）只对 arm、ppc64 与 darwin/arm64 的外部链接为真 | — |
| 数据、DWARF、pclntab、符号表与段的符号 | `data.go` 的 `addstrdata`、`addgostring`、`addinitarrdata`、`GCProg.Init`、`fixZeroSizedSymbols`、`buildinfo`；`dwarf.go` 的全部 13 处；`pcln.go` 的 4 处；`symtab.go` 的 13 处（含 `textsectionmap`）；`elf.go:1437`（`addgonote`）；`lib.go:869`（`runtime.tlsg`，TLS 数据）、`lib.go:930`（moduledata）；`lib.go:136`、`143`（`mkArchSym`、`mkArchSymVec`：`setArchSyms` 登记的除 `runtime.unreachableMethod` 外都是段符号）；`lib.go:2950`（`defineInternal`：直接的调用者在 `data.go:3241` ~ `3247` 定义只读数据；经 `xdefine` 定义的是段与区间的边界标记，其中只有下面“文本段的标记”与不适用于 K 的 `runtime.text.N` 是文本） | 适用（部分只在共享库、c-archive 等模式下执行，同样是数据） | **不能**：名字都是固定的链接器名字（`runtime.*` 的数据标记、`go:*`、段名），类型都不是文本；`addstrdata` 处理的是 `-X` 的字符串变量（数据） |
| 已是根的文本 | `lib.go:136`（`mkArchSym`，由 `setArchSyms` 在 `lib.go:155` 为 `runtime.unreachableMethod` 调用） | 适用（`main.go:420`，在 `deadcode` 之后） | 不新增：它已是 `deadcode` 的根（`deadcode.go:84`、`103` ~ `110`），经 `mark` 打印 |
| **FIPS 的边界标记** | `fips140.go:157`、`168`（`loadfips`，由 `lib.go:689` 在 `loadlib` 中调用，早于 `deadcode`） | 适用（前提 5） | 不是包代码，但产生两个**文本**符号 `go:textfipsstart`、`go:textfipsend`（`fips140.go:127` ~ `128`，`SetLocal`，`fips140.go:174` ~ `176` 加入 `ctxt.Textp`）；不经 `mark`，不打印 |
| **文本段的标记** | `data.go:2710` ~ `2711`（`textaddress` 中以 `xdefine` 定义 `runtime.text`（`STEXT`）与 `runtime.etext`（`STEXTEND`）） | 适用 | 不是包代码；两者由 `genelfsym` 按名字写入符号表（§6），不经 `mark` |
| **map 初始化的替换** | `deadcode.go:344`（`mapinitcleanup`，`-pruneweakmap` 默认为 true，`main.go:113`） | 适用 | **能**：包初始化函数到已不可达的 `map.init` 的弱调用被改写为 `runtime.mapinitnoop`，后者直接置为可达，不经 `mark`（不打印，也不入工作队列）。它是 runtime 包的 Go 函数，不是 libc 的函数；是否出现取决于实际是否发生了改写 |

### 2.3 结论与适用范围

- 在 K 与 §1.3 的前提下，`mark` 之外进入产物文本或符号表的文本符号只有：`runtime.text`、`runtime.etext`、`go:textfipsstart`、`go:textfipsend`，以及条件性的 `runtime.mapinitnoop`。这就是计划 §7 第 5 条所说的“S-2 的合成符号清单”（提案 M-4 写定它的用法）。
- 除 `runtime.mapinitnoop` 之外，没有任何包代码的 Go 函数（包括 `modernc.org/libc` 的全部函数）能不经 `mark` 而可达；所以 libc 的文本符号可达，当且仅当它在 `-dumpdep` 中作为 `to` 出现（还须 §7 的名字规则）。
- **限制**：
  - 结论依赖调用点的执行条件，而不是对任意配置的证明。K 之外（PIE、cgo、外部链接、插件、其他架构）须重审。
  - 宿主对象的前提（前提 2）只由模块缓存的检索支持；除 libc、sqlite、x/sys 之外，其他模块的缓存没有重算 dirhash。
  - `mark` 内的 `fmt.Printf`（`deadcode.go:375`）不检查写入错误。链接器打印不完整时，链接器自己不会失败；对文本符号，这由计划 §7 第 5 条的交叉核对发现（A 中有、`-dumpdep` 中没有 → T），对数据符号不发现（计划 §7 已写明不依赖数据符号的行）。

## 3. S-3：派生符号的命名与 T0 的映射

### 3.1 命名的来源

| 形式 | 名字 | 固定源码 |
|---|---|---|
| 函数字面量（外层不是闭包） | `F.func<N>` | `cmd/compile/internal/ir/func.go:443` ~ `446`、`503` |
| 闭包中的闭包 | `F.func<N>.<M>`（后缀为 `.`） | 同上（`outerfn.OClosure != nil` 时 `suffix` 保持 `"."`） |
| range-over-func 的循环体 | `F-range<N>` | `func.go:447` ~ `448` |
| `go` / `defer` 的包装 | `F.gowrap<N>`、`F.deferwrap<N>` | `func.go:449` ~ `452` |
| 包级的闭包 | `<pkg>.glob..func<N>` | `func.go:438`、`458` ~ `461` |
| 内联产生的闭包 | 外层取**被内联函数**的名字与包（`func.go:465` ~ `486`）；编译期的 `#hash#` 后缀在写对象文件时去掉（`cmd/internal/obj/objfile.go:280` ~ `294`）；闭包都标为内容寻址（`func.go:542`） | 所以 `modernc.org/libc.F.func1` 可以出现在**其他包**的对象中，多个不同的闭包可以同名；链接器把内容寻址的文本符号写为 `STB_LOCAL`（`cmd/link/internal/ld/symtab.go:113` ~ `120`） |
| 方法 | `T.M`、`(*T).M` | `ir/expr.go:1189` 起 |
| 方法值的包装 | `T.M-fm` | `noder/reader.go:4152`、`walk/closure.go:220` |
| ABI 包装 | 与被包装函数**同名**、不同 ABI（`ssagen/abi.go:274` ~ `281`，`dupok`） | 见 §3.2 |

### 3.2 ABI 包装

- 编译器假设每个函数都会以 ABIInternal 被引用（`ssagen/abi.go:201` ~ `204`），汇编按 symabis 记录 ABI0 的引用；每个缺少的 ABI 生成一个包装（`abi.go:232`、`236` ~ `248`）。
- 因此在 K 上：
  - Go 定义、又被汇编以 `CALL ·F(SB)` 调用的函数 F：有 ABIInternal 的 F 与 ABI0 的包装 F；
  - 汇编定义、在 Go 中声明的函数 Y：有 ABI0 的汇编体 Y 与 ABIInternal 的包装 Y。
- `-dumpdep` 打印 `SymName`，两者同名；ELF 的符号表中，非 ABIInternal 的文本符号在存在同名的 ABIInternal 文本符号时改名为 `<name>.abi0`（`symtab.go:826` ~ `845`，`RegabiWrappers` 在 amd64 上默认开启）。
- 这里原有一个例子：“包装可能把被包装函数内联进去，那时 F 的代码在名为 `F.abi0` 的符号中”。它在固定工具链上没有源码依据，已撤回（N-132-1，§170）。
  - 固定源码的顺序是先做去虚化与内联（`cmd/compile/internal/gc/main.go:259`），之后才生成 ABI 包装（`main.go:280` ~ `281`，注释写明这一步不受益于内联）。
  - `makeABIWrapper`（`ssagen/abi.go:252` 起）只生成对 F 的调用或尾调用（`abi.go:336` ~ `347`），这条路径上没有再次调用内联器。
  - 本文不推断 F 的代码会出现在 `F.abi0` 中，也没有运行过对照 C6 或观察到这种情形。
- 上面的命名与映射结论不受影响：同名而 ABI 不同，ELF 中的名字按 `mangleABIName` 改写；`F.abi0` 作为 F 的派生符号在 A 中时，按计划 §7 第 6 条判 present。

### 3.3 T0 在 K 上的固定枚举

位置都是 `modernc.org/libc@v1.77.1` 中 K 编入的文件。“汇编包装”列是 `abi0_linux_amd64.s` 中 `CALL` 该函数的 `TEXT` 符号（行号是 `TEXT` 所在行）；对应的 Go 声明在 `abi0_linux_amd64.go`（例如 `Yexecve` 在第 2415 行）。

| 类别 | 函数 | 定义 | 能力的依据 | 汇编包装 |
|---|---|---|---|---|
| 对象引用 | `Xsystem` | `libc_musl.go:1187` | `exec.Command("sh", "-c", …)`、`Run` | `Ysystem`（`:25880`） |
| 对象引用 | `Xsignal` | `libc_musl.go:1034` | `gosignal.Reset`、`Ignore`、`Notify` | `Ysignal`（`:24160`） |
| 对象引用（链） | `___libc_sigaction` | `libc_musl.go:1115` | 调用 `Xsignal`（`:1128`） | 无 |
| 对象引用（链） | `X__sigaction` | `ccgo_linux_amd64.go:133597` | 调用 `___libc_sigaction`（`:133618`） | `Y__sigaction`（`:7114`） |
| 对象引用（链） | `Xsigaction` | `ccgo_linux_amd64.go:133626` | 调用 `X__sigaction`（`:133631`） | `Ysigaction`（`:24027`） |
| 原始系统调用 | `Xexecve` | `ccgo_linux_amd64.go:126113` | `X__syscall3(…, SYS_execve, …)`（`:126119`） | `Yexecve`（`:12258`） |
| 原始系统调用 | `Xfexecve` | `ccgo_linux_amd64.go:126210` | `X__syscall5(…, SYS_execveat, …)`（`:126220`），并调用 `Xexecve`（`:126225`） | `Yfexecve`（`:12965`） |
| 原始系统调用 | `Xvfork` | `ccgo_linux_amd64.go:126580` | `X__syscall0(…, SYS_fork)`（`:126586`） | `Yvfork`（`:27337`） |
| 原始系统调用 | `Xwaitid` | `ccgo_linux_amd64.go:126597` | `___syscall_cp(…, SYS_waitid, …)`（`:126602`） | `Ywaitid`（`:27670`） |
| 原始系统调用 | `Xwaitpid` | `ccgo_linux_amd64.go:126605` | `___syscall_cp(…, SYS_wait4, …)`（`:126610`） | `Ywaitpid`（`:27689`） |
| 原始系统调用 | `Xwait4` | `ccgo_linux_amd64.go:30015` | `X__syscall4(…, SYS_wait4, …)`（`:30032`） | `Ywait4`（`:27651`） |
| 原始系统调用 | `Xpclose` | `ccgo_linux_amd64.go:137717` | `X__syscall4(…, SYS_wait4, …)`（`:137731`） | `Ypclose`（`:20377`） |
| 同族（直接调用上述函数） | `Xexecv` | `ccgo_linux_amd64.go:126105` | 调用 `Xexecve`（`:126110`） | `Yexecv`（`:12243`） |
| 同族 | `Xexecle` | `ccgo_linux_amd64.go:126017` | 调用 `Xexecve`（`:126057`） | `Yexecle`（`:12209`） |
| 同族 | `X__execvpe` | `ccgo_linux_amd64.go:126122` | 调用 `Xexecve`（`:126140`、`:126167`） | `Y__execvpe`（`:4391`） |
| 同族 | `Xwait` | `ccgo_linux_amd64.go:126589` | 调用 `Xwaitpid`（`:126594`） | `Ywait`（`:27621`） |
| 同族 | `Xwait3` | `ccgo_linux_amd64.go:30007` | 调用 `Xwait4`（`:30012`） | `Ywait3`（`:27634`） |
| 同族（间接一层） | `Xexecl`、`Xexeclp`、`Xexecvp`、`Xexecvpe` | `ccgo_linux_amd64.go:125973`、`126061`、`126194`、`126202` | 逐个函数体核对：`Xexecl` → `Xexecv`；`Xexeclp` → `Xexecvp`；`Xexecvp`、`Xexecvpe` → `X__execvpe`；最终都到 `Xexecve` | `Yexecl`（`:12192`）、`Yexeclp`（`:12226`）、`Yexecvp`（`:12275`）、`Yexecvpe`（`:12290`） |

另外列出、不判定能力的：

- `Xfork`（`libc_musl.go:1026`，返回 `ENOSYS`）与 `Yfork`（`:13655`）；`Xpopen`（`libc_musl.go:1469`，`panic(todo(""))`）与 `Ypopen`（`:20474`）。
- **变量号的通用入口**：`X__syscall0` ~ `X__syscall6`（`syscall_musl.go:87` ~ `169`）、`___syscall_cp`（`syscall_musl.go:14`）、`Xsyscall`（`ccgo_linux_amd64.go:114794`）与 `Ysyscall`（`:25803`）。它们以参数接收系统调用号，被大量转译函数调用；它们是否在 A 中，须运行确认。对上表任一目标得到 absent，都不排除经这些入口、以其他位置的常量或变量调用同一系统调用；上表只覆盖在 K 编入的全部 Go 文件中文本检索到的、以 `SYS_execve`、`SYS_execveat`、`SYS_fork`、`SYS_vfork`、`SYS_clone`、`SYS_clone3`、`SYS_wait4`、`SYS_waitid`、`SYS_rt_sigaction` 的名字直接出现的位置（`SYS_clone`、`SYS_clone3`、`SYS_vfork`、`SYS_rt_sigaction` 在 K 编入的文件中没有作为调用实参出现）。
- K 编入的文件中没有 `Xposix_spawn` 本身，只有 `Xposix_spawn_file_actions_*` 与 `Xposix_spawnattr_*`（属性的读写，不启动进程）。

这保持计划 §4.3 与 F-5 的限定：**T0 不是完整的例外清单**；以上枚举只来自对 K 编入文件的文本检索，看不到经字符串、反射、`unsafe` 或函数指针表构造的到达方式。

### 3.4 映射

对 T0 中的每个函数 F（包路径 `modernc.org/libc`），在 A 中按下列名字判定（规范化见提案 M-3）：

| 证据 | ELF 符号表中的名字 | `-dumpdep` 中的名字 |
|---|---|---|
| 主符号 | `modernc.org/libc.F` | 同 |
| ABI0 包装（F 被汇编调用时） | `modernc.org/libc.F.abi0` | `modernc.org/libc.F`（与主符号同名） |
| 闭包与包装 | `modernc.org/libc.F` 之后紧接 `.func<N>`、`.<M>`、`-range<N>`、`.gowrap<N>`、`.deferwrap<N>` 的组合，可以有多个同名（内容寻址） | 同 |
| 汇编包装 Y | `modernc.org/libc.YF`（ABIInternal 包装）与 `modernc.org/libc.YF.abi0`（汇编体） | 两者都是 `modernc.org/libc.YF` |
| 内联实例 | DWARF 中 `DW_AT_abstract_origin` 的 `DW_AT_name` 为 `modernc.org/libc.F` 的 `DW_TAG_inlined_subroutine` | 无 |

- 前缀须按分隔符匹配：`Xexecv` 是 `Xexecve`、`Xexecvp` 与 `Xexecvpe` 的字面前缀，只能以 `F` 之后紧接 `.`、`-` 或名字结束来认定。
- 未知的情况（→ T，不得 absent，按计划 §4.2）：以 `modernc.org/libc.F.` 或 `modernc.org/libc.F-` 开头、但不符合上面语法的名字；目标的主符号不在同一次构建的归档中（不能当作“已被消除”）。
- 每个目标实际有哪些派生符号，**须运行之后从归档取得**。例如每个转译函数都有 `if __ccgo_strace { … defer func() { … }() }`，而 `__ccgo_strace` 是常量 false（`straceoff.go:10`）；这个闭包是否仍被编译为 `F.func1`，本文没有从编译器源码确定，不作推断。

## 4. S-4：内联与 DWARF 的对应

### 4.1 DWARF 的开启条件

- 编译器：`Flag.Dwarf` 默认为 `GOARCH != "wasm"`（`cmd/compile/internal/base/flag.go:166`），`GenDwarfInl` 默认为 2（`flag.go:173`）；没有 DWARF 时内联记录也关闭（`gc/main.go:152` ~ `159`）。cmd/go 只在 `OmitDebug`、plan9 或 wasm 时传 `-dwarf=false`（前提 4）。
- 链接器：`dwarfEnabled` 在没有 `-w`、ELF 内部链接时为真（`cmd/link/internal/ld/dwarf.go:1720` ~ `1745`）。每个编译单元中、属于 `unit.Textp` 的函数，取它的 DWARF 信息辅助符号写入（`dwarf.go:2011`、`1798` ~ `1813`）；被引用的抽象函数 DIE 一并写入（`dwarf.go:1834` 起）。
- 压缩：默认压缩 DWARF 段；ELF 上保留原段名并置 `SHF_COMPRESSED`（`dwarf.go:2455` ~ `2464`）。`debug/elf` 支持 zlib 与 zstd（`debug/elf/file.go:135` ~ `160`）。

### 4.2 内联记录如何产生

- 每个函数经 `Flushplist` 生成 DIE（`cmd/internal/obj/plist.go:173`、`obj/dwarf.go:344` ~ `394`）；Go 编译的函数（包括 ABI 包装）都有 DIE，汇编的函数没有内联。
- `assembleInlines`（`cmd/compile/internal/dwarfgen/dwinl.go:29` ~ `202`）逐条读函数的指令：
  - 指令位置的内联索引（`posInlIndex`，`dwinl.go:292` ~ `301`）非负，就登记该内联调用及其全部祖先（`dwinl.go:41` ~ `50`）；
  - 再按相邻指令的内联索引切分 PC 范围（`dwinl.go:150` ~ `173`），长度为零的范围不记（`addRange`，`dwinl.go:313` ~ `315`）；
  - 由内联产生的变量也会登记调用（`dwinl.go:70` ~ `77`）。
- 写出时，自身没有 PC 范围、且后代都为空的内联调用被省略（`cmd/internal/dwarf/dwarf.go:1113` ~ `1131` 的 `isEmptyInlinedCall`，`1135` ~ `1153` 的 `inlChildren`）；其余都写为 `DW_TAG_inlined_subroutine`，抽象来源指向被内联函数的抽象 DIE（`dwarf.go:1289` ~ `1316`）。

### 4.3 结论

**可以由源码推出的（限定形式）**：在 §1.3 的前提下，若调用者是 A 中的函数，而且至少有一条长度非零的指令、其位置归属于被内联的 F 或 F 的内联后代，那么 A 的 DWARF 中有一个以 F 为抽象来源的 `DW_TAG_inlined_subroutine`。

**不能由源码推出的（强形式）**：“F 的代码以内联的方式在 A 中，就一定有这样的记录”。源码中有两类遗漏：

1. **空实例**：F 的内联体没有留下任何指令（例如全部被常量折叠或消除）。这时 F 的代码确实不在该调用者中，不是错误。
2. **归属的丢失**：内联体产生的计算，若在 SSA 阶段与调用者的等价计算合并（例如公共子表达式消除保留了调用者的值），或由编译器生成、位置不带内联索引的指令实现，就归属于调用者的位置，不产生 F 的记录。本文没有从源码排除这种情况，也没有证明它对 T0 不会发生。

所以计划 §3.2 的前提“默认构建对每个内联实例都写出这一项”在强形式上**未成立**；按这个前提定义的 §7 第 7 条（“F 的代码不在产物 A 中”）不能只由符号表与 DWARF 得出。

- 对 T0 的一个可能的补充论证：每个目标的函数体都含有不能被消除或与调用者合并的调用（`X__syscall*`、`___syscall_cp`、`exec.Command`、`gosignal.*` 等），只要 F 被内联，这些调用（或被一并内联的被调函数的指令）就归属于 F 的内联子树。这需要对每个目标逐一的静态论证，也改变了 absent 的含义；本文只把它写成提案 M-2，不据此宣称通过。
- 适用范围：只适用于默认构建（DWARF 开启、没有 `-N -l`、没有 `-dwarf=false`）的 K；剥离或 `-w` 的产物没有这项证据（计划 §7 已判 T）。

## 5. S-6：缓存、输出的转发与归档

### 5.1 跳过链接的条件

链接动作先调用 `useCache(a, linkActionID, a.Package.Target, …)`（`cmd/go/internal/work/exec.go:1630`）。`go build -o <out>` 时 `a.Package.Target` 就是 `<out>`（`build.go:545`）。

| 情形 | 结果 | `-x` 中的痕迹 |
|---|---|---|
| `<out>` 已存在，且它的 build ID 以本次的 action ID 开头（`buildid.go:484` ~ `495`） | 跳过链接，不重放输出 | 没有链接命令 |
| 主包编译动作的特例：`<out>` 的 build ID 中记录的链接 ID 与重新计算的一致（`buildid.go:496` ~ `531`）；它与上一种情形在同一次构建中同时成立 | 主包的编译与链接都跳过，并从缓存**重放**之前保存的编译输出与链接输出（输出为空时不重放） | 没有链接命令；重放时有 `cat <GOCACHE 中的文件>  # internal`（`showStdout`，`buildid.go:627`、`638`） |
| 缓存中有链接动作的输出文件（`buildid.go:564` 起） | 跳过并重放 | 同上；`go build` 默认不缓存可执行文件（`buildid.go:756` 起只在 `CacheExecutable` 时），所以在 K 的 `go build` 上不预期出现 |
| 以上都不成立 | 运行链接（`exec.go:1657`），并开始保存输出（`buildid.go:623`） | 有链接命令 |

- 所以“输出非空”不能证明链接实际运行：缓存中保存的旧的链接输出（`updateBuildID` 在每次链接之后以主包的 action ID 保存，`buildid.go:676` ~ `688`）可以在跳过时被重放。计划 §7 第 2 条“`-x` 的记录中没有 A 或 B 的链接命令 → T(identity)”仍然成立，并须同时拒绝重放的痕迹（提案 M-5）。
- 计划要求的“新建的空 `GOCACHE`、新的输出目录”使前三种情形都不会发生；对照 C10 正是要构造前两种情形（§8.2）。对 B 重复构建时，旧的 `-dumpdep` 会被重放，所以 C10 必须同时检查链接命令的缺失与重放的痕迹。

### 5.2 `-dumpdep` 输出的转发

1. 链接器把每一行以 `fmt.Printf` 写到自己的标准输出（`cmd/link/internal/ld/deadcode.go:375`）。
2. cmd/go 运行工具时，把标准输出与标准错误设为**同一个**缓冲区（`work/shell.go:630` ~ `631`）；`os/exec` 在两者为同一写入端时让子进程的两个描述符共用一个管道（`os/exec/exec.go:580` ~ `585`）。所以链接器的标准错误（警告等）按写入的先后与 `-dumpdep` 交错。
3. 工具成功、输出非空时，`reportCmd`（`shell.go:444` ~ `531`）：
   - 若不以换行结尾，补一个换行（`shell.go:479` ~ `481`）；
   - 把行首或空白之后出现的工作目录替换为 `$WORK`（`shell.go:484`，`replacePrefix` 见 `shell.go:535` 起）；
   - 链接时 `dir` 为 `"."`（`gc.go:664`），目录的相对化实际上不改写内容；
   - 在前面加一行 `# <导入路径>`（`cmdError.Error`，`shell.go:557` ~ `567`；链接的描述是主包的导入路径，`gc.go:676`），即 `# github.com/XKHoshizora/specmetry/cmd/specmetry`。
4. 因为链接动作在运行之前已开始保存输出，这段文字先存入 `a.output`（`shell.go:523` ~ `525`），在链接动作返回时由 `flushOutput` 一次写出（`exec.go:1633`、`buildid.go:648` ~ `651`）；默认的打印目的地是 `os.Stderr`（`load/printer.go:49` ~ `54`）。在它之前，`-x` 已打印链接命令与 `go tool buildid -w … # internal`（`buildid.go:664` ~ `666`）；之后是安装到 `-o` 的命令等。

结论：

- **计划 §5.1 第 5 步“标准输出单独写入文件”的前提不成立**：`go build` 的标准输出中没有 `-dumpdep`，按原写法采集到的文件为空（按计划 §7 第 3 条即 T(format)）。
- `-dumpdep` 在 `go build` 的标准错误中，是一段以 `# <导入路径>` 开头、与链接器标准错误合并、经 `$WORK` 改写的文字；它的结束没有显式的标记。cmd/go 在这条路径上不截断输出（整段在内存中保存，一次写出）。
- 需要调整采集与解析的方法：提案 M-1。

### 5.3 可取得的归档与身份

- 每个新编译的包，`-x` 记录编译命令（`-p <导入路径>`、`-o $WORK/b<NNN>/_pkg_.a`），之后在存入缓存时打印 `cp $WORK/b<NNN>/_pkg_.a <GOCACHE>/<xx>/<hex>-d # internal`（`buildid.go:742` ~ `745`）。
- 缓存文件名中的 `<hex>` 是文件内容的 SHA-256（`cmd/go/internal/cache/cache.go:564` ~ `575` 计算 output ID，`117` ~ `119`、`319` ~ `320` 组成路径），所以可以对读取的归档重算哈希、核对它与文件名相符。这支持关联与追溯，不认证来源。
- `$WORK` 在命令返回之后被删除，读取的应是这份 `GOCACHE` 中的文件（计划 §6 的 S-6 所问）。
- 存入缓存失败时，cmd/go 不打印 `cp` 行，也不报错（`buildid.go:744` ~ `745` 只在成功时打印）；缺少某个包的 `cp` 行 → 不能定位归档 → T(identity)。
- 链接的 `importcfg.link` 以 `cat >… << 'EOF' # internal` 的形式记录（`shell.go:263` ~ `273`、`exec.go:1670` ~ `1688`）；这条命令以 `cat ` 开头，工作目录**不**替换为 `$WORK`（`shell.go:357`），所以其中是绝对路径，其他命令中是 `$WORK`。比较时须按这两种形式归一。

## 6. S-7：可达文本与 ELF 符号表

- 产物的文本列表 `ctxt.Textp` 在 `linksetup` 的末尾由 `AssignTextSymbolOrder` 组成（`lib.go:942` ~ `953`；`loader/loader.go:2792` 起）：先取已在列表中、且可达的外部符号（K 上是 FIPS 的两个标记），再取 Go 对象文件中**全部可达的文本符号**（`loader.go:2811` ~ `2850`），每个符号至多一次（dupok 的符号经 `assignedToUnit` 去重）。
- K 上，之后对 `ctxt.Textp` 的改动只有：
  - 按符号类型的稳定排序，使 FIPS 的文本集中在两个标记之间（`data.go:2706` ~ `2708`），不增删符号；
  - 跳板与 build ID 的插入都不适用（§2.2）。
- `linksetup` 之后被置为可达的文本符号不会进入列表；K 上只有 `setArchSyms` 再次标记已经是根的 `runtime.unreachableMethod`（§2.2）。
- 写出：没有 `-s` 时 `asmElfSym` 写符号表（`elf.go:2176` ~ `2179`）。`genelfsym`（`symtab.go:192` ~ `268`）：
  - 按名字写 `runtime.text`；
  - 为 `ctxt.Textp` 中的每个符号调用 `putelfsym`（类型 `STT_FUNC`），不按“不在符号表中”的属性过滤；
  - 最后写 `runtime.etext`（`STEXTEND` 属于文本，`sym/symkind.go:191` ~ `193`）。
  - 数据的循环只处理 `SELFRXSECT` 及之后的类型（`symtab.go:250`），文本类型（`sym/symkind.go:47` ~ `51`）不会以数据重复写出。
- `putelfsym`（`symtab.go:79` ~ `184`）对每个符号恰好在一个绑定的轮次中写出（`asmElfSym` 先 `STB_LOCAL` 后 `STB_GLOBAL`，`symtab.go:284` ~ `290`）。缺少段时报告错误（`symtab.go:96` ~ `103`），链接失败而不是静默遗漏。
- 名字：`SymExtname`（K 上的 Go 符号即名字本身，前提是没有 cgo 导出与动态导入）经 `mangleABIName` 改写（§3.2），非动态链接时把 `·` 换成 `.`（`symtab.go:159` ~ `162`）。内容寻址的文本符号与文件局部的符号写为 `STB_LOCAL`，可以同名。

结论：在 K 与 §1.3 的前提下，A 的 ELF 符号表中的 `STT_FUNC` 条目，恰好是 `runtime.text`、`runtime.etext` 与 `linksetup` 结束时可达的全部文本符号（含 §2.3 的合成符号），每个一次，名字按上述规则改写；可达的文本符号不会被遗漏。遗漏只发生在：剥离（`-s`，计划已判 T）、外部链接（不属于 K），或链接器报错（链接失败，T(tool)）。

## 7. S-1 与 S-5 的补充核对

- **S-5**：`-dumpdep` 的标志只在 `main.go:76` 定义、在 `deadcode.go:366` 读取，链接器中没有其他读取；它只决定是否打印，不改变可达性。维持原结论。
- **S-1 的更正**：
  1. 原文“每个符号至多作为 `to` 出现一次”不准确。类型符号在转为接口使用时会被清除可达、再标记一次（§2.1 的三处；`deadcode.go:272` ~ `273` 的注释写明“至多两次”），所以可以出现两次（第二次带 ` <UsedInIface>`）。文本符号不会被清除，至多一次。
  2. 同一个**名字**可以属于不同的符号：ABI0 与 ABIInternal 的同名对（§3.2）、同名的内容寻址闭包（§3.1）。所以 `to` 的名字重复不能当作格式错误，也不能假设一个名字对应一个符号。
  3. 文本符号的名字可以带 ` <ReflectMethod>`（`deadcode.go:386` ~ `388`）。
  4. K 上的根行：`_ -> _rt0_amd64_linux`（入口符号，`lib.go:421`；按两个版本查找，`deadcode.go:103` ~ `110`）、`_ -> runtime.unreachableMethod`、`_ -> go:main.inittasks`（`inittask.go:44`、`deadcode.go:131` ~ `132`）。计划 §7 第 3 条“缺少预期的根行”可以据此写定（提案 M-3）。
  5. `runtime.mapinitnoop` 可以可达而不出现（§2.2）。

## 8. 配置 K 的方法输入清单

### 8.1 目标与映射

- 目标函数、所在文件与汇编包装：§3.3 的表。实际编入的来源：§1.2（静态求值）。映射与未知的情况：§3.4。
- 判定 present 的证据：主符号、ABI0 包装、闭包与包装、汇编包装 Y 的两个符号、内联实例（§3.4 的五种）。
- absent 在现方法下不可得出（§4.3），待提案 M-2 的决定。

### 8.2 对照：预期、命中条件与证据类别

证据类别：**S** = 本文的静态源码依据；**R** = 须运行才能取得。对照程序是一个单独的 test-only 小模块，只在诊断中构建；下面只描述拟用的构造，不提交代码。

| # | 拟用的构造 | 预期 | 命中条件（不满足 → T(control)，不算覆盖） | 证据类别 |
|---|---|---|---|---|
| C1 | `main` 直接调用一个 `//go:noinline` 的函数 | present（主符号） | 主符号在对照程序的归档中（`go tool nm`） | 预期 S（§6）；命中 R |
| C2 | 只在包级变量的初始化中调用的 `//go:noinline` 函数 | present | 同上；`-dumpdep` 中它的 `from` 是该包的初始化函数 | S；R |
| C3 | 只存入全局函数表、经回调调用的函数 | present | 同上 | S；R |
| C4 | 只经接口与 `reflect.Value.MethodByName(name)` 到达的导出方法，`name` 不是编译期常量 | present（`reflectSeen`） | 调用 `MethodByName` 的函数在 `-dumpdep` 中带 ` <ReflectMethod>`；`name` 为常量时走的是 `R_USENAMEDMETHOD`（`deadcode.go:248` ~ `254`），不是本对照 | S（`deadcode.go:143`、`462`）；R |
| C5 | 只有一个调用者、会被内联的小函数；它的函数体调用一个 `//go:noinline` 的函数。**拟**使至少一条调用指令归属于内联实例：这是构造想达到的条件，不由“含有副作用的调用”保证（N-132-2，§170） | present（DWARF 的内联实例；自身的符号可以不存在） | 两项都须命中，缺任一项 → T(control)：(1) 独立的内联诊断：带 `-gcflags=-m` 的另一次构建中有这次调用的 `inlining call to …`；(2) A 本身的 DWARF 中有以该函数为抽象来源、PC 范围非空的 `DW_TAG_inlined_subroutine`。诊断不能代替 A 本身的命中 | 分开计：S 只是 §4.2 的限定形式（有归属的指令才有记录）；“调用指令保留内联归属”是构造的预测，不是 S；命中是 R。注：没有那条 `noinline` 调用时，空实例不产生记录（§4.3 第 1 类），对照会因构造而失败。这个对照与其他健康对照的成功，都不构成全部目标的归属完备性证明 |
| C6 | Go 函数 F 只被一个汇编 `TEXT ·W(SB)` 以 `CALL ·F(SB)` 调用，W 在 Go 中声明并被调用（与 libc 的 `Y*` 同形） | present：ELF 中有 F、`F.abi0`、W 与 `W.abi0`（§3.2） | 汇编源文件中 W 确实 `CALL` F；四个名字都在对照产物中 | S；R |
| C7 | 只被一个不可达的函数调用的函数 | absent | 两个函数的主符号都在归档中（确认它们被编译，而不是根本不存在） | S（§2、§6）；R |
| C8 | 从未被引用的导出函数 | absent | 主符号在归档中 | S；R |
| C9 | B 的 `-dumpdep` 段落（按 M-1 取出）的两个副本：(i) 删去最后一行；(ii) 删去 A 的某个文本符号名字作为 `to` 的**全部**行 | (ii)：T(consistency)；(i)：按原计划的条件（N-128-1） | (ii) 所选的名字在 A 的符号表中只属于一个文本符号，或删去该名字的全部行（§7 第 2 点：同名可以有多行） | S；R |
| C10 | 第一次构建之后保留 `<out>` 与 `GOCACHE`，以同一命令再构建一次 | T（`-x` 中没有链接命令） | 第二次的 `-x` 中确实没有链接命令（§5.1 的前两种情形）；若链接实际运行 → T(control) 并记录；出现重放的 `cat … # internal` 行时同样 T（M-5） | S（§5.1）；R |

真实产物中的正对照（预期 present；不是 present → T）：

| 名字（ELF / `-dumpdep`） | 定义 | 到达方式 |
|---|---|---|
| `main.main` | `cmd/specmetry` | 运行时的入口 |
| `modernc.org/sqlite.(*Driver).Open` | `modernc.org/sqlite@v1.60.1/driver.go:287` | `database/sql` 经接口调用驱动；仓库在 `internal/store/sqlite/store.go:20` 导入驱动并经 `sql.Open("sqlite", …)` 使用（如 `readonly.go:173`） |
| `modernc.org/sqlite.newConn` | `conn.go:56` | 由打开连接的路径调用 |
| `modernc.org/libc.Xmalloc` | `modernc.org/libc@v1.77.1/mem_musl.go:20`（K 编入的定义；其他 `Xmalloc` 在 K 上不编入） | 由转译的 SQLite 调用 |

命中条件：各自的主符号在同一次构建的归档中。它们在 A 中实际以哪种形式出现（主符号、内联或包装），须运行才能知道（R）。

### 8.3 身份的取得与核对

| 对象 | 取得 | 核对 |
|---|---|---|
| Go 工具链 | 按链接计划 §8.3 的身份核对（`LookPath`、`EvalSymlinks`、`VERSION`、`GOVERSION`、`runtime.Version()`） | 不变 |
| `compile`、`asm`、`link` | `-x` 记录中的绝对路径（`$GOROOT/pkg/tool/linux_amd64/…`） | 路径位于已核对的 `GOROOT` 之下；记录 SHA-256 |
| 包的归档 | §5.3 的 `-p`、`-o` 与 `cp … # internal` 三者关联 | 对缓存文件重算 SHA-256，须等于文件名中的十六进制串；每个必需的包（至少 `modernc.org/libc`）都有 `cp` 行 |
| 前端 | 链接计划 §2 的 `web/dist` 清单 | 不变 |
| 产物 A、B | SHA-256、`go version -m`；ELF 的检查：没有 `PT_INTERP` 与 `.dynamic`（前提 2）、有 `.symtab`、有 DWARF 段（含 `SHF_COMPRESSED` 的解压） | 不符 → T(identity) 或 T(consistency) |
| `-dumpdep` | 按提案 M-1 从 B 的标准错误中取出的段落 | 段落的字节数与 SHA-256 写入证据；它的标题行与前置的链接命令按 M-1 核对 |

### 8.4 仍须运行才能取得的证据

- K 的实际文件与包集合（`go list -deps -json`），与 §1.2 的静态求值核对。
- 前提 1 ~ 4 的 `-x` 记录与产物检查；链接命令的完整参数（包括 cmd/go 是否追加 `-X=runtime.godebugDefault=…`，`gc.go:628` ~ `630`，它只影响数据）。
- 每个目标在归档中的实际符号（主符号、ABI0 包装、闭包），以及 `runtime.mapinitnoop` 是否出现。
- 全部对照的命中与结果；A 与 B 的对应（计划 §5.2）。
- 链接计划 §8.3 的 runner 前提（subreaper、`__WALL`、pidfd 等）与身份核对。

## 9. 方法调整的提案（未采纳，交独立审查）

| # | 起因 | 提案 | 影响 |
|---|---|---|---|
| M-1 | S-6：`-dumpdep` 不在标准输出上 | 采集 B 的 `go build` 的**标准错误**为一个文件；在其中取出段落：它紧接在 B 的链接命令与 `go tool buildid -w … # internal` 之后，以恰好一行 `# github.com/XKHoshizora/specmetry/cmd/specmetry` 开始，之后连续的、能按 `from -> to` 解析的行属于段落；第一行不能解析的行结束段落，它必须是已知的 `-x` 命令形式（安装与清理的命令），否则 T(format)；段落中出现任何不能解析的行（例如链接器的警告）也是 T(format)；名字中出现 `$WORK` → T(format)。另一选项：以 `-toolexec` 的包装原样采集链接器的标准输出；它新增一个工具，须另行设计与授权，本文不推荐先行采用 | 改写计划 §5.1 第 5 步与 §7 第 3 条；对照 C9 改为作用于取出的段落；不改变以产物 A 为主证据的形式 |
| M-2 | S-4：强形式未成立 | 把 absent 的含义收窄为：“A 中没有 F 及其派生的符号、没有 F 的汇编包装，也没有任何指令归属于 F 的内联子树（本方法、配置 K）”；并为每个目标增加一项执行之前的静态论证：函数体含有不能被消除或与调用者合并的调用。论证不成立的目标 → T，不得 absent | 改变 §7 第 7 条的结论文字与它对 Q-B3-1 (a) 的意义：absent 不再等于“代码不在产物中”。这是否足以支持 (a)，由负责人决定；不推出 F 的能力在运行时不可达 |
| M-3 | S-1、S-3、S-7：名字的差异与重复 | 比较之前的规范化：ELF 名字去掉 `mangleABIName` 加上的 `.abi0`（只在存在同名的 ABIInternal 文本符号时），`·` 视同 `.`；`-dumpdep` 的名字去掉 ` <UsedInIface>`、` <ReflectMethod>`；`to` 的名字可以重复；根行写定为 §7 第 4 点的三行 | 改写计划 §7 的第 3、5 条与 §4.2 的映射；C9 (ii) 按 §8.2 构造 |
| M-4 | S-2：合成的文本符号 | 计划 §7 第 5 条的清单固定为 `runtime.text`、`runtime.etext`、`go:textfipsstart`、`go:textfipsend`、`runtime.mapinitnoop`；A 中任何其他不是 `to` 的文本符号 → T(consistency) | 只是审计结果的落实；`runtime.mapinitnoop` 无条件列入，它不是目标 |
| M-5 | S-6：跳过时的重放 | 在计划 §7 第 2 条之外，`-x` 的记录中出现以 `cat ` 开头、以 `# internal` 结尾、读取 `GOCACHE` 中文件的命令（输出的重放）→ T(identity) | 收紧既有的规则；C10 的命中条件随之写定 |

- 这些提案都不改写已关闭的收尾协议（链接计划 §8.3、§8.4），不增加时间、资源或步骤。
- M-1 与 M-2 是执行路径继续的前提；M-3 ~ M-5 是解析与分类的细化。

## 10. 未完成的条件与待决清单

**本审计之后仍未完成**（都不是已通过）：

- 本文的独立审查，以及 M-1 ~ M-5 的审查与决定。
- §8.4 中须运行才能取得的全部证据。
- 链接计划 §8.6 的条件：驱动程序的代码与测试、场所与 workflow、runner 的前提与身份的取证。

**依赖本审计结论的决定**：

| 事项 | 依赖 |
|---|---|
| Q-B3-1 (a) 的可行性验证是否继续 | M-1、M-2 的决定；M-2 若不获采纳，按本方法不能得到 absent |
| F-2（以产物为主证据、`-dumpdep` 为交叉核对） | M-1、M-3、M-4 |
| F-4（S 审计） | 本文的独立审查 |
| F-1、F-6（场所、驱动程序、时间线与资源） | 只在决定继续 Q-B3-1 (a) 的验证之后才有意义 |

**可以独立决定的**（不依赖本审计）：

- E-8 (b)（生产构建的 `CGO_ENABLED`）；注意：若不固定为 0，K 随之改变，本文须重审。
- X1 ~ X4、E-1 ~ E-10、U 项、F-7、F-8。
- F-5（T0 的完整性方法），以及 Q-B3-1 的 (b)、(c)。

本文不批准依赖例外，不接受新的 L，不决定生产的 CGO、X / E / U、F-8、场所或资源；Q-B3-1 (a) 本轮只获审计准备的授权。

## 附录 A：S-2 的逐处清单

固定提交 `2aae100` 的 `src/cmd/link/internal/` 之下；种类：D = `SetAttrReachable(…, true)`，R = `SetReachable(true)`，C = `CreateSymForUpdate`（`loader/symbolbuilder.go:59` 置为可达）。共 126 处，其中 `deadcode.go:362` 是 `mark` 本身。逐处的判定按 §2.2 的类别。

| 文件 | 调用点（行号、种类、所在函数） |
|---|---|
| `arm/asm.go` | 404 D `trampoline`; 611 R `addpltreloc` |
| `arm64/asm.go` | 1293 D `gensymlate`; 1390 D `trampoline` |
| `ld/data.go` | 952 C `windynrelocsyms`; 1276 C `addstrdata`; 1299 C `addgostring`; 1313 C `addinitarrdata`; 1352 C `Init`; 1499 C `fixZeroSizedSymbols`; 1520 C `fixZeroSizedSymbols`; 1524 C `fixZeroSizedSymbols`; 1527 C `fixZeroSizedSymbols`; 1531 C `fixZeroSizedSymbols`; 1539 C `fixZeroSizedSymbols`; 1543 C `fixZeroSizedSymbols`; 1546 C `fixZeroSizedSymbols`; 1550 C `fixZeroSizedSymbols`; 1555 C `fixZeroSizedSymbols`; 1560 C `fixZeroSizedSymbols`; 1565 C `fixZeroSizedSymbols`; 2608 C `textbuildid`; 2625 C `buildinfo`; 2655 C `buildinfo`; 2921 C `assignAddress`; 3352 R `AddTramp` |
| `ld/deadcode.go` | 344 D `mapinitcleanup`; 362 D `mark` (mark) |
| `ld/dwarf.go` | 227 C `writeabbrev`; 316 D `newdie`; 501 D `dotypedef`; 1083 D `importInfoSymbol`; 1707 C `writegdbscript`; 1807 D `dwarfVisitFunction`; 1811 D `dwarfVisitFunction`; 2205 C `dwarfGenerateDebugSyms`; 2207 R `dwarfGenerateDebugSyms`; 2214 R `dwarfGenerateDebugSyms`; 2298 D `dwarfGenerateDebugSyms`; 2465 D `dwarfcompress`; 2554 R `writeDebugMiscSecHdr` |
| `ld/elf.go` | 1023 C `elfdynhash`; 1074 C `elfdynhash`; 1077 C `elfdynhash`; 1117 C `elfdynhash`; 1130 C `elfdynhash`; 1149 C `elfdynhash`; 1437 C `addgonote`; 1468 C `doelf`; 1478 C `doelf`; 1486 C `doelf`; 1490 C `doelf`; 1499 C `doelf`; 1504 C `doelf`; 1507 C `doelf`; 1514 C `doelf`; 1523 C `doelf`; 1526 C `doelf`; 1529 C `doelf`; 1533 C `doelf`; 1598 R `doelf`; 1630 C `doelf`; 1632 R `doelf` |
| `ld/fips140.go` | 157 C `loadfips`; 168 C `loadfips` |
| `ld/ld.go` | 229 D `PrepareAddmoduledata`; 235 D `PrepareAddmoduledata`; 248 D `PrepareAddmoduledata` |
| `ld/lib.go` | 136 D `mkArchSym`; 143 D `mkArchSymVec`; 750 C `loadWindowsHostArchives`; 753 R `loadWindowsHostArchives`; 869 D `linksetup`; 930 D `linksetup`; 938 D `linksetup`; 2950 C `defineInternal` |
| `ld/macho.go` | 489 R `domacho`; 496 R `domacho`; 502 R `domacho`; 511 R `domacho`; 517 R `domacho`; 522 R `domacho`; 530 R `domacho`; 1073 C `machosymtab`; 1074 C `machosymtab`; 1406 C `machoDyldInfo`; 1407 C `machoDyldInfo`; 1504 C `machoCodeSigSym` |
| `ld/pcln.go` | 64 D `addGeneratedSym`; 197 D `genInlTreeSym`; 974 D `pclntab`; 1091 D `findfunctab` |
| `ld/pe.go` | 1269 C `Peinit`; 1278 C `Peinit`; 1297 D `Peinit`; 1393 C `initdynimport`; 1401 C `initdynimport` |
| `ld/seh.go` | 32 C `collectSEHAMD64` |
| `ld/symtab.go` | 371 C `textsectionmap`; 455 C `symtab`; 460 C `symtab`; 466 C `symtab`; 474 C `symtab`; 553 C `symtab`; 561 C `symtab`; 565 C `symtab`; 679 C `symtab`; 703 C `symtab`; 748 C `symtab`; 762 D `symtab`; 788 C `symtab` |
| `ld/xcoff.go` | 1182 R `adddynimpsym`; 1186 C `adddynimpsym`; 1286 C `doxcoff`; 1333 R `doxcoff` |
| `loong64/asm.go` | 798 D `trampoline` |
| `ppc64/asm.go` | 115 C `genstub`; 237 C `genpltstub`; 479 D `rewriteABIFuncReloc`; 1260 C `gentramp`; 1693 C `ensureglinkresolver` |
| `riscv64/asm.go` | 238 R `genSymsLate`; 369 R `elfsetupplt`; 421 R `addpltsym`; 723 D `trampoline` |
| `x86/asm.go` | 74 C `gentext` |
