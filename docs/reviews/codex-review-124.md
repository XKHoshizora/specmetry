# Codex Review 124 — PR #48 / Review 123 定向复审

**结论：B-123-1 可关闭，N-123-1 收口；B-123-2 的 Prepare／Batch.Close 主体修复成立，但 Windows 查找器仍会丢弃第二次及之后的关闭失败，保留残余 B-123-2-R。固定对象技术 NO-GO，暂不建议合并。**

残余不导致已观察到的成功放行：独立反例仍退出 6，第一次失败没有被后续成功清除。问题是实际已经产生的另一项原始关闭错误没有进入返回链，未兑现本轮关闭结果完整保存的要求。没有观察到原生泄漏，不把接缝证据写成自然故障。

## 1. 固定身份、范围与授权

开始核验：2026-11-01 11:52:46 +09:00。读取当前 AGENTS.md、完整 Review 123、处理记录 §158／§159、生产计划 §11.12 和直接关联冻结条文、错误契约及调用链。

| 项目 | 核验结果 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #48](https://github.com/XKHoshizora/specmetry/pull/48)，open、未合并 |
| 本地／远程分支 | `feat/wp2.5-batch2` |
| HEAD | `b6542be72315260d5ad04049627fcf773b7aaa3a` |
| HEAD 树 | `e186e00c9d812d479e1bacb8498e4ee73c25789d` |
| base／GitHub master | `47f568066c4e4e18cbf28fff2a01cfdda7d34ee4` |
| 上轮受审 HEAD | `1a2eda18ba137ba31cb796ce22ad90c415a027db` |
| 当前合并快照 | `60a42aae992dd1eab0e96882ad030076fe09adf6` |
| 快照父提交，依次 | 上述 base、上述 HEAD |
| 快照树 | `e186e00c9d812d479e1bacb8498e4ee73c25789d`，与 HEAD 相同 |

本地工作区和暂存区开始时干净；本地旧 master 引用仍为 `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`，不是本次 base，未更新或用于取差异。受审 HEAD 与固定 base 的 merge-base 是 `47f5680`。远程身份通过 GitHub PR、两个分支 ref 和提交 API 核验，没有切换对象。

本轮上轮 HEAD → HEAD 为 **19 文件、1029 行增加、58 行删除**，逐份检查：归档与状态／计划、错误辅助层、Prepare／Batch.Close、三个拒绝入口、Windows 路径关闭、测试访问层、七项新增回归以及既有测试清理的调整。Review 123 已认可的其余第二批范围按原限定沿用，没有重审无关原型或第三批能力。

### 1.1 归档及实施顺序

Review 123 单独归档于 `4b05c81da6994a4eb40b0a6cd9eca59c0d25692e`，该提交仅含报告；当前文件与归档 blob 及上一轮交付一致：26619 字节，SHA-256：

`e713eebaffa18ce5d0e59e07ca1391998084f71f0b98692d6b98374c063459da`

随后 `9f93b41f44050761784417284acebdddc48b0922` 先登记 §158／计划 §11.12；`6e5747e4c6d592fe70cf683b15bf09482872b3b7` 才实施修复及回归。修复 CI 的 errorlint 失败后，`445b2a2366a426a0a3ea63fa148ce8a2cd1b3c6c` 先记录 §159，最终 `b6542be` 只在指定身份比较行增加注释。

旧处理记录完整保留为字节前缀，只追加 §158／§159；历史报告未改写。Schema、共享 apperr、gitx、SQLite／迁移、derive／toderive／dbdump、factgen/legality、冻结规范、原型、依赖与 workflow 未变。没有公开命令注册、检查进程、全局闸门或第四批任务管理。状态同步没有扩大授权。

## 2. B-123-1：关闭

### 2.1 实际边界与全部检查

统一边界为 `internal/checkrun/errors.go:21` 的 `refusal`。三个实际调用点均已替换：

