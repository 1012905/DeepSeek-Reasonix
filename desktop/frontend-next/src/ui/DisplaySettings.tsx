import type { AgentPort } from "../port/port";
import { ChangedFilesSetting } from "./ChangedFilesSetting";
import { FoldWorkSetting } from "./FoldWorkSetting";
import { TurnNarrationSetting } from "./TurnNarrationSetting";
import { ReceiptSetting } from "./ReceiptSetting";

// What the conversation screen shows, each kept by the kernel the same way.
export function DisplaySettings({ port }: { port: AgentPort }) {
  return (
    <>
      <ChangedFilesSetting port={port} />
      <FoldWorkSetting port={port} />
      <TurnNarrationSetting port={port} />
      <ReceiptSetting port={port} />
    </>
  );
}
