同一会话并发续跑祖先子任务时，不再重复创建尚未发布的转录副本。

## 修复

- 在现有 Store 内预留目标会话与祖先来源，覆盖首次复制到发布元数据之间的窗口。关联 #14025。
- 竞争续跑返回忙碌提示；不同会话和来源仍独立，已完成副本继续复用，不改变存储格式或工具权限。关联 #14025。

---

<!-- lang:en -->

## Fixed

- Reserve ancestor continuation copies within one Store before metadata publication, preventing competing calls from creating duplicate transcripts. Refs #14025.
- Busy continuations are refused; independent parents and sources remain separate. Completed copies are reused without storage-format or permission changes. Refs #14025.