- `selection.go` 的显式参数选择拒绝；
- 同文件 `staticRules` 的 D3/D4 拒绝；
- `interpreter.go:204` 的解释器不可用拒绝。

三个入口仍先遍历全部参数／所选 AC，汇集该步骤的问题，最后才进入 refusal；没有对输入或检查循环新增 200 项限制。`errors.go:22` 先取得完整问题数，超过 200 才保留前 200 项；不排序、不补造位置，不改变原始错误码。后面的有效选择及配置固定流程未改。

省略信息放在受控 message：`N problems: the first 200 are listed, N-200 left out`。199／200 不虚报省略，201 为 200 条详情、总数 201、省略 1；只有第 201 项有问题时，详情仍准确指向它，不能因前 200 项正常而空过。

### 2.2 回归与组合

`release_test.go:69` 的 `TestRefusalDetailsBoundary` 调用的是实际 choose／staticRules／interpreters 的测试访问入口，不是另造一份 refusal 模型。三入口各执行 199／200／201，再各执行“仅第 201 项有问题”，共 12 组条件，属于一个顶层测试，不能算成 12 个独立测试名称。

每组经实际 `apperr.ToEnvelope`、JSON 序列化、实际 `contract.ValidateJSON("error.v1", ...)`；同时核对退出码、数量、每条真实位置及 message。解释器最后一项对照前 200 项使用当前平台可用的解释器，最后 pwsh 在空 PATH 中不可用；有实际查找路径，并非将全部项都无效化。

虽然详情数量期望使用 MaxDetails，但未仅依赖该常量自证：实际 Schema 的 200 上限、message 中固定的 first 200 和未超 200 时不得出现 left out 均是额外约束。固定位置来源于夹具索引和契约，没有用返回结果反算。

`TestPrepareManyProblemsAndACloseFailure` 经实际 Prepare 验证 201 个参数问题叠加 DB 关闭接缝错误：错误码仍为 invalid_input／2，200 个位置保留，数量说明在清理说明之前，错误仍可 `errors.Is` 取得。两种说明都在 message 中，未借 details 挤掉原项。三个入口的固定受控 message 加本批两类资源关闭说明远小于 2000 字符；这些组合不会先被 ToEnvelope 截断。该判断不扩大为任意外部错误文本都不会被截断。

**证据**：本机独立执行 TestRefusalDetailsBoundary 通过；三个平台 CI 的该测试及实际 Prepare 组合通过。组合在本机因 Git 夹具准备失败没有进入目标断言，不能写成本机已复现。

**裁定：B-123-1 在本批三个拒绝入口及当前清理组合范围内关闭。** 没有改变 Schema、共享 apperr 或其他调用行为。

## 3. B-123-2：主体修复成立，仍有一项残余

### 3.1 Prepare、Batch.Close 与错误展示

`prepare.go:81` 的失败 defer 现在调用 `failed(err, b.Close())`；`errors.go:74` 复制原步骤的 apperr，保留 Code、Details 和 Current，添加受控清理说明，再把该副本、原步骤错误和关闭错误都放入返回链。

- `apperr.From` 首先取得带清理说明的副本，ToEnvelope 的最终展示不只保留错误链内部信息。
- 原始步骤对象仍在链中，可由 errors.Is 取得；原始 Close 错误经 ReleaseError.Unwrap 保留，errors.Is／As 仍可访问。
- 不用关闭错误覆盖原步骤代码或 details。比如输出类型拒绝仍为 internal／1，参数拒绝仍为 invalid_input／2。
- `ReleaseError.Error` 使用受控资源名称，未回显原生错误文本、路径或注入文本；原始错误仍供程序检查。

`prepare.go:198` 的 Batch.Close 每个资源只关闭一次，保存第一次的 `*ReleaseError`，后续调用返回同一缓存值；成功时重复调用仍为 nil，host.release 也只执行一次。`closed` 的注释明确是“已经尝试”，不是释放证明。本轮没有新增并发 Close 的调用契约或声称其可并发使用。

