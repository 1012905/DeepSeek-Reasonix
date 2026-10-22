## 修复

- 修复最终回答后的工作区观察遇到阻塞 I/O 时，会话仍长时间显示运行中的问题；观察现在遵守既有时限，停止操作也会取消观察和证据封存的等待，未完成的观察会记录为证据缺口。（#13953）

---
<!-- lang:en -->

## Fixed

- Fixed blocked workspace observation keeping sessions active after a final answer.
  Stop now cancels observation and evidence-seal waits; missing observations remain explicit gaps. (#13953)
