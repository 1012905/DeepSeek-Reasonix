import { t } from "../i18n";

// The settings page names a tool by what it does, in the interface's language. The
// kernel's description is written for the model and is only the fallback for a tool
// this build has no line for, such as one an MCP server declares.
export function toolBlurb(name: string): string | undefined {
  switch (name) {
    case "ask":
      return t("向你提问：遇到只有你能决定或提供的事时，模型用它来问你。");
    case "await_user":
      return t("把控制权交还给你，任务清单保持未完成，等你给出下一步需要的东西。");
    case "bash":
      return t("在命令行里执行命令并返回输出，用于构建、测试、git、包管理等。");
    case "bash_output":
      return t("读取后台任务新产生的输出。");
    case "browser_act":
      return t("在浏览器页面上执行真实操作，比如点击和输入。");
    case "browser_open":
      return t("打开一个网页并返回页面快照。");
    case "browser_read":
      return t("读取浏览器页面的内容：快照、查找或截图。");
    case "complete_step":
      return t("记录已批准计划中某一步的完成情况和依据。");
    case "compress":
      return t("压缩当前对话中指定的一段，不删除可见历史；只在你明确要求时使用。");
    case "context_budget":
      return t("报告当前对话在自动压缩前还剩多少上下文空间。");
    case "edit_file":
      return t("把文件里一处精确的文本替换成另一段，适合定点修改。");
    case "kill_shell":
      return t("终止一个仍在运行的后台任务。");
    case "read_file":
      return t("读取文本文件，可指定起始行和行数，输出带行号。");
    case "recall":
      return t("找回已折叠的对话内容：按压缩摘要里的位置读取，或在折叠区域里搜索。");
    case "setting":
      return t("查找并修改你自己的设置；改动前会先征得你同意。");
    case "todo_write":
      return t("记录并更新当前工作的任务清单，每次发送完整清单。");
    case "update_goal":
      return t("汇报当前目标这一轮的处理结果。");
    case "use_capability":
      return t("按稳定的 ID 搜索并调用延后加载的工具、技能和 MCP。");
    case "view_image":
      return t("查看工作区里的一张图片（PNG、JPEG、GIF、WebP），需要支持图片的模型。");
    case "wait":
      return t("等后台任务结束，返回每个任务的状态和最终输出。");
    case "write_file":
      return t("写入完整的文件内容，覆盖原文件并创建所需目录；只改一部分请优先用 edit_file 或 multi_edit。");
    default:
      return undefined;
  }
}