`released` 先得到真实 Close 返回值，再以 errors.Join 追加接缝错误，不替换真实失败。Scratch 与 DB 分别记录，任何一项出错都会保留。既有 gitx／SQLite 接口语义没有改动。

### 3.2 Windows 单次关闭与持有责任

`interpreter_windows.go:19`～`:37` 将 finalName 与 CloseHandle 分开：真实路径读取先完成；关闭失败返回空路径和 ReleaseError，并合并可能同时发生的读取错误。lookup 在 find 前后都检查保存的关闭失败，因此“第一个候选关闭出错、后面的候选健康”不能恢复成功；根路径的关闭失败也优先于普通 unverified 分类。实际 Prepare 会返回 environment／6，受控说明可见，不误报普通 shell_unavailable。

“不再重试、进程退出释放”在本批仅指尚未确认释放的进程所持文件／目录句柄、DB 连接资源；不是已完成清理，不证明目录被删除或事务已经提交，也不能把调用返回等同于宿主退出。Windows 原始句柄即使不再保留可操作的 Go 引用，仍是该进程的内核资源；本批未许可再操作未知或失败句柄。没有因此新增 C-4／D21 禁用触发，第四批的任务清单／晚到任务错误管理仍未接入。

测试自己的 Batch／DB 清理已改为检查错误；Windows 的保护夹具在 Cleanup 中解除保护并关闭自有句柄，分别检查两个 API 的结果。测试清理成功不反写被测路径为已成功释放。

以上主体接线及“单个句柄失败＋后续健康”的回归成立。但不能据此宣布所有关闭结果已经完整保存，见下项。

### B-123-2-R：同一查找器的第二次关闭失败被主动丢弃

**位置**：`internal/checkrun/interpreter.go:48`～`:58`、`:64`～`:70`，以及 `:101`～`:125` 的同源候选循环。

`finder.final` 只有在 `f.released == nil` 时才保存新的 ReleaseError。与此同时，newFinder 遇到错误会继续处理其他登记根；onPath 在 accept 失败后也可以继续解析其他候选。因此后续关闭操作仍然真实发生，但它们的错误不再进入保存的结果。

**允许的事件顺序**：

1. 本项目有两个可读的登记 worktree 根 A、B（或查找两个候选）；查找器依次调用 finalPath。
2. A 的句柄已经取得，关闭返回错误 E1；f.released 保存 E1，newFinder 继续。
3. B 也取得自己的句柄，关闭返回另一错误 E2；finalPath 已返回 E2，但 `&& f.released == nil` 阻止保存，外层循环又丢弃本次 err。
4. lookup／interpreters／Prepare 最后仍退出 6，但返回链只有 E1，无法取得 E2。后来的普通成功不清除 E1，却也不能补回已丢失的 E2。

**独立复现**：在固定 HEAD 的仓库外副本中，用已提交的 OnRelease 接缝，对两个自建目录对应的两个 handle 关闭各追加不同错误，不改 finalPath／finder／判定实现：

```text
handle closes=2 exit=6 first=true second=false
required close results not all retained
```

实际编译成功并进入断言，测试因 `errors.Is(err, E2) == false` 失败、退出 1。没有设置保护或故意制造原生关闭故障；这是原始调用之后追加的两项接缝错误，证明聚合／消费遗漏，不证明自然故障或泄漏。

最小复现的核心为：在 `package checkrun_test` 的隔离测试中，OnRelease 仅在 resource==handle 时按调用顺序返回 E1、E2；`NewFinder([]string{A, B}, "", nil)` 后调用 Lookup，再核对调用数、exit=6 和两个 errors.Is。A、B 都由该测试创建，不访问未知对象。

**实际影响及依据**：没有发现这条路径错误返回成功，原生结束／清理也没有被证实为成功；这里保留的是错误完整性阻塞。它违反 §158 第 4 项及计划 §11.12 的关闭结果完整消费、原始结果可检查要求，也是 Review 123 B-123-2 所要求的新关闭链不丢结果的直接残余。不能用“已有一个关闭错误足以拒绝”代替保存已经发生的另一项关闭结果。现有七项回归未构造两个不同路径关闭都失败的情况。

