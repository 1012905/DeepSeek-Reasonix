## 修复

- Child file reads keep their own change-protection record, including deferred
  tools and nested tasks, so they cannot refresh a parent's stale view. (#14025)
- 子代理的直接和动态文件读取使用独立记录，包含嵌套任务，避免误刷新父代理的
  过期文件视图并覆盖用户修改。 (#14025)
