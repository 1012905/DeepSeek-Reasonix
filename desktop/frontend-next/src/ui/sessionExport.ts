import { download } from "../port/download";
import { host } from "../port/host";
import type { HubPort, TreeSession } from "../port/hub";

export async function saveSessionFile(hub: Pick<HubPort, "exportSession">, session: TreeSession) {
  const exported = await hub.exportSession(session.path);
  const records = exported.content
    .split(/\r?\n/)
    .filter((line) => line.trim())
    .map((line) => JSON.parse(line) as unknown);
  const content = JSON.stringify(
    { version: 1, title: session.title || session.name, exportedAt: new Date().toISOString(), records },
    null,
    2,
  );
  const clean = (session.title || exported.name || session.name).replace(/[<>:"/\\|?*\x00-\x1f]/g, "-").trim() || "reasonix-session";
  const filename = `${clean}.json`;
  const saved = await host().saveText(filename, content);
  if (saved === null) download(filename, content);
}