**最小修正方向**：二选一即可，不需要扩大生产范围：

- 首次 ReleaseError 后立即结束该查找步骤，不再把它当成普通缺失继续新建路径句柄；已取得句柄的关闭结果全部消费；或者
- 继续遍历时累积每次实际返回的 ReleaseError，而不是仅保存首项，保持受控展示和全部原始错误可达。

补两个独立错误的定向对照，同时保留“一个失败＋后续健康仍拒绝”、普通不存在时可继续寻找、原有关闭一次与真实错误不被注入覆盖的对照。不需要改共享 apperr、扩大 Schema、增加禁用条件或实现第四批。

**裁定：B-123-2 尚不能整体关闭；Prepare／Batch.Close 和 Windows 单次关闭的修复按上述范围认可，残余为 B-123-2-R。** 未发现其外的新增 BLOCKER；不撤回 Review 123 对其他第二批内容的限定结论。

## 4. N-123-1、lint 及直接回归

**N-123-1 收口**：计划 §11.9 已准确改为“不再打开登记的写事务；固定读取所用的事务（DB.WriteTx）不写入”。没有新增 ReadTx、额外登记写入或修改读取固定／seq／last_seen_at 语义。

`6e5747e → b6542be` 的生产代码完全相同，internal 中只有 `release_test.go:224` 增加单行 `nolint:errorlint`。比较表达式 `second != first` 原样保留，没有改成较弱的 errors.Is／内部指针检查，没有全局关闭 errorlint 或修改配置；豁免理由与缓存错误身份要求一致。

直接回归检查：

- 三个入口仍完整选择／校验，未取消合法反证或把输入限于 200；拒绝阶段顺序未改。
- 原 G1 准入、禁用状态、清单为空仍拒绝、已准入每项再查接口未改；没有增加关闭失败的禁用政策。
- 解释器的来源、PATH 规则、worktree 排除、PD-26 与对象读取顺序未放宽；普通路径不存在与新关闭失败是分开的状态。
- 测试访问入口仅导出已存在的实际函数及接缝；生产中的接缝默认 nil，没有由测试实现取代被测步骤。
- 关闭说明及数量说明的当前组合保持原步骤码；残余不涉及更改冻结结果、可靠退出码、qualifying 或历史读取。

已删除登记 worktree 的拒绝仍符合现行冻结 §19.8；待决问题维持 Review 123 的裁定，不自行放宽、删除登记或接受为 L。建议仍最迟在公开命令／用户验收前决定；若修订影响第四批预检接线，应先明确其影响。factgen/legality、维护事项及冻结证据限制保持原范围。

## 5. 独立验证与未验证边界

实际环境：Windows/amd64。先确认 go shim 路径，再显式使用已存在的 `C:\Users\hoshi\AppData\Local\mise\installs\go\1.27.1\bin\go.exe`；实际版本为 go1.27.1 windows/amd64。设置 GOTOOLCHAIN=local、GOPROXY=off、GOSUMDB=off，没有调用 shim 安装工具，没有升级或提高权限。CI 的 Go 1.27.3 单独采证。

### 5.1 已提交的七项定向回归：一次

```text
go test -tags nowebui -count=1 -timeout 90s ./internal/checkrun -v
  -run '^(TestRefusalDetailsBoundary|TestPrepareRefusalKeepsCloseFailures|TestBatchCloseKeepsItsFirstResult|TestPrepareManyProblemsAndACloseFailure|TestFinalPathConsumesItsClose|TestLookupFailsOnACloseFailure|TestPrepareHandleCloseFailure)$'
```

