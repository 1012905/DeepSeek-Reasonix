## 修复

- 工具 hook 可能写文件时，即使目标工具只读，也先进入 checkpoint 写屏障并记入
  覆盖缺口；准入失败时拒绝执行 hook。Refs #14025
- Admit possible tool-hook writes through the checkpoint barrier even around
  readers, preserving undo retirement and explicit coverage gaps. Refs #14025
