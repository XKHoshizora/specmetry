# WP2.5 第三批：原 Q-B3-1 (a) 强命题的限定证明准备（T0、配置 K）

> 状态：**证明准备（纯文档），待独立审查**（2026-11-05；review-resolution §170、§171；[链接验证计划](wp2.5-c9-link-proof-plan.md)；[链接方法审计](wp2.5-c9-link-method-audit.md)）。
>
> - 本文只回答一个问题：在固定的 Go 1.27.3、原生产构建形态与配置 K 下，能否为 T0 建立足以支持“目标代码不在产物中”的完备性论证。
> - 本文不批准依赖例外，不接受新的 L，不选择 Q-B3-1 的 (b) / (c)，不批准更弱的保证，不决定生产的 CGO、X / E / U、F-8、场所或资源，也不授权执行。
> - 编写本文时没有运行任何 Go 命令、构建、链接、验证分析器、驱动程序或实验，没有安装工具链。证据是固定源码的只读文本、文本检索、Git 与只读的哈希核对。检索与闭包计算用临时的只读脚本完成，脚本不提交（§3.1 写明算法）。
> - 凡写“须运行”的，都不是本文的证据；凡写“预期”的，都只是推测，不是结论。

## 0. 结论

### 0.1 答案

**部分能。** 对 T0 的 21 个函数：

| 组 | 目标 | 状态 | 依据 |
|---|---|---|---|
| 甲：不可内联 | `Xfexecve`、`Xwait4`、`Xpclose`、`Xexecle`、`X__execvpe`、`Xexecl`、`Xexeclp`、`X__sigaction`（8 个） | **已有可审查的充分论证（限定）**；`X__sigaction` 另以 K 的文件选择为前提（丙） | 函数体含 `defer`，固定编译器不内联它们（§2.3）。F 的函数体只编译进 F 自己的文本符号，F 的代码若在 A 中，A 的符号表必有 F 的名字或其派生名字（§4.1）。另有宿主集合的独立论证（§3.2；`X__sigaction` 除外） |
| 乙：可能被内联，宿主集合小 | `Xsystem`、`Xexecve`、`Xexecv`、`Xexecvp`、`Xexecvpe`、`Xvfork`、`Xwaitid`、`Xwaitpid`、`Xwait`、`Xwait3`（10 个） | **已有可审查的充分论证（限定）** | 宿主集合（§3）在两个模块的全部非测试文件上计算，不依赖文件选择，只含 libc 的函数与汇编包装（§4.2） |
| 丙：依赖文件与包的选择 | `Xsigaction`、`___libc_sigaction`（另：甲组的 `X__sigaction`） | **充分论证另以 K 的文件与包选择为前提**（须 `go list` 核对） | 按 K 的静态文件选择（与编入的包集合），宿主集合为 2、26（`X__sigaction` 为 4）个名字；不按选择时，非 K 文件中的同名函数使集合扩大到不可用（§4.3） |
| 丁：信号处置的入口 | `Xsignal`（1 个） | **未成立** | 宿主中有方法与通用入口（`(*TLS).Free`、`X__syscall2`、`X__syscall3` 等），方法包装的名字规则没有写定；而且静态源码表明，只要 `(*TLS).Free` 的代码或 `X__syscall2`、`X__syscall3` 自身的符号在 A 中，`Xsignal` 的代码就在 A 中（§5） |

- 这些论证**不依赖 DWARF 的内联记录**，所以不需要 S-4 的强形式，也不需要方法提案 M-2 的弱 absent；absent 的含义仍是原命题“F 的代码不在产物 A 中”，没有收窄。
- 甲、乙、丙的论证只给出**充分条件**：A 的符号表中没有规定的名字 → F 的代码不在 A 中。规定的名字出现时，结论不是 absent。是否真的没有这些名字，**须运行**（读取 A 的符号表）才知道；本文不预测。
- 这改变了原计划 §7 第 6、7 条判定 absent 的依据（原依据是符号表加 DWARF），是方法的变更。本文把它写成提案 **H-1**（§8），交独立审查，不自行采用。

### 0.2 对 Q-B3-1 (a) 的意义

- 对**启动与回收**一族（exec、fork、wait 与 `Xsystem`，共 17 个：甲组除 `X__sigaction` 之外的 7 个与乙组的 10 个），原 (a) 的强命题有继续定稿的技术基础（限定）：论证、前提与须运行的证据都已列明。
- 对**改变信号处置**一族（4 个），就能力整体而言，原 (a) 目前没有技术基础：
  - `Xsigaction`、`X__sigaction`、`___libc_sigaction` 有以文件与包的选择为前提的论证（丙）；
  - 但 `Xsignal` 未成立，而且静态源码给出它很可能在产物中的依据（§5.2）；
  - T0 之外，K 编入的 libc 还有直接发送信号与改变处置的位置（`Xabort`、`resetSigDfl`、`(*TLS).raiseSignal`、`(*TLS).killSignal`），不在 T0 中（§6.2）。
- 即使 T0 的强命题全部成立，也不能推出能力整体不可达，也不能推出依赖例外可以批准（§6）。F-5（目标完整性）仍独立未决。
- 需要另作决定的方向列在 §7.2，本文不选择。

## 1. 固定的对象与方法的边界

