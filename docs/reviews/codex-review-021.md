# Codex 审查 021：WP1.5 Review 020 修复最终定向复审

- 审查日期：2026-10-15
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #8](https://github.com/XKHoshizora/specmetry/pull/8)
- 审查目标：`0af9d193bd3fa149e721eaa3e3db54345e975730`
- 增量基准：`2abca0014a43b5abf5e67d9b587077b0caeeb480`
- 前轮报告：[Codex Review 020](codex-review-020.md)
- CI：[Actions run 38621326456](https://github.com/XKHoshizora/specmetry/actions/runs/38621326456)
- 范围：Review 020 的 B1～B5、相关回归及 N1/N2 测试有效性；不重新全面审查已通过的工作包，不提前要求 WP1.6 或 M2 功能。
- 操作记录：复审期间原仓库保持只读，故障注入均在隔离临时环境中执行，未连接真实远端。本报告在复审完成后按用户要求单独保存，未修改其他仓库文件，未提交、推送或合并。

**最终结论：NO-GO。** Review 020 的 **B2～B5 可以关闭，B1 仍存在可复现的绕过**。当前不建议验收 WP1.5 或合并 PR #8。

审查目标为 `0af9d193`，增量基准为 `2abca001`。已核对[指定 CI](https://github.com/XKHoshizora/specmetry/actions/runs/38621326456)：8/8 成功，三平台测试的合并提交确实包含目标提交；Ubuntu 启用了 Race Detector。以下反例均在隔离目录执行，未连接真实远端。

## BLOCKER

### B1 未关闭：部分克隆识别和协议限制仍可绕过

这是原 B1 的残余漏洞，包含三个相关缺口。

**① 部分克隆检测未完整遵循 Git 的布尔值语义。**

位置：[gitx.go:559](../../internal/gitx/gitx.go#L559)。

实现只将空值、`true`、`yes`、`on`、`1` 视为 true。但真实 Git 2.32.0 将：

```ini
[remote "origin"]
    promisor = 2
```

解释为 true，当前 `PartialClone` 却返回 false。

独立复现中，仓库没有设置 `extensions.partialClone`，只有上述合法的 promisor 配置。删除一个已提交 Blob 后调用 `CountLines`，旧版 Git 仍启动了内部 Fetch，并修改 `.git/config`，追加部分克隆过滤配置。

**影响：**部分克隆可进入本应禁止的对象读取路径，违反“读取之前拒绝”和仓库只读契约。

**② `specmetry-no-transport` 是被允许的协议名，并不代表禁止全部协议。**

位置：[gitx.go:349](../../internal/gitx/gitx.go#L349)。

组合反例如下：

1. 使用真实 Git 2.32.0。
2. 设置 `remote.origin.promisor=2`。
3. 设置 URL 为 `specmetry-no-transport::unused`。
4. 在临时 PATH 中放置 `git-remote-specmetry-no-transport`，仅向临时标记文件追加一行后退出。
5. 删除已提交文件的 Blob，通过公开 `propose --json` 引用该文件。

**实际结果：**

- 本地 remote helper 被执行，标记文件出现三行；
- `.git/config` 被修改；
- 随后 `propose` 才返回退出码 6；
- Proposal 数据和 seq 保持不变，但 Git 与外部文件副作用已经发生。

这证明返回错误不能补救此前的越界执行。测试 helper 没有网络行为；实际 helper 具有出站能力时，现有限制无法保证核心无出站请求。

**③ 部分克隆的负结果缓存不会随配置变化失效。**

位置：[gitx.go:446](../../internal/gitx/gitx.go#L446)。

真实 Git 2.32.0 的独立反例：

1. 在普通仓库调用 `CommitOf`，缓存“不是部分克隆”。
2. 将同一仓库配置改为 `remote.origin.promisor=true`，并移除 Blob。
3. 再调用 `CountLines`。

配置检查被缓存跳过，内部 Fetch 再次修改 `.git/config`。比较快照的起点在故障注入完成之后，因此该变化确实来自被测读取操作。

**最小修复建议：**

- 用 Git 自身的布尔值解析能力处理 promisor；无法可靠解释配置时保守拒绝。
- 将协议允许列表设为真正的空列表，并用真实 Git 2.32 验证所有传输和自定义 helper 均被拒绝。
- 取消旧版 Git 路径上的负结果缓存，避免配置变化后继续沿用“安全”判断。
- 补充上述数值布尔值、配置变化和同名 helper 的回归测试，继续比较完整 Git 目录及外部标记。

无需重构存储层或改变已批准的产品方向。

## NON-BLOCKER

未发现需要另列的新问题。原 N1、N2 已合理修复：

- **N1：**测试文件确实超过 1 MiB，多字节字符实际跨块；匹配后出现 NUL、截断 UTF-8 仍得到 `not_checkable`。取消测试在首块读取之后触发，确实覆盖块间取消及无部分结果。
- **N2：**直接提交 Index，验证前明确断言提交树包含 `120000`、`160000` 和 `100755` 条目，原先被 `git add -A` 移除条目的前置条件问题已解决。

## ACCEPTED RISK

- 部分克隆中的 Code Ref 可以按本轮明确的边界直接拒绝；限制本身可以接受，识别必须可靠。
- Context 取消在普通文件的分块读取之间生效；本轮测试证明的是这一保证。
- WP1.6 的审核、正式编号和 Baseline 应用，及 M2 状态推导继续保持原范围，不作为本轮阻塞项。

## 修复关闭情况

| 项目 | 结论与验证 |
|---|---|
| **B1** | **未关闭。** 标准配置下的缺失 Blob、Tree、Commit 测试通过，但真实 Git 2.32 的上述绕过可造成副作用。 |
| **B2** | **关闭。** 独立验证 `sources_json`、`final_payload_json` 的 INSERT/UPDATE、嵌套非 NULL `excerpt` 和维护模式；`source_refs` 同样受保护。拒绝后事实、seq、维护状态回滚，打开及关闭数据库后均未发现 canary；脱敏审核结果仍可写入。 |
| **B3** | **关闭。** 新增、更新 Requirement 的 `goal:null` 可以提交，payload 保留 null；不存在、类型错误、已退役的非空 Goal 引用仍拒绝，没有自动应用 Baseline。 |
| **B4** | **关闭。** 静态校验和 exists 判断共用解析器，溢出、倒置和超限输入拒绝且不写入；合法上限、空文件和实际行数判断通过。另实际运行了 Windows 32 位解析测试，与 64 位结果一致。 |
| **B5** | **关闭。** Linux 无写端 FIFO、普通文件/FIFO 替换、公开 `propose` 和 `source list` 回归通过；替换测试实际观察到普通文件与特殊文件两类结果。Windows NUL、目录及正常文件回归通过，失败输出未发现新增路径回显。 |

真实 Git 2.31.1 的安全拒绝也已复验通过，包含有效 Trace2 对照；FSMonitor、环境配置注入、Hook、Trace 和错误脱敏回归通过。

Windows 本轮定向测试通过；Linux 测试在 WSL 中配合真实 Git 2.32.0、2.43.0 执行。macOS 和 Ubuntu Race Detector 依据实际 CI 日志，未宣称完成本地复验。

## 最终回答

1. **B1～B5 尚未全部关闭。**
2. **Git 2.32 已经真实版本验证，验证发现了残余漏洞。**
3. **hash_only 数据库第二道防线有效。**
4. **N1、N2 的无效测试前置条件已修复。**
5. 除 B1 的残余安全问题外，未发现新的实质性隐私或数据一致性缺陷。
6. **WP1.5 暂不能正式验收，PR #8 暂不能合并。**

**最终代码审查：NO-GO。** 复审期间原仓库保持不变，未修改、提交、推送或合并。本文件为复审完成后按用户要求保存的完整报告。
