import { t } from "../i18n";
import type { ModelRate } from "../port/port";
import { shortId } from "./ModelLimits";

export interface PriceText {
  inp: string;
  out: string;
  hit: string;
  cur: string;
}

const CURRENCIES = ["CNY", "USD"] as const;
const blank = (cur = "CNY"): PriceText => ({ inp: "", out: "", hit: "", cur });
const typed = (n: number) => String(n);

export function priceTextOf(own: ModelRate | undefined): PriceText {
  return own ? { inp: typed(own.input), out: typed(own.output), hit: own.cacheHit === own.input ? "" : typed(own.cacheHit), cur: own.currency || "CNY" } : blank();
}

// Digits and at most one point: a price is not an integer, and a sign or an
// exponent is how a typo becomes a rate nobody meant.
export const decimal = (s: string) => {
  const cleaned = s.replace(/[^\d.]/g, "");
  const at = cleaned.indexOf(".");
  return at < 0 ? cleaned : cleaned.slice(0, at + 1) + cleaned.slice(at + 1).replace(/\./g, "");
};

const parsed = (s: string) => (s === "" || s === "." ? null : Number(s));

// A row is a price once input and output are both typed; one of them alone is a
// half-finished row, which is neither a price nor an empty one.
export const incomplete = (row: PriceText | undefined) => !!row && (row.inp === "") !== (row.out === "");

// An empty row clears the model's own price; an incomplete one changes nothing.
// A blank cached-input price is the input price: no discount is assumed.
export function pricesToSend(picked: string[], text: Record<string, PriceText>, stored: Record<string, ModelRate>): Record<string, ModelRate | null> {
  const out: Record<string, ModelRate | null> = {};
  for (const model of picked) {
    const row = text[model] ?? blank();
    if (incomplete(row)) {
      if (stored[model]) out[model] = stored[model];
      continue;
    }
    const input = parsed(row.inp);
    const output = parsed(row.out);
    if (input === null || output === null) {
      if (stored[model]) out[model] = null;
      continue;
    }
    out[model] = { input, output, cacheHit: parsed(row.hit) ?? input, currency: row.cur };
  }
  return out;
}

export function ModelPrices({
  models, value, onChange, official = false, stored = {},
}: {
  models: string[];
  value: Record<string, PriceText>;
  onChange: (next: Record<string, PriceText>) => void;
  official?: boolean;
  stored?: Record<string, ModelRate>;
}) {
  const set = (model: string, patch: Partial<PriceText>) =>
    onChange({ ...value, [model]: { ...(value[model] ?? blank()), ...patch } });
  return (
    <div className="ml-group mp-group" role="group" aria-label={t("按模型设置价格")}>
      <div className="ml-head">
        <strong>{t("按模型设置价格")}</strong>
        <span>{t("每百万 token 的单价。留空表示未知，用量页会如实显示「未知」，而不是 0；填 0 表示免费。缓存命中价留空时按输入价计算。")}</span>
      </div>
      <ul className="me-list ml-list">
        {models.map((model) => {
          const cur = value[model] ?? blank();
          const own = cur.inp !== "" || cur.out !== "";
          const foreign = official && own && stored[model] !== undefined && stored[model]!.official === false;
          const freeCached = foreign && cur.hit !== "" && Number(cur.hit) === 0 && Number(cur.inp) > 0;
          return (
            <li key={model} className="ml-row mp-row" data-model={model} data-own={own || undefined}>
              <span className="me-name" title={model}>{shortId(model)}</span>
              <span className="unit-field">
                <input data-action="provider.draft" data-value="model-price-input" aria-label={t("{model} 的输入单价", { model })}
                  aria-invalid={incomplete(cur) && cur.inp === "" ? true : undefined} inputMode="decimal" value={cur.inp} placeholder={t("未知")}
                  onChange={(e) => set(model, { inp: decimal(e.target.value) })} /><i>{t("输入")}</i>
              </span>
              <span className="unit-field">
                <input data-action="provider.draft" data-value="model-price-output" aria-label={t("{model} 的输出单价", { model })}
                  aria-invalid={incomplete(cur) && cur.out === "" ? true : undefined} inputMode="decimal" value={cur.out} placeholder={t("未知")}
                  onChange={(e) => set(model, { out: decimal(e.target.value) })} /><i>{t("输出")}</i>
              </span>
              <span className="unit-field">
                <input data-action="provider.draft" data-value="model-price-cache" aria-label={t("{model} 的缓存命中单价", { model })}
                  inputMode="decimal" value={cur.hit} placeholder={t("同输入价")}
                  onChange={(e) => set(model, { hit: decimal(e.target.value) })} /><i>{t("缓存命中")}</i>
              </span>
              <select data-action="provider.draft" data-value="model-price-currency" aria-label={t("{model} 的货币", { model })} value={cur.cur}
                onChange={(e) => set(model, { cur: e.target.value })}>
                {CURRENCIES.map((c) => <option key={c} value={c}>{c}</option>)}
              </select>
              {incomplete(cur) && <span className="why">{t("输入价和输出价需要都填写，才会保存这一行。")}</span>}
              {foreign && (
                <span className="why mp-foreign">
                  <span>{t("这个价格不在官方价目表里，选了人民币或美元显示时无法换算。")}</span>
                  {freeCached && <span>{t("缓存命中价是 0：命中的 token 会按免费计，费用会偏低。")}</span>}
                  <button type="button" className="act" data-action="provider.draft" data-value="model-price-restore" aria-label={t("{model} 恢复官方价", { model })}
                    onClick={() => set(model, { inp: "", out: "", hit: "" })}>{t("恢复官方价")}</button>
                </span>
              )}
            </li>
          );
        })}
      </ul>
    </div>
  );
}