| 对象 | 固定值 | 核对 |
|---|---|---|
| 仓库 | master `9c5860676de2b714f42a7210dca887bc61c6ec41`（树 `95ad173b4724975707b483fc8033601cfdf2dc06`） | Git |
| Go | `go1.27.3`，提交 `2aae1000ea1d4b28923b96db28acf4e3b99d6aff`，树 `09275a3482ffe831661a72c202c6261e0eb6ae7f` | 与链接方法审计 §1.1 相同的只读浅取；本轮重新核对了提交、树与 `VERSION` |
| 模块 | `go.mod` 列出的全部 23 个模块 | 按 go.sum 的方法独立重算本机模块缓存的 dirhash，23 个都与 go.sum 一致（附录 B） |
| 配置 K | 链接验证计划 §2：`linux/amd64`、`CGO_ENABLED=0`、`GOEXPERIMENT` 为空、`GOAMD64=v1`、`GOFLAGS` 除 `-mod=readonly` 之外为空 | 须运行（§7.1 的 R1） |
| 构建形态 | 原生产构建形态：`go build -o <out> ./cmd/specmetry`，没有 `-trimpath`、`-ldflags`、`-gcflags`、`-pgo` 或构建标签 | 同上；本文的论证不使用改变构建的产物 |

- 本文不以关闭内联、改变编译参数或更换工具版本的产物代替原产物 A。
- 文件选择是对 go/build 规则的静态求值（与链接方法审计 §1.2 相同），不是 `go list` 的结果。丙组依赖它，甲、乙组的宿主集合论证不依赖它（§3.2）。

## 2. 通用引理：固定流水线中代码与名字的关系

### 2.1 L1：产物的机器码只在文本符号中

- 链接器写 `.text` 节时，只写 `ctxt.Textp` 中的符号与填充：
  - `cmd/link/internal/ld/asmb.go:41` ~ `46` 对 `.text` 调用 `CodeblkPad`；
  - `CodeblkPad` 只遍历 `ctxt.Textp`（`data.go:1020` ~ `1021`）；
  - amd64 不覆盖 `Asmb`，填充为 `0xCC`（`amd64/obj.go:49`）。
- K 上没有宿主对象（链接方法审计 §1.3 前提 2）。`Textp` 由 Go 对象文件中可达的文本符号与链接器合成的四个标记组成（审计 §2.3、§6）；合成的标记不含来自包代码的指令。
- 每个文本符号的内容只来自一个函数：
  - 编译器对每个 `ir.Func` 单独生成指令并提交：`ssagen.Compile`（`ssagen/pgen.go:303` ~ `329`）→ `pp.Flush`（`objw/prog.go:110` ~ `112`）→ `obj.Flushplist`；
  - `Flushplist` 按 `ATEXT` 把指令分给各自的符号，再逐个符号汇编（`cmd/internal/obj/plist.go:24` ~ `56`、`161` ~ `175`）；
  - 汇编文件的每个 `TEXT` 块同样是一个符号，只含该块中写出的指令。
- 例外：内容寻址的符号可以合并（L6）。

### 2.2 L2：一个函数的代码进入另一个函数，只有内联一条路

- 一个函数的机器码来自它自己的 IR（内联之后）。
- 编译器把另一个函数的函数体放进当前函数的唯一方式，是在调用点内联。
  - 内联的入口是 `TryInlineCall`（`inline/inl.go:903` ~ `916`）。被调用者由 `inlCallee` 从调用表达式解析（`inl.go:923` ~ `957`），只接受三种形式：函数名（`PFUNC` 的 `ONAME`）、方法表达式（`OMETHEXPR`）与闭包字面量（`OCLOSURE`）。
  - 解析之前先求 `ir.StaticValue`（`ir/expr.go:879` ~ `946`）。它只穿过 `OCONVNOP`、括号、内联调用的单一结果，以及只赋值一次的**局部**变量（`expr.go:910` 要求 `PAUTO`）；包级变量不穿过。
  - 所以被内联的函数，必须在调用者的源码中，或在某个已被内联进来的函数的函数体中，以名字、方法表达式、闭包或局部的静态值出现。
- 其他可能把代码带进来的步骤，逐一核对：
  - **静态去虚化**：把接口调用改为对具体类型的方法调用，选择子保持原来的方法名（`devirtualize/devirtualize.go:25` ~ `160`，`141` 行以 `sel.Sel` 构造）。所以它只涉及**方法**，而且调用点的源码中有该方法名。
  - **PGO**：
    - 按 profile 的去虚化与内联，只在有 profile 时运行（`inline/interleaved/interleaved.go:30` ~ `37`、`44` ~ `47`）；
    - profile 只来自编译参数 `-pgoprofile`（`gc/main.go:237` ~ `246`）；cmd/go 只在包带有 profile 时传它（`work/gc.go:118` ~ `120`）；
    - `-pgo` 默认为 `auto`（`work/build.go:350`），只取主包目录下的 `default.pgo`（`load/pkg.go:3061` ~ `3076`）；
    - 固定的仓库中没有任何 `*.pgo` 文件（本轮检索）。
    - PGO 的去虚化可以引入源码中没有出现的被调用者，所以它是本论证必须排除的前提（R1）。
  - **方法包装**：编译器生成的方法包装在生成之后自行内联（`noder/reader.go:4190` ~ `4193`），只涉及方法。
  - **ABI 包装**：在内联之后生成，函数体只是对 F 的调用或尾调用（`gc/main.go:259` 内联、`280` ~ `281` 生成包装；`ssagen/abi.go:252` ~ `347`），不再内联。
  - **内建函数（intrinsic）**：按“架构、包路径、函数名”查固定的表（`ssagen/intrinsics.go:2388` ~ `2416`）；`cmd/compile/internal` 的源码中没有 `modernc`，所以表中没有 libc 的函数。
  - **常量条件**：常量为假的 `if` 分支在写出 IR 时就被丢弃（`noder/writer.go:1669` ~ `1687`，`staticBool` 在 `3084` 起）。K 上 `__ccgo_strace` 是常量 false（`straceoff.go:10`，构建约束 `!libc.strace`），所以转译函数开头的跟踪分支及其中的 `defer` 闭包不进入 IR。

