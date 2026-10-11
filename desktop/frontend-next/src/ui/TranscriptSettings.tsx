import type { AgentPort } from "../port/port";
import { ChangedFilesSetting } from "./ChangedFilesSetting";
import { FoldWorkSetting } from "./FoldWorkSetting";

/** What the conversation view shows around a turn: the changed-files pill and
 *  whether a finished turn's work is folded. */
export function TranscriptSettings({ port }: { port: AgentPort }) {
  return (
    <>
      <ChangedFilesSetting port={port} />
      <FoldWorkSetting port={port} />
    </>
  );
}
