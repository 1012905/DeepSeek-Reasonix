// Session rows carry hover Archive and Delete buttons only when this machine turned them on. The reveal is a
// width and opacity transition on :hover, so whether the buttons are really visible and hit-testable is
// a question for a real browser.
import { chromium } from "playwright";

const BASE = process.env.PERF_URL ?? "http://localhost:4399/perf.html";
const SHOTS = process.env.SHOTS;
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
try {
  for (const scheme of ["light", "dark"]) {
    for (const on of [false, true]) {
      const ctx = await browser.newContext({ locale: "zh-CN", colorScheme: scheme, viewport: { width: 1400, height: 800 } });
      if (on) await ctx.addInitScript(() => localStorage.setItem("rx-session-quick-actions", "on"));
      const page = await ctx.newPage();
      await page.goto(BASE, { waitUntil: "networkidle" });
      await page.waitForSelector(".sessrow", { timeout: 20000 });
      const row = page.locator(".sessrow").first();
      await row.scrollIntoViewIfNeeded();
      const box = await row.boundingBox();
      await page.mouse.move(box.x + 40, box.y + box.height / 2);
      await page.waitForTimeout(300);
      const quick = page.locator(".sessrow").first().locator(".sessquick");
      if (!on) {
        check(`${scheme}: off draws no quick buttons on a hovered row`, (await quick.count()) === 0);
      } else {
        const state = await quick.evaluate((el) => {
          const r = el.getBoundingClientRect();
          const [archive, trash] = el.querySelectorAll("button");
          const hit = (b) => {
            const q = b.getBoundingClientRect();
            const at = document.elementFromPoint(q.left + q.width / 2, q.top + q.height / 2);
            return at === b || b.contains(at);
          };
          return { opacity: Number(getComputedStyle(el).opacity), width: r.width, archive: hit(archive), trash: hit(trash) };
        });
        check(`${scheme}: on reveals both buttons on a hovered row`, state.opacity === 1 && state.width > 40 && state.archive && state.trash, JSON.stringify(state));
      }
      if (SHOTS) await page.screenshot({ path: `${SHOTS}/sessquick-${on ? "on" : "off"}-${scheme}.png`, clip: { x: 0, y: 0, width: 420, height: 800 } });
      await ctx.close();
    }
  }
} finally {
  await browser.close();
}
if (fails.length) {
  console.error(`\n${fails.length} failed: ${fails.join("; ")}`);
  process.exit(1);
}
console.log("\nall passed");
