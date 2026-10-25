## 修复

- Background task/fleet resources are finalized even when the Manager declines
  launch. Cleanup errors remain visible and do not skip sibling resources. Refs #14025
- 后台 task/fleet 即使未被 Manager 启动，也会清理写入登记、transcript 与进度资源；
  清理错误可诊断，且不会跳过其余资源。Refs #14025
