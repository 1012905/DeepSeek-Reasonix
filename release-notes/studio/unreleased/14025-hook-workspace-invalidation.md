## 修复

- 修正宿主 hook 失效通知：工具被拒绝、只读调用或嵌套调用也能向已接入的接收器
  报告可能的工作区变化；不把 hook 副作用记作工具已执行。Refs #14025
- Notify opted-in host sinks of possible hook writes, including refused, reader
  and nested calls; keep tool execution independent. Refs #14025
