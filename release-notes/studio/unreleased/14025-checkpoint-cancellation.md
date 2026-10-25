## 修复

- Cancelled calls leave checkpoint waits before hooks/tools. Repeated permit
  cleanup preserves other writers; background registration stays separate. Refs #14025
- 取消工具调用后可退出检查点准入等待，不再等待独占方释放后继续进入同步 hook 或
  工具；重复清理同一许可不会释放其他写入者。后台登记及异步 hook 生命周期另行处理。Refs #14025
