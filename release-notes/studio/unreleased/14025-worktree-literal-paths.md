## 修复

- 隔离整合按字面文件名检查和写入，避免特殊文件名触发 Git 模式展开并覆盖相邻
  用户修改；修复换行、引号等文件名的哈希输入。Refs #14025
- Isolated integration treats filenames literally and preserves unrelated edits.
  Hashing handles quoted and newline-containing names safely. Refs #14025