### 2.3 L3：含 `defer` 的函数不被内联

- `CanInline` 用 `hairyVisitor` 遍历函数体；遇到 `ODEFER`、`OGO` 或 `OTAILCALL` 即判定“太复杂”（`inl.go:693` ~ `695`），这时不设置 `fn.Inl`（`inl.go:302` ~ `312`）。
- 内联要求 `HaveInlineBody`（`inl.go:914`），其实现首先要求 `fn.Inl` 不为空（`noder/reader.go:3691` ~ `3698`）。
- 常量条件的分支不计入（`inl.go:729` ~ `741`），而且已在写出 IR 时丢弃（L2）。所以只有不在跟踪分支中的 `defer` 起作用。
- 这与 PGO 无关：PGO 改变预算，不改变这条规则。

### 2.4 L4：名字

- 函数、闭包、`go` / `defer` 包装、`-range` 循环体与 ABI 包装的名字由固定源码确定（链接方法审计 §3.1、§3.2）。
  - 闭包与包装的名字以所在函数的名字为前缀；内联产生的闭包，外层取**被内联函数**的名字与包（`ir/func.go:465` ~ `486`）。
  - ELF 中非 ABIInternal 的同名文本符号加 `.abi0`（`symtab.go:826` ~ `845`）。
- 可达的文本符号都写入 ELF 符号表，每个一次（审计 §6，S-7；前提是不剥离）。
- 宿主的派生名字：设 `n` 为 `modernc.org/libc.h`（`h` 为宿主的名字），名字等于 `n`，或以 `n` 开头、紧接 `.` 或 `-`，就属于 `h`。这一规则覆盖 `.abi0`、`.func<N>`、`.gowrap<N>`、`.deferwrap<N>`、`-range<N>` 及其组合；分隔符的要求排除 `Xwait` 与 `Xwait3` 之类的字面前缀。

### 2.5 L5：引用需要名字与导入

- Go 的名字解析：包 Q 引用包 P 的函数，必须导入 P（或点导入），并在源码中写出它的标识符。
- `//go:linkname` 的拉取声明没有函数体，不能被内联（L3 的 `HaveInlineBody`），只会产生对符号的重定位，使目标自己的符号可达。
- 汇编以 `CALL ·F(SB)` 引用，只产生重定位，汇编中不含 Go 的函数体。
- 本轮检索：
  - 23 个模块与仓库中，导入 `modernc.org/libc` 的只有 `modernc.org/sqlite` 与 libc 自身（附录 B）。
  - 23 个模块、仓库与固定的标准库中，都没有指向 `modernc.org/libc` 的 `//go:linkname`。
  - libc 自身没有 `//go:linkname`。
  - 其他模块的汇编没有引用 libc 的符号。
  - 标准库不能导入模块。

### 2.6 L6：内容寻址的合并（限制）

- 内容寻址的符号（闭包等）按内容哈希（数据与重定位）合并，保留其中一份的对象索引，名字与字节来自同一份（`link/internal/loader/loader.go:399` ~ `437`）。
- 所以同字节的两个闭包在 A 中只有一个名字。若保留的是宿主的那份，名字属于宿主，结论保守为“未得到 absent”；若保留的是别处的那份，A 中的字节是别处源码编译的同字节代码，不是 F 的函数体编译出的代码。
- 对甲、乙、丙组：宿主的函数体在 K 上除跟踪分支之外，只有 `defer func() { Xrealloc(tls, …, 0) }()`（`Xexecl`、`Xexeclp`、`Xexecle`、`X__execvpe`）与 `defer tls.Free(…)`（`Xfexecve`、`Xwait4`、`Xpclose`、`X__sigaction`、`X__assert_fail`）两种字面量与包装（附录 A），它们都不调用 T0 或宿主。这一限制对这些组没有实际影响。

### 2.7 定理（宿主集合的充分条件）

**宿主集合 H(F)** 是满足下面条件的最小集合：

- F ∈ H(F)；
- 一个顶层声明（函数、方法或汇编 `TEXT` 块）的源码中，在注释之外以标识符的形式写出 H(F) 中某个成员的名字，这个声明就属于 H(F)；
- 包级的变量与常量声明若写出成员的名字，对应包的初始化函数与包级闭包都属于 H(F)。

**定理**：设 H(F) 中没有方法与泛型函数。在 §7.1 的前提下，若 A 的 ELF 符号表中没有名字属于 H(F) 的成员（按 L4），则 A 中没有由 F 的函数体编译出的指令。

证明的梗概（逆否）：

1. 设 A 中有由 F 的函数体编译出的指令。由 L1，它在某个可达的文本符号 s 中，s 的内容来自一个函数 g 的 IR，或一个汇编块。
2. 汇编块只含它自己写出的指令，不含 F 的函数体。所以 s 来自 g。
3. g 的 IR 含有 F 的函数体。由 L2，PGO 已由前提排除，这只能经一串内联：g 写出 h₁，h₁ 的函数体写出 h₂，…，最后写出 F（“写出”包括以名字调用、作为局部的静态值或闭包）。
4. 由宿主的定义，h₁、…、F 与 g（或 g 所在的声明）都在 H(F) 中。
   - 若这串经过静态去虚化或方法包装，其中必有一环是方法，而写出下一环的方法本身就是宿主，与“H(F) 中没有方法”矛盾。
   - 包级声明中的写出，使包的初始化函数成为宿主（定义的第三条）。
