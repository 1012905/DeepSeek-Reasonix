import { HttpError, type Attachment, type DroppedRef } from "./port";
import { SseHttp } from "./sse_http";
import { SseTheme } from "./sse_theme";

// Where a file enters a turn: bytes the page holds, or paths the shell knows.
// The kernel's cap on one uploaded file (serve/attachments.go). A file past it is
// refused here, before it is read: turning a modpack into base64 is several
// copies of it in the renderer's memory, and the kernel would refuse it anyway.
export const MAX_ATTACHMENT_BYTES = 25 * 1024 * 1024;

export class SseIntake extends SseTheme {
  // JSON, not raw bytes: csrfGuard admits nothing else, and that guard is what
  // stops a cross-site form posting here at all.
  async attach(blob: Blob, name?: string) {
    if (blob.size > MAX_ATTACHMENT_BYTES) {
      const limit_mb = MAX_ATTACHMENT_BYTES >> 20;
      const detail = `attachment is larger than ${limit_mb} MB`;
      throw new HttpError(413, detail, { code: "attachment.too_large", error: detail, params: { limit_mb } });
    }
    let data: string;
    try {
      const buf = new Uint8Array(await blob.arrayBuffer());
      let bin = "";
      for (let i = 0; i < buf.length; i += 0x8000) bin += String.fromCharCode(...buf.subarray(i, i + 0x8000));
      data = btoa(bin);
    } catch (e) {
      const detail = String(e);
      throw new HttpError(0, detail, { code: "attachment.unreadable", error: detail, params: { detail } });
    }
    const res = await fetch(this.base + "/attachments", {
      method: "POST",
      headers: { "content-type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({ mime: blob.type, name: name ?? "", data }),
    });
    if (!res.ok) await SseHttp.fail("/attachments", res);
    return (await res.json()) as Attachment;
  }

  dropRefs(paths: string[]) {
    return this.post0<DroppedRef[]>("/drop", { paths });
  }
}
