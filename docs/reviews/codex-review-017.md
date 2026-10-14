# Codex 审查 017：WP1.1 最终定向复审

- 审查日期：2026-10-14
- 仓库：`XKHoshizora/specmetry`
- PR：[PR #6](https://github.com/XKHoshizora/specmetry/pull/6)
- 最终审查目标：`7bb4b84ce0736f52fd503f9b0aaf9fff26729de9`
- 增量基准：上一轮审查目标 `66d5948ba3af293642ee1bc63220620ee3766144`
- 相关修复：`ef8cad3` 关闭旧版 Git 版本探测的 Trace2 泄露；`7bb4b84` 修复 SQLite `TestOpenLockTimeout` 的测试辅助进程问题。
- 上一轮报告：[Codex Review 016](codex-review-016.md)
- 目标 CI：[CI 第 19 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38546544133)
- 范围：Review 016 最后一个 B1、真实 Git 2.31.1 的安全拒绝路径、SQLite 锁测试辅助进程的引用生命周期与 GC、崩溃释放、修复前后对照，以及最终提交对应的 CI。B2、B3、N1、N2 和既有工作包不重新全面复核；doctor 的 Git Trace 问题保持独立遗留任务。

**结论：GO。** 对最终提交 `7bb4b84ce0736f52fd503f9b0aaf9fff26729de9` 的定向复审通过。Review 016 最后一个 B1 可以正式关闭，WP1.1 可以验收并合并 PR #6。

审查期间原仓库保持只读，独立反例与修复前后对照均在隔离的临时副本、临时仓库和数据目录中运行，未提交、推送或合并。本报告随后按照用户要求单独保存。

## BLOCKER

**无。**

## 定向复审结果

### 1. B1 已关闭

[受控环境](../../internal/gitx/gitx.go#L246) 在首次 `git --version` 执行前就设置三个 Trace2 关闭变量：

```text
GIT_TRACE2=0
GIT_TRACE2_EVENT=0
GIT_TRACE2_PERF=0
```

继承的 `GIT_*` 仍全部过滤。没有发现环境隔离回归。

### 2. 真实 Git 2.31.1 安全拒绝路径通过

复用了上一轮构建的真实二进制，独立验证：

- 对照调用实际生成三种含 canary 的 Trace2 文件，前置条件有效。
- 最终 CLI 返回退出码 **6**，审计确认只执行 `--version`。
- stdout、stderr 不包含 canary；没有外部跟踪文件、数据库或 Marker。
- `.git` 文件内容及引用保持不变。

仓库中的真实旧版回归测试也连续 **3/3 通过**。这与替身测试分开验证，没有混同。

### 3. SQLite 测试辅助进程的根因与修复成立

原辅助进程丢弃 `openLock`，使底层 `os.File` 可被垃圾回收；finalizer 关闭描述符后，锁提前释放。[新增的 `defer lock.release()`](../../internal/store/sqlite/store_test.go#L375) 会保留接收者引用，覆盖整个等待期间。

独立对照结果：

- 临时恢复“丢弃引用”、保留强制 GC：Windows、Linux 均 **3/3 失败**，原断言捕获提前释放。
- 修复版：Windows 连续 **10 轮**、Linux 连续 **20 轮**通过超时与崩溃释放测试。
- 另行确认：GC 后仍持锁，强制终止进程后由操作系统释放，两平台各 **3/3 通过**。

没有扩大 Timeout、减弱断言或跳过失败测试。`7bb4b84` 只改变测试代码和处理记录，未改变生产存储行为。

### 4. 最终提交的 CI 已核实

[CI 第 19 次运行](https://github.com/XKHoshizora/specmetry/actions/runs/38546544133) 对应最终提交，**8/8 成功**。三平台均实际通过相关锁测试和替身测试；真实旧 Git 测试在 CI 中跳过，由本次独立 Linux 复验补足。

### 5. 未发现新的实质性 BLOCKER

未发现新的实质性安全、隐私或数据一致性 BLOCKER。

### 6. WP1.1 可以正式验收并合并

可以正式验收 WP1.1，并合并 PR #6。

此结论仅针对 WP1.1，不代表后续工作包或整个 M1 已完成。

## NON-BLOCKER

doctor 的 Git Trace 写入继续保留为独立遗留安全任务，**发布前必须关闭**；它不是本次 WP1.1 的新增阻塞项。

## 验证边界与保留事项

- Windows Git／CLI 回归及相关包静态检查通过；正常初始化实际验证使用 Git 2.43.0、2.56.0，没有将替身结果当作真实 Git 2.32.0 实测。
- 本机 Race Detector 尝试因缺少 `gcc` 未能执行；最终 Ubuntu CI 使用 `-race`，相关测试通过。
- 此前 37 次运行中未记录名称的失败，仍不能确定就是 `TestOpenLockTimeout`，也不据此扩大阻塞范围。

## 最终答复

| 问题 | 结论 |
|---|---|
| Review 016 的最后一个 B1 是否真正关闭？ | **是，可以正式关闭。** |
| 真实 Git 2.31.1 的安全拒绝路径是否通过？ | **是。** 已独立验证真实二进制行为，不以替身测试代替。 |
| SQLite `TestOpenLockTimeout` 的根因与修复是否成立？ | **成立。** 强制 GC 对照稳定复现引用丢失，修复版保持持锁并正确释放。 |
| 最新三平台 CI 是否对应最终提交，且全部成功？ | **是，8/8 成功。** |
| 是否出现新的实质性 BLOCKER？ | **未发现。** |
| WP1.1 是否可以正式验收并合并 PR #6？ | **可以。最终结论：GO。** |