5. s 是 g 自己或它的派生符号：g 中闭包与包装的符号，或内联产生、以被内联函数命名的闭包。由 L4，s 的名字属于 H(F) 的某个成员；没有泛型函数，所以不需要实例化名字的规则。
6. s 可达，由 S-7，它的名字（按 `mangleABIName` 改写）在 A 的符号表中。

H(F) 中有方法时，方法包装（提升方法、指针接收者、`-fm` 方法值）以接收者类型命名，第 4、5 步的名字关系没有写定，定理不适用（§5.1）。

L6 是合并时的限制：A 中可能留有与 F 的闭包字节相同、但来自别处源码的代码。

**推论（甲组）**：若 F 不可内联（L3），第 3 步中 g 只能是 F 自己（或 F 的闭包与包装），所以只需检查 F 自己的名字与派生名字。

## 3. 宿主集合的计算

### 3.1 算法

1. 范围：`modernc.org/libc@v1.77.1` 与 `modernc.org/sqlite@v1.60.1` 的全部 `.go` 与 `.s` 文件，排除 `_test.go`（`go build` 从不编译它们）与 `testdata`。只有这两个模块可能写出 libc 的标识符（L5）。
2. 每轮：
   - 以 `grep -w` 检索当前前沿的名字（整词，`_` 与数字算作词的一部分）；
   - 对每处命中，取它所在的顶层声明：行首的 `func`、`var`、`const`、`type`，或汇编的 `TEXT ·name(SB)`；
   - 命中在 `//` 之后的算注释，不扩展。
3. 命中所在的函数或汇编块，其名字加入下一轮；方法以方法名加入。`init` 不加入，因为 Go 不允许引用 `init`。包级声明与类型声明中的命中单独列出、逐处检查（§3.2 的结果中只有同名的类型声明）。
4. 直到没有新名字。
5. 闭包计算三种范围：
   - **全集**：全部文件，不论构建约束；
   - **K**：只从 K 选入的文件扩展；
   - **K + 包**：另外只从 K 实际编入的包（libc 根包、`libc/uuid/uuid`、`libc/sys/types`、sqlite 根包、`lib`、`vtab`）扩展。

这是对源码文本的保守超集：注释外的字符串、同名的局部变量与其他配置的同名函数也会加入，只会使集合变大（结论更保守），不会遗漏写出名字的声明。

### 3.2 结果

| 目标 | 全集 | K | 宿主（全集；未注明者 K 相同） |
|---|---:|---:|---|
| `Xsystem` | 2 | 2 | `Xsystem`、`Ysystem` |
| `Xexecve` | 18 | 18 | `Xexecve`、`Xexecv`、`Xexecl`、`Xexecle`、`Xexeclp`、`Xexecvp`、`Xexecvpe`、`X__execvpe`、`Xfexecve` 与它们的 `Y*` 汇编包装 |
| `Xfexecve` | 2 | 2 | `Xfexecve`、`Yfexecve` |
| `Xexecv` | 4 | 4 | `Xexecv`、`Xexecl`、`Yexecv`、`Yexecl` |
| `Xexecle` | 2 | 2 | `Xexecle`、`Yexecle` |
| `X__execvpe` | 8 | 8 | `X__execvpe`、`Xexecvp`、`Xexecvpe`、`Xexeclp` 与 `Y*` |
| `Xexecl` | 2 | 2 | `Xexecl`、`Yexecl` |
| `Xexeclp` | 2 | 2 | `Xexeclp`、`Yexeclp` |
| `Xexecvp` | 4 | 4 | `Xexecvp`、`Xexeclp`、`Yexecvp`、`Yexeclp` |
| `Xexecvpe` | 2 | 2 | `Xexecvpe`、`Yexecvpe` |
| `Xvfork` | 2 | 2 | `Xvfork`、`Yvfork` |
| `Xwaitid` | 2 | 2 | `Xwaitid`、`Ywaitid` |
| `Xwaitpid` | 4 | 4 | `Xwaitpid`、`Xwait`、`Ywaitpid`、`Ywait` |
| `Xwait` | 2 | 2 | `Xwait`、`Ywait` |
| `Xwait4` | 6 | 4 | `Xwait4`、`Xwait3`、`Ywait4`、`Ywait3`；全集另有 `X__wait4_time32`、`X__wait3_time32`（只在 386 与 arm 的文件中） |
| `Xwait3` | 3 | 2 | `Xwait3`、`Ywait3`；全集另有 `X__wait3_time32` |
| `Xpclose` | 2 | 2 | `Xpclose`、`Ypclose` |
| `Xsigaction` | 5523 | 2 | K：`Xsigaction`、`Ysigaction`。全集经 darwin 与 freebsd 的 `Xabort`（`libc_darwin.go:1621`、`libc_freebsd.go:1356` 调用 `Xsigaction`）扩大 |
| `X__sigaction` | 5523 | 4 | K：`X__sigaction`、`Xsigaction`、`Y__sigaction`、`Ysigaction`。全集同上 |
| `___libc_sigaction` | — | 3697；K + 包：26 | K + 包：`___libc_sigaction`、`X__sigaction`、`Xsigaction`、`Xabort`、`X__assert_fail`、`X__builtin_abort`、`X__builtin_trap`、`X__builtin___memcpy_chk`、`X__builtin___memmove_chk`、`X__builtin___memset_chk`、`X__builtin___snprintf_chk`、`X__builtin___strncat_chk`、`X__builtin___strncpy_chk`、`X__builtin___vsnprintf_chk` 与 12 个 `Y*`。只按 K 的文件时，未编入的 `sqlite/vec`、`sqlite/vfs` 包写出 `libc.Xabort`，使集合扩大 |
| `Xsignal` | — | 5080；K + 包：4899 | 见 §5 |

