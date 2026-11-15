# 第二阶段测试 workflow 的第 1 次运行：run 43680105408

| 项 | 值 |
|---|---|
| workflow | `.github/workflows/c9-tool-test.yml`（blob `28352ff49af7053b666a6780746a470e40217698`） |
| 运行 | run 43680105408，attempt 1，`push` 到 `c9test/phase2`，2026-11-15T05:12:57Z ~ 05:15:02Z，**failure**；没有重跑 |
| 代码点 | `267012b71ea149e036038e3bc7d3eafa4e08b1c7`，树 `6dfff00592d460c537654ced835f1d810cd48491`，工具模块子树 `bb314889c4788e527dfc7e84fd2b52095957fdaa`。三个作业都核对 HEAD 等于 `GITHUB_SHA`、工作区干净，`RUNNER_ENVIRONMENT` 为 `github-hosted` |
| 工具链 | Ubuntu：`/opt/hostedtoolcache/go/1.27.3/x64/bin/go`，`go1.27.3 linux/amd64`，SHA-256 `5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b`。macOS：`/Users/runner/hostedtoolcache/go/1.27.3/arm64/bin/go`，`go1.27.3 darwin/arm64`，`aa355b078a81158d418079a3f8c776719eb99cc6f527e39ffa3250c66566aafb`。Windows：`/c/hostedtoolcache/windows/go/1.27.3/x64/bin/go`，`go1.27.3 windows/amd64`，`83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`（与本机第一阶段使用的 go1.27.3 的 `go.exe` 相同） |
| 命令 | 见 workflow 文件：身份核对；`go mod download` 与 `go mod verify`；`go vet ./...`、`-tags c9evidence_run`、`-tags c9evidence_realcall`，Linux 上 `gofmt -l`，正式入口只编译（`go test -tags c9evidence_run -c`，不运行）；`go test -count=1 -v ./...`；Linux 上 `go test -race -count=1 -v ./...`；`C9EVIDENCE_REALCALL=1 go test -tags c9evidence_realcall -count=1 -v ./internal/realcall/`；残留进程的观测 |

## 各作业的结果

| 作业 | 结论 | 步骤 |
|---|---|---|
| ubuntu-latest（132708668167） | success | 全部成功：步骤 A（vet、格式、正式入口编译、单元测试、`-race`）；步骤 B：RC-1（linux/amd64 0.29 s、linux/arm64 0.32 s）、RC-2、RC-3、RC-4（结论 4、退出 0，0.61 s）全部通过 |
| macos-latest（132708668081） | success | 全部成功（`-race` 只在 Linux，跳过）；步骤 B：RC-1（darwin/arm64 0.47 s、darwin/amd64 0.34 s）、RC-2、RC-3、RC-4（结论 4、退出 0，0.86 s）全部通过 |
| windows-latest（132708668297） | **failure** | 步骤 A 全部成功（含 LC-9 的 11 个适用变体）。步骤 B 失败，见下 |

## Windows 步骤 B 的失败（保留，按 §8 分析）

- **RC-1、RC-2**：A1-a 守卫失败：`resolve candidate: The system cannot find the path specified.`（`filepath.EvalSymlinks` 对 `exec.LookPath("go")` 的结果失败）。守卫失败之后没有任何真实调用（RC-1 的登记表为 0 条记录；RC-2 在建立调用之前停止）。这一版守卫没有记录 `LookPath` 的原始结果，所以本次运行不能确定是哪一个路径分量失败。
- **RC-3**：同一个守卫失败发生在测试自己启动的 goroutine 中（`listSpec` 在 goroutine 里调用了 `t.Fatalf`），测试等待 “到达 Wait” 60 秒后失败。这是测试代码的缺陷，不是协议的结果。
- **RC-4**：作业演练的结论为 2（未完成）：R 的两个配置都因 A1-a 失败而未评估，LC-1 未决；**LC-2 ~ LC-9 在 Windows runner 上全部命中**（`controls.jsonl`：109 个命中、9 个不适用，即 9 个只在 Linux 的 LC-9 变体；没有不命中或未决），LC-2 的符号链接变体在 GitHub 的 Windows runner 上可以创建链接。
- 残留进程的观测：三个作业都打印 “no go or test process”（只是观测，不作为清理的证据）。

## 文件

- `job-ubuntu.log`、`job-macos.log`、`job-windows.log`：`gh run view --log --job` 的原样输出（含 ANSI 转义）。
- `c9-tool-test-<runner>.tar.gz`：各作业上传的 artifact（`unit.log`、Linux 的 `race.log`、`realcall.log` 与 `realcall/` 之下的 RC 结果和 RC-4 的记录文件），重新打包；`ARTIFACTS.sha256` 是解包之后每个原始文件的 SHA-256。
- `run.json`、`jobs.json`：运行与作业（含每一步的结论）的元数据，取自 GitHub 的接口。
- `SHA256SUMS`：本目录中其余文件的 SHA-256。
