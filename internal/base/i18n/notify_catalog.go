package i18n

// The text of every desktop notification, per language; each catalogue embeds its own.

var notifyEN = NotifyMessages{
	UnattendedMessages: unattendedEN,
	PaidSearchMessages: PaidSearchMessages{PaidSearchApprovalReasonFmt: "web_search wants to send this query to DeepSeek's paid Search API: %[3]s. DeepSeek bills %[2]s. Allow it for %[1]s?"},
	AskTimeoutMessages: AskTimeoutMessages{
		CmdAskTimeout: "skip a question nobody answers after a wait", AskTimeoutFmt: "ask timeout: %s — set it with /ask-timeout <seconds|off>", AskTimeoutOff: "off", AskTimeoutBadFmt: "ask-timeout: %q is neither a number of seconds nor off", AskSkipFmt: "skipped in %s — press any key to keep it",
	},
	NotifyTitle:         "Reasonix",
	NotifyTurnDone:      "The turn finished",
	NotifyTurnFailed:    "The turn failed",
	NotifyApproval:      "Waiting for your approval",
	NotifyAsk:           "The model asked you something",
	NotifyFeedbackReply: "Your feedback has a new reply",
	NotifyDiskLow:       "Disk space is low: %s left on the drive holding %s",
}

var notifyZH = NotifyMessages{
	UnattendedMessages: unattendedZH,
	PaidSearchMessages: PaidSearchMessages{PaidSearchApprovalReasonFmt: "web_search 想把这条搜索词发给 DeepSeek 的付费搜索接口：%[3]s。DeepSeek 按次计费：%[2]s。是否允许使用 %[1]s？"},
	AskTimeoutMessages: AskTimeoutMessages{
		CmdAskTimeout: "设置提问无人回答多久后自动跳过", AskTimeoutFmt: "提问等待超时：%s — 用 /ask-timeout <秒数|off> 修改", AskTimeoutOff: "关闭", AskTimeoutBadFmt: "ask-timeout：%q 既不是秒数也不是 off", AskSkipFmt: "%s 后自动跳过 — 按任意键保留",
	},
	NotifyTitle:         "Reasonix",
	NotifyTurnDone:      "这一轮结束了",
	NotifyTurnFailed:    "这一轮失败了",
	NotifyApproval:      "有操作在等你批准",
	NotifyAsk:           "模型问了你一个问题",
	NotifyFeedbackReply: "你的反馈有新回复",
	NotifyDiskLow:       "磁盘空间不足：存放 %[2]s 的磁盘只剩 %[1]s",
}

var notifyZHTW = NotifyMessages{
	UnattendedMessages: unattendedZHTW,
	PaidSearchMessages: PaidSearchMessages{PaidSearchApprovalReasonFmt: "web_search 想把這條搜尋詞傳給 DeepSeek 的付費搜尋介面：%[3]s。DeepSeek 按次計費：%[2]s。是否允許使用 %[1]s？"},
	AskTimeoutMessages: AskTimeoutMessages{
		CmdAskTimeout: "設定提問無人回答多久後自動跳過", AskTimeoutFmt: "提問等待逾時：%s — 用 /ask-timeout <秒數|off> 修改", AskTimeoutOff: "關閉", AskTimeoutBadFmt: "ask-timeout：%q 既不是秒數也不是 off", AskSkipFmt: "%s 後自動跳過 — 按任意鍵保留",
	},
	NotifyTitle:         "Reasonix",
	NotifyTurnDone:      "這一輪結束了",
	NotifyTurnFailed:    "這一輪失敗了",
	NotifyApproval:      "有操作在等你核准",
	NotifyAsk:           "模型問了你一個問題",
	NotifyFeedbackReply: "你的回饋有新回覆",
	NotifyDiskLow:       "磁碟空間不足：存放 %[2]s 的磁碟只剩 %[1]s",
}
