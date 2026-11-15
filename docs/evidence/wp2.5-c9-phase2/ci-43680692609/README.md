# 第二阶段测试 workflow 的第 2 次运行：run 43680692609

| 项 | 值 |
|---|---|
| workflow | `.github/workflows/c9-tool-test.yml`（这一代码点的 blob 见 `job-*.log` 的身份步骤） |
| 运行 | run 43680692609，attempt 1，`push` 到 `c9test/phase2`，**failure**；没有重跑 |
| 代码点 | `f1b405848127fd7e4cdd2985a5c3a5664dac5f8e`，工具模块子树 `aab881f5758cb4809610a420da8da70839ad0631`（相对第 1 次运行：守卫记录 `LookPath` 的原始结果并写进错误；新增 RC-0 诊断；RC-3 在测试 goroutine 中取得调用规格；Windows 的路径观测步骤） |
| 工具链 | 三个 runner 都是 setup-go 安装的 go1.27.3，路径与 SHA-256 同第 1 次运行 |

## 各作业的结果

| 作业 | 结论 | 说明 |
|---|---|---|
| macos-latest（132710439886） | success | 步骤 A 与步骤 B 全部成功：RC-0、RC-1（darwin/arm64 0.29 s、darwin/amd64 0.18 s）、RC-2、RC-3、RC-4 通过 |
| ubuntu-latest（132710439823） | **failure** | 步骤 A 的 `-race` 一步失败：第一阶段已有的测试 `TestCase2aManifestCloseLateParentFirst`（`internal/chain/walkthrough_test.go:121`）读取尚未发布的 `s.Writer.Manifest.Close`，空指针 panic。按“只在 A 之后”，步骤 B 被跳过。同一测试在第 1 次运行的 `-race` 中通过 |
| windows-latest（132710439672） | **failure** | 步骤 A 全部成功。步骤 B：RC-0 记录了守卫输入的逐分量结果；RC-1 ~ RC-4 都因 A1-a 守卫失败而没有任何真实调用 |

## 分析

- **Windows 的守卫**（事实，RC-0 与路径观测步骤）：`exec.LookPath("go")` 得到 `C:\hostedtoolcache\windows\go\1.27.3\x64\bin\go.exe`。其中 `C:\hostedtoolcache\windows\go\1.27.3\x64` 是 junction，目标为 `D:\hostedtoolcache\windows\go\1.27.3\x64`；它的 `Lstat` 模式为 `?rw-rw-rw-`（`ModeIrregular`，没有 `ModeDir`）。`filepath.EvalSymlinks` 对 junction 本身返回原路径，对它之下的 `bin` 与 `go.exe` 都返回 “The system cannot find the path specified.”。固定 Go 1.27.3 的源码：`path/filepath/symlink.go:90 ~ 93` 中，不是符号链接、也不是目录的中间分量即返回 `syscall.ENOTDIR`；Windows 上 `ENOTDIR` 即 `ERROR_PATH_NOT_FOUND`（`syscall/zerrors_windows.go:9`）；Go 1.23 起 mount point 的模式是 `ModeIrregular`（`os/types_windows.go` 的 `mode` 与 `winsymlink`）。所以 §12.14 第 1 条的守卫（经 `EvalSymlinks` 解析候选）对 setup-go 在 windows-latest 上的安装必然失败。守卫的规则不改。
- **Ubuntu 的 `-race` 失败**：测试侧的前提缺失。诊断任务在快照点登记后，在自己的 goroutine 中写三行快照行；`manifestCloseLate` 之后，2a、2b 两个测试把假时钟推过 `D`，并假定诊断任务已经写完这三行、正在等待写者。没有任何同步保证这一点：诊断任务较慢时，它在 `D` 之后仍有待写的行，按规则走到期出口（工具的行为符合规则），测试随后读取写者尚未发布的关闭结果。工具的判定代码不改；修正只在测试中加入这一前提。

## 文件

同第 1 次运行（`job-<runner>.log`、artifact 的 tar.gz 与 `ARTIFACTS.sha256`、`run.json`、`jobs.json`、`SHA256SUMS`）。Ubuntu 的 artifact 中没有 `realcall`（步骤 B 被跳过）。