- 甲、乙组（共 17 个，不含 `X__sigaction`）：全集的宿主都是 libc 根包的普通函数与汇编包装，没有方法、泛型、包级声明或函数值的引用；sqlite 中没有任何一处写出 T0 的名字。
- 在上表中宿主不多于 26 个的各个闭包里，注释之外的命中所在的声明只有函数与汇编块。唯一的例外是 darwin 子包中名为 `X__sigaction` 的类型声明（`signal/signal_darwin_*.go`、`stdlib/stdlib_darwin_*.go`）：它是同名的类型，不是对函数的引用，也不在 K 中。
- `Y*` 是汇编块（`abi0_linux_amd64.s`），只以 `CALL` 调用目标；它们在 Go 中只有无函数体的声明（`abi0_linux_amd64.go`），没有被 Go 代码引用。

## 4. 逐组的论证

### 4.1 甲组：不可内联（8 个）

| 目标 | 定义（K） | 不在跟踪分支中的 `defer` | 其他论证（全集宿主） |
|---|---|---|---|
| `Xfexecve` | `ccgo_linux_amd64.go:126210` ~ `126230` | `:126216` `defer tls.Free(32)` | 2 个 |
| `Xwait4` | `:30015` ~ `30045` | `:30021` `defer tls.Free(32)` | 6 个 |
| `Xpclose` | `:137717` ~ `137741` | `:137723` `defer tls.Free(16)` | 2 个 |
| `Xexecle` | `:126017` ~ `126059` | `:126027` `defer func() { Xrealloc(tls, argv, 0) }()` | 2 个 |
| `X__execvpe` | `:126122` ~ `126192` | `:126132` `defer func() { Xrealloc(tls, b, 0) }()` | 8 个 |
| `Xexecl` | `:125973` ~ `126015` | `:125983` 同 `Xexecle` | 2 个 |
| `Xexeclp` | `:126061` ~ `126103` | `:126071` 同上 | 2 个 |
| `X__sigaction` | `:133597` ~ `133624` | `:133603` `defer tls.Free(16)` | 只有按 K 的 4 个（丙） |

- **必然留下的表示**：F 的代码若在 A 中，A 的符号表中必有 `modernc.org/libc.F`，或 F 的派生名字（`F.abi0`、`F.func<N>`、`F.deferwrap<N>`）。
- **为什么保持**：由 L3，F 不被内联；由 L1 与 L2，F 的函数体只编译进 F 自己的符号；闭包与 `defer` 包装各自成为以 F 为前缀的符号（L4）；可达的符号都在符号表中（S-7）。
- **不能排除的**：
  - 定义 F 的文件是否就是上表的文件，依赖 K 的文件选择（R4）。所以除 `X__sigaction` 外另给出宿主集合的论证：它不依赖选择，两条论证互相独立。
  - 编译器与工具链的身份（R2）。
  - L6（不影响：闭包只调用 `Xrealloc`，`deferwrap` 只调用 `(*TLS).Free`）。
- 甲组不依赖 PGO 不存在这一前提（L3 与预算无关）；宿主集合的那条论证仍依赖它。

### 4.2 乙组：可能被内联，宿主集合小（10 个）

| 目标 | 定义（K） | 宿主（全集） |
|---|---|---|
| `Xsystem` | `libc_musl.go:1187` ~ `1206` | 2 |
| `Xexecve` | `ccgo_linux_amd64.go:126113` ~ `126120` | 18 |
| `Xexecv` | `:126105` ~ `126111` | 4 |
| `Xexecvp` | `:126194` ~ `126200` | 4 |
| `Xexecvpe` | `:126202` ~ `126208` | 2 |
| `Xvfork` | `:126580` ~ `126587` | 2 |
| `Xwaitid` | `:126597` ~ `126603` | 2 |
| `Xwaitpid` | `:126605` ~ `126611` | 4 |
| `Xwait` | `:126589` ~ `126595` | 2 |
| `Xwait3` | `:30007` ~ `30013` | 3 |

- **必然留下的表示**：F 的代码若在 A 中，A 的符号表中必有 H(F) 某个成员的名字或派生名字（§2.7 的定理）。
- **为什么保持**：§2.7 的证明。宿主都是 libc 根包的普通函数与汇编块，没有方法，所以去虚化与方法包装不涉及它们。
- **不能排除的**：
  - PGO（R1，必须由运行确认没有启用）；
  - 工具链与模块的身份（R2、R3）；
  - 剥离（R5）。
- 不依赖文件选择：宿主集合在全部文件上计算，其他配置的同名函数只使集合更大。
- 本文不推断这些函数是否会被内联；论证对内联与否都成立。

### 4.3 丙组：依赖文件与包的选择

- `Xsigaction`（`ccgo_linux_amd64.go:133626` ~ `133632`，K 的宿主 2 个）与 `X__sigaction`（甲组，K 的宿主 4 个）：
  - 全集的宿主经 darwin 与 freebsd 的 `Xabort` 扩大。K 编入的 `Xabort`（`libc_musl.go:669` ~ `675`）不写出这两个名字，而是写出 `___libc_sigaction`。
  - 所以论证以“K 实际编入的是 `ccgo_linux_amd64.go` 与 `libc_musl.go`，不是其他配置的文件”为前提（R4）。
