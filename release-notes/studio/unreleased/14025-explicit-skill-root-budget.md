显式子代理技能调用遵守已有任务根预算。

## 修复

- slash、composer 多选及 Goal 技能调用共享已有 token/wall 根预算，保留零额度不限和原有收尾。关联 #14025。
- CLI profile run/preview 复用同步回合准入及取消；忙碌、关闭、draining 和工作区 authority 检查同样适用。关联 #14025。
- 本次不提供硬预留、根金额汇总或完整隔离恢复保障。关联 #14025。

---

<!-- lang:en -->

## Fixed

- Explicit slash, composer and Goal subagent skills share the existing token/wall root, preserving unlimited zero and finalization. Refs #14025.
- CLI profile run/preview use synchronous turn admission and cancellation, including busy, closed, draining and workspace authority checks. Refs #14025.
- Hard reservations, root money accounting and complete isolation/recovery remain outside this change. Refs #14025.
