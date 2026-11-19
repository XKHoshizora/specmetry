# 本机验证 6：Review 155 修复之后的代码点

| 项 | 值 |
|---|---|
| 代码点 | `5e6c3a4b93b5189c807bee4d179494e837a02b30`，树 `b6ac8721dedcd6d152aaab4facf74cb7f616e2b6`，工具模块子树 `755ffc0c2465959fa5528d07b5c079154e8e4feb` |
| 工具链 | `go1.27.3 windows/amd64`，SHA-256 `83b7ba66215e696414fa49462d0db2da4f4cf15c7bebd42f0b189fa04866999d`；`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=` |
| 时间 | 2026-11-16T16:21:37Z ~ 16:22:30Z（UTC） |
| 工作区 | 运行前后各有 2 个变更路径：未跟踪的 `local-5/` 与正在写入的本目录；被测代码没有未提交的改动 |

| 文件 | 内容 | 结果 |
|---|---|---|
| `windows-unit.log` | Windows 原生：`go test -count=1 -v ./...` | 退出 0；538 条 `--- PASS`（顶层 237，子测试 301），0 条 `--- FAIL`，6 条 SKIP：`TestR154RequestOtherKind`、`TestR155OverflowOtherSignal`、`TestR155ExpiryOtherSignal`、`TestR155OverflowSIGPIPEHit`、`TestR155ExitedBeforeRequest`（只在 Linux 或 Unix）与 `TestTakeNegatives/symbolic_link`（本机没有创建符号链接的权限）；LC-9 的 11 个变体命中、9 个不适用 |
| `windows-static.log` | 三个 GOOS × 三种标签集合的 `go vet`，`gofmt -l`，`go mod verify`，正式入口只编译 | 都退出 0；`gofmt -l` 没有输出 |
| `wsl-linux.log` | 交叉编译的 linux/amd64 测试二进制在 WSL（内核 6.18.33.2-microsoft-standard-WSL2）中运行（只有伪工具、接缝与构造的快照） | 14 个包都退出 0；544 条 `--- PASS`（顶层 242，子测试 302），0 条 `--- FAIL`，没有 SKIP；LC-9 的 20 个变体命中 |

- 测试的数目：顶层 242 个、子测试 302 个（Windows 上其中顶层 5 个、子测试 1 个为 SKIP）。上一代码点 `a0ed8dc` 是顶层 229、子测试 279；增加的 13 个顶层测试是 Review 155 的回归（toolctl 的 H1、H2、C2，controls 的 H3、H4、S1 ~ S6，keep 的 H5，realcall 的 C1），23 个子测试是 C1 的构造用例。
- `realcall` 包在没有构建标签时只编译 `doc.go` 与 `rc3_test.go`：只有 RC-3 判定的纯函数与它在构造快照上的单元测试，不启动进程，不做真实调用。
- 这些不是三平台的证据，没有 `-race`，也没有运行 RC-0 ~ RC-4（本机不执行采集器的真实 Go 调用）。
- `SHA256SUMS` 是三个日志的 SHA-256。
