# 第二阶段测试 workflow 的第 5 次运行：run 43906849891（Review 155 修复之后的固定代码点）

| 项 | 值 |
|---|---|
| workflow | `.github/workflows/c9-tool-test.yml`，blob `4387acdc683a2014c235f3991bf854a30d6c7c2c`（与第 3、4 次运行相同，本轮没有改） |
| 运行 | run 43906849891，attempt 1，`push` 到 `c9test/phase2`，2026-11-16T16:22:57Z ~ 16:24:29Z，**success（3/3）**；没有重跑 |
| 代码点 | `5e6c3a4b93b5189c807bee4d179494e837a02b30`，树 `b6ac8721dedcd6d152aaab4facf74cb7f616e2b6`，工具模块子树 `755ffc0c2465959fa5528d07b5c079154e8e4feb`。三个作业的身份步骤都打印 HEAD 等于 `GITHUB_SHA`、工作区干净、上面的树与子树、workflow 的 blob，以及 `RUNNER_ENVIRONMENT github-hosted` |
| 作业 | ubuntu-latest 133437885177、macos-latest 133437883940、windows-latest 133437884693 |
| 工具链 | 三个 runner 都是 setup-go 安装的 go1.27.3：Ubuntu `/opt/hostedtoolcache/go/1.27.3/x64/bin/go`（SHA-256 `5f5eb6d0…0b5b`），macOS `/Users/runner/hostedtoolcache/go/1.27.3/arm64/bin/go`（`aa355b07…aafb`），Windows `C:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe`（`83b7ba66…999d`）。Windows 的步骤 B 之前同第 3、4 次运行，把 junction 的目标 `D:\…\bin` 放到 PATH 最前，步骤 B 的 `go` 为 `/d/hostedtoolcache/windows/go/1.27.3/x64/bin/go`，`go.exe` 的 SHA-256 同为 `83B7BA66…999D` |
| artifact | Windows 14245962553、Ubuntu 14245592701、macOS 14245042864，三个都关联 `5e6c3a4`。本目录保存的是重新打包的 `tar.gz`，不与 GitHub ZIP 的 digest 作字面比较 |

## 各作业的结果（全部 success）

| 项 | ubuntu-latest | macos-latest | windows-latest |
|---|---|---|---|
| 步骤 A：单元测试（`unit.log`） | 14 个包 ok；544 条 PASS（顶层 242，子测试 302），0 FAIL，0 SKIP | 541 条 PASS（顶层 239，子测试 302），0 FAIL，3 SKIP（只在 Linux：`TestR154RequestOtherKind`、`TestR155ExpiryOtherSignal`、`TestR155ExitedBeforeRequest`） | 539 条 PASS（顶层 237，子测试 302），0 FAIL，5 SKIP（上面三个，以及没有信号的 `TestR155OverflowOtherSignal`、`TestR155OverflowSIGPIPEHit`） |
| 步骤 A：`-race`（`race.log`） | 544 条 PASS，0 FAIL，没有 `DATA RACE` 告警 | 不运行 | 不运行 |
| Review 155 的回归 | H1 ~ H4、S1 ~ S6、S7（R1-e）、C1、C2 都通过；H5（keep）通过 | 适用的都通过（S5、C2 SKIP） | 适用的都通过（S1、S2、S5、C2 SKIP） |
| LC-9（完整预期含退出身份与故障集合） | 20 个命中 | 11 个命中、9 个不适用 | 11 个命中、9 个不适用 |
| RC-0 | 守卫的输入逐分量记录 | 同左 | 同左（经 junction 目标的路径） |
| RC-1（两个配置） | linux/amd64 0.20 s、linux/arm64 0.18 s；28 项检查全部成立；K 一致；三次调用都满足第 11 条（四个准入、`Cmd` 与进程被持有、四个描述符确认释放、故障为零），pidfd 各关闭一次 | darwin/amd64 0.25 s、darwin/arm64 0.38 s；同左（没有 pidfd） | windows/amd64 0.38 s、windows/arm64 0.31 s；同左；守卫候选 `D:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe` |
| RC-1 中 T2 的传递 | 第 2 种调用的参数数组 5 个元素，第 4 个是 688 字节的 T2 | 同左 | 同左 |
| RC-2（上限 1 KiB） | 超量；`go` 的结束为 `signaled SIGPIPE`；故障集合恰为 `wait operation exit-status signaled SIGPIPE`；四个描述符确认释放；没有请求 | 同左 | 超量；`exited 0`；故障集合为空 |
| RC-3（到期；按合法分支判定） | 分支“由请求结束”：父端在 `W` 到期（假时钟 `2026-11-15T00:01:40Z`）、调用判定失败、第 11 条不成立、有晚到的结果；请求准入、经 pidfd 发起 1 次、nil；子进程的结束 `signaled SIGKILL`；唯一的 `Wait` 返回、B 类；故障集合恰为 `wait operation exit-status signaled SIGKILL`；没有清理未知 | 分支“没有请求（darwin）”：同样到期与失败；没有请求；清理未知恰为平台的一条；`exited 0`；故障集合为空 | 分支“由请求结束”：同样到期与失败；`Kill` 1 次、nil；`exited 1`；故障集合恰为 `wait operation exit-status exited 1` |
| RC-4（作业演练） | 结论 4、退出 0，0.39 s；登记表 6 条全部满足第 11 条，故障为零；`controls.jsonl` 119 行（表头与 118 个结果），118 个命中 | 结论 4、退出 0，0.79 s；6 条全部满足；109 个命中、9 个不适用 | 结论 4、退出 0，0.76 s；6 条全部满足；109 个命中、9 个不适用 |
| 步骤 B 的测试（`realcall.log`） | 31 条 PASS（顶层 6：RC-0 ~ RC-4 与 C1；子测试 25：RC-1 的两个配置与 C1 的 23 个构造用例） | 同左 | 同左 |
| 残留进程的观测 | “no go or test process” | 同左 | 同左 |

- RC-3 的分支是本次实际发生的分支；“请求之前已经结束”的分支本次没有发生，只由 C1 的构造用例与 C2（Linux 上的伪工具）覆盖。RC-3 不声明真实 `go` 在请求时仍在运行：父端停在 `Wait` 之前只暂停父端。
- 回归中 S2 ~ S5 与 R4 的日志里有 `got` / `want` 的差异行：那是被构造的反例按预期不命中时写出的理由，不是失败。
- 残留进程的观测只是观测：测试环境在作业之后销毁，不作为控制协议已确认清理的证据（§200）；进程寿命的持有者（`keep`）只给责任一个所有者，也不是清理的证据。
- RC-1 ~ RC-4 的对象是临时的夹具模块 `example.com/c9fx`，不是本仓库 R；RC-4 只说明健康路径上收尾的实际接线，不是正式取证。

## 文件

同第 1 ~ 4 次运行：`job-<runner>.log`（`gh api …/actions/jobs/<id>/logs` 的原样输出）；`c9-tool-test-<runner>.tar.gz`（重新打包的 artifact，内部原始文件 Ubuntu 14 个、macOS 13 个、Windows 13 个）与 `ARTIFACTS.sha256`（每个原始文件的 SHA-256，共 40 项）；`run.json`、`jobs.json`；`SHA256SUMS`（本目录其余文件的 SHA-256，含本 README）。
