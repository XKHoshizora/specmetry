# 第二阶段测试 workflow 的第 3 次运行：run 43681144176（固定代码点）

| 项 | 值 |
|---|---|
| workflow | `.github/workflows/c9-tool-test.yml`，blob `4387acdc683a2014c235f3991bf854a30d6c7c2c` |
| 运行 | run 43681144176，attempt 1，`push` 到 `c9test/phase2`，2026-11-15T05:27:17Z ~ 05:28:41Z，**success（3/3）**；没有重跑 |
| 代码点 | `ed84d091f3d352d46cdf525880998ba6ef569b91`，树 `65c53940869b9ef28b525966d85eaecaa97a0c62`，工具模块子树 `c943f922827282e2868ef4da84fe8e370f96f126`。三个作业都核对 HEAD 等于 `GITHUB_SHA`、工作区干净，`RUNNER_ENVIRONMENT` 为 `github-hosted` |
| 工具链 | 同第 1 次运行：Ubuntu `go1.27.3 linux/amd64`（SHA-256 `5f5eb6d0…0b5b`），macOS `go1.27.3 darwin/arm64`（`aa355b07…aafb`），Windows `go1.27.3 windows/amd64`（`83b7ba66…999d`）。Windows 的步骤 B 之前，workflow 把 junction 的目标 `D:\hostedtoolcache\windows\go\1.27.3\x64\bin` 放到 PATH 最前：步骤 B 使用的 `go` 为 `/d/hostedtoolcache/windows/go/1.27.3/x64/bin/go`，`go.exe` 的 SHA-256 同为 `83B7BA66…999D` |

## 各作业的结果（全部 success）

| 项 | ubuntu-latest（132711825257） | macos-latest（132711825153） | windows-latest（132711825263） |
|---|---|---|---|
| 步骤 A：单元测试（`unit.log`） | 12 个包 ok；487 PASS、0 FAIL、0 SKIP | 同左 | 同左（LC-2 的符号链接在 runner 上可以创建） |
| 步骤 A：`-race`（`race.log`） | 487 PASS、0 FAIL，没有 `DATA RACE` 告警 | 不运行 | 不运行 |
| LC-9 | 20 个命中 | 11 个命中、9 个不适用（只在 Linux 的变体） | 11 个命中、9 个不适用 |
| RC-0 | 守卫的输入逐分量记录 | 同左 | 同左（经 junction 目标的路径，`EvalSymlinks` 成功） |
| RC-1（两个配置） | linux/amd64 0.28 s、linux/arm64 0.26 s；28 项检查全部成立；K 一致；三次调用都满足第 11 条，pidfd 各关闭一次 | darwin/arm64 0.53 s、darwin/amd64 0.32 s；同左（没有 pidfd） | windows/amd64 0.38 s、windows/arm64 0.32 s；同左；守卫候选为 `D:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe` |
| RC-1 中 T2 的传递 | 第 2 种调用的参数数组 5 个元素，第 4 个是 688 字节的 T2；输出按上下文文本 v1 解码，B2 成立 | 同左 | 同左 |
| RC-2（上限 1 KiB） | 标准输出超量，读端关闭为 nil；`go` 因 SIGPIPE 结束（`signal: broken pipe`）；失败 | 同左 | 超量，读端关闭为 nil；`go` 退出 0；失败 |
| RC-3（到期） | 经 pidfd 请求一次，结果为 nil；`go` 被 SIGKILL 结束（`signal: killed`）；pidfd 关闭一次；失败 | 不发请求，记清理未知；`go` 自己完成（标准输出 27998 字节），晚到的回收；失败 | `Kill` 一次，结果为 nil；退出码 1；失败 |
| RC-4（作业演练） | 结论 4、退出 0，0.58 s；登记表 6 条全部满足第 11 条；`controls.jsonl` 118 个命中 | 结论 4、退出 0，0.80 s；6 条全部满足；109 个命中、9 个不适用 | 结论 4、退出 0，0.78 s；6 条全部满足；109 个命中、9 个不适用 |
| 残留进程的观测 | “no go or test process” | 同左 | 同左 |

- 残留进程的观测只是观测：测试环境在作业之后销毁，不作为控制协议已确认清理的证据（§200）。
- RC-1 ~ RC-4 的对象是临时的夹具模块 `example.com/c9fx`，不是本仓库 R；RC-4 是夹具上的演练，不是正式取证。

## 文件

同第 1 次运行：`job-<runner>.log`；`c9-tool-test-<runner>.tar.gz` 与 `ARTIFACTS.sha256`；`run.json`、`jobs.json`；`SHA256SUMS`。
