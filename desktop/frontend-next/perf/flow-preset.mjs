// A line-width preset still works on a column a hand has already dragged: the
// dragged width outranks the stylesheet, so choosing a preset has to drop it.
import assert from "node:assert/strict";
import { chromium } from "playwright";

const PAGE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?pref=zh&turns=3";
const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1900, height: 900 } });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(PAGE, { waitUntil: "networkidle" });
  await page.waitForSelector(".flowgrip-l");
  const flow = () => page.locator(".flow").first().evaluate((el) => Math.round(el.getBoundingClientRect().width));
  const readable = await flow();
  const box = await page.locator(".flowgrip-l").boundingBox();
  await page.mouse.move(box.x + 3, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + 3 - 60, box.y + box.height / 2, { steps: 6 });
  await page.mouse.up();
  await page.waitForTimeout(200);
  const dragged = await flow();
  assert.ok(dragged > readable, `the hand widened the column (${readable} -> ${dragged})`);

  const choose = async (name) => {
    await page.evaluate(() => document.querySelector('[data-action="chrome.settings"]')?.click());
    await page.waitForSelector(".prefs-nav button");
    await page.locator(".prefs-nav button", { hasText: "外观" }).first().click();
    await page.locator('[data-action="appearance.measure"]', { hasText: new RegExp(`^${name}$`) }).click();
    await page.keyboard.press("Escape");
    await page.waitForTimeout(400);
    return flow();
  };
  const full = await choose("铺满窗口");
  assert.ok(full > dragged, `the full-width preset widens a dragged column (${dragged} -> ${full})`);
  const standard = await choose("适宜阅读");
  assert.equal(standard, readable, `the reading preset returns to the readable measure (${full} -> ${standard})`);
  assert.deepEqual(errors, []);
  console.log(`PASS a preset overrides a dragged width: ${readable} -> ${dragged} (dragged) -> ${full} (full) -> ${standard} (reading)`);
} finally {
  await browser.close();
}
