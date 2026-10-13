"use strict";

// Windows refuses a background process the foreground, and a focus() it refused
// only flashes the taskbar button. A window that is topmost for one call is
// raised past that rule, and handed back to the normal stacking order at once.
function raise(win, platform = process.platform) {
  win.show();
  win.focus();
  if (platform !== "win32" || win.isFocused()) return;
  win.setAlwaysOnTop(true);
  win.focus();
  win.setAlwaysOnTop(false);
}

module.exports = { raise };
