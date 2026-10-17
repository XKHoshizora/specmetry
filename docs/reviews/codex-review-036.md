# Codex Review 036：M1 退出条件补证据的独立审查

- 审查日期：2026-10-17（Asia/Tokyo）
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #16](https://github.com/XKHoshizora/specmetry/pull/16)
- 目标提交：`e1d2ea7fea4ceabf58cf831d160ee7d8c0122875`
- 基准提交：`19600ec0176263c965ec34ef85519ae1dbafee2a`
- CI：[GitHub Actions 39007328726](https://github.com/XKHoshizora/specmetry/actions/runs/39007328726)
- 原始退出条件证据报告：[PR #15](https://github.com/XKHoshizora/specmetry/pull/15)，报告提交 `cf72f208c5c8f6efb682e7326adb15ca8880561f` 中的 `docs/m1-exit-evidence.md`。
- 范围：仅审查 PR #16 新增的五项 P0 测试证据、正式二进制测试、集成证据定义，以及 SRC-07 的真实调用链和持久化结论。不重新全面审查已获 GO 的九个工作包。
- 方法：固定提交差异及必要调用链静态审查、原始证据报告核对、隔离副本中的定向测试、三平台 CI 元数据与日志核查。
- 补丁身份：上述固定提交范围的完整 binary/full-index diff，SHA-256 为 `b8d04a67e1b4ac11479c48e4c5bb171bb7cca992d369ca4d1635044b742a5ee2`。

**最终结论：GO。PR #16 可以合并。**

实际差异仅包含测试、测试辅助程序和文档，没有生产业务代码变更。

## BLOCKER

无。未发现足以否定新增证据的测试缺陷或直接回归。

## 五项 P0 补证据

五项新增测试均有有效前置条件，断言覆盖预期业务事实，而不只是退出码。

| 验收项 | 独立审查结论 |
|---|---|
| **CH-01** | [m1exit_test.go:167](../../internal/cli/m1exit_test.go#L167) 对 `propose --apply`、`question dismiss`，分别设置 `CLAUDECODE` 和 `CLAUDE_CODE_ENTRYPOINT`，在真实 PTY 中验证 exit 3、拒绝原因、无提示、canary 不泄露及业务状态不变。零读取由进程内计数器直接验证，并与生产执行体开头的通道检查相互印证；PTY 测试证明无输入也能结束。 |
| **CH-02** | [m1exit_test.go:212](../../internal/cli/m1exit_test.go#L212) 覆盖 `source ack`、`source add --allow-any-type`、`propose --apply`、`question dismiss`，实际保持 stdin 为终端、将 stdout 转到文件；带和不带 `--json` 均拒绝且无写入。JSON 文件经过完整解码、EOF 检查及 `error.v1` 校验。另有 stdin 非终端的子进程验证。 |
| **BL-21** | [m1exit_test.go:244](../../internal/cli/m1exit_test.go#L244) 明确断言环境变量提示、准确字段指针及 `github_token` 规则；检查文本和 JSON 输出、数据目录及写入计数，填补了原先只验证拒绝、未验证提示的缺口。 |
| **BL-27** | [m1exit_test.go:285](../../internal/cli/m1exit_test.go#L285) 通过两份实际 Proposal 构造冲突，并先断言条目确为 `conflict`、Baseline 为 v2。确认冲突但缺少类别时，两种接受方式均被拒绝；提供类别后的成功对照验证 v3、最终优先级和保存的类别。 |
| **PRV-07** | [m1exit_test.go:322](../../internal/cli/m1exit_test.go#L322) 覆盖 summary、item note、嵌套 final_data 和 scope note；断言准确指针、规则、确认前拒绝、不回显及状态不变。无密钥的合法对照能够成功审核。 |

### CH-01 / CH-02 的证明边界

- Agent 变量由真实子进程环境传入，生产 `executeRoot` 执行实际通道检测，不是用注入的 `agent_cli` 代替全部进程级验证。
- Agent 变量的值为运行时生成的 canary；测试核查输出和数据目录均不包含其值。
- 正常人类通道的对照会显示确认，拒绝确认后不写入，避免将所有操作无条件拒绝也误认为正确实现。
- CH-01 的 PTY 测试不发送确认输入；如果误进入阻塞的人工确认路径，测试不会按预期正常结束。精确的输入读取次数为 0，则由额外的进程内 `readCounter` 验证。两者的证明能力没有混同。
- CH-02 的 stdout 文件会实际读回；带 `--json` 时要求整份输出是一份合法 `error.v1`，不只是查找 JSON 前缀或某个错误字符串。
- 无写入断言覆盖 seq、Question、Proposal、Baseline、Source、Target、Scope 等相关事实。

### BL-21 / BL-27 / PRV-07 的前置条件

- BL-21 的合成密钥位于实际 AC Check command；除了 exit 2，还断言 `/items/2/data/check/command`、`secret_detected: github_token`，以及“reference an environment variable by name instead”提示。该命令只是提案输入，没有被执行，不产生网络请求。
- BL-27 在缺少 `change_class` 之前先验证冲突已实际形成，且审核使用当前版本并明确 `acknowledged_conflict=true`。拒绝不是由未确认冲突或过期版本代替触发；两种接受方式都验证了 `/items/0/change_class`。成功对照证明合法审核仍可应用。
- PRV-07 使用人类通道进入真实 Review 命令树，因而不是在通道层提前拒绝。四个位置的密钥都要求由同一整份文档扫描产生准确 Pointer 和规则编号，文本与 JSON 错误不泄露密钥，确认提示之前拒绝，相关业务状态不变。
- PRV-07 的 `/scope/note` 场景同时带有业务前置条件不成立的 Scope；扫描发生在这些业务检查之前。测试要求准确的密钥 Pointer 和规则，因此不会将后续 Scope 拒绝误当作密钥防线生效。

## 正式二进制测试

**正式二进制证据有效。**

[TestBuiltBinary](../../internal/cli/binary_test.go#L51) 实际构建 `./cmd/specmetry`，经过 [main.go:12](../../cmd/specmetry/main.go#L12)，并启动独立子进程。它验证了：

- `init --json`、`source add --json`、`source list --json`、`target show --json` 的成功输出：完整 JSON、EOF 和各自 Schema。
- 普通成功、未知命令、项目未初始化、非终端人类专属操作等真实进程退出码。
- Agent 变量、PTY、stdout 重定向、取消及正常人工确认。
- 确认成功后，Question 实际成为 `dismissed / human_cli`。

[buildSpecmetry](../../internal/cli/completion_test.go#L24) 运行 `go build -tags nowebui -o <临时路径> ./cmd/specmetry`；[runSpecmetry](../../internal/cli/completion_test.go#L46) 通过 `exec.Command` 启动程序，并从进程结果读取真实退出码。

PTY 启动器 [execHelper](../../internal/cli/m1exit_test.go#L56) 从子进程的 `ExitError.ExitCode()` 取得退出码，再输出测试标记；没有把启动器自身的成功退出当成业务成功。它将真实程序的 stdin、stdout、stderr 接到伪终端，重定向场景则将子进程 stdout 接到文件。

测试二进制 helper 与正式程序的差异已披露，未发现会导致本次证据错误通过的替代行为：

- `cli` helper 使用生产命令树和 `executeRoot`，但额外注册测试专用命令，不经过生产 `main`。
- `cli-stdout-to-file` 在测试进程内替换 `os.Stdout` 并使用同一生产入口；正式二进制测试另有真实子进程 stdout 句柄重定向证据。
- `exec` helper 是启动器，真正被检测 stdin/stdout、执行业务及退出的是它启动的生产二进制。

`nowebui` 构建验证的是生产 CLI 入口，不是正式发布包。文档准确说明了 Dashboard 构建及 M6 打包验证的边界，未把这项冒烟测试当成所有业务验收或发布包验收的替代。

新测试及相关构建、PTY 路径中没有发现会跳过本轮关键断言的环境条件；构建或 PTY 建立失败会使测试失败，而不是被标为通过。

## SRC-07：不解析文档语义——PASS

审查标准：Specmetry 读取来源文档内容仅用于计算哈希和核对提案明确给出的片段，不进行文档语义解析、自动提取需求或保存来源全文。

**正式独立审查结论：PASS。** 结论来自实际调用链，不是仅根据注释或依赖库清单推断。

| 路径 | 源码证据与结论 |
|---|---|
| Source Add | [source.go:227](../../internal/cli/source.go#L227) 解析路径，按扩展名判断文档类型，再计算哈希；[Register](../../internal/source/registry.go#L204) 只保存路径、类型、位置、哈希、大小及登记事实。 |
| 哈希与观测 | [HashFile](../../internal/source/source.go#L49) 分块计算 SHA-256 和大小；[Observe](../../internal/source/registry.go#L58) 比较哈希；[Acknowledge](../../internal/source/registry.go#L221) 与 [SetHash](../../internal/source/source.go#L33) 处理确认事实及来源集合，不处理正文语义。 |
| 片段核对 | [CheckSources](../../internal/proposal/store.go#L89) 收集提案明确提供的片段；[CheckExcerpts](../../internal/proposal/excerpt.go#L58) 去除 Unicode 空白后执行流式字符串匹配，返回校验标记和哈希，不抽取需求。 |
| 初始化 | [ListFiles](../../internal/project/files.go#L28) 获取路径、文件类型和大小；[DetectStage](../../internal/project/rules.go#L249) 据此识别阶段，没有读取来源正文推断客户要求。 |
| 持久化 | `CheckedSource` 不携带片段原文；Question 和 Baseline 的引用写入将 `excerpt` 设为 NULL。迁移 0004 对引用及 Proposal JSON 提供 hash_only 约束。 |

具体追踪结果：

1. Source Add 的类型白名单由 `project.IsDocumentType` 按文件扩展名判断；人工指定的 Kind 是枚举输入，没有根据正文生成分类。登记函数没有接收正文参数。
2. `HashFile` 使用有界缓冲区逐块更新 SHA-256 和大小，不返回文本，也没有将文本传入持久化层。
3. Source List/Observe、Ack 及来源集合哈希只处理路径、当前哈希、已确认哈希和确认事实。审核沿用引用时的 `review.Recheck` 同样只重新计算来源哈希。
4. `CheckSources` 的待匹配模式来自 Proposal 的显式 Source Ref。`CheckExcerpts` 做 UTF-8/NUL 判断、Unicode 空白规范化与 KMP 流式匹配；它没有根据标题、段落、文档结构或语言含义生成 Goal、Requirement、AC 或 Question。
5. PDF、Office 等不可直接校验的类型不会被解码为文档结构；返回 `not_checkable`。`verified` 只表示给定片段匹配，不表示语义正确或需求覆盖完整。
6. 初始化的候选来源、阶段和测试命令提示由路径、扩展名、大小及文件名规则决定。没有从客户文档正文推断未确认需求。
7. `hash_only` 下，`proposal_items.sources_json` 保存 `CheckedSource`，只有 Source ID、locator、excerpt hash、校验结果和 source hash；Question 引用及正式 Baseline 引用的 INSERT 明确使用 `excerpt = NULL`。
8. 迁移 0004 的 `proposal_items_hash_only_insert/update` 和 `source_refs_hash_only_insert/update` 提供第二道防线；项目初始化使用 `hash_only`，M1 没有提供切换 full 模式的命令。
9. 对生产文件内容读取路径进行了有限穷举核对：来源正文走哈希和片段匹配；其余直接读取涉及配置、Marker、Git exclude、显式交换输入及 Git 能力探测事件等不同用途。未发现将来源正文送入语义提取、spool、日志或输出的可达路径。

这里的 UTF-8 检查、空白规范化和明确片段匹配属于允许的引用校验，不是文档语义解析。当前 M1 来源调用链没有将正文用于自动提取需求或将全文持久化。

此外，独立运行了以下既有隐私测试，均通过：

- `TestSourceContentIsNeverStored`：来源正文 canary 经 Add、List、Ack、Remove 不进入输出或数据目录。
- `TestProposeNeverStoresExcerpts`：提案引用片段不进入输出或持久化数据。
- `TestReviewNeverStoresExcerpts`：审核、读取和历史输出不包含片段原文，正式引用仍为 NULL。

## 验收定义与文档

三类集成测试证据定义符合实际证明能力：

- **进程内**：证明命令树的业务行为、输入校验、事务、数据、JSON 和返回的退出码值；不能单独证明操作系统 TTY、重定向、实际进程退出状态或 main 接线。注入通道不证明通道检测。
- **进程级**：测试二进制 helper 在子进程和真实 PTY/ConPTY 中调用生产入口，能够验证环境变量、终端、重定向、提示和进程边界；不等同于正式发布二进制。
- **正式二进制**：构建并运行 `cmd/specmetry`，补上 main、操作系统句柄和实际进程退出码证据；`nowebui` 与 Dashboard/发布包的差异已经明确披露。

没有要求将所有既有进程内业务测试改写为正式二进制测试，也没有用新的分类掩盖需要进程级验证的 CH 边界。

**正式授权依据：** 审查时发现 PR #15/#16 的公开讨论中没有可引用的批准记录，因此向产品负责人核实。产品负责人于本次对话明确回复：**“已正式批准，可记录本次确认”**。该确认适用于 acceptance-criteria §1 的三类集成测试证据定义；本轮据此认定不再存在待确认的产品决策。

SRC-07 的 M1/M2 划分与 Review 019、data-model §4.5 和 review-resolution §24 的既有边界一致：

- M1：来源登记、人工例外、重复登记不自动确认变化，以及本次“不解析语义”的独立审查。
- M2：`SOURCE_CHANGED` 差距及其对交付状态的影响。

这不是新增延期，没有将原本应在 M1 完成的来源写入行为移至 M2。

## NON-BLOCKER

没有新增阻塞验收的证据问题。以下既有事项继续独立保留：

- **Windows 数据库创建期间偶发 `Access is denied`**：此次三平台通过不代表已经解决，不应抹去此前失败记录，也不应一般性忽略权限错误。
- **项目名称代码围栏安全修复**：仍是 M1 最终验收前的独立任务，不要求 PR #16 一并实现。
- **原始退出条件证据报告更新**：仍需明确引用合并后的提交和本次 SRC-07 审查结论，不能将旧报告原有的 INCOMPLETE 记录无依据地直接改为 PASS。

以上未解决事项没有被归类为已接受风险。

## PRODUCT DECISION

无待决事项。三类证据定义的正式授权已在本次对话中确认；SRC-07 阶段边界沿用已有批准记录。

## ACCEPTED RISK

未新增接受风险。

`nowebui` 冒烟测试与正式发布包的差异是已经披露的验证范围边界，不代表已证明发布包或完整 Dashboard 行为。M6 的打包与分发验证仍按既定计划完成。

## 测试与 CI 证据

### 本机独立运行

Windows 隔离副本中，以下六项新增测试全部通过，没有跳过：

- `TestBuiltBinary`。
- `TestAgentVariablesRefuseTheRemainingHumanOnlyCommands`。
- `TestRedirectionRefusesTheRemainingHumanOnlyCommands`。
- `TestProposeSecretInACheckNamesAnEnvironmentVariable`。
- `TestReviewConflictItemNeedsAChangeClass`。
- `TestReviewRefusesASecretAnywhereInTheDocument`。

上述定向运行共约 57 秒。另一次运行中的三项来源正文与片段隐私测试全部通过，共约 13 秒。

所有运行使用隔离副本、临时仓库和临时 SPECMETRY_HOME。没有操作真实项目数据或真实远端，没有在原仓库执行会生成文件的测试。

本轮独立运行的是目标提交上的现有测试。PR 描述中的五类变异验证作为作者提供的辅助证据读取，没有冒称全部由本轮独立重跑。

### GitHub Actions

已核查 [CI 39007328726](https://github.com/XKHoshizora/specmetry/actions/runs/39007328726) 的元数据、作业身份和相关日志：

- 首轮 attempt 1，8/8 success。
- 全部作业对应 `e1d2ea7fea4ceabf58cf831d160ee7d8c0122875`。
- Windows、Ubuntu、macOS 均实际执行上述六项新增测试并通过，没有 SKIP。
- Ubuntu 的 Go 测试启用 Race Detector。
- 三平台 `TestCheckWhileTheDatabaseIsCreated` 此次通过；这不是其既有间歇性失败已关闭的证据。
- Dashboard 构建任务通过，但没有把它扩大表述为完整发布包验收。

跨平台结论来自 CI 日志核查；本机独立运行环境为 Windows，两类证据未混同。

## 最终回答

| 问题 | 结论 |
|---|---|
| CH-01、CH-02 的新增进程级证据是否有效？ | 有效。真实 PTY/ConPTY、Agent 变量、重定向和业务无写入断言覆盖了原缺口；正式二进制另提供生产 main 与真实句柄、退出码证据。 |
| BL-21、BL-27、PRV-07 是否真正覆盖原缺口？ | 是。分别新增明确提示断言、真实 conflict 前置条件和两种接受方式、Review 整份文档嵌套字段的密钥拒绝证据。 |
| SRC-07 的“不解析文档语义”能否正式通过？ | **PASS**。已按实际读取、校验、持久化和输出调用链独立审查。 |
| 三类集成证据定义是否符合证明能力？ | 是。定义区分业务行为、操作系统进程行为和正式 main 接线；产品负责人已在本次对话确认正式批准。 |
| PR #16 是否存在直接回归或实质性测试缺陷？ | 未发现。生产代码没有变化，新增证据具备有效前置条件和业务断言。 |
| PR #16 是否可以合并？ | 可以合并，**GO**。 |
| 原始六项 INCOMPLETE 是否可以全部关闭？ | 缺失证据均已补齐；可以在相应提交及审查记录合并、证据报告更新后全部关闭。 |

**最终结论：GO。**

**这不等于 M1 整体验收 GO。** 项目名称安全修复、证据报告更新和里程碑最终核查仍需单独完成。本轮不启动 M2。

审查前后原仓库均干净，审查过程未修改、提交、推送或合并原仓库。本文仅在随后收到用户明确保存要求后新增；未修改其他文件，未创建提交，未推送或合并。
