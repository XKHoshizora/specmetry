# 第二阶段测试 workflow 的第 4 次运行：run 43737429057（Review 154 修复之后的固定代码点）

| 项 | 值 |
|---|---|
| workflow | `.github/workflows/c9-tool-test.yml`，blob `4387acdc683a2014c235f3991bf854a30d6c7c2c`（与第 3 次运行相同，本轮没有改） |
| 运行 | run 43737429057，attempt 1，`push` 到 `c9test/phase2`，2026-11-15T16:47:18Z ~ 16:48:49Z，**success（3/3）**；没有重跑 |
| 代码点 | `a0ed8dc0800bc64b8f1c88e715f00fa600d053fd`，树 `9d120b675a08a18ede3fdd2b647c7839c971840e`，工具模块子树 `76a5c49c037b279ba783913cc95a56dd6e93c9eb`。三个作业的身份步骤都打印 HEAD 等于 `GITHUB_SHA`、工作区干净、上面的树与子树、workflow 的 blob，以及 `RUNNER_ENVIRONMENT github-hosted` |
| 工具链 | 三个 runner 都是 setup-go 安装的 go1.27.3：Ubuntu `/opt/hostedtoolcache/go/1.27.3/x64/bin/go`（SHA-256 `5f5eb6d0…0b5b`），macOS `/Users/runner/hostedtoolcache/go/1.27.3/arm64/bin/go`（`aa355b07…aafb`），Windows `C:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe`（`83b7ba66…999d`）。Windows 的步骤 B 之前同第 3 次运行，把 junction 的目标 `D:\…\bin` 放到 PATH 最前，步骤 B 的 `go` 为 `/d/hostedtoolcache/windows/go/1.27.3/x64/bin/go`，`go.exe` 的 SHA-256 同为 `83B7BA66…999D` |
| artifact | Windows 14171091579、Ubuntu 14170881316、macOS 14170044774，三个都关联 `a0ed8dc`。本目录保存的是重新打包的 `tar.gz`，不与 GitHub ZIP 的 digest 作字面比较 |

## 各作业的结果（全部 success）

| 项 | ubuntu-latest（132883013142） | macos-latest（132883013235） | windows-latest（132883013375） |
|---|---|---|---|
| 步骤 A：单元测试（`unit.log`） | 12 个包 ok；508 条 PASS（顶层 229，子测试 279），0 FAIL，0 SKIP | 507 条 PASS（顶层 228，子测试 279），0 FAIL，1 SKIP（`TestR154RequestOtherKind`，只在 Linux） | 同 macOS |
| 步骤 A：`-race`（`race.log`） | 508 条 PASS，0 FAIL，没有 `DATA RACE` 告警 | 不运行 | 不运行 |
| Review 154 的回归 | toolctl 的 R1-a ~ R1-e、R2-a ~ R2-f，controls 的 R3-a ~ R3-e、R4-a ~ R4-c 都通过 | 同左，R4-c 为 SKIP | 同左，R4-c 为 SKIP |
| LC-9 | 20 个命中（完整预期含故障集合） | 11 个命中、9 个不适用 | 11 个命中、9 个不适用（`start-failure-not-executable` 的真实错误为 errno 216，与本机相同） |
| RC-0 | 守卫的输入逐分量记录 | 同左 | 同左（经 junction 目标的路径，`EvalSymlinks` 成功） |
| RC-1（两个配置） | linux/amd64 0.24 s、linux/arm64 0.21 s；28 项检查全部成立；K 一致；三次调用都满足第 11 条：四个准入都准入、`Cmd` 与进程被持有、四个描述符确认释放、故障为零，pidfd 各关闭一次 | darwin/amd64 0.26 s、darwin/arm64 0.40 s；同左（没有 pidfd） | windows/amd64 0.51 s、windows/arm64 0.43 s；同左；守卫候选 `D:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe` |
| RC-1 中 T2 的传递 | 第 2 种调用的参数数组 5 个元素，第 4 个是 688 字节的 T2；输出按上下文文本 v1 解码，B2 成立 | 同左 | 同左 |
| RC-2（上限 1 KiB） | 标准输出超量；`go` 以 SIGPIPE 结束（`signal: broken pipe`）；故障集合恰为 `wait operation exit-status`；四个描述符确认释放；没有请求 | 同左 | 超量；`go` 退出 0；故障集合为空 |
| RC-3（到期） | 经 pidfd 请求一次，结果为 nil；`go` 被 SIGKILL 结束（`signal: killed`）；故障集合恰为 `wait operation exit-status`；pidfd 关闭一次 | 不发请求，记清理未知；`go` 自己完成，退出 0；故障集合为空 | `Kill` 一次，结果为 nil；退出码 1；故障集合恰为 `wait operation exit-status` |
| RC-4（作业演练） | 结论 4、退出 0，0.46 s；登记表 6 条全部满足第 11 条，故障为零；`controls.jsonl` 119 行（表头与 118 个结果），118 个命中 | 结论 4、退出 0，0.65 s；6 条全部满足；109 个命中、9 个不适用 | 结论 4、退出 0，1.07 s；6 条全部满足；109 个命中、9 个不适用 |
| 残留进程的观测 | “no go or test process” | 同左 | 同左 |

- 回归中 R4-a、R4-b、R4-c 与 R3-e 的日志里有 `got` / `want` 的差异行：那是被构造的反例按预期不命中时写出的理由，不是失败。
- 残留进程的观测只是观测：测试环境在作业之后销毁，不作为控制协议已确认清理的证据（§200）。
- RC-1 ~ RC-4 的对象是临时的夹具模块 `example.com/c9fx`，不是本仓库 R；RC-4 是夹具上的演练，只说明健康路径上收尾的实际接线，不是正式取证。

## 文件

同第 1 ~ 3 次运行：`job-<runner>.log`（`gh run view --log --job` 的原样输出）；`c9-tool-test-<runner>.tar.gz`（重新打包的 artifact，内部原始文件 Ubuntu 14 个、macOS 13 个、Windows 13 个）与 `ARTIFACTS.sha256`（每个原始文件的 SHA-256，共 40 项）；`run.json`、`jobs.json`；`SHA256SUMS`（本目录其余文件的 SHA-256，含本 README）。
