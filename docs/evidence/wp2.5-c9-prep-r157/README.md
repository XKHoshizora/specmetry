# Review 157 的本机验证：`c9-evidence.yml` 的身份读取与 junction 目标的判定

本目录是 review-resolution §208 登记、§209 记录的本机验证的原始材料：N-157-1（junction 目标的完整路径判定）与 N-157-2（身份读取的失败消费）两项 workflow 精化，以及 YAML 层面“其余设置不变”的比较。

**这些是本机的脚本验证，不是 runner 上的正式 workflow 执行。** 没有运行 Go、工具模块、正式入口或采集器，没有联网，没有建立或推送任何分支，没有触发任何 workflow。

| 项 | 值 |
|---|---|
| 被测的 workflow | 提交 `43b9f99999d29edd4d7c5f1cb6e4387a77144e73` 中 `.github/workflows/c9-evidence.yml` 的 blob `9ef8b58766200b7ca35aa6757a5f226d1e562b0b` |
| 对照 | Review 157 受审的 `dcc01bd` 中的 blob `852c032dc4c47c78a0897d13c1f0969ab5869dc6`；`master` `c34baf8` 中的 blob `11d7f1f14d1dea6daa85e1d9285d15a77f8b0302` |
| 环境 | 本机 Windows 11；bash 5.3.15（MSYS2）、git 2.56.1.windows.1、PowerShell 7.6.10、Node 26.10.0、`web/node_modules` 中的 prettier 3.9.10 |
| 运行 | `bash run-all.sh <仓库> <临时目录>`；驱动与三个夹具脚本都在本目录 |

## 文件

| 文件 | 内容 |
|---|---|
| `run-all.sh` | 驱动：取出三份 workflow，运行下面三项，写入 `outputs/`；临时目录与本机用户目录替换为 `<W>`、`<HOME>`，最后检查输出中没有残留的用户路径 |
| `yaml-check.mjs` | 以 prettier 的 YAML 解析器把三份 workflow 转为数据，打印步骤脚本之外的全部设置，按步骤名比较，并把每个 `run` 块取出为文件 |
| `bash-cases.sh` | 身份、LE-4 与 Collect 三个 bash 步骤的夹具用例 |
| `pwsh-cases.ps1` | Windows PATH 步骤的夹具用例 |
| `outputs/yaml-check.out` | YAML 的比较 |
| `outputs/syntax.out` | 每个 `run` 块的 `bash -n` 与 PowerShell 解析器的结果 |
| `outputs/bash-cases.out`、`outputs/pwsh-cases.out` | 每个用例的命令、原始输出与退出状态 |
| `SHA256SUMS` | 本目录全部文件（它自己除外）的 SHA-256 |

## 方法

- **步骤的运行方式**：每个 `run` 块都从 YAML 中取出原文。bash 块按 GitHub 的方式运行（`bash --noprofile --norc -eo pipefail <file>`）；PowerShell 块前置 `$ErrorActionPreference = 'stop'`、后接退出码的检查，与 GitHub 对 `pwsh` 步骤的包装相同。
- **bash 夹具**：每个用例一个新的临时 Git 仓库（一次提交；关闭用户与系统的 Git 配置）。`go` 是一个打印固定文字的 shell 脚本，不是 Go 工具链。`git` 的读取失败或返回空串，由一个只拦截指定参数、其余都交给真实 git 的包装脚本构造。
- **路径转换**：本机的 MSYS2 运行时会把 `HEAD:.github/workflows/c9-evidence.yml` 这类参数当作路径列表改写。第二阶段 windows-latest 上的 Git Bash 原样传递了这个参数（[`ci-43906849891/job-windows.log`](../wp2.5-c9-phase2/ci-43906849891/job-windows.log) 第 218 行），所以夹具以 `MSYS_NO_PATHCONV=1` 运行。
- **PowerShell 夹具**：`go.exe` 是 Windows 自带程序的副本，只计算哈希，不运行。真实的用例使用一个真实的临时 junction 与一个普通目录。真实的 junction 只保存完整的目标，所以其他目标字符串由一个名为 `Get-Item` 的函数替身给出（PowerShell 中同名的函数优先于 cmdlet）；替身只回答 `LinkType` 与 `Target`。为了可读，输出统一为 UTF-8、去掉颜色控制码、PowerShell 与 .NET 的信息用英文；这不改变步骤的行为。

