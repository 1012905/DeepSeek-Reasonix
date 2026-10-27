任务费用门禁与记账事件使用同一份报价。

## 修复

- 本地 Agent 费用门禁采用发生时刻的权威报价，避免高峰费率下按基础价累计而多执行工具轮。关联 #14025。
- 使用固定点金额并保留原币种桶，显示币种不改变门禁；缺失用量仍标记不完整。共享根费用与硬预留不在本次范围。关联 #14025。

---

<!-- lang:en -->

## Fixed

- Local Agent cost gates consume the same occurrence-time quote as accounting, preventing base-rate accounting from allowing an extra tool round during a peak-rate window. Refs #14025.
- Fixed-point original-currency buckets preserve incomplete usage. Display currency does not change the gate; shared root money and hard reservations remain separate. Refs #14025.
