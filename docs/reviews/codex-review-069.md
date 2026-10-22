# Codex Review 069：PR #31 Windows CI 超时预算调整独立审查

## 一、结论

**GO。未发现 BLOCKER；没有本 PR 新增的 NON-BLOCKER。固定 HEAD 的 CI 已正式 completed/success、attempt 1、8/8，PR #31 可以建议合并。**

Windows 的全部包来自实时 `go list -tags nowebui ./...`，恰好分成非 CLI 与 CLI 两组；前者保留默认每包 10 分钟，后者单独使用 `-timeout 20m`。包枚举失败使步骤失败；任何一组失败使步骤失败，且第一组失败后第二组仍执行。没有新增自动重试、失败过滤、测试删减或固定包数量要求。

Ubuntu/macOS 的原执行方式、Ubuntu 的 `-race`、Linux root 跨用户步骤及其他 Job 均保持不变。文档忠实登记了已批准的风险缓解，没有将超时预算调整写成根因修复或 M-16 关闭。

**M-16 继续未解决。** 本次成功不证明 20 分钟始终足够，也不证明 CI 变慢的原因；Windows CLI 的挂起报告可能从 10 分钟延迟至 20 分钟，是负责人已批准的本次取舍，不延伸为其他保证豁免。本报告不修改或重新审查 PR #30。

## 二、固定身份与范围

