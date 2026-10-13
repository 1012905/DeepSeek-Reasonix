import { StrictMode, createElement } from "react";
import { createRoot } from "react-dom/client";
import { boot as bootLang } from "./i18n";
import { track as trackWidth } from "./ui/viewport";
import "./styles/tokens.css";
import "./styles/app.css";
import "./styles/studio.css";
import "./styles/feedback.css";
import "./styles/community.css";
import { App } from "./ui/App";
import { MockHub } from "./port/mock_hub";

const q = new URLSearchParams(location.search);
const theme = q.get("theme");
if (theme) document.documentElement.dataset.theme = theme;
bootLang(); trackWidth();
(async () => {
  const w = window as any;
  const hub = new MockHub();
  const [rt] = await hub.runtimes();
  const pf = hub.portFor.bind(hub);
  (hub as any).portFor = (r: any) => { const p: any = pf(r); p.providerSetup = async () => null; p.welcomeSeen = async () => true; return p; };
  const port: any = hub.portFor(rt);
  let tabs = q.get("tabs") === "0" ? [] : [
    { id: "b1", target: "v1", title: "Reasonix 文档", url: "https://docs.reasonix.io", active: true },
    { id: "b2", target: "v2", title: "Vite 官方指南", url: "https://vite.dev/guide", active: false },
    { id: "b3", target: "v3", title: "localhost:5173", url: "http://localhost:5173", active: false },
  ];
  port.configProblem = async () => null;
  port.browserTabs = async () => tabs;
  const fail = q.get("fail");
  port.browserClose = async (id: string) => {
    if (fail && (fail === "all" || id === fail)) throw new Error("页面正忙，无法关闭");
    tabs = tabs.filter((t) => t.id !== id);
  };
  w.__port = port; w.__hub = hub;
  createRoot(document.getElementById("root")!).render(createElement(StrictMode, null, createElement(App, { hub })));
})();
