# Codex Review 123 — PR #48 / WP2.5 第二批独立审查

**结论：固定对象技术 NO-GO，暂不建议合并。发现两项 BLOCKER：B-123-1（拒绝信息超过 200 项时违反既有 error.v1 契约）和 B-123-2（新资源关闭链丢弃关闭错误）。** 最终 CI 正式为 8/8 success，但没有覆盖这两个错误分支，不能替代本轮源码及反例核验。

第二批的授权已生效，实施登记先于代码；参数与选择、一致读取、D3/D4、解释器、PD-26、空批次及 G1 准入的大部分接线符合批次边界。没有注册公开 `check run`，没有启动检查进程或接入第三批能力。下述问题均属于本批已实现的契约和资源责任，不以尚未完成后续 Runner 为阻塞理由。

## 1. 固定身份与审查范围

开始核验：2026-11-01 02:47:10 +09:00。读取当前 AGENTS.md，检查工作区、受审分支、GitHub PR 与远程 master。开始时工作区和暂存区干净。

| 项目 | 核验值 |
|---|---|
| 仓库 | `C:\Users\hoshi\Desktop\specmetry` |
| PR | [XKHoshizora/specmetry #48](https://github.com/XKHoshizora/specmetry/pull/48)，open、未合并 |
| 本地及远程受审分支 | `feat/wp2.5-batch2` |
| HEAD | `1a2eda18ba137ba31cb796ce22ad90c415a027db` |
| HEAD 树 | `4d1cae71587fa378d58b388bec2ac6a6645cbd67` |
| base／GitHub master | `47f568066c4e4e18cbf28fff2a01cfdda7d34ee4` |
| HEAD 与 base 的 merge-base | 上述固定 base |
| 最终 CI 合并快照 | `4a0b6a004fe1f8a3a82030366ac242b7211dfcae` |
| 快照父提交，依次 | 上述 base、上述 HEAD |
| 快照树 | `4d1cae71587fa378d58b388bec2ac6a6645cbd67`，与 HEAD 相同 |

本地另有旧的 `master` 引用 `ecb83ba433079ab87fbe57eedd5977f9b1f42ea2`，它不是本次 base，也未用于取差异或下结论；未更新、切换或覆盖该引用。受审分支、base 对象及 GitHub master 均按上表核验。结束阶段的原生命令 `git ls-remote` 遇到 DNS 解析失败，远程身份由 GitHub 连接器的 PR、两个分支 ref 和提交 API 独立核对；没有申请额外权限或改网络设置。

完整 `base → HEAD` 为 **30 文件、3366 行增加、9 行删除**，不是仅审最终 lint 注释。读取五份文档变化、baseline／proposal 的业务校验、`internal/checkrun` 的全部新增实现及测试、CLI 写入路径测试、`gitx.ExecPath` 及测试，并追踪现有事务、worktree 登记、Git 受控环境、私有临时区和错误信封的直接调用链。

依据冻结文档 `f99d648`／技术基准 `f2b5c87`，按规范 §22 的优先关系，重点对照 §3、§4.1、§5.3／§5.6、§10.3、§17.3、§19.1／§19.8／§19.9，生产计划 §4.2／§11，处理记录 §155～§157及 Review 120～122。

## 2. 授权、归档与变化边界

Review 122 原样归档在 `13771e4`，该提交仅含报告；当前文件与归档 blob 完全相同：13676 字节，SHA-256：

`087e38da7373188881e7411879c8578e44d9fcd1a2a44a9113b9cf9abb7c1fff`

PR #47 已合并为固定 base `47f5680`，父提交依次为 `ad592c33306828e96b59f6e7be77dc4af8995bd9`、`b43f9cd6dcb5262d78eb0d42f21dbcf0846cd95c`，树为 `34715f8d7e3ae5478d6009554039e6aba6bc1e10`，与该最终 HEAD 相同。[master CI 41214044060](https://github.com/XKHoshizora/specmetry/actions/runs/41214044060) 为 push、attempt 1、success；八份日志均实际检出 `47f5680`。因此 §155 的第二批条件授权已生效，并非仅依据作者的状态描述。

实施顺序核验如下：

1. `e5201fddd1eabbced24d6262aa8bcd37c3ac4fd3` 先登记计划 §11／处理记录 §156，只有文档。
2. 随后依次实施 D3/D4（`9537fca`）、ExecPath（`913e4a4`）、参数与宿主（`e23e609`）、第 3～6 步（`9456306`）、解释器（`bad52e0`）、第 8～10 步（`f9d6a28`）、D10 说明（`79ebbb8`）及 finder 调整（`501612d`）。
3. 首次 CI 的 lint 失败之后，`6622597` 先记录 §157 的定向收尾授权；最终 `1a2eda1` 的生产差异只有 `os.Stat(final)` 那一行的 `nolint:gosec` 注释。

处理记录的 base 版本是当前版本的完整字节前缀，只追加 §156／§157；历史审查报告未改写。冻结规范、Schema／示例、迁移、第一批存储与纯函数实现、derive／toderive／dbdump、原型、依赖及 workflow 均未变化。AGENTS 与实施计划的状态同步在负责人明确保留的范围内；data-model §8.3 只把 V-P04 的状态同步为已实现，没有改变规则。

## 3. BLOCKER

### B-123-1：三个拒绝入口没有限制 details，合法规模的输入能产生不合法的错误信封

**位置**：

- `internal/checkrun/selection.go:137`、`:141`：显式参数每个问题直接追加，随后全部传给 `WithDetails`。
- 同文件 `:171`～`:180`：D3/D4 每个受影响 AC 直接追加。
- `internal/checkrun/interpreter.go:156`～`:169`：每个受影响 AC 的 `shell_unavailable` 直接追加。
- 关联：`internal/apperr/apperr.go:77`、`:160` 没有数量裁剪；`schemas/error.v1.schema.json:32` 明确 `maxItems: 200`。

**具体反例**：在已初始化项目中给出 201 个不存在的 AC 参数。选项组合合法，进入第 5 步后产生 201 个 details，退出码 2。`noCheckRun` 只处理 message，不减少 details；转为 `error.v1` 后不符合既有 Schema。冻结 §3 没有把参数数量限制为 200，不能把该输入当作不存在的调用。

另外，201 个已保存且违反 D3/D4 的检查，或者 201 个使用不可用解释器的所选 AC，同样分别产生不合法的 201 项信封，退出码仍为 2／6。并不需要真正运行检查或写入事实才能到达。

**独立验证**：在固定 HEAD 的仓库外隔离副本中，只增加审查用纯函数测试，不改受审实现。分别调用实际 `fixed.choose`、`staticRules`、`interpreters`，再经实际 `apperr.ToEnvelope`、JSON 序列化和实际 `contract` 校验：

| 入口 | 200 项对照 | 201 项反例 |
|---|---|---|
| 选择拒绝 | details=200，exit=2，契约通过 | details=201，exit=2，契约失败 |
| D3/D4 拒绝 | details=200，exit=2，契约通过 | details=201，exit=2，契约失败 |
| 解释器拒绝 | details=200，exit=6，契约通过 | details=201，exit=6，契约失败 |

该运行正常编译并进入六项断言；三项反例由契约断言失败检出，不是编译失败。解释器反例使用已有“根最终路径不可核验”的状态，不执行解释器或新机制实验。

最小复现的核心如下，放在隔离副本的 `package checkrun` 测试中，使用该副本现有 contract 和 apperr；不需要数据库：

```go
args := make([]string, 201)
for i := range args { args[i] = fmt.Sprintf("AC-001-%d", i+1) }
_, err := (fixed{}).choose(Options{Criteria: args, Isolation: "off"})
raw, _ := json.Marshal(apperr.ToEnvelope(err))
v, _ := contract.New()
// err 为选择拒绝，len(apperr.From(err).Details) == 201；下行返回契约错误。
schemaErr := v.ValidateJSON("error.v1", raw)
```

**违反依据及影响**：冻结规范 §1 的契约表、§17.3 D10、§21.3／§22 采纳的既有 error.v1 数量上限，以及本批计划 §11.8“每个拒绝都通过 error.v1 校验”。当前虽拒绝执行、没有错误地产生检查事实，但返回的契约不成立；调用方不能依赖已承诺的错误结构。本批已授权 D10，不能推到第七批才处理。

**最小修正方向**：在本批错误构造的公共边界统一实施数量限制，并用受控 message／保留的汇总位置表达总数和省略数，整个 details 仍不得超过 200 项。保留已展示项的真实参数位置／AC 对应与原退出码，不增加 Schema 上限、不静默改成只处理前 200 个参数。补 199／200／201 边界和三个入口的契约对照；本轮未实施修正。

### B-123-2：新关闭链没有把关闭结果完整传给调用者

**位置**：`internal/checkrun/prepare.go:77`～`:81`、`:206`～`:221`；同源的新增 Windows 观察句柄关闭在 `internal/checkrun/interpreter_windows.go:27`。测试辅助路径 `internal/checkrun/prepare_test.go:228` 也忽略了 `Batch.Close` 的结果。

**允许的事件顺序与源码证明**：

1. `Prepare` 已成功取得 DB；Git 项目在第 8 步也成功取得 `gitx.Scratch`。
2. 之后目标读取失败，或第 9 步因不支持的 Writer 拒绝，函数准备返回原步骤错误和 nil Batch。
3. defer 调用 `b.Close()`。该方法确实收集 `Scratch.Close` 与 DB.Close 的错误，但 defer 用 `_ =` 丢弃整个返回值，最终只留下原步骤错误。
4. `b.closed` 已置 true，再调用 Close 返回 nil；`Prepare` 的调用者甚至没有收到 Batch，无法从返回对象取得第一次关闭的失败结果。

`gitx.Scratch.Close` 的现有实现会把 witness、目录及 root 的 Close 错误返回，`DB.Close` 也返回错误；新编排层明确丢弃了这些结果。这不是要求重审或改写 gitx 方案 B，而是新调用者没有消费现有接口已经提供的结果。

Windows 新 `finalPath` 也在成功读出路径之后无条件忽略 `CloseHandle` 错误，因此该错误不能阻止 finder 接受候选或让成功的 Prepare 报告资源问题。它与“路径最终值读取成功”是两个维度，不能用前者证明观察句柄已释放。

**影响及分级**：关闭失败不证明资源一定仍打开，也不证明它已释放；本轮没有复现原生 Close／CloseHandle 故障，不能声称观察到真实泄漏。但上述返回值丢失是确定的控制流，违反本批计划 §11.4 `Prepare`“之前打开的都已关闭”的责任与用户要求的资源错误消费。失败返回时缺失清理事实，Windows 路径还可能在关闭出错后照常返回成功。当前尚无检查进程，不涉及 F1 或越界发信号；仍是本批新增资源管理自身的问题。

**最小修正方向**：让失败收尾保存并传递第一次关闭结果，保留原步骤错误及独立的受控清理说明；明确关闭失败时的释放知识和后续责任，不以 `closed=true`／第二次 nil 充当释放证明。Windows finalPath 也须消费自己取得的句柄的关闭结果。避免重复 Close；不擅自为此新增 C-4／D21 的禁用条件或更改既有错误优先规则。增加实际编排的关闭故障接缝及健康对照，确认原错误和额外关闭错误同时保留；注入不能覆盖真实 Close 的失败。测试自己的 Cleanup 应检查返回值。无需为本修正提前实现第四批任务管理。

## 4. 其余实现与分批边界

以下为当前可采纳的局部结论，不覆盖上述两项失败分支。

| 项目 | 源码与现有测试核对 | 裁定 |
|---|---|---|
| 参数及整体拒绝 | `options.go` 在任何访问前拒绝 AC／all 两者都有或都无、auto 和非法 isolation；required 紧接着退出 6。`choose` 保留参数位置，重复指向后出现的位置，列出同一步的各项问题 | 符合 D15／D16；数量边界见 B-123-1 |
| 选择、顺序与范围 | 显式 AC 要求 active、automated 且有 check；all 遍历全部符合条件的基准 AC，不以交付 scope 过滤；`baseline.LoadAll` 使用既有数值编号比较 | 符合 E1 和 all 的范围；测试含 9／10 的数值排序、范围外 AC |
| 固定读取 | `selection.go:46` 的一次 `DB.WriteTx` 读取项目、W、排除根、AC、保存哈希及目标；结束后复制 check、inputs、shell，后续不重读当前配置替换 | 满足 D13。BEGIN IMMEDIATE 会取得既有写锁，但回调不改行、不调用 NextSeq；不是新增 seq 的写入 |
| 事务生命周期 | WriteTx 在读取完成后结束；rows 的终态与关闭结果在正常读取出口合并检查；失败返回，不带活跃事务进入解释器／Git 对象阶段 | 没有将事务延长到后续检查；未新增存储重试 |
| 提前出口及副作用 | 参数 → required／host → 项目／DB → 路径扫描 → 选择 → 静态规则 → 解释器 → PD-26／Scratch／S／空批次 HEAD → 输出类型 → 空批次登记 | 条件拒绝按步骤裁定；成功获取的资源关闭结果有 B-123-2。测试读取全部表和 seq，不只检查 runs |
| D3/D4 写入 | `baseline.Check.LineBreakProblem` 是共用业务规则；proposal 的新增／本次 check 更新、审核编辑、baseline 写入适用；没提供 check 的更新不重验旧 check | cmd 拒绝 CR/LF，sh／bash／缺省拒绝 CR；pwsh／powershell 不扩大限制。旧读取、原配置及 hash 未改，Schema 未收紧 |
| 解释器来源 | Unix sh 固定 /bin/sh；bash／pwsh 按绝对 PATH 项；Windows cmd／powershell 用系统目录，pwsh 精确 pwsh.exe，不扩 PATHEXT | 空项与相对项跳过；普通文件、Unix 执行位、最终路径和分隔符边界排除均有接线 |
| Git for Windows | 新 ExecPath 复用 gitx 的实际 Git 选择、版本检查、受控环境与取消／退出错误处理；只读 --exec-path。Windows 校验 R/env/libexec/git-core、允许 env、usr/bin shell 及 msys-2.0.dll，R 也排除 worktree | 没有用 PATH 上任意 sh 代替。PathPrefix 只是为第六批保存值，本批没有修改宿主／gitx PATH |
| PD-26 与 S | `CheckObjectReads` 位于 NewScratch、CommitOf 之前；目标 ref 在固定目标上解析一次，无目标或无 commit 为无 S；空批次另外读取当前 HEAD | 顺序符合 §4.1；PD-26 及对象失败用例实际在 CI 执行 |
| 输出类型 | 当前仅允许非 nil 的原始 *os.File；JSON 时只检查转发目的 stderr；不关闭调用者 stdout／stderr | 受控内存目的端与封闭／复制能力明确留第四批，未谎称完成 |
| G1 | 包级 process、同锁检查并增加 active；同锁置位与加入持有项；移除清单不复位；Recheck 在同锁读禁用状态 | 测试接缝调用实际 host 原语和同一个 prepare；生产每项循环／置位调用点留第四～六批，没有计为已接线 |
| 空批次 | 已登记时不调用 EnsureWorktree；未登记时新事务再查路径，已被并发登记则不更新；真正首次登记才分配一个 seq。bodyDone 区分提交前错误与提交不确定 | 不新增 check_runs；已登记 last_seen_at 不变；文档校验发生在登记后，失败不谎称撤销已提交登记 |
| D9／D10 | 空 runs 是非 nil 数组，TargetSnapshot 使用固定 S 和本次 HEAD；拒绝补“no check was run”，不把 exit 6 解释为安全重跑 | 小规模现有信封与空批次测试成立；大规模 D10 仍有 B-123-1 |

新增 `gitx.ExecPath` 没有改变 gitx 既有全局参数、禁用自动 fetch 的机制、环境过滤或取消返回路径。`TestExecPath` 验证 GIT_EXEC_PATH 的诱饵被过滤；该新增测试没有单独构造取消和非零返回，对这些分支的本轮结论来自复用调用链的源码，不冒称已有专门动态覆盖。

单行 `nolint:gosec` 仅抑制该次 Stat 的 G703：这里按冻结规则读取外部候选路径元数据，后续仍检查文件类型、执行位及 worktree 排除。理由及范围成立；它不是对执行文件的认证，不证明规范化可消除所有路径竞争，也没有增加信号或执行授权。

## 5. 待决规范问题及非阻塞意见

### Q-WORKTREE：已删除的登记路径

`newFinder` 对本项目所有登记根取最终路径；任何一个失败都置 `unverified`，lookup 因而拒绝。`TestLookupWithAnUnresolvableWorktree` 与 Prepare 对照覆盖此行为。这**符合当前冻结 §19.8**，没有为绕过缺失根而放宽归属或来源检查。

删除过已登记链接 worktree 后，该项目之后的非空调用可能持续被拒绝，空批次仍能走自己的路径。计划 §11.11 已如实登记；负责人没有批准放宽，也没有接受为 L。本轮不据此单独阻塞第二批合并，不把当前符合规范的拒绝改判成实现缺陷。

建议最迟在第七批公开命令注册／面向用户验收之前，由负责人明确保留现状还是授权规范修订；若调整会改变排除根与解释器预检，应在第四批端到端执行准备接线之前确定影响。未决定时，后续只能继续现有保守规则，不能自行忽略旧路径、删除登记或把未知当成安全。

### N-123-1：把“没有写事务”限定为没有额外的登记写事务

计划 §11.9 第 757 行称已登记空批次“不打开写事务”，而 §11.5 与实际 readFixed 均使用 DB.WriteTx 做不写行的快照读取。实现满足零新增事实／seq／last_seen_at 的语义，不需为了措辞另增 ReadTx 或修改已验收存储接口；建议文字改为“不再打开登记写事务，固定读取事务不写入”。这是非阻塞精化，不是允许额外业务写入。

### 其他保持的限制

- Scoop shim 未构造；便携、MinGit 的目录夹具不等于实际安装器全流程验证。按现有计划如实保留适用性限制，不扩大解释器来源。
- G1 清单观察任务、已结束记录、晚到任务错误和受控输出目的端属于第四批；当前 host 数据结构和准入测试不证明这些能力已完成。
- factgen/legality 差额仍须在首次用于验证新 Runner 历史之前另行处理；没有因本批绿色而关闭。
- 维护事项、N-087-1、冻结时的 H4/H6 等限制不混入本批修复，不关闭 M-16，也不重做整套原型审查。

## 6. 审查者独立验证

环境：Windows/amd64；显式使用已存在的 `C:\Users\hoshi\AppData\Local\mise\installs\go\1.27.1\bin\go.exe`，实际 `go version go1.27.1 windows/amd64`。没有调用 mise 的 go shim，没有安装升级；设置 GOTOOLCHAIN=local、GOPROXY=off、GOSUMDB=off。它不是 CI 的 Go 1.27.3，证据范围分开。

### 6.1 现有定向测试：一次，保留首次失败

```text
go test -tags nowebui -count=1 -timeout 90s -v
  ./internal/checkrun ./internal/baseline ./internal/proposal ./internal/gitx
  -run '^(TestOptionsCheck|TestHostRefusesOnceDisabled|TestHostRecheck|TestHostAdmissionRacesDisabling|TestWithin|TestLookupWithAnUnresolvableWorktree|TestLookupGitForWindowsLayouts|TestLookupOnThisMachine|TestLookupPwshOnWindows|TestLineBreakProblem|TestCheckStaticRefusesLineBreaks|TestCheckEditedDataRefusesLineBreaks|TestExecPath)$'
```

13 个选中的顶层测试中，11 个 PASS、2 个在夹具准备处失败，整次退出非零：

- `TestLookupPwshOnWindows` 在 `mklink /J` 遇到 Access is denied，没有到达联接／短文件名排除断言。
- `TestExecPath` 在夹具 `git init` 写 `.git/config` 时遇到 Permission denied，没有到达目标 ExecPath 断言。
- 参数、宿主拒绝／再检查／并发、路径边界、缺失根、Windows 布局及本机来源、LineBreakProblem 和两项 proposal 校验均通过。

没有提高权限或重跑上述失败来取得绿色；环境拒绝不作为产品失败，也不计为未到达断言的验证成功。

### 6.2 存储／编排定向测试：一次

```text
go test -tags nowebui -count=1 -timeout 90s -v ./internal/checkrun
  -run '^(TestPrepareFixesTheConfiguration|TestPrepareEmptyBatchRegistrationFails)$'
```

两个测试都在自建仓库的 git init 配置写入时被拒绝，退出非零，未进入固定读取／登记失败的目标断言。本次没有运行到迁移锁；不能把 Review 122 当时的 `.migrate.lock` 限制冒称为本次失败位置。相应动态断言另采本轮 CI 证据。

### 6.3 隔离纯函数反例：一次，三个边界对照与三个检出

预先限定为 B-123-1 的三个既有入口，每个仅跑 200／201 两种数量；不运行用户检查、进程树或新平台机制。用 `git archive` 导出固定 HEAD，在仓库外自有临时副本中加入 `TestReview123D10`，执行一次：

```text
go test -tags nowebui -count=1 -timeout 90s -v ./internal/checkrun -run '^TestReview123D10$'
```

审查用测试文件 SHA-256 为 `271c4c4d34dac4cf23a8d74bd92a7fca33c763087fa76fce2b5d49e310fa3b77`。结果见 §3：三项 200 对照 PASS，三项 201 契约断言 FAIL，整次退出 1。该哈希仅识别本轮夹具；结论来自实际运行与调用链，不以哈希代替证据。

测试已结束；核对隔离目录的绝对路径和本轮唯一名称后删除，确认不存在。原仓库未添加该测试。B-123-2 采用静态返回值／出口证明，没有声称模拟或取得原生关闭故障。

`git diff --check base..HEAD` 通过。没有运行全量本机套件、原型或用户配置检查，也没有复现作者其他本地运行、跨平台 vet 或反向检查。

## 7. CI：正式状态、实际检出及适用性

### 7.1 首次失败保留

[run 41219007149](https://github.com/XKHoshizora/specmetry/actions/runs/41219007149)：pull_request、attempt 1、completed/failure，7/8，唯一失败为 Go lint（job `124589462772`）。日志实际检出：

- HEAD：`501612da21ee8410b3b7a0914c9a161622db4126`；
- 合并快照：`f3765c4a39eef0ad7d015849f738a7dddbc594e9`；
- 父提交：固定 base、上述旧 HEAD；树 `157dbb18cda9a12c4a6647702ca1b31f0f242608`；
- `internal/checkrun/interpreter.go:90:20: G703: Path traversal via taint analysis (gosec)`，随后 issues found。

该旧快照不在本地对象库中，父提交和树由 GitHub 提交 API 核验。没有重跑这次运行，最终绿色属于新固定 HEAD 的新 run。

### 7.2 最终运行

[run 41224740139](https://github.com/XKHoshizora/specmetry/actions/runs/41224740139)：pull_request、attempt 1、completed/success，八个作业全部 success。读取八份原始日志，每份 checkout 的 `git log -1` 均为完整 `4a0b6a004fe1f8a3a82030366ac242b7211dfcae`，不是仅用 job.head_sha 推定。

| 作业 | Job ID | 正式结果／实际 checkout |
|---|---|---|
| Web | 124606437040 | success／4a0b6a0 |
| Go lint | 124606437186 | success／4a0b6a0 |
| Go tests (Windows) | 124606437246 | success／4a0b6a0 |
| Go tests (Ubuntu) | 124606437373 | success／4a0b6a0 |
| Go tests (macOS) | 124606437479 | success／4a0b6a0 |
| Build with dashboard (macOS) | 124606641058 | success／4a0b6a0 |
| Build with dashboard (Windows) | 124606641077 | success／4a0b6a0 |
| Build with dashboard (Ubuntu) | 124606641179 | success／4a0b6a0 |

Go 日志为 1.27.3，Windows/amd64、Linux/amd64、Darwin/arm64。三个平台各有 27 个带测试的包输出 ok；`[no test files]` 的包不计入 27。Windows 的 CLI 单独运行与其余包合起来恰好覆盖这些包，没有因分包重复统计。

本轮 Prepare、G1、固定配置、选择／扫描／顺序、PD-26、输出、S、空批次及提交错误等目标测试均实际执行并 PASS；baseline/proposal 规则与 CLI 的 `TestCheckLineBreaksOnEveryWritePath` 也在三平台通过。Windows 另实际执行布局矩阵、系统来源、PATH 排除；Unix 使用自身专用路径／符号链接测试，不能写成 Windows 测试在所有平台执行。

Windows `TestLookupPwshOnWindows` 日志为 PASS，没有“no short name on this volume”那条未覆盖说明；结合源码的两分支和逐项正反断言，可确认该次短文件名分支被加入并检查，而非仅凭测试名称推定。联接同样未走准备失败。没有 Scoop 专用运行，不能从这些 PASS 推出其 shim 已测。

Ubuntu 普通步骤确实按 RUNNER_OS=Linux 选择 `-race` 并传给全包测试；八项 CrossUser 在该普通非 root 步骤 SKIP，随后独立编译 store.test、以 sudo 运行 CrossUser，八项全部 PASS。root 步骤没有声称带 race，两种证据分别保留。

### 7.3 FAIL／SKIP 及次数口径

最终八份日志没有测试 FAIL，三个普通 Go 测试日志的 SKIP 集合与固定 base 的 master CI **逐名相同**：Windows 10、Ubuntu 15、macOS 7。没有新增批次测试 SKIP。

三个平台共同的七项为：

- CLI：`TestClaimWithARealGitThatFetches`、`TestInitRefusesRealOldGit`、`TestARealGitThatFetches`；
- gitx：`TestResolveCommitsWithAGitThatFetches`、`TestWP23ReadsWithAGitThatFetches`；
- sqlite：`TestM14ReadOnlyProbe`、`TestM14OpenPath`。

Windows 另三项：gitx 的 `TestResolveCommitsWithUnreadableStorage`、`TestClassifyWithUnreadableStorage`，CLI 的 `TestSourceListUnreadableSource`。Ubuntu 另八项是上述普通步骤的 CrossUser，后续 root PASS 不回改其 SKIP 事实。

checkrun 包耗时为 Ubuntu 28.949 s、macOS 10.506 s、Windows 33.389 s，只是该次观测；不是上限、性能承诺或 M-16 关闭依据。常规 CI 不运行独立原型模块，也不证明后续平台执行、任务移交或完整 Runner 已通过。

## 8. 分层裁定与最小下一步

| 问题 | 裁定 |
|---|---|
| 授权及实施登记 | 符合；先登记后代码，合并核验与 master CI 条件已生效 |
| 第二批实现整体 | NO-GO：B-123-1、B-123-2 尚需处理 |
| 固定 PR 合并建议 | 暂不建议合并；最终常规 CI 条件已满足，但不能覆盖实现阻塞 |
| D3/D4、读取固定、选择、空批次等局部规则 | 按 §4 和实际测试范围采证，不扩展成完整 WP2.5 通过 |
| 已删除登记 worktree | 符合现行冻结规则；单独待负责人决定，不擅自接受为 L，不作为本批额外阻塞 |
| 后续能力与状态 | 第三批未授权；第四批任务／输出接线及第七批公开入口仍未完成，不计为当前完成项 |

建议负责人先决定是否授权仅修复两项当前阻塞及对应回归：错误信封边界、关闭结果的持续消费与责任；N-123-1 可同批做文字精化。任何规范政策、已删除 worktree 的处置、CI 资源或后续批次分别决定。本报告没有执行这些修正，没有触发或重跑 CI，也不产生合并授权。

冻结技术基准及已批准产品选择保持；不验收 WP2.5／M2，不授权第三批。结束身份和工作区的最终回执见下一节。

## 9. 结束核验

结束核验：2026-11-01 02:56:58 +09:00。再次读取 GitHub PR 和两个远程分支 ref：PR 仍 open、未合并，远程受审分支／PR HEAD 仍为 `1a2eda18ba137ba31cb796ce22ad90c415a027db`，base／GitHub master 仍为 `47f568066c4e4e18cbf28fff2a01cfdda7d34ee4`；本地受审分支 HEAD 与树仍为固定值，没有对象漂移。

`git diff --exit-code` 与暂存区检查均无改动；工作区唯一新增为未跟踪、未提交的 `docs/reviews/codex-review-123.md`。隔离纯函数夹具已删除，没有终止未知进程，没有修改实现／测试／冻结规范，没有触发或重跑 CI，没有提交、推送或合并。完成审查后暂停，等待负责人决定。