- 日期：2026-10-22，Asia/Tokyo。
- PR：[XKHoshizora/specmetry#31](https://github.com/XKHoshizora/specmetry/pull/31)。
- 固定 HEAD：`0de32c5b4a319619b49a27fe1c8aaac1c80c5d8d`。
- 固定 base：`d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`。
- HEAD 的唯一父提交即上述 base；已核对包含关系。
- HEAD tree：`7406acf74d1e359d795a58d768b09650d00aad62`。
- 开始及结束核对的远程 HEAD/base 均与固定对象一致，PR Open，`auto_merge=null`，没有自动更换审查目标。

差异为一个提交、三个文件，29 行新增／3 行删除：

| 文件 | 变化 |
|---|---|
| [.github/workflows/ci.yml](../../.github/workflows/ci.yml) | 仅修改 Go tests 的 Test 步骤：非 Windows 保留原命令；Windows 枚举、分组、检查及依次执行两组 |
| [docs/m1-exit-evidence.md](../m1-exit-evidence.md) §9，M-16 行 | 补充 555.293 秒观测、负责人批准的超时预算、代价及未解决状态 |
| [docs/implementation-plan.md](../implementation-plan.md) 第 147 行 | 同步 M-16 风险缓解及未解决状态 |

没有生产代码、测试实现、Schema、迁移、验收标准或其他工作流变化。已读取适用 AGENTS.md，按固定 Git 对象读取上述文件，并在仓库外导出目标副本；未切换或清理共享工作区。

本轮开始时已有未跟踪的 `docs/reviews/codex-review-068.md`。审查期间共享工作区另出现 `internal/proposal/integrity_test.go`、`internal/cli/coderef_integrity_test.go` 未跟踪文件及 `internal/gitx/resolve_test.go` 的工作区修改。这些并行改动均非本轮产生，未读取其改动内容、覆盖或处理，不纳入固定对象；已有 Review 068 也未修改。Review 069 文件起始不存在，本轮在原仓库仅新增本报告。

## 三、分组完整性与退出码

### 1. 全部包恰好进入一组

[ci.yml](../../.github/workflows/ci.yml) 第 44～55 行：

- `all=$(go list -tags nowebui ./...)` 使用与测试相同的 build tag 和包模式，没有硬编码包清单或数量。
- `grep -vxF "$cli"` 只排除完整匹配的 CLI 导入路径。不是正则子串筛选，类似名称的其他包不会被排除。
- `n_cli` 必须为 1，`dups` 必须为空，且 `n_rest + 1 == n_all`；不满足则在测试之前退出。
- 两组分别为原集合除 CLI 的全部包，以及 CLI 本身。因此当前有效输入下没有遗漏、交集或重复。

独立在固定目标副本执行真实 `go list` 得到 31 个包；与 Windows、Ubuntu、macOS 的 CI 包结果逐项比较，集合一致且无重复。31 是该快照的观测值，源码中没有这个常量。

另在隔离副本添加一个真实的新 Go 包，重新运行原脚本，得到 32 包、31 个非 CLI 包、1 个 CLI 包；记录的参数中新增包恰好出现一次。没有修改分组脚本或基准包清单来容纳新增包。这证明本次分组不把 31 固化为长期要求；该临时包未加入原仓库。

### 2. 枚举及测试失败不会被吞掉

CI 实际使用 `bash --noprofile --norc -e -o pipefail {0}`，日志已核对。

- 第 45 行是独立的赋值命令，返回 `go list` 的状态；`go list` 失败会触发 `-e`，不能拿部分输出继续测试。
- 筛选／计数处的 `|| true` 用于允许 grep 没有匹配；它没有包住 `go list` 或测试组。包集合仍须通过后续完整性检查。
- 第 56～60 行以 `rc=0` 开始，两次测试分别 `|| rc=1`，最后 `exit $rc`。第一组失败不会提前跳过 CLI；CLI 成功也不会将已记录的失败改回成功。
- 没有循环执行、按失败类型忽略结果、自动重跑、`continue-on-error` 或针对失败包的排除。
- 非 Windows 分支中，失败的 `go test` 在 `-e` 下会先使步骤失败，后面的 `exit 0` 不能覆盖它。

上述不只依靠静态解释，已在相同 Bash 启动参数下独立注入枚举／测试退出码，结果见第五节。

### 3. 参数与其他执行路径

第 58 行非 CLI 组保留 `-tags nowebui -count=1 -v`，没有显式超时，因而保留 Go 默认每包 10 分钟；第 59 行 CLI 组保留相同参数并增加 `-timeout 20m`。已用本机 `go help testflag` 核对默认 10 分钟及该选项的语义。

Ubuntu/macOS 在第 31～36 行执行原来的 `go test -tags nowebui -count=1 -v $race ./...`；只有 Linux 将 `race` 设置为 `-race`。Linux root 步骤第 61～68 行与 base 逐字未变：先构建 sqlite 测试程序，再以 sudo 执行 `CrossUser`、`-test.count=1`、`-test.v`。该独立 root 步骤没有新增 `-race`，不能把普通步骤的 Race Detector 结果算到它头上。

Job 名称、三平台矩阵、`fail-fast: false`、触发器、权限、并发设置、lint/Web/dashboard build Job 及另一个 diagnostics 工作流均未修改。

## 四、正式 CI 身份与实际执行证据

[run 39600735762](https://github.com/XKHoshizora/specmetry/actions/runs/39600735762) 已正式 **completed/success**，事件 `pull_request`，attempt 1，HEAD 为本轮固定目标。不是根据“未见 FAIL”推断成功，也没有本轮重跑。

实际 checkout 的合并快照：`3c901fb10544e901bd477cef04bb5b7a3f16df9f`。

- 第一父提交：`d8f8a50a7f01d9fec9ce7dd699495ff829c9229f`。
- 第二父提交：`0de32c5b4a319619b49a27fe1c8aaac1c80c5d8d`。
- 完整 tree：`7406acf74d1e359d795a58d768b09650d00aad62`，与本轮 HEAD tree 一致。

已读取八个 Job 的正式状态与实际日志，全部检出上述合并快照：

| Job ID | Job | 正式结果 |
|---|---|---|
| 119224273417 | Go tests (macos-latest) | success |
| 119224273500 | Go tests (ubuntu-latest) | success |
| 119224273522 | Go lint | success |
| 119224273541 | Go tests (windows-latest) | success |
| 119224273730 | Web | success |
| 119224429413 | Build with dashboard (ubuntu-latest) | success |
| 119224429488 | Build with dashboard (windows-latest) | success |
| 119224429678 | Build with dashboard (macos-latest) | success |

CI 的三平台 Go 均为 1.27.3；本机校准为 1.27.1，来源分开记录。

### Windows 顺序及包集合

Job 119224273541 日志：

1. 输出 `packages: 31 in ./..., 30 without .../internal/cli, 1 of it`。
2. 第一组报告全部 30 个非 CLI 包：24 个 `ok`、6 个 `[no test files]`。`gitx` 为 242.576 秒。
3. 第一组最后一个包结果之后才进入 CLI 的测试输出，随后单独报告 `internal/cli 353.092s`。
4. 总共 31 个包结果，25 个 `ok`、6 个无测试文件，没有重复；包集合与固定副本的真实 `go list` 完全相同。

CLI 本次用时少于 10 分钟，不能据此说 20 分钟预算已被耗尽验证，或它今后一定足够。显式参数、Go 选项语义和分组执行证据支持“预算已经按批准方案生效”；耗时及根因是另一件事。

### Ubuntu/macOS、Race Detector 和跳过项

- Ubuntu 和 macOS 各报告相同的 31 包、无重复；各为 25 个 `ok`、6 个无测试文件。CLI 分别为 445.737 秒、141.044 秒。
- Ubuntu 的运行器身份、检出脚本、实际 Bash 执行步骤与 Linux 分支共同支持普通测试携带 `-race`；本机分支校准另核对了展开后的命令参数。这里没有以“没有竞态告警”代替 `-race` 执行证据，也没有要求为本 PR 新增日志标记。
- Ubuntu 普通步骤中 8 个需 root 的 `TestCrossUser*` 先按设计 SKIP，随后独立 root 步骤实际执行这 8 项并 PASS。普通 `-race` 与 root 实跑是不同证据。
- Windows/macOS 的 root 步骤按 `if: runner.os == 'Linux'` 跳过，不属于跳过整个 Job；八个 Job 本身均未跳过。
- 三平台的 `TestInitRefusesRealOldGit`、`TestARealGitThatFetches`、`TestWP23ReadsWithAGitThatFetches` 因专用 Git 条件未提供而跳过；M-14 两项诊断测试按默认关闭策略跳过，不记作诊断通过。
- Windows 另有 `TestClassifyWithUnreadableStorage`、`TestSourceListUnreadableSource` 的权限条件跳过；平台排除或未编译的专属用例也不能算作该平台通过。

上述跳过沿用原平台／启用条件，本 PR 未增加过滤或削减测试。CI 证明的是固定树下这次执行，不证明所有测试分支、旧 Git、跨账户机制或维护根因已经穷尽验证。

## 五、本轮独立定向验证

在仓库外 `review069` 隔离副本，从目标 YAML 提取 Test 的完整 `run:` 块，未改脚本逻辑。提取脚本 SHA-256：`7691a28cbd48cfa9593f64e83bdb7262c365eecd45918a495de77a49b4b10a27`。

平台为 Windows/amd64；Git Bash 5.3.15，Go 1.27.1。以 CI 同样的 `--noprofile --norc -e -o pipefail` 启动。测试替身只记录 `go test` 参数并按场景返回退出码，正常包枚举转交真实 Go；重复／缺失／空列表是明确注入的异常枚举。设置 `GOPROXY=off`、`GOSUMDB=off`、`GOTOOLCHAIN=local`，不访问网络。

| 场景 | 实际脚本退出码 | `go test` 调用数／结果 |
|---|---:|---|
| Windows 正常 | 0 | 2；30 个非 CLI 包，然后 CLI；参数和集合断言通过 |
| `go list` 返回 17 | 17 | 0；没有进入测试 |
| 第一组返回 11，CLI 成功 | 1 | 2；CLI 仍执行 |
| 第一组成功，CLI 返回 23 | 1 | 2 |
| 两组均失败 | 1 | 2；无第三次自动重试 |
| 枚举含重复包 | 1 | 0 |
| 枚举不含 CLI | 1 | 0 |
| 枚举为空但返回成功 | 1 | 0 |
| `RUNNER_OS=Linux`，测试成功 | 0 | 1；完整参数含 `-race ./...` |
| `RUNNER_OS=macOS`，测试成功 | 0 | 1；完整参数为原命令，不含 `-race` |
| Linux／macOS 的测试返回 11 | 各为 11 | 各 1；没有被后续 `exit 0` 覆盖 |
| 隔离副本真实新增 Go 包 | 0 | 两组自动覆盖 32 包，新增包恰好一次 |

还对记录的参数做了独立断言：所有相关 Windows 场景第一组正好为基准包集合除 CLI，第二组正好为 `test -tags nowebui -count=1 -v -timeout 20m <CLI>`；Linux/macOS 正常与失败场景参数均与原路径一致。不是只看最后退出码或作者提供的表格。

首次在受限沙箱启动 Git Bash 时，它在执行脚本之前以 `NtCreateDirectoryObject ... 0xC0000022` 初始化失败；没有产生测试调用，不能算作流程测试或产品失败。保留该日志后，在获准的隔离执行环境完成上述校准。没有重跑 GitHub CI，也没有把本机环境失败归为 M-16。

这些校准证明脚本的分组、参数、控制流和退出码，不冒称本机完整跑过三平台测试。实际测试成功来自第四节的正式 CI；Linux/macOS 分支的本机校准是在 Windows Bash 中选择环境变量，不是本机 Linux/macOS 运行。

## 六、文档、维护项与最终裁定

两处文档均保留：原 600.048 秒超时、后续 555.293 秒观察、未证实的运行器资源竞争解释、与 M-10 的区分。它们准确说明：

- 负责人已批准 Windows CLI 的预算调整，其他包和其他平台不变。
- 调整是风险缓解，代价为 CLI 挂起可能更晚报告。
- 根因未验证，M-16 保持未解决，不是接受了根因未知下的所有风险。
- 没有改变 M-14／M-15、其他维护事项或现有工作包验收状态。

**BLOCKER：无。**

**NON-BLOCKER：无本 PR 新发现。** M-16 仍是独立、未解决的维护事项；不因本次 CI 成功关闭。没有提出仅为增加标记、重跑整仓或修改无关超时的任务。

**最终 GO；固定对象的审查和 CI 条件已满足，可以建议负责人合并 PR #31。** 如合并前 HEAD/base 或合并内容变化，应仅核对新增内容及其 CI 适用性；本结论不自动延伸到其他树，不替负责人执行合并或宣布任何工作包／M2 验收。

本轮在原仓库仅新增本未提交报告，使用仓库相对链接；未修改其他文件，未提交、推送、合并、改基或重跑 CI，未修改或重新审查 PR #30。
