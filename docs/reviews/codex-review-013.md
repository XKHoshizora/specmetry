# Codex 审查 013：WP1.0 修复提交最终定向代码复审

- 审查日期：2026-10-14
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #4](https://github.com/XKHoshizora/specmetry/pull/4)
- 审查目标：`a206a3ac4bc24411012a64db5c78591a480089d5`
- 修复前提交：`0e58c62e57ba459ce4042cc8321d4797ddd8041a`
- PR 基准：`cee87b499f5af8f455dbdec299c8cb6069ef5269`
- 前次报告：[codex-review-012.md](codex-review-012.md)
- 目标 CI：[CI 第 11 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38496208946)
- 范围：针对前次 B1（Git 环境隔离）、N1（Proposal Oracle）、N2（Manifest 一致性）的修复及必要回归。不重新审查整个 M0、WP1.3 或 M1 设计，不要求提前实现 WP1.1、WP1.2、WP2.9。

**最终结论：GO。** B1、N1、N2 均可关闭。未发现新的实质性 BLOCKER，**可以合并 PR #4**。

审查期间未修改原仓库，未提交、推送、合并或向 GitHub 发布评论。独立验证使用仓库外临时副本，所有攻击性测试只使用临时仓库。本报告随后按照用户要求单独保存。

## BLOCKER

**无。B1 的原始攻击路径已被关闭。**

### B1：Git 环境隔离——已关闭

