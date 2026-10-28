## 修复

- 应用隔离结果时，拒绝被换成目录链接或普通文件的父目录，避免重定向删除；保留原结果 ID，恢复目录后可重试。 (#14025)
- 保留正常删除、新目录创建、最终链接删除和整工作区文件转目录行为。 (#14025)

<!-- lang:en -->

## Fixed

- Refuse changed directory-link or file ancestors during isolated application; prevent redirected deletion and retain the original result ID for retry after restoring the directory. (#14025)
- Preserve ordinary deletion, new-directory creation, leaf-link deletion, and whole-workspace file-to-directory conversion. (#14025)
