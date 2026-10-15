// Over a wallpaper the tab strip sits above the conversation, whose own backdrop
// fades out on its other three sides. A strip that ends on a hard line reads as a
// seam. Whether the strip's lower edge fades is a property of the render, so it
// is read from the browser: the strip draws no bottom border over a wallpaper and
// carries a gradient below its own box.
import { chromium } from "playwright";

const BASE = process.env.PERF_URL ?? "http://localhost:4399/perf.html";
const fails = [];
const check = (name, ok, detail = "") => {
  console.log(`${ok ? "  ok" : "FAIL"}  ${name}${detail ? "  — " + detail : ""}`);
  if (!ok) fails.push(name);
};

const browser = await chromium.launch();
try {
  for (const scheme of ["light", "dark"]) {
    const ctx = await browser.newContext({ viewport: { width: 1300, height: 860 }, colorScheme: scheme, locale: "zh-CN" });
    const page = await ctx.newPage();
    await page.goto(`${BASE}?pack=photo&turns=3`, { waitUntil: "networkidle" });
    await page.waitForSelector(".tabs");
    const edge = await page.evaluate(() => {
      const tabs = document.querySelector(".tabs");
      const own = getComputedStyle(tabs);
      const fade = getComputedStyle(tabs, "::after");
      return { border: own.borderBottomColor, fadeHeight: parseFloat(fade.height) || 0, fadeImage: fade.backgroundImage };
    });
    check(`${scheme}: the tab strip draws no hard bottom line over a wallpaper`, edge.border === "rgba(0, 0, 0, 0)", edge.border);
    check(`${scheme}: it fades into the conversation below`, edge.fadeHeight >= 12 && edge.fadeImage.includes("linear-gradient"), `${edge.fadeHeight}px`);
    await ctx.close();
  }
} finally {
  await browser.close();
}
if (fails.length) {
  console.error(`\n${fails.length} failed: ${fails.join("; ")}`);
  process.exit(1);
}
console.log("\nall passed");
