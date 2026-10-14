// Opening the branch menu while the pointer is still over the branch chip must not leave the hover card showing:
// the two surfaces would stack their shadows. Visibility under :hover is a styling question, so it is asked of a real browser.
import { chromium } from "playwright";

const BASE = process.env.PERF_URL ?? "http://localhost:4399/perf.html?pref=zh&turns=2";
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
try {
  const page = await (await browser.newContext({ locale: "zh-CN", viewport: { width: 1280, height: 800 } })).newPage();
  await page.goto(BASE, { waitUntil: "networkidle" });
  await page.waitForSelector(".app", { timeout: 20000 });
  const chip = page.locator('.studio-branch-pop .studio-branch').first();
  await chip.waitFor();
  const card = page.locator(".studio-branch-card").first();
  const shown = () => card.evaluate((el) => getComputedStyle(el).visibility === "visible" && Number(getComputedStyle(el).opacity) > 0.5);
  await chip.hover();
  await page.waitForTimeout(400);
  check("the hover card shows over a closed chip", await shown());
  await chip.click();
  await page.waitForTimeout(500);
  check("the branch menu opened", (await chip.getAttribute("aria-expanded")) === "true");
  check("the hover card is gone while the menu is open", !(await shown()));
} finally {
  await browser.close();
}
if (fails.length) {
  console.error(`\n${fails.length} failed: ${fails.join("; ")}`);
  process.exit(1);
}
console.log("\nall passed");
