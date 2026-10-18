import { useCallback, useEffect, useState } from "react";
import type { AgentPort, WorkspaceChange } from "../port/port";
import type { GitRepoRow } from "../port/workspace";
import { useCommitCard } from "./CommitCard";
import { usePushCard } from "./GitPushCard";
import { StudioIcon } from "./StudioIcon";
import { WorkbenchChanges } from "./WorkbenchChanges";
import { WorkbenchHistory } from "./WorkbenchHistory";

interface Props {
  port: AgentPort;
  // The workspace's own repository's changes, kept current by the pane. A folder
  // of projects has none of its own: each repository below it reads its own.
  changes: WorkspaceChange[];
  repos: GitRepoRow[];
  // Moves when the agent has written, so a repository the pane does not watch is read again.
  wrote: number;
  // The workspace-relative path whose diff is on screen, "" when none is.
  current: string;
  onOpen: (path: string) => void;
  onChanged?: () => void;
  remote: boolean;
}

const join = (repo: string, path: string) => (repo ? `${repo}/${path}` : path);

// The Git panel: for each repository of the workspace, what changed as a tree to
// stage, discard and commit, the push of the branch, and its history. A workspace
// that is one repository shows just that; a folder of projects shows one section per
// repository, each acting on its own.
export function WorkbenchGit({ port, changes, repos, wrote, current, onOpen, onChanged, remote }: Props) {
  if (repos.length === 0) return null;
  const several = repos.length > 1 || repos[0].dir !== "";
  if (!several) {
    return <RepoSection port={port} repo="" changes={changes} current={current} onOpen={onOpen} onChanged={onChanged} remote={remote} />;
  }
  return (
    <div className="workbench-git" data-repos={repos.length}>
      {repos.map((row) => (
        <Folder key={row.dir} port={port} row={row} wrote={wrote} current={current} onOpen={onOpen} remote={remote} />
      ))}
    </div>
  );
}

function Folder({ port, row, wrote, current, onOpen, remote }: { port: AgentPort; row: GitRepoRow; wrote: number; current: string; onOpen: (path: string) => void; remote: boolean }) {
  const [open, setOpen] = useState(true);
  const [changes, setChanges] = useState<WorkspaceChange[]>([]);
  const [again, setAgain] = useState(0);
  useEffect(() => {
    let live = true;
    port.changes(row.dir).then((r) => live && setChanges(r.changes), () => live && setChanges([]));
    return () => { live = false; };
  }, [port, row.dir, wrote, again]);
  const reload = useCallback(() => setAgain((n) => n + 1), []);
  return (
    <section className="workbench-repo" data-repo={row.dir}>
      <button type="button" className="workbench-repo-head" data-action="workbench.repo" data-target={row.dir} aria-expanded={open} onClick={() => setOpen(!open)}>
        <StudioIcon name={open ? "down" : "chevron"} aria-hidden="true" />
        <b>{row.name || row.dir}</b>
        <StudioIcon name="branch" aria-hidden="true" />
        <span>{row.branch}</span>
        <small>{changes.length}</small>
      </button>
      {open && <RepoSection port={port} repo={row.dir} changes={changes} current={current} onOpen={onOpen} onChanged={reload} remote={remote} />}
    </section>
  );
}

function RepoSection({ port, repo, changes, current, onOpen, onChanged, remote }: {
  port: AgentPort; repo: string; changes: WorkspaceChange[]; current: string; onOpen: (path: string) => void; onChanged?: () => void; remote: boolean;
}) {
  const refresh = () => onChanged?.();
  const commit = useCommitCard(port, refresh, repo || undefined);
  const push = usePushCard(port, repo || undefined, refresh);
  const inside = repo && current.startsWith(repo + "/") ? current.slice(repo.length + 1) : repo ? "" : current;
  return (
    <>
      <WorkbenchChanges
        port={port}
        repo={repo || undefined}
        changes={changes}
        current={inside}
        onOpen={(path) => onOpen(join(repo, path))}
        onChanged={onChanged}
        commit={commit}
        actions={remote ? null : push.button}
        below={remote ? null : push.card}
      />
      {!remote && <WorkbenchHistory port={port} repo={repo || undefined} />}
    </>
  );
}
