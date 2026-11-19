# 第二阶段的本机验证 2（固定代码点 `ed84d09`）

| 项 | 值 |
|---|---|
| 代码点 | `ed84d091f3d352d46cdf525880998ba6ef569b91`，树 `65c53940869b9ef28b525966d85eaecaa97a0c62`，工具模块子树 `c943f922827282e2868ef4da84fe8e370f96f126`；运行前后工作区都干净 |
| 工具链 | 同本机验证 1（`go1.27.3 windows/amd64`，SHA-256 `83b7ba66…999d`；`GOTOOLCHAIN=local GOPROXY=off GOWORK=off GOFLAGS=`） |
| 时间 | 2026-11-15T05:29Z ~ 05:31Z |

| 文件 | 内容 | 结果 |
|---|---|---|
| `windows-unit.log` | Windows 原生：`go test -count=1 -v ./...` | 退出 0；486 个 `--- PASS`，0 个 `--- FAIL`；1 个 SKIP（`TestTakeNegatives/symbolic_link`，本机没有创建符号链接的权限） |
| `windows-static.log` | 三个 GOOS × 三种标签集合的 `go vet`，`gofmt -l`，`go mod verify`，正式入口只编译 | 都退出 0；`gofmt -l` 没有输出 |
| `wsl-linux.log` | 交叉编译的 linux/amd64 测试二进制在 WSL 中运行（只有伪工具与接缝） | 12 个包都退出 0；487 个 `--- PASS`，0 个 `--- FAIL`，没有 SKIP |

- 与本机验证 1 相同，这些不是三平台的证据，没有 `-race`，也没有运行 RC-1 ~ RC-4。
- `SHA256SUMS` 是三个日志的 SHA-256。
