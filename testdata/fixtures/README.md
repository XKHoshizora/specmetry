# 项目测试夹具

规格见 [acceptance-criteria.md §1.1](../../docs/acceptance-criteria.md#11-测试夹具testdatafixtures)。测试用 `testutil.BuildFixture(t, "<名称>")` 在临时目录中把夹具构造成 git 仓库，仓库中不提交嵌套的 `.git`。

| 夹具 | 阶段（architecture §6.1.1） | 内容 |
|---|---|---|
| `fx-docs-only` | `docs_only` | 客户需求说明（8 条需求，R8 没有可判定的验收描述，R3、R7 各有一处歧义）；README、CHANGELOG、`node_modules/x/README.md` 和空文件 `docs/empty.md` 用于候选来源的排除规则；没有代码。附带录制的 `proposal.json` |
| `fx-in-dev` | `in_development` | 同一份需求说明，加上只用 Go 标准库的订单服务：`main` 实现 R1～R5，`feature/x` 再实现 R6 |
| `fx-incomplete` | `incomplete_docs` | 只有两段话的 README，加上实现了 6 个功能的便签代码 |

## 目录约定

每个夹具一个目录：

- `fixture.json`：构造清单。`commits` 按顺序把列出的文件从 `repo/`（或 `from` 指定的目录）复制进仓库，并以固定的时间提交；`branches` 从指定分支创建分支并提交；最后回到 `main`。`expected_refs` 固定每个分支的提交 ID。
- `repo/`：`main` 分支的最终内容。构造后，主 worktree 干净，跟踪的文件恰好就是 `repo/` 中的文件。
- `branches/<名称>/`：只在某个分支上出现的文件。
- 录制的 Agent 输出（例如 `fx-docs-only/proposal.json`）放在夹具目录下，不属于被构造的仓库。

## 稳定性

- 提交者身份、提交时间固定，git 配置与本机隔离，文件按原样复制（仓库的 `.gitattributes` 保证检出为 LF）。所以三个平台上构造出的提交 ID 相同，测试会对照 `expected_refs` 检查。
- 夹具中没有本机路径、真实账号、网络地址或密钥，测试会扫描检查。
- 修改夹具文件或清单后，提交 ID 会变化：确认改动无误后，把测试报告的新 ID 写回 `expected_refs`。

## 尚未包含的内容

`fx-in-dev` 中故意失败的测试和录制的 Claim、其余夹具和期望结果（oracle）在 WP2.9 等后续工作包中补齐。