## 结果

**bash 步骤**（退出状态；“旧”指 `dcc01bd`）：

| 用例 | 本轮 | 旧 |
|---|---|---|
| 身份：成功 | 0，全部值打印 | 0 |
| 身份：HEAD 与 `GITHUB_SHA` 不同 | 1 | 1 |
| 身份：工作区不干净 | 1 | 1 |
| 身份：子树路径不存在 | 1（cannot read the tool module subtree） | **0**，继续打印 |
| 身份：`git status` 读取失败 | 1（cannot read the worktree status） | **0**，被当作干净 |
| 身份：树的读取返回空串 | 1（the tree is empty） | **0** |
| 身份：workflow 的 blob 读取失败 | 1（cannot read the workflow blob） | **0** |
| 身份：`go` 不在 PATH 上 | 1（go is not on PATH） | 127（`go version` 失败） |
| LE-4：成功 | 0 | 0 |
| LE-4：没有 `go.mod` | 1 | **0** |
| LE-4：`go.sum` 的 blob 读取返回空串 | 1 | **0** |
| LE-4：`git status` 读取失败 | 1 | **0**，被当作干净 |
| LE-4：下载改变了工作区 | 1 | 1 |
| Collect：成功 | 0 | 0 |
| Collect：`go` 不在 PATH 上 | 1（go is not on PATH） | 127 |

**PowerShell 步骤**：

| 用例 | 本轮 | 旧 |
|---|---|---|
| 真实的 junction | 0，目标的 `bin` 写入 `GITHUB_PATH` | 0，同左 |
| 普通目录（没有 junction） | 0，PATH 不变 | 0，同左 |
| 目标为盘符的完整路径 | 0 | 0 |
| 目标为 `\\?\` 设备路径 | 0 | 0 |
| 目标为 null | 1（the junction's target is empty） | 1 |
| 目标为空串 | 1（同上） | 1 |
| 目标为 `C:relative` | 1（not a fully qualified path） | 1，但是在哈希一步：路径被按当前目录解析之后找不到文件 |
| 目标为 `C:relative`，当前目录下有同样字节的 `relative\bin\go.exe` | 1（not a fully qualified path） | **0**，把相对路径 `C:relative\bin` 写入 `GITHUB_PATH` |
| 目标为 `\relative` | 1（not a fully qualified path） | 1，在哈希一步（按当前盘的根解析） |
| 目标为 `relative` | 1（not a fully qualified path） | 1 |
| 两个目标 | 1（does not have exactly one target） | 1 |
| 目标的 `go.exe` 字节不同 | 1（is not the go.exe on PATH） | 1 |
| 目标下没有 `go.exe` | 1（`Get-FileHash` 找不到文件） | 1 |

**YAML**：三份文件的 `name`、`on`、`permissions`、`concurrency` 与作业的全部设置相同；`dcc01bd` 与本轮都只多出 Windows 的 PATH 步骤；本轮相对 `master`，身份、LE-4 与 Collect 三个步骤只有 `run` 不同（LE-4 的 `go mod download` 一行不变）。全部 `run` 块的语法检查都没有错误。

## 限制

- 本机脚本验证，不是 runner 上的执行。新的 PATH 步骤与读取检查都没有在 runner 上运行过。
- `go` 与 `go.exe` 都是替身，没有运行 Go；正式入口、工具模块与采集器都没有运行。
- `Get-Item` 的函数替身只用来给出真实 junction 产生不了的目标字符串；它不证明 runner 上的 junction 会返回这些值。
- 本机的 PowerShell、.NET 与 bash 版本不一定与 runner 相同；`MSYS_NO_PATHCONV=1` 是为了与第二阶段 runner 上观察到的参数传递一致。
- 相同的 SHA-256 只说明两次读到的字节相同，不证明安装的来源、整个 GOROOT 的内容或文件之后不变。
