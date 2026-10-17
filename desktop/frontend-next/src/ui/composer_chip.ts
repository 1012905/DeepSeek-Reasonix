import type { Attachment } from "../port/port";
import type { PasteChip } from "./pastekeep";

export type Chip =
  | {
      k: "attachment";
      id: string;
      name: string;
      state: "adding" | "ready" | "failed";
      a?: Attachment;
      blob?: File;
      url?: string;
      error?: string;
    }
  | PasteChip
  | { k: "quote"; id: string; body: string; turn?: number; source?: string; lines: number };
