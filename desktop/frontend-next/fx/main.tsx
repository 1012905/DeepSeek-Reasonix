import "../src/styles/tokens.css";
import "../src/styles/app.css";
import "../src/styles/composition-picker.css";
import "../src/styles/dock.css";
import "../src/styles/studio.css";
import "../src/styles/dialogcenter.css";
import "../src/styles/stophint.css";
import "../src/styles/phonecomposer.css";
import "../src/styles/tips.css";
import "../src/styles/skillpin.css";
import "../src/styles/waitnote.css";
import "../src/styles/bring.css";
import "../src/styles/peerthreads.css";
import "../src/styles/line.css";
import "../src/styles/notice.css";
import "../src/styles/peers.css";
import "../src/styles/contentmatches.css";
import "../src/styles/jsontext.css";
import "../src/styles/narrow.css";
import "../src/styles/workfold.css";
import "../src/styles/workbench-launcher.css";
import "../src/styles/feedback.css";
import "../src/styles/community.css";
import "../src/styles/themes.css";
import "../src/styles/sponsor.css";
import "../src/styles/chart.css";
import "../src/styles/versions.css";
import "../src/styles/changedpill.css";
import "../src/styles/askskip.css";
import "../src/styles/pastekeep.css";
import "../src/styles/ctxitems.css";
import "../src/styles/sidepanel.css";
import "../src/styles/scrolljump.css";
import "../src/styles/sessionnotes.css";
import "../src/styles/providerpresets.css";
import "../src/styles/sidebardrop.css";
import "../src/styles/sessionquick.css";
import "../src/styles/sessionpin.css";
import "../src/styles/treerows.css";
import "../src/styles/workbench-changes.css";
import "../src/styles/gitgraph.css";
import "../src/styles/modelmenu.css";
import "../src/styles/composer-menu.css";
import "../src/styles/runsignals.css";
import "../src/styles/toasts.css";
import "../src/styles/overview.css";
import { createRoot } from "react-dom/client";
import { boot as bootLang } from "../src/i18n";
import { MockPort } from "../src/port/mock";
import { AgentCompositionPicker } from "../src/ui/AgentCompositionPicker";
import { RunLine } from "../src/ui/RunLine";

localStorage.setItem("rx-show-agent-composition", "on");
const q = new URLSearchParams(location.search);
const live = q.get("live") === "1";
document.documentElement.dataset.theme = q.get("theme") ?? "dark";
const port = new MockPort();
(port as any).agentComposition = async () => ({ id: "builtin:minimal", name: "轻装搭档", revision: "fixture", digest: "fixture", started: true, ...(live ? { formatVersion: 2, resourceMode: "live" } : {}) });
bootLang?.();
createRoot(document.getElementById("root")!).render(
  <div style={{ padding: 24, maxWidth: 760 }}>
    <RunLine label="空闲" running={false} blocked={false} sent={0} received={0} estimated={false}>
      <AgentCompositionPicker port={port} locked running={false} onChanged={() => {}} onSettings={() => {}} onError={() => {}} />
    </RunLine>
  </div>,
);
