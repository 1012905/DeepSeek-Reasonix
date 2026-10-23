// With the icon rail hidden, the sidebar's rows line up and read alike: every row's icon sits on one vertical line and every
// label starts on another, from the new-session button down to the wallet and feedback rows at the foot.
import { chromium } from "playwright";

const BASE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?pref=zh&ws=1&sess=1&turns=1";
const ROWS = [
  ["新建会话", ".studio-new-task"],
  ["搜索与快捷操作", ".studio-search"],
  ["文件", ".studio-quicknav > button:nth-child(1)"],
  ["工具与集成", ".studio-quicknav > button:nth-child(2)"],
  ["运行环境", ".studio-quicknav > button:nth-child(3)"],
  ["钱包与用量", ".studio-wallet:not(.studio-feedback)"],
  ["发送反馈", ".studio-feedback"],
];
const TOLERANCE = 1;
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
try {
  for (const scheme of ["light", "dark"]) {
    const ctx = await browser.newContext({ locale: "zh-CN", colorScheme: scheme, viewport: { width: 1280, height: 900 }, reducedMotion: "reduce" });
    await ctx.addInitScript(() => { try { localStorage.setItem("rx-nav-rail", "off"); } catch {} });
    const page = await ctx.newPage();
    await page.goto(BASE, { waitUntil: "networkidle" });
    await page.waitForSelector(".app");
    const seen = [];
    for (const [name, sel] of ROWS) {
      const at = await page.evaluate((s) => {
        const row = document.querySelector(s);
        if (!row) return null;
        const svg = row.querySelector("svg").getBoundingClientRect();
        const walk = document.createTreeWalker(row, NodeFilter.SHOW_TEXT);
        let text = null;
        for (let n = walk.nextNode(); n && !text; n = walk.nextNode()) {
          if (!n.textContent.trim()) continue;
          const range = document.createRange();
          range.selectNodeContents(n);
          text = range.getBoundingClientRect().x;
        }
        const label = [...row.querySelectorAll("*")].find((e) => e.children.length === 0 && e.textContent.trim() && !e.closest("kbd,small,.fbk-badge")) ?? row;
        const cs = getComputedStyle(label);
        return { icon: svg.x + svg.width / 2, text, look: `${cs.fontSize}/${cs.fontWeight}/${cs.color}`, fill: getComputedStyle(row).backgroundColor };
      }, sel);
      if (at) seen.push({ name, ...at });
    }
    const spread = (key) => Math.max(...seen.map((r) => r[key])) - Math.min(...seen.map((r) => r[key]));
    const show = (key) => seen.map((r) => `${r.name} ${r[key].toFixed(1)}`).join("; ");
    check(`${scheme}: all ${ROWS.length} rows are drawn`, seen.length === ROWS.length, `${seen.length}`);
    check(`${scheme}: the icons share one vertical line`, spread("icon") <= TOLERANCE, `spread ${spread("icon").toFixed(1)}px — ${show("icon")}`);
    check(`${scheme}: the labels start on one vertical line`, spread("text") <= TOLERANCE, `spread ${spread("text").toFixed(1)}px — ${show("text")}`);
    const looks = new Set(seen.map((r) => r.look));
    check(`${scheme}: every row's label has one size, weight and colour`, looks.size === 1, seen.map((r) => `${r.name} ${r.look}`).join("; "));
    const fill = seen.find((r) => r.name === "新建会话")?.fill ?? "";
    check(`${scheme}: New session has no fill at rest`, /rgba\(0, 0, 0, 0\)|transparent/.test(fill), fill);
    await ctx.close();
  }
} finally {
  await browser.close();
}
if (fails.length) {
  console.error(`\n${fails.length} failed: ${fails.join("; ")}`);
  process.exit(1);
}