- `___libc_sigaction`（`libc_musl.go:1115` ~ `1132`）：
  - 宿主包括 `Xabort` 与 10 个调用 `Xabort` 的断言、检查或内建函数（附录 A），连同汇编包装共 26 个名字。
  - 前提是 K 编入的包只有 §3.1 列出的六个（R4）；否则 `sqlite/vec` 与 `sqlite/vfs` 中写出 `libc.Xabort` 的函数也成为宿主。
  - 在 K 编入的 sqlite 文件中，没有任何一处写出这 26 个名字。它们是否在 A 中，须运行；本文不预测。
- 若 R4 不成立，这三个目标的论证回到全集，宿主集合过大，实际上不能得到 absent。

## 5. 丁组：`Xsignal`

### 5.1 断点

- K + 包范围的宿主集合有 4899 个名字，包括方法：
  - `(*TLS).deliverSignal`（`libc_musl.go:484`）写出 `Xsignal`（`:498`）；
  - `(*TLS).checkSignal`（`:473`）与 `(*TLS).raiseSignal`（`:524`）调用 `deliverSignal`；
  - `(*TLS).Free`（`:463` ~ `471`）调用 `checkSignal`。
- 方法名是 `Free` 这样的常见标识符，闭包按名字扩展到数千个函数。
- **断点一**：宿主中有方法。方法包装（提升方法、指针接收者包装、`-fm` 方法值）会以接收者类型命名（`noder/reader.go` 的 `methodWrapper` 与 `wrapMethodValue`），本文没有写定它们与宿主的名字关系。§2.7 的定理只对没有方法的宿主集合给出了完整的名字规则。
- **断点二**：即使补上名字规则，宿主中有转译代码的通用入口。只要其中任一个在 A 中，就不能得到 absent。

### 5.2 静态蕴含：`Xsignal` 很可能在产物中

固定源码中有三条到 `Xsignal` 的调用链，各环的条件都是运行时的数据：

| 起点 | 链 | 条件 |
|---|---|---|
| `(*TLS).Free` | `Free` → `checkSignal`（`:470`）→ `deliverSignal`（`:476`）→ `Xsignal`（`:498`） | `tls.checkSignals`、通道中有待处理的信号、处置带 `SA_RESETHAND` |
| `X__syscall2`（`syscall_musl.go:111`） | 调用号为 `SYS_tkill`、目标是当前线程时 → `raiseSignal`（`:117`）→ `deliverSignal`（`:529`）→ `Xsignal` | 调用号是参数 |
| `X__syscall3`（`:132`） | 调用号为 `SYS_tgkill` 时同上（`:136`） | 同上 |

- 编译器只消除可以证明不可达的代码。这些条件都依赖运行时的数据，所以：
  - **`(*TLS).Free` 的代码**（自身的符号，或被内联的副本）在 A 中，则 `checkSignal`、`deliverSignal` 与 `Xsignal` 的代码在 A 中，以自身的符号或内联的形式。
  - **`X__syscall2`、`X__syscall3` 自身的文本符号**在 A 中，同样如此。
  - 被内联、调用号为常量的副本，那个分支可能被常量折叠，所以对它们只就自身的符号下结论。
- `(*TLS).Free` 是转译代码释放栈帧的方法，sqlite 的转译代码大量调用它。本文**预期**它在 A 中，但这是须运行才能确认的事实，不作为证据。
- 结论：`Xsignal` 的强命题未成立。按上面的静态蕴含，原 (a) 对 `Xsignal` 所需的“不在产物中”在 K 上很可能不成立。

## 6. F-5：目标完整性与用途的限制

### 6.1 通用入口

- `X__syscall0` ~ `X__syscall6`、`___syscall_cp` 与 `Xsyscall` 以参数接收系统调用号，经 `unix.Syscall` / `unix.Syscall6` 进入内核（`syscall_musl.go:14` ~ `170`）。
- T0 的 absent 只说明 T0 中函数的代码不在 A 中；它不排除其他位置以常量或变量把 `execve`、`fork`、`wait4`、`kill`、`rt_sigaction` 等调用号交给这些入口。

### 6.2 T0 之外的直接位置（本轮新发现）

在 K 编入的 libc 文件中，检索 `unix.`、`syscall.`、`exec.`、`os.` 与 `signal` 的进程和信号操作，找到以下几处不在 T0 中的位置。表中“操作”一列的直接系统调用都不经 T0 的函数：

| 位置 | 操作 | 到达 |
|---|---|---|
| `Xabort`（`libc_musl.go:669` ~ `675`） | 调用 `___libc_sigaction`；`unix.Tgkill`、`unix.Kill` 向自身发送 `SIGABRT` | 由 `X__assert_fail` 与 `X__builtin_*` 检查函数调用 |
| `resetSigDfl`（`abort_linux.go:37` ~ `42`） | `unix.RawSyscall6(unix.SYS_RT_SIGACTION, …)` 与 `SYS_RT_SIGPROCMASK`，直接改变内核的信号处置 | 由 `Xabort` 调用 |
| `(*TLS).raiseSignal`（`libc_musl.go:524` ~ `539`） | 运行 C 的处理函数，或 `unix.Kill` 发给本进程 | 由 `X__syscall2`、`X__syscall3` 在 `SYS_tkill`、`SYS_tgkill` 时调用 |
| `(*TLS).killSignal`（`:351` ~ `368`） | 投入目标线程的通道，或 `unix.Kill` 发给本进程 | 同上（目标是另一个模拟线程时） |

