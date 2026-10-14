// The transcript's two jump buttons: absent where nothing scrolls or within 48px
// of their end, present and clear of the composer elsewhere, taking no layout
// height, and landing on the end when no expanded block lies that way.
import { chromium } from "playwright";
import { mkdirSync } from "node:fs";
import { join } from "node:path";

const PAGE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?turns=60";
const SHOTS = process.env.SCROLLJUMP_SHOTS;
if (SHOTS) mkdirSync(SHOTS, { recursive: true });
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
for (const scheme of ["light", "dark"]) {
  const ctx = await browser.newContext({ locale: "zh-CN", viewport: { width: 1280, height: 800 }, colorScheme: scheme });
  const page = await ctx.newPage();
  page.on("pageerror", (e) => fails.push("page error: " + e.message));
  await page.goto(PAGE, { waitUntil: "domcontentloaded" });
  await page.waitForSelector('[data-pane="flow"] .flow');
  await page.waitForTimeout(1200);
  const root = page.locator('[data-pane="flow"]').first();
  const up = page.locator('.scrolljump button[data-value="up"]');
  const down = page.locator('.scrolljump button[data-value="down"]');
  const geo = () => root.evaluate((el) => ({ top: el.scrollTop, c: el.clientHeight, h: el.scrollHeight }));

  await root.evaluate((el) => el.scrollTo({ top: el.scrollHeight }));
  await page.waitForTimeout(400);
  check(`${scheme}: at the end only the up button is offered`, (await up.count()) === 1 && (await down.count()) === 0);

  await root.evaluate((el) => el.scrollTo({ top: 0 }));
  await page.waitForTimeout(400);
  check(`${scheme}: at the start only the down button is offered`, (await up.count()) === 0 && (await down.count()) === 1);

  await root.evaluate((el) => el.scrollTo({ top: el.scrollHeight * 0.4 }));
  await page.waitForTimeout(500);
  check(`${scheme}: mid-way both are offered`, (await up.count()) === 1 && (await down.count()) === 1);
  const box = await page.evaluate(() => {
    const host = document.querySelector(".scrolljump");
    const scroller = document.querySelector('[data-pane="flow"]').getBoundingClientRect();
    const compose = document.querySelector(".compose").getBoundingClientRect();
    const btns = [...host.querySelectorAll("button")].map((b) => b.getBoundingClientRect());
    return { host: host.getBoundingClientRect().height, scrollerBottom: scroller.bottom, composeTop: compose.top,
      right: Math.max(...btns.map((b) => b.right)), scrollerRight: scroller.right, bottom: Math.max(...btns.map((b) => b.bottom)) };
  });
  check(`${scheme}: the buttons take no height`, box.host === 0, JSON.stringify(box));
  check(`${scheme}: the buttons sit inside the transcript, above the composer`, box.bottom <= box.scrollerBottom + 0.5 && box.bottom <= box.composeTop + 0.5 && box.right <= box.scrollerRight);
  if (SHOTS) await page.screenshot({ path: join(SHOTS, `${scheme}-mid.png`) });

  await up.click();
  await page.waitForTimeout(500);
  let g = await geo();
  check(`${scheme}: up with nothing expanded lands on the start`, g.top < 60, `top=${g.top.toFixed(0)}`);
  await root.evaluate((el) => el.scrollTo({ top: el.scrollHeight * 0.4 }));
  await page.waitForTimeout(500);
  for (let n = 0; n < 8 && g.h - g.top - g.c >= 60; n++) {
    await down.click();
    await page.waitForTimeout(500);
    g = await geo();
  }
  check(`${scheme}: down reaches the end`, g.h - g.top - g.c < 60, `left=${(g.h - g.top - g.c).toFixed(0)}`);
  await ctx.close();
}
await browser.close();
if (fails.length) {
  console.error(`\n${fails.length} failed`);
  process.exit(1);
}
