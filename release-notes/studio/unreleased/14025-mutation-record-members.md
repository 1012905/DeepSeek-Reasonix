## 修复

- Refuse repeated fields and case aliases in mutation records before admission or dead-owner cleanup, retaining their bytes. Writer formats and runtime activation stay unchanged. (#14025)

- 写入事务注册表在准入和死亡持有者回收前拒绝重复字段及大小写别名，保留原字节供兼容实现检查；正常写入格式不变，本次不启用共享事务。 (#14025)