| 顶层测试 | 本轮独立结果 |
|---|---|
| TestRefusalDetailsBoundary | PASS；三个入口的全部边界与第 201 项对照 |
| TestFinalPathConsumesItsClose | PASS；健康、受保护句柄导致的真实 API 失败、接缝及两者同时发生 |
| TestLookupFailsOnACloseFailure | PASS；候选关闭失败＋后续健康，以及根关闭失败 |
| TestPrepareRefusalKeepsCloseFailures | Git 夹具 init 写 `.git/config` 被拒，未进入目标断言 |
| TestBatchCloseKeepsItsFirstResult | 同上，未进入目标断言 |
| TestPrepareManyProblemsAndACloseFailure | 同上，未进入目标断言 |
| TestPrepareHandleCloseFailure | 同上，未进入目标断言 |

整次退出非零；不能写为七项全部通过。没有绕过权限或反复运行求绿色，未到达断言的四项不计为本机证据，也不当作产品失败。

Windows 保护句柄回归是测试有意设置 HANDLE_FLAG_PROTECT_FROM_CLOSE，使 CloseHandle 实际返回 ERROR_INVALID_HANDLE；不是自然发生的故障。原生返回和接缝错误共同保留的证据来自该检查；测试解除保护并关闭自有句柄的 Cleanup 完成，没有清理错误。Scratch／DB 没有构造原生关闭失败，仍只有计划内注入和健康路径的证据。

### 5.2 一次隔离补充核验

预先限定为同一 finder 的两个自有路径句柄、两个不同关闭接缝错误，执行一次 TestReview124TwoPathCloseErrors，go test 的测试期限 90 s。不执行用户项目检查或进程树实验。结果见 §3：正常编译，两个接缝命中，exit=6；第二个错误缺失，被断言检出。

审查用测试文件 SHA-256：`a6e53e428ce780434ca5e84089b7c736771eade787b58b448880ce5340d93760`。哈希仅识别文件，结论来自实际输出与源码。测试结束后核对本轮临时目录的绝对路径和唯一名称，再删除隔离副本，确认不存在；原仓库未加入该测试。

没有独立复现作者其他本机运行／变异，没有运行全量本机套件、原型或用户配置的检查。`git diff --check` 通过。未新增平台实验，未扩大权限。

## 6. CI：正式状态、实际 checkout 与历史

### 6.1 四次运行分别保留

