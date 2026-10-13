"use strict";

function setUnreadBadge(unread, win, electron = require("electron"), platform = process.platform) {
  if (platform === "darwin") {
    electron.app.dock.setBadge(unread ? "•" : "");
  } else if (platform === "win32") {
    let image = null;
    if (unread) {
      const pixels = Buffer.alloc(16 * 16 * 4);
      for (let y = 0; y < 16; y++) {
        for (let x = 0; x < 16; x++) {
          if ((x - 7.5) ** 2 + (y - 7.5) ** 2 > 49) continue;
          // Native bitmaps use BGRA, including on Windows ARM.
          pixels.set([64, 64, 224, 255], (y * 16 + x) * 4);
        }
      }
      image = electron.nativeImage.createFromBitmap(pixels, { width: 16, height: 16 });
    }
    win.setOverlayIcon(image, unread ? "Reasonix Studio" : "");
  } else {
    electron.app.setBadgeCount(unread ? 1 : 0);
  }
}

module.exports = { setUnreadBadge };
