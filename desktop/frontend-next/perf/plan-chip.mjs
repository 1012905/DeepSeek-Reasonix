// The plan chip sits at the right edge of the tab row, beside the side-question
// toggle. Two auto margins in one flex row split the free space and leave the
// chip floating in the middle; the real stylesheet and real geometry decide it.
import { chromium } from "playwright";

const BASE = process.env.PERF_URL ?? "http://localhost:4399/perf.html";
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
for (const scheme of ["light", "dark"]) {
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 }, colorScheme: scheme });
  page.on("pageerror", (e) => fails.push("page error: " + e.message));
  await page.goto(`${BASE}?pref=zh&turns=3`, { waitUntil: "networkidle" });
  await page.waitForSelector(".pane > .tabs .sidetoggle");
  await page.evaluate(() => {
    const toggle = document.querySelector(".pane > .tabs .sidetoggle");
    const chip = document.createElement("div");
    chip.className = "plan-peek";
    chip.innerHTML = '<button type="button" class="tab plan-peek-trigger">计划 <small>0/12</small></button>';
    toggle.before(chip);
  });
  if (process.env.BEFORE) await page.addStyleTag({ content: ".plan-peek + .sidetoggle { margin-left: auto }" });
  await page.waitForTimeout(200);
  const g = await page.evaluate(() => {
    const chip = document.querySelector(".pane > .tabs .plan-peek").getBoundingClientRect();
    const toggle = document.querySelector(".pane > .tabs .sidetoggle").getBoundingClientRect();
    const row = document.querySelector(".pane > .tabs").getBoundingClientRect();
    return { gap: Math.round(toggle.left - chip.right), chipRight: Math.round(chip.right), rowRight: Math.round(row.right) };
  });
  check(`${scheme}: the plan chip sits against the side toggle, not mid-row`, g.gap >= 0 && g.gap <= 16, JSON.stringify(g));
  if (process.env.SHOT) await page.locator(".pane > .tabs").screenshot({ path: `${process.env.SHOT}-${scheme}.png` });
  await page.close();
}
await browser.close();
if (fails.length) {
  console.log(`\n${fails.length} failed`);
  process.exit(1);
}
console.log("\nall ok");
