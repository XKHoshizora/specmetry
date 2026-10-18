# Codex Review 053：PR #26 夹具自动维护隔离独立审查

## 结论与固定身份

**GO。未发现 BLOCKER。PR #26 的夹具修复可以合并；保留一项关于历史根因与后台分离措辞的 NON-BLOCKER。** 这不表示原 Ubuntu 偶发失败已获得排他性的根因证明，也不关闭 M-14／M-15，不授权 WP2.3。

| 项目 | 本次核查结果 |
|---|---|
| 仓库／PR | [XKHoshizora/specmetry PR #26](https://github.com/XKHoshizora/specmetry/pull/26) |
| base | `a236e5350ff92dcffd9bc05d09b1f3aac3996704` |
| HEAD | `ba75ff310e82c116c0eaa31a7a2ba8719a828677` |
| CI | [39127098623](https://github.com/XKHoshizora/specmetry/actions/runs/39127098623)，Attempt 1，8/8 success |
| CI 实际检出 | `f4b7b6fa31a5cfd9aa987368a28e422443ad307c` |
| 检出提交的两个父提交 | 依次为上述 base、HEAD |
| HEAD 与 CI 检出提交的内容树 | 均为 `54726dd495061280247187f69460bba6c402989e` |
| 远程身份 | 开始及收尾核查均与上述 HEAD／base 一致，PR 为 Open |
| 实际补丁 | 只修改 `internal/testutil/git.go`，新增 `internal/testutil/maintenance_test.go`；141 行增加、4 行删除 |

原工作区未切换、清理或用于动态反例。审查按固定 Git 对象读取；运行与变异验证均在仓库外隔离副本进行。本报告是本轮唯一新增的原仓库文件。

## BLOCKER

无。

## NON-BLOCKER

### N1：原 CI 根因和后台分离的表述应保留证据限定

**位置：** [git.go](../../internal/testutil/git.go) 第 58～61 行、[maintenance_test.go（固定目标）](https://github.com/XKHoshizora/specmetry/blob/ba75ff310e82c116c0eaa31a7a2ba8719a828677/internal/testutil/maintenance_test.go#L13) 第 13～19 行，以及 PR 描述 Summary 的 “the cause found”。

原 CI 日志确实记录了读取 `.git/objects/maintenance.lock` 时文件已经不存在；自动维护能创建该锁，移除本次开关后也确实能观察到维护子进程。因此修复针对的机制有依据。

但这些证据仍不能单独证明原 Ubuntu 失败时具体进程的生命周期、唯一根因及全部 Git 版本的后台分离行为。本轮正对照显式关闭 detach，证明的是观测器能够看见维护子进程；它不证明当时有后台子进程在 commit 返回后删锁。PR 正文已正确把原 Ubuntu 根因列为推断，但 Summary 和代码注释比正文更确定。

**影响：** 可能让后续维护者把合理机制解释误读为已完成的原始故障复现；不影响本次私有配置修复的实际有效性。

**最小建议：** 将概括性措辞改为“与自动维护锁竞争一致；本修复消除已验证的自动维护启动路径，原 CI 的具体竞争未直接复现”。把“Linux/macOS 上会后台分离”限定为相应 Git 版本、配置和适用条件。无须为接受本夹具修复穷尽证明历史根因，本项不阻塞合并。

## 实现与测试审查

### 1. 两个配置项有效，作用域仍然隔离

[git.go](../../internal/testutil/git.go) 第 28、54～86 行把原来的私有空全局配置改为：

```ini
[maintenance]
    auto = false
[gc]
    auto = 0
```

配置文件仍位于本次夹具的私有临时目录，经 `GIT_CONFIG_GLOBAL` 显式提供。系统配置被禁用，HOME／XDG_CONFIG_HOME 和模板目录仍指向隔离目录。没有向用户真实全局配置写入，也没有改动生产 Git 执行代码。

两个开关分别抑制普通 Git 命令的自动 maintenance 调用和自动 gc 启发式；后者不只是把一个阈值调低。该解释与 Git 的版本化官方文档一致：[maintenance 配置](https://github.com/git/git/blob/v2.55.0/Documentation/config/maintenance.adoc)、[gc 配置](https://github.com/git/git/blob/v2.55.0/Documentation/config/gc.adoc)。

边界是“夹具的默认受控 Git 环境”。测试主动使用 `-c maintenance.auto=true`、仓库本地显式覆盖或手动执行 maintenance/gc，仍能改变行为；本修复没有承诺禁止这些有意操作。正对照正是有意启用维护，不构成配置失效。

`NewGitRepo` 与 `NewSHA256GitRepo` 共用该构造路径；后续提交与 Worktree 操作继续使用相同受控环境。未发现扩大生产行为的变化。

### 2. Trace2 对照有效，新增断言能够发现回归

[maintenance_test.go（固定目标）](https://github.com/XKHoshizora/specmetry/blob/ba75ff310e82c116c0eaa31a7a2ba8719a828677/internal/testutil/maintenance_test.go#L23) 第 23～118 行通过真实 Git commit 取得 Trace2 事件：

- 正对照只对该命令恢复维护，并把 maintenance／gc 的自动后台分离关闭；实际观察到 `git maintenance run --auto --quiet --no-detach`。
- 普通负对照检查没有 maintenance／gc 子进程，并查询两个开关的有效值。
- 两个继承环境变量场景在构造 GitRepo 之前注入启用维护的配置，检查隔离后仍然没有维护子进程。
- Trace 文件缺失、空文件、非法 JSON、扫描错误及 Git 命令失败都会使测试失败；不是“读不到日志便算没有子进程”。

本轮在隔离副本仅把 `fixtureGitConfig` 恢复为空，保留新增测试不变。结果为：

| 场景 | 固定目标 | 移除配置的隔离变异 |
|---|---|---|
| 能观察到维护的正对照 | PASS | PASS |
| 普通夹具提交不启动维护 | PASS | FAIL，发现 maintenance 子进程 |
| 继承 GIT_CONFIG_COUNT 不能启用维护 | PASS | FAIL，发现 maintenance 子进程 |
| 继承 GIT_CONFIG_PARAMETERS 不能启用维护 | PASS | FAIL，发现 maintenance 子进程 |
| 私有全局配置只有两个预期设置 | PASS | FAIL，实际为空 |

因此测试不只是验证配置常量自身，还实际观测到了有／无修复的子进程差异。变异测试的非零退出码是预期证据，不是目标提交失败。

该日志读取器是可信隔离测试中的行为观测器，不是 PD-26 的安全能力分类器。它没有验证完整的 SID 和根进程生命周期；本次正对照、真实命令成功与具体 child_start 断言足以支持其窄范围用途，不能把它宣传为任意 Trace2 输入的完整性证明。

### 3. 环境、Hook、模板与快照保证未被放宽

[git.go](../../internal/testutil/git.go) 第 104～114 行仍按大小写不敏感方式过滤所有继承的 `GIT_*`，以及 HOME／XDG_CONFIG_HOME。两种配置注入不会覆盖私有文件；此前 Hook、模板、对象目录、Index 和外部仓库定向的隔离检查仍然存在。

[原有环境隔离测试](../../internal/testutil/gitenv_test.go) 仍经公开 BuildFixture 路径：

- 对另一个临时仓库取完整前后快照；
- 检查外部仓库没有变化；
- 检查注入 Hook 没有执行；
- 核对夹具自己的 Git 目录与固定 Commit SHA。

本轮没有新增快照忽略项或把消失的 maintenance.lock 当作可忽略错误。[snapshot.go](../../internal/testutil/snapshot.go) 在 base 与 HEAD 的 blob 均为 `9cb674c55bc412538bd0bf59b835a53985c70baa`；SnapshotTree、SettledSnapshot、AssertTreeUnchanged 均未改变。

故意使用用户环境的 [plainGit 所在测试文件](../../internal/cli/init_security_test.go) 在两端的 blob 均为 `ab9f32404bd63cc99380b08af27448935192e11b`。其用户环境正对照没有改成受控夹具环境，也没有被放宽。

## 独立运行、CI 与证据边界

### 本轮独立运行

平台为 Windows，Git `2.56.0.windows.2`、Go `1.27.1 windows/amd64`。使用固定目标的仓库外副本，未连接真实远端。

以下定向命令退出 0，用时约 80 秒；两个新增测试及其子测试均实际运行，没有 SKIP：

```text
go test -tags nowebui -mod=readonly -count=1 -v -timeout 5m   -run '^(TestFixtureCommitsStartNoAutomaticMaintenance|TestFixtureGitConfigIsOnlyMaintenance|TestBuildFixtureIgnoresInheritedGitEnvironment|TestGitRepo.*|TestFixturesAreSelfContained)$'   ./internal/testutil
```

实际通过的顶层测试为：

- TestFixtureCommitsStartNoAutomaticMaintenance；
- TestFixtureGitConfigIsOnlyMaintenance；
- TestBuildFixtureIgnoresInheritedGitEnvironment，包含九种注入组合；
- TestGitRepoRefusesAForeignRepository；
- TestGitRepoCommitsAndWorktrees；
- TestFixturesAreSelfContained。

移除配置的隔离变异只运行两个新增测试，退出 1，失败点与上表一致，用时约 35 秒。该变异等价于恢复本补丁前的空私有配置，不是重放原 Ubuntu 偶发竞争。

另一次较宽的本机运行包含 TestProjectFixtures，达到自设的 3 分钟上限，堆栈停在 applications 子测试等待其应用命令；整个运行退出 1。该批**不记为通过**，也没有据此确定耗时根因或宣称存在生产死锁。随后缩小范围得到上述独立结果。没有修改仓库测试超时、断言或跳过规则；TestProjectFixtures 的完整成功证据来自下述 CI，而不是这次超时运行。

### CI 独立核查

读取了 run 元数据、8 个 Job 的结果及检出日志。所有 Job 检出同一合并快照，两个父提交和内容树均与本次固定身份一致。

| Job | 结果 |
|---|---|
| Go tests：Windows、Ubuntu、macOS | 3/3 success |
| Go lint | success |
| Web | success |
| Build with dashboard：Windows、Ubuntu、macOS | 3/3 success |

三平台 Go 日志均明确记录两个新增测试及全部维护子测试 RUN／PASS，没有按条件跳过；正对照均记录 maintenance 子进程。原有环境隔离、拒绝外部 Git 目录、提交／Worktree、夹具自包含和固定 SHA 的 ProjectFixtures 测试也实际通过。

- Ubuntu：Git 2.55.0，Go 1.27.2；测试步骤在 Linux 选择 `-race`，实际执行 `go test -tags nowebui -count=1 -v $race ./...` 并成功，支持本轮 Ubuntu Race Detector 证据。
- macOS：Git 2.56.0，Go 1.27.2。
- Windows：Git 2.56.0.windows.2，Go 1.27.2。
- Linux 专属 root 跨用户步骤成功；macOS／Windows 对该步骤按设计跳过，不应混称全部平台均执行。

本轮没有在本机独立重跑 Linux／macOS 或旧版 Git；这些平台的新测试结论来自已核查的 CI 日志。本机也没有重跑全部生产模块或自行触发 Actions。

### 原观察、机制和根因的区别

独立读取[原 Ubuntu 失败 Job](https://github.com/XKHoshizora/specmetry/actions/runs/39121984050/job/117659263503)后，确认原 Attempt 1 的确出现：

```text
TestBuildFixtureIgnoresInheritedGitEnvironment/GIT_CONFIG_COUNT_hooks_path
gitenv_test.go:70: open .../.git/objects/maintenance.lock: no such file or directory
```

对应源码第 70 行在 outer.Commit 之后、注入变量和 BuildFixture 之前调用 SettledSnapshot。这能排除“该次失败是随后注入的 GIT_CONFIG_COUNT 已经生效”的读法，但日志没有提供当时维护进程的 Trace2 轨迹。

PR 作者披露的锁轮询、人工创建／移除锁的快照竞争、WSL／真实旧 Git 记录属于作者证据；本轮没有重新运行这些实验。人工竞争能证明快照对并发消失文件的反应，不能单独证明原 CI 就由同一个 Git 后台机制触发。

本轮新增的独立证据是：真实 commit 的维护子进程正对照、移除开关后的可检测回归、修复后的受控环境测试，以及三平台 CI 日志核查。没有把最新 CI 成功写成原失败运行已重跑，或所有历史竞争均被排他定位。

## 最终裁定与接手边界

1. **维护／gc 隔离：通过。** 私有配置生效，没有修改用户配置或生产 Git 行为。
2. **测试有效性：通过。** Trace2 正对照与移除修复的变异结果相互支持；继承配置注入仍受隔离。
3. **快照与 plainGit：保持。** 没有忽略、弱化断言或替换故意使用的用户环境。
4. **CI：固定提交 Attempt 1，8/8 成功；三平台相关测试及 Ubuntu Race Detector 证据成立。**
5. **最终 GO，PR #26 可以合并。** N1 为非阻塞措辞修正建议，不要求继续施压复现全部历史根因。

后续由负责人决定是否合并；若 HEAD 或 base 改变，应核对新增差异与 CI 适用性。本轮不处理 PR #24／#25，不修改生产诊断或重试，不关闭 M-14／M-15，不宣布其他工作包验收，也不授权或启动 WP2.3。未新增接受风险。

