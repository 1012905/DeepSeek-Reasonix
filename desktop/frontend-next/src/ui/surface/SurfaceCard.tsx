import { lazySurface } from "./a2ui/lazy";
import type { Item } from "../../state/session";
import { parseSurface } from "./spec";
import { ExtensionView } from "../cards/ExtensionView";
import { t } from "../../i18n";
import "../RestoreNotice.css";
import type { Tool } from "../../port/wire";
import { surfaceOfTool, surfaceFields } from "./call";
import "./SurfaceCard.css";

const NativeSurface = lazySurface(() => import("./a2ui/form"));

export function surfaceTool(tool: Tool, running: boolean) {
  const spec = surfaceOfTool(tool);
  if (spec && surfaceFields(spec).length) return null;
  if (spec && !running && tool.output !== undefined) return <ExtensionView body={spec.body} />;
  return undefined;
}

export interface SurfaceCardProps {
  item: Extract<Item, { t: "ask" }>;
  onAnswer: (itemId: string, id: string, answers: { questionId: string; selected: string[]; value?: Record<string, unknown> }[]) => Promise<void>;
}

export function SurfaceCard(props: SurfaceCardProps) {
  const spec = parseSurface(props.item.ask.questions[0]?.surface);
  if (!spec || props.item.surfaceExpired) return <div className="restore-notice" role="status" data-code="surface.expired">{t("界面已失效，请等待新的请求")}</div>;
  return <NativeSurface key={props.item.ask.id} {...props} spec={spec} />;
}