- 链接方法审计 §3.3 与链接验证计划 §4.1 的 T0，按 libc 自己的 `SYS_*` 名字与对象引用检索。它们没有覆盖 `golang.org/x/sys/unix` 的 `SYS_RT_SIGACTION`、`Kill`、`Tgkill`，所以漏掉了这几处。本节只登记这一事实，不改写已合并的审计文档。
- 本节的检索本身也是文本检索，不是完整性的证明：经函数指针表、`unsafe`、反射或字符串构造的到达方式仍然看不到。
- `Xsystem` 的能力来自标准库的 `os/exec`；仓库自己也使用 `os/exec`，它的代码在 A 中，与 `Xsystem` 是否在 A 中无关。T0 的 absent 不涉及标准库。

### 6.3 用途的限制

- 即使甲、乙、丙组的强命题在某次 A 上全部成立，也只说明这些函数的代码不在该 A 中。它不推出：
  - libc 的启动、回收或信号能力在产物中整体不可达；
  - 依赖例外可以批准；
  - 其他配置、其他产物或今后的版本。
- 信号一族已有反向的静态依据（§5.2、§6.2），不能用 T0 的结果代表“信号能力不在产物中”。

## 7. 前提、须运行的证据与结论

### 7.1 须运行才能取得的前提

| # | 前提 | 用于 | 取得 |
|---|---|---|---|
| R1 | PGO 没有启用；构建形态与 K 相同 | 乙组、丙组、甲组的宿主论证 | `-x` 的编译命令中没有 `-pgoprofile`；`go version -m A` 中没有 `-pgo`；`go env` 与链接验证计划 §2 一致 |
| R2 | 工具链是固定提交的 Go 1.27.3 | 全部 | 链接验证计划 §2、§8.3 的身份核对 |
| R3 | 模块集合与内容与 go.mod、go.sum 相同 | 全部 | `go mod verify`；`go list -m` 与附录 B 一致 |
| R4 | K 编入的文件与包与 §1、§3.1 的静态求值相同 | 甲组的不可内联论证、丙组 | `go list -deps -json ./cmd/specmetry` 中 libc 与 sqlite 各包的 `GoFiles`、`SFiles` |
| R5 | A 不剥离，有 `.symtab` | 全部 | ELF 检查 |
| R6 | 判定：A 的符号表中是否有 H(F) 的名字（L4 的规则） | 全部 | 读取 A 的 `STT_FUNC` 名字 |

- 判定只用产物 A 的符号表，不需要 DWARF，也不需要 `-dumpdep`。原计划中的产物 B、`-dumpdep` 的交叉核对与对照是否保留，属于方法的定稿，本文不展开（M-1、M-3、M-5 本轮暂不展开）。
- 可以另行取得的旁证（不代替 A）：一次独立的 `-gcflags=-m` 诊断构建，确认甲组的“不可内联”原因。它不是主证据，未命中也不改变 §2.3 的源码论证，只用于发现工具链身份的不一致。
- 拟用的判定程序与夹具只作描述，不提交代码：
  - 读取 ELF 的符号表；
  - 按 L4 的前缀与分隔符规则匹配；
  - 先用一个已知存在的正对照（如 `modernc.org/libc.Xmalloc`）确认名字格式的读取。正对照未命中时为 T。

### 7.2 结论

**原 (a) 是否有继续定稿的技术基础**：

- **启动与回收一族（17 个：甲组除 `X__sigaction` 之外的 7 个与乙组的 10 个）**：有。论证不依赖 DWARF，不收窄 absent；须定稿的是提案 H-1（§8）与 R1 ~ R6 的取证方式，并经独立复审。
- **`Xsigaction`、`X__sigaction`、`___libc_sigaction`**：有，但另以文件与包的选择为前提（R4）。
- **`Xsignal` 与信号处置的能力整体**：没有。
  - `Xsignal` 未成立，而且有很可能在产物中的静态依据；
  - T0 漏掉了 `Xabort`、`resetSigDfl`、`raiseSignal`、`killSignal` 等直接的信号操作。

**缺口**：

1. 信号一族：`Xsignal` 的方法宿主与通用入口（§5）。§6.2 的位置不在 T0 中：
   - 若把它们加入目标集合，`(*TLS).raiseSignal`、`(*TLS).killSignal` 是方法，有同样的方法宿主问题，并经 `X__syscall2`、`X__syscall3` 到达；
   - `Xabort` 与 `resetSigDfl` 的宿主集合（K + 包）分别为 21、22 个名字。除 `resetSigDfl` 自己之外，都在 `___libc_sigaction` 那 26 个名字之中，都是函数与汇编包装。本文只做了这一计算，没有把它们写成目标。
2. F-5：目标集合的完整性没有方法；通用入口使有限 T0 不能代表能力整体。
3. 前提 R1 ~ R6 都须运行；执行未授权。

**需要另作决定的替代方向**（本文不选择，也不推荐其中任一项）：

- 把 (a) 的范围限于启动与回收，信号一族另行处理。这改变 (a) 覆盖的范围，须负责人明确决定，并同步规范、例外条件与验收。
- Q-B3-1 (b)：只有精确例外、没有不可达证明。这是更弱的保证，须负责人明确选择并写清放宽的范围。
- Q-B3-1 (c)：更换数据库驱动。
- 改变依赖本身（例如固定一份去掉这些能力的 libc），这会引入依赖的变更与维护，须另定范围与授权。

## 8. 提案 H-1（未采纳，交独立审查）

