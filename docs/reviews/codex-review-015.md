# Codex 审查 015：WP1.1 项目注册与初始化独立增量代码审查

- 审查日期：2026-10-14
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #6](https://github.com/XKHoshizora/specmetry/pull/6)
- 审查目标：`86ac50dc64fa1ccf7f5fe96b783cacba4ded6f5b`
- 基准：`origin/master = 4c5a0383812c58fa241eda4a5e73894fc70726f2`
- 目标 CI：[CI 第 16 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38513528148)
- 依据：AGENTS.md、CLAUDE.md、architecture §6.1.1、§6.1.2、§10、TD-25、implementation-plan WP1.1、acceptance-criteria 中对应的初始化与 Worktree 验收项，以及 `init.v1` 交换契约。
- 范围：WP1.1 的项目身份、Worktree 归属、Git 调用、初始化事务、Marker、阶段识别、候选来源、隐私和输出契约增量。不重新审查 M0、WP1.0、WP1.2、WP1.3，不提前要求 WP1.4～WP1.8 或 M2 的后续业务功能。

**结论：NO-GO。** `86ac50d` 存在 **3 项 BLOCKER**，当前不建议验收 WP1.1 或合并 PR #6。

审查以 `origin/master` 的 `4c5a038` 为基准。已检查实际差异，并在仓库外临时副本、临时仓库中运行测试和反例。审查期间原仓库未修改，未提交、推送或合并。本报告随后按照用户要求单独保存。

## BLOCKER

### B1：Git 读取命令仍能执行外部程序，并修改仓库 Ref

