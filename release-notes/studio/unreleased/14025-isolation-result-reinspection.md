## 修复

- 隔离成果在其他会话应用或丢弃后，旧会话不能再靠缓存写回；候选发生变化时，
  要求重新检查路径后再应用。Refs #14025
- Reinspect isolated results before settlement so stale sessions cannot revive
  settled work. Changed candidates require fresh path admission. Refs #14025