| 项 | 内容 |
|---|---|
| 起因 | 链接方法审计 S-4：DWARF 的内联记录不能支持“F 的代码不在 A 中”；Review 132 认定 M-2 的弱结论不足以支持原 (a) |
| 提案 | 链接验证计划 §7 第 6、7 条改为：present = A 的符号表中有 F 的名字或派生名字；absent = 整体没有 T，且 A 的符号表中没有 H(F) 任一成员的名字或派生名字（L4），H(F) 按 §3 在执行之前固定写入；既不是 present 也不能得到 absent 的（宿主的名字出现而 F 自己的没有）→ 新的类别“未得到 absent”，不是 absent，也不是 T |
| 前提 | §7.1 的 R1 ~ R6；H(F) 没有方法、泛型与包级声明（否则该目标“未成立”）；`Xsignal` 不适用 |
| 不改变 | absent 的含义（仍是 F 的代码不在 A 中）；以产物 A 为主证据；已关闭的收尾协议（计划 §8.3、§8.4） |
| 影响 | 判定不再依赖 DWARF 与 S-4；`-dumpdep` 不再是 absent 的必要条件；是否保留产物 B、C5、C9 等对照，须随方法定稿另行决定。计划 §4.2、§7 与对照表须同步修订，并经独立复审 |

- 甲组可以采用更简单的形式：H(F) 取 F 自己（L3 的推论）。是否采用，同样交审查。
- 本提案不批准例外，不改变冻结的规则；采纳与否由负责人决定。

## 9. 待决清单

| 事项 | 依赖本文 |
|---|---|
| 本文与提案 H-1 的独立审查 | 是 |
| (a) 的范围是否限于启动与回收；信号一族如何处理 | 是：§5、§6.2 的发现 |
| F-5（目标集合的完整性方法） | 是：§6 |
| M-1、M-3、M-5 的定稿，产物 B 与对照的去留 | 部分：H-1 若采纳，absent 不再依赖它们 |
| 执行（R1 ~ R6 的取证、判定程序、场所、资源） | 是；执行未授权 |
| Q-B3-1 (b) / (c)、生产 CGO、X / E / U、F-7 / F-8、场所与资源 | 否：可以独立决定 |

## 附录 A：宿主函数体的核对（K）

除跟踪分支（L2，已丢弃）之外，宿主函数体中的函数字面量、`defer`、`go` 与 `goto`：

| 函数 | 位置 | 发现 |
|---|---|---|
| `Xsystem` | `libc_musl.go:1187` ~ `1206` | 无 |
| `Xexecve`、`Xexecv`、`Xexecvp`、`Xexecvpe`、`Xvfork`、`Xwaitid`、`Xwaitpid`、`Xwait`、`Xwait3`、`Xsigaction` | `ccgo_linux_amd64.go`（§4.2 的行号；`Xsigaction` 为 `:133626` ~ `133632`） | 无 |
| `Xfexecve`、`Xwait4`、`Xpclose`、`X__sigaction`、`X__assert_fail`（`:24633` ~ `24641`） | 同上 | `defer tls.Free(n)` |
| `Xexecle`、`X__execvpe`、`Xexecl`、`Xexeclp` | 同上 | `defer func() { Xrealloc(tls, …, 0) }()` 与 `goto` |
| `___libc_sigaction`、`Xsignal`、`Xabort` | `libc_musl.go:1115` ~ `1132`、`1034` ~ `1048`、`669` ~ `675` | 无 |
| `X__builtin_abort`、`X__builtin_trap`、`X__builtin___*_chk` | `builtin.go:51` ~ `395`、`libc_all.go:265` ~ `285` | 无 |

## 附录 B：模块的 dirhash 与导入

对本机模块缓存中 `go.mod` 列出的 23 个模块，按 go.sum 的方法独立重算 dirhash，都与仓库的 go.sum 一致：

- 直接依赖：
  - `github.com/aymanbagabas/go-pty` v0.2.3
  - `github.com/dlclark/regexp2` v1.12.0
  - `github.com/pelletier/go-toml/v2` v2.4.3
  - `github.com/santhosh-tekuri/jsonschema/v6` v6.0.3
  - `github.com/spf13/cobra` v1.10.2
  - `golang.org/x/mod` v0.42.0
  - `golang.org/x/sys` v0.49.0
  - `golang.org/x/term` v0.47.0
  - `modernc.org/sqlite` v1.60.1
- 间接依赖：
  - `github.com/creack/pty` v1.1.24
  - `github.com/dustin/go-humanize` v1.0.1
  - `github.com/google/uuid` v1.6.0
  - `github.com/inconshreveable/mousetrap` v1.1.0
  - `github.com/mattn/go-isatty` v0.0.24
  - `github.com/ncruces/go-strftime` v1.0.0
  - `github.com/remyoudompheng/bigfft` v0.0.0-20230129092748-24d4a6f8daec
  - `github.com/spf13/pflag` v1.0.9
  - `github.com/u-root/u-root` v0.16.0
  - `golang.org/x/crypto` v0.51.0
  - `golang.org/x/text` v0.37.0
  - `modernc.org/libc` v1.77.1
  - `modernc.org/mathutil` v1.7.1
  - `modernc.org/memory` v1.12.1

检索的结果：

- 导入 `modernc.org/libc` 的 Go 文件只在 `modernc.org/sqlite`（218 个）与 libc 自身（150 个）中；其余 21 个模块与仓库都没有。
- 各模块的汇编中都没有引用 libc 的符号。

重算的 dirhash 只证明所读目录与 go.sum 记录的内容一致，不认证上游原产，也不证明执行时的构建使用这些目录（R3）。
