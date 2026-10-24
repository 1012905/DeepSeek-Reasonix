import assert from "node:assert/strict";
import { chromium } from "playwright";

const base = process.env.PERF_URL ?? "http://localhost:4399/perf.html?pref=zh&turns=0";
const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1600, height: 860 } });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(base, { waitUntil: "networkidle" });
  await page.waitForSelector(".flowgrip-l");
  const width = () => page.locator(".compose").evaluate((el) => Math.round(el.getBoundingClientRect().width));
  const start = await width();
  const drag = async (dx) => {
    const box = await page.locator(".flowgrip-l").boundingBox();
    await page.mouse.move(box.x + 3, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + 3 + dx, box.y + box.height / 2, { steps: 8 });
    await page.mouse.up();
    await page.waitForTimeout(200);
    return width();
  };
  const mid = await drag(40);
  assert.ok(mid < start && mid >= 640, `a drag inward goes under the readable measure (${start} -> ${mid})`);
  const floor = await drag(400);
  assert.equal(floor, 640, `the column stops at 640 (${floor})`);
  const slider = page.locator(".flowgrip-l");
  assert.equal(await slider.getAttribute("aria-valuemin"), "640");
  await slider.focus();
  await page.keyboard.press("Home");
  await page.waitForTimeout(200);
  assert.equal(await width(), start, "Home returns to the readable measure");
  assert.deepEqual(errors, []);
  console.log(`PASS the column is continuous from 640 up: ${start} -> ${mid} -> ${floor} -> ${start}`);
} finally {
  await browser.close();
}