| Run | 固定 HEAD／attempt | 正式结果 | 采证说明 |
|---|---|---|---|
| [41219007149](https://github.com/XKHoshizora/specmetry/actions/runs/41219007149) | 501612d／1 | failure，7/8 | 原 G703 lint failure；沿用 Review 123 的日志核验，本轮重新确认正式元数据 |
| [41224740139](https://github.com/XKHoshizora/specmetry/actions/runs/41224740139) | 1a2eda1／1 | success，8/8 | Review 123 对象，不能替代本次修复证据 |
| [41228654793](https://github.com/XKHoshizora/specmetry/actions/runs/41228654793) | 6e5747e／1 | failure，7/8 | 本轮读取八份日志，唯一失败为测试 errorlint |
| [41232924416](https://github.com/XKHoshizora/specmetry/actions/runs/41232924416) | b6542be／1 | success，8/8 | 本轮最终固定对象，新运行，不是重跑旧 run |

它们均为 pull_request、attempt 1；没有将历史合并为“首次全部通过”。41228654793 八份日志实际 checkout 为 `74eb8883f16b57b697a9c9984ff7b3e48be7a8b4`，父提交依次为固定 base、`6e5747e4c6d592fe70cf683b15bf09482872b3b7`，树 `1f073432d143b031e283b4e8a462a27f6d200d01`。lint job 124618024905 报 `release_test.go:224:26` 的 errorlint，原比较行仍在。其余七项 success。

### 6.2 最终八作业

读取全部原始日志；每份 checkout 的 `git log -1` 都是完整 `60a42aae992dd1eab0e96882ad030076fe09adf6`，父提交及同树关系另经提交 API 核验，没有仅看 job.head_sha。

| 作业 | Job ID | 正式结果／实际 checkout |
|---|---|---|
| Go lint | 124630665778 | success／60a42aa |
| Go tests (Windows) | 124630665988 | success／60a42aa |
| Go tests (Ubuntu) | 124630666024 | success／60a42aa |
| Web | 124630666029 | success／60a42aa |
| Go tests (macOS) | 124630666117 | success／60a42aa |
| Build with dashboard (macOS) | 124630907444 | success／60a42aa |
| Build with dashboard (Windows) | 124630907476 | success／60a42aa |
| Build with dashboard (Ubuntu) | 124630907478 | success／60a42aa |

Go 实际为 1.27.3：Linux/amd64、Windows/amd64、Darwin/arm64。三个平台各有 27 个带测试包输出 ok；无测试文件的包另计，Windows 的 CLI 分包不重复统计。

四个通用回归在三个平台均实际执行并 PASS；三个 Windows 专用回归在 Windows 均实际执行并 PASS。Linux／macOS 没有编译 `_windows_test.go`，不是把这些测试 SKIP 后也算通过。七项均按顶层名字识别，未重复计入内部循环或日志重印。

最终测试没有 FAIL；SKIP 集合与修复前的 lint-failure run 逐名相同：Ubuntu 15、macOS 7、Windows 10，也与 Review 123 记录的集合相符。新增七项没有 SKIP。Scoop shim 仍未构造，不能因缺少专用 SKIP 行而认定覆盖。

Ubuntu 普通测试步骤按 RUNNER_OS=Linux 选择并传入 `-race`；其中八项 CrossUser 因非 root 跳过。随后独立编译 store.test，以 sudo 运行 CrossUser，八项全部 PASS；该 root 步骤不宣称带 race，不回改前一步 SKIP。其余历史平台限制保持。

checkrun 包该次耗时为 Ubuntu 37.855 s、macOS 11.976 s、Windows 27.686 s，只是观测，不是上界或 M-16 解决证据。常规 CI 不运行原型模块，绿色不证明第四批管理或完整 WP2.5 已完成。

## 7. 分层裁定及下一步

| 项目 | 裁定 |
|---|---|
| B-123-1 | 关闭：实际三个入口的数量边界、全量检查、契约及当前清理组合成立 |
| B-123-2 | 主体修复成立；残余 B-123-2-R 尚未关闭，不能整体收口 |
| N-123-1 | 收口，未改变存储接口或零业务写入语义 |
| 其他直接回退 | 未发现额外 BLOCKER；其余局部结论按 Review 123 原限定保持 |
| 固定对象技术结论 | NO-GO |
| 当前合并建议 | 暂不建议合并；正式 CI 条件满足，但仍缺关闭结果完整性修正 |

最小下一步是由负责人决定是否授权仅处理 B-123-2-R 及相应定向回归，保持受审规格、范围、预算和其他断言。无需重做第一批、所有解释器场景或旧原型；本报告没有执行修复，没有代触发／重跑 CI，没有授权合并或第三批。

不修改已删除 worktree 的规范，不将其接受为 L；factgen/legality、维护事项及冻结限制不关闭。不验收 WP2.5／M2。

## 8. 结束核验

结束核验：2026-11-01 11:58:04 +09:00。再次读取 GitHub PR、两个分支 ref 及本地身份：PR 仍 open、未合并，受审分支／PR HEAD 仍为 `b6542be72315260d5ad04049627fcf773b7aaa3a`，树仍为 `e186e00c9d812d479e1bacb8498e4ee73c25789d`，base／GitHub master 仍为 `47f568066c4e4e18cbf28fff2a01cfdda7d34ee4`；没有对象漂移。

工作区唯一新增为未跟踪、未提交的 `docs/reviews/codex-review-124.md`；已跟踪文件和暂存区无改动。审查用隔离副本已删除，没有终止未知进程，没有修改实现或冻结规范，没有触发或重跑 CI，没有提交、推送或合并。报告完成后暂停，等待负责人决定。