**位置：** [gitx.go:239](../../internal/gitx/gitx.go#L239)、[gitx.go:272](../../internal/gitx/gitx.go#L272)。

`filterEnv` 只过滤仓库定位变量，保留了 `GIT_CONFIG_COUNT`、`GIT_CONFIG_PARAMETERS`、全局配置及 Trace 变量；调用 Git 时也未禁用 FSMonitor。`GIT_OPTIONAL_LOCKS=0` 无法阻止这些副作用。

独立反例通过公开 `init --json --no-exclude` 路径验证：

- 分别通过仓库配置、`GIT_CONFIG_COUNT`、`GIT_CONFIG_PARAMETERS`、全局配置设置 FSMonitor。
- FSMonitor 程序写入临时外部文件，并创建 `refs/heads/review-polluted`。
- 四种情况下 `init` 都返回成功，外部文件和新 Ref 均实际存在。
- 单独设置 `GIT_TRACE`，也能让 `init` 在指定的外部位置创建 Trace 文件。

这违反 architecture §6.1.2、§10.6 的只读 Git 调用与仓库写入边界。当前命令虽然只执行读取子命令，仍可能间接执行程序、修改历史引用或产生日志。

**最小修复：** 过滤配置注入和 Trace 环境变量；明确控制外部配置的使用范围，并为读取命令显式禁用 FSMonitor 等可执行扩展。增加公开 CLI 反例，断言外部文件、Refs、HEAD、Index 和配置不变。无需重写 Git 解析层。

### B2：默认交付目标 Ref 绕过密钥扫描

**位置：** [init.go:163](../../internal/project/init.go#L163)、[init.go:389](../../internal/project/init.go#L389)。

名称和持久化路径经过扫描，但主 Worktree 的分支 Ref 在扫描之后读取，随后直接写入 `delivery_targets.ref`，并进入成功输出。

独立反例：

1. 在临时仓库创建分支，名称使用运行时拼接的 `"ghp_" + "A1b2"×9`。
2. 使用安全的项目名称执行公开 `init --json`。
3. 命令返回 0；该合成 Token 同时出现在数据库的目标 Ref 和 JSON 输出中。

这是现有 `github_token` 规则能够识别的输入，属于扫描接入遗漏，违反 architecture §10.8。Schema 校验通过无法弥补这个缺口。

**最小修复：** 在打开数据库之前扫描将被保存的默认目标 Ref；命中时返回退出码 2，只报告字段与规则。回归测试应同时检查输出、Marker、数据库及 `seq`，不能只检查扫描辅助函数。

### B3：Git 错误原文会泄露配置值

**位置：** [gitx.go:286](../../internal/gitx/gitx.go#L286)。

Git stderr 的第一行被直接拼入应用错误，随后进入 `error.v1`。`Locate` 又发生在路径密钥扫描之前，因此错误中的目录也未经该扫描。

独立反例：

- 通过 `GIT_CONFIG_COUNT` 给 `core.bare` 设置合成敏感 canary。
- 执行公开 `init --json`。
- 命令正确返回退出码 6，但 JSON 错误完整包含 `bad boolean config value '<canary>'`。
- 将同类值写入临时仓库本地配置，也能复现。

因此，仅修复 B1 的环境变量过滤仍不足以关闭此问题；本地配置解析失败同样可能回显内容。

**最小修复：** 对外返回固定的 Git 操作错误类别，保留退出码 6，不拼接原始 stderr、配置值或未经检查的路径。增加环境配置和本地配置两个公开 CLI canary 测试。

## NON-BLOCKER

### N1：Marker 冲突时，“不写入任何内容”的表述与文件副作用不一致

**位置：** [init.go:115](../../internal/cli/init.go#L115)、[init.go:349](../../internal/project/init.go#L349)、[init_test.go:734](../../internal/cli/init_test.go#L734)。

已有合法 Marker 指向本地不存在的项目、且使用全新数据目录时，`init` 会先创建数据库，再在事务中拒绝冲突。错误却声称 `Nothing was changed`。

独立验证确认：没有错误登记项目，但留下了新建数据库。现有测试只检查项目数为零，不能证明文件完全不变。

**建议：** 对于“已有 Marker、本地数据库不存在”的情况，在打开数据库前直接拒绝；测试应在查询辅助函数运行之前检查文件副作用。当前未发现业务记录或身份被错误修改，因此不列为 BLOCKER。

### N2：非 Git 扫描不响应取消，候选数上限不限制扫描成本

**位置：** [files.go:44](../../internal/project/files.go#L44)、[rules.go:237](../../internal/project/rules.go#L237)。

非 Git 路径遍历不检查 `ctx.Err()`，并保存完整文件清单；200 条限制只在候选输出阶段应用。

独立测试中，调用前已取消的 Context 仍完成扫描并返回 100 个文件、`nil` 错误。大型目录会有线性扫描及内存成本。

**建议：** 在遍历和文件处理循环中检查取消。当前没有复现数据损坏，不要求本轮重构扫描架构。

## ACCEPTED RISK

- **仓库移动、复制：** Marker 与数据库冲突时保守拒绝符合 TD-25。独立复制 Marker 的反例未产生错误身份复用，已有 `seq` 保持不变。迁移能力可以延期。
- **SHA-256 仓库：** 独立验证其快照输出为 `null`，初始化进度仍为 `initialized`，没有错误报告交付就绪。当前限制可以接受。
- **阶段识别：** 仅凭路径和大小给出初始阶段、候选来源，不等于确认客户需求；当前实现没有据此创建需求或验收事实。
- WP1.4～WP1.8、M2 的后续业务验收不属于本轮缺陷。

## 验证结果与覆盖边界

- 在目标提交临时副本运行原有 `go test -tags nowebui -count=1 -v ./...`、`go vet -tags nowebui ./...`，均通过。
- 34 个合法 Schema 示例、108 个非法反例通过预期检查，包含本次新增的 11 个反例。测试会先验证基础示例合法，未发现基础输入非法导致的假通过。
- 三个固定夹具经公开 CLI 得到预期阶段；候选排序、排除规则、`requirements.txt` 的依赖清单归类符合设计。阶段测试使用明确预期值，没有仅重复 Manifest。
- 独立运行 8 个初始化进程：全部成功，只登记一个项目、一个 Worktree。
- 独立强制数据库提交后 Marker 写入失败；修复障碍后再次公开 Init，正确恢复同一项目。
- Unicode 路径、缺失 Marker 后重建、Windows 目录联接拒绝均验证正常。
- 本机两项符号链接测试因权限不足跳过；已核实三平台 CI 日志中实际运行并通过，不能将其记为本机通过。
- [目标提交 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38513528148) 确为 8/8 成功，Linux 测试配置包含 Race Detector。但现有测试未覆盖 B1～B3，CI 全绿不能关闭这些反例。

五项设计细化中，默认目标取主 Worktree 分支、Detached HEAD／无提交分支处理、名称规则、exclude 追加和依赖清单归类整体符合已批准方向；上述扫描遗漏和冲突副作用仍需处理。

## 最终答复

| 最终问题 | 结论 |
|---|---|
| 项目身份与 Worktree 归属是否正确？ | 正常调用路径正确，未发现重复登记或错误归属。 |
| Init 是否安全、可重复、幂等？ | 重复执行、并发登记和失败恢复正常；安全边界尚未通过。 |
| 是否存在仓库污染、隐私泄露或错误登记风险？ | 已复现仓库 Ref 污染、密钥持久化及配置值泄露；未复现错误身份登记。 |
| 三个夹具是否支持真实验收？ | 支持本轮阶段识别、候选发现和 Worktree 验收，不能代替安全反例。 |
| 是否可以验收并合并 PR #6？ | **暂不可以，先关闭 B1～B3 并补充回归测试。** |
| 最终代码审查结论 | **NO-GO**。 |
