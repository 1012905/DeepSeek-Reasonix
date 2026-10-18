import { useEffect, useState } from "react";
import type { AgentPort } from "../port/port";
import type { GitRepoRow } from "../port/workspace";

// The workspace's repositories as the kernel lists them: itself when it is one,
// otherwise those found below it. Empty until answered, and empty for a workspace
// git knows nothing about, which is not a failure. Read again when `again` moves.
export function useGitRepos(port: AgentPort, again = 0): GitRepoRow[] {
  const [repos, setRepos] = useState<GitRepoRow[]>([]);
  useEffect(() => {
    let live = true;
    port.gitRepos().then((r) => live && setRepos(r.repos), () => live && setRepos([]));
    return () => { live = false; };
  }, [port, again]);
  return repos;
}
