// The side question opens from the button in the composer, above the composer
// and inside the conversation's own column — never over what sits beside or
// above it, whether the file panel is open or not and at any width.
import { chromium } from "playwright";

const PAGE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?pref=zh&ws=2&sess=6&turns=3";
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};
const shotDir = process.env.SHOTS;
const box = (el) => (el ? (({ left, top, right, bottom }) => ({ left, top, right, bottom }))(el.getBoundingClientRect()) : null);

const browser = await chromium.launch();
for (const scheme of ["light", "dark"]) {
  for (const [width, files] of [[1400, false], [1400, true], [760, false], [760, true]]) {
    const label = `${scheme} ${width}px${files ? " with the file panel" : ""}`;
    const page = await browser.newPage({ viewport: { width, height: 800 }, locale: "zh-CN", colorScheme: scheme });
    page.on("pageerror", (e) => fails.push("页面异常: " + e.message));
    await page.goto(PAGE, { waitUntil: "networkidle" });
    await page.waitForSelector(".compose");
    if (files) {
      await page.locator('.navbtn[aria-label="文件"]').first().click();
      await page.waitForSelector(".workbench-filelist button");
    }
    await page.locator('[data-action="side.toggle"]').click();
    await page.waitForSelector(".sidepanel");
    await page.waitForTimeout(250);
    const r = await page.evaluate(() => {
      const box = (el) => (el ? (({ left, top, right, bottom }) => ({ left, top, right, bottom }))(el.getBoundingClientRect()) : null);
      const panel = document.querySelector(".sidepanel");
      const compose = panel.closest(".compose");
      const pane = compose.closest(".pane");
      const beside = [...document.querySelectorAll(".workbench-filelist, .workbench-explorer-head")].filter((el) => !pane.contains(el)).map(box);
      return { panel: box(panel), compose: box(compose), pane: box(pane), beside, vw: innerWidth, vh: innerHeight, button: box(document.querySelector('[data-action="side.toggle"]')) };
    });
    const p = r.panel;
    check(`${label}: it sits above the composer`, p.bottom <= r.compose.top + 1 && r.compose.top - p.bottom < 24, `bottom ${Math.round(p.bottom)}, composer top ${Math.round(r.compose.top)}`);
    check(`${label}: it stays inside its pane`, p.left >= r.pane.left - 1 && p.right <= r.pane.right + 1 && p.top >= r.pane.top - 1, `panel ${Math.round(p.left)}–${Math.round(p.right)}, pane ${Math.round(r.pane.left)}–${Math.round(r.pane.right)}`);
    check(`${label}: it stays inside the window`, p.left >= 0 && p.right <= r.vw && p.top >= 0 && p.bottom <= r.vh);
    check(`${label}: it does not cover the file panel`, r.beside.every((b) => p.right <= b.left || p.left >= b.right || p.bottom <= b.top || p.top >= b.bottom), files ? `${r.beside.length} file panel parts` : "no file panel");
    check(`${label}: it opens above the button that asked`, p.bottom <= r.button.top && p.left <= r.button.left + 1, `button ${Math.round(r.button.left)}`);
    if (shotDir && !(width === 760 && files)) await page.screenshot({ path: `${shotDir}/${scheme}-${width}${files ? "-files" : ""}.png` });
    await page.close();
  }
}
await browser.close();
if (fails.length) {
  console.log(`\n${fails.length} failed`);
  process.exit(1);
}