**位置：** [git.go:85](../../internal/testutil/git.go#L85)、[git.go:100](../../internal/testutil/git.go#L100)。

静态检查确认：

- `inheritedEnvWithoutGit` 按变量名大小写无关的方式，清除所有继承的 `GIT_*`、`HOME`、`XDG_CONFIG_HOME`，再设置受控环境。Windows 混合大小写不会保留注入变量。
- 初始化固定 `--object-format=sha1`，使用私有 HOME、XDG 目录、空全局配置和空模板，关闭系统配置，并通过 `GIT_CEILING_DIRECTORIES` 阻止向上发现其他仓库。
- `requireOwnGitDir` 在首次配置修改及 `Commit`、`CommitAt` 的 staging 之前检查实际 Git 目录；不匹配时立即终止。
- `BuildFixture`、`NewGitRepo`、`CommitAt`、`Commit`、`WorktreeFor` 及其 Git 子进程使用同一套受控环境，没有发现继承环境变量的遗漏调用路径。

### 原始攻击的独立复现

独立探针在外部临时仓库中预置了工作文件和已暂存修改，并在攻击前后直接遍历、哈希实际文件，包括完整的 `.git` 内容，不只依赖“夹具构建成功”的结果。

注入范围包含 Windows 混合大小写的：

- Git 仓库、工作目录、公共目录定位变量。
- Index、对象目录和备用对象目录变量。
- `GIT_DEFAULT_HASH=sha256`。
- `GIT_CONFIG_COUNT`、`GIT_CONFIG_PARAMETERS` 及指向 Hook 路径的配置。
- 系统和全局配置路径、HOME、XDG 配置目录。
- 带可执行 Hook 的模板、身份和日期变量。

这些注入经过 `BuildFixture`（包含 `fx-in-dev` 分支构造）、`NewGitRepo`、`Commit`、`CommitAt` 和 `WorktreeFor` 公开调用路径验证。

**结果：外部临时仓库的 HEAD、Index、配置、对象和工作文件字节全部不变，外部用户配置及模板不变，注入 Hook 未执行。** 固定夹具 SHA 保持一致，直接创建的测试仓库使用 SHA-1，已有分支 worktree 的 HEAD 和重复调用行为正常。

### 提前终止路径

新增的 `TestGitRepoRefusesAForeignRepository` 覆盖 `Commit`：将测试仓库的 `.git` 替换为指向外部临时仓库的 gitfile，提交在 staging 前被拒绝，外部仓库完整快照不变。

本轮另行独立验证了 `CommitAt` 的同类路径：明确因“不是预期测试仓库”而提前终止，外部仓库文件保持不变。没有将其他 Git 执行错误误认为仓库身份保护生效。

### 新增测试的有效性

[gitenv_test.go](../../internal/testutil/gitenv_test.go) 对外部仓库执行完整快照比对，包含 HEAD、索引、配置、对象和工作文件；同时检查夹具有自己的 `.git`、固定 SHA 正确和注入 Hook 没有执行。因此这些测试证明了外部仓库隔离，而不只是证明最终夹具构建成功。

## NON-BLOCKER

**无新增问题。N1、N2 已修复。**

### N1：Proposal Oracle——已修复

**位置：** [proposal_check_test.go:225](../../internal/testutil/proposal_check_test.go#L225)、[proposal_check_test.go:65](../../internal/testutil/proposal_check_test.go#L65)。

Oracle 从原始 brief 独立定义预期，不从待检查 Proposal 的声明推导结果：

- 1 个 Goal、8 个 Requirement、9 个 AC、4 个 Question。
- 各需求小节应有的 AC 数分别为 R1:1、R2:2、R3:1、R4:1、R5:2、R6:1、R7:1、R8:0。
- R3、R7 的问题必须保留对应歧义原文。
- R8 保留验收方式问题，不得出现 AC。
- 唯一阻塞问题来自“待确认”小节，必须归属项目级别。

AC 与 Requirement 的关联通过来源片段所属小节验证，不再仅比较“多少个需求有 AC”的集合大小。问题的 `about` 也必须对应正确需求小节。

三个变异在仓库外临时副本中，通过现有 `TestProjectFixtures/fx-docs-only/recorded_proposal` 验收路径验证，均明确失败，且失败原因正确：

| 变异 | 实际失败原因 |
|---|---|
| 删除 c3 | `section R2 has 1 criteria, the brief yields 2` |
| 删除三个非阻塞问题 | `1 questions, the brief raises 4` |
| 将 c9 错误关联到 R8 | 引用来自 R7，却关联 R8；同时报告 R7 缺少 AC |

这些变异仍是合法的 JSON 和 Schema 输入；失败来自应有的内容、数量和关联检查，而不是构造前置条件不成立或无关解析错误。

原文归属判断要求引用片段只匹配一个小节。本轮额外制造了跨小节重复片段，检查明确报告应当只匹配一个小节，没有任意选择第一个匹配位置。当前固定 brief 的有效引用通过，未发现明显错误归属。

### N2：Manifest 一致性——已修复

**位置：** [fixture_test.go:162](../../internal/testutil/fixture_test.go#L162)、[fx-in-dev/fixture.json:6](../../testdata/fixtures/fx-in-dev/fixture.json#L6)。

`fx-docs-only` 和 `fx-in-dev` 的 brief 字节相同；注记现在都正确保留 R8 缺少可判定验收描述，以及 R3“合适数量”、R7“尽快”的歧义。测试对两份 `FixtureBrief` 执行完整相等检查，防止同一份文档携带矛盾的预期元数据。

本次只调整构造仓库外的 Manifest 注记，没有修改夹具仓库内容。三个夹具的历史和固定 SHA 均未改变。

## ACCEPTED RISK

- Oracle 针对当前固定 brief，不承担通用自然语言需求判定。
- 故意失败的测试、录制 Claim 和其他 oracle 仍按计划留到 WP2.9；WP1.1、WP1.2 尚未实现，不影响本轮验收。

## 必要回归与执行证据

### 本地独立验证

验证使用 Windows、Go 1.27.1，在仓库外临时副本中完成。原始目标提交的全量测试与另行编译的独立探针分开执行；攻击和变异没有写入原仓库。

| 验证 | 结果 |
|---|---|
| `go test -tags nowebui -count=1 -v ./...` | 通过 |
| `go vet -tags nowebui ./...` | 通过 |
| 原有九类 Git 环境注入测试 | 全部通过，检查外部完整快照、固定 SHA 及 Hook 标记 |
| Windows 混合大小写注入及多个公开调用路径 | 通过；外部仓库、配置及模板文件字节不变 |
| `Commit` 外部 gitfile 拒绝 | 通过；staging 前终止，外部仓库不变 |
| 独立 `CommitAt` 外部 gitfile 拒绝 | 通过；因仓库身份不符终止，外部仓库不变 |
| 原始合法 Proposal | 通过 |
| 三种 Proposal 文件变异经现有验收路径检查 | 均按预期失败，原因与对应缺陷一致 |
| 跨小节重复引用片段 | 被明确、保守拒绝 |
| brief 字节与 Manifest 注记一致性 | 通过 |
| 三个夹具重复构造、历史、固定 SHA、分支及样例应用 | 通过 |

未发现 M0、WP1.3 的明显回归。两项既有符号链接测试因系统权限不足跳过，未记为通过；本轮不据此重新宣称整个 M0 已完成独立平台验收。

构造器不依赖真实 Git 全局配置或真实用户资料；样例应用只使用标准库，没有引入外部网络服务依赖。未发现本次修复引入的明显新文件写入安全问题。

### GitHub CI

已核实 [目标 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38496208946) 对应审查目标提交，8/8 任务成功，并读取 Windows、Linux、macOS 三个平台的 Go 测试日志，确认下列测试实际执行并通过，没有仅凭任务绿灯推测：

- 夹具重复构造、Git 状态、阶段、brief、Proposal、分支和样例应用检查。
- 九类继承 Git 环境变量注入测试。
- 外部 gitfile 拒绝测试。
- Proposal 三种变异检查，日志中的拒绝原因分别对应 AC 数量、问题数量和需求归属错误。

三平台重复构造均通过固定 `expected_refs` 比较。CI 使用 Go 1.27.2；本地使用 Go 1.27.1。Linux 测试配置包含 Race Detector。本轮未在本地独立运行 Linux 或 macOS，其平台结论来自目标提交的实际 CI 日志。

## 最终回答

| 问题 | 回答 |
|---|---|
| B1 是否真正关闭？ | **是。** 原始环境变量攻击路径已关闭，独立外部仓库快照验证通过。 |
| N1、N2 是否已经修复？ | **是。** 三种变异正确失败，独立 Oracle 和 Manifest 一致性检查生效。 |
| 是否存在新的测试污染、错误验收或跨平台确定性问题？ | **未发现实质性问题。** |
| 当前 WP1.0 是否可以通过代码审查？ | **是。** |
| 是否允许合并 PR #4？ | **可以。** 目标提交已核实 CI 8/8 成功。 |
| 最终结论 | **GO。** |

该结论仅覆盖 WP1.0 及本轮修复，不代表整个 M1 已验收。无需为关闭本轮问题继续进行无必要的底层重构。

原仓库在审查结束时保持干净，HEAD 未变。本报告保存操作仅新增 `docs/reviews/codex-review-013.md`；未修改其他文件，未提交、推送或合并。
