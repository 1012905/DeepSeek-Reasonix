import { useCallback, useEffect, useState } from "react";
import { t } from "../i18n";
import { reason } from "../i18n/kernel";
import { HttpError } from "../port/http_error";
import type { AgentPort, WebAccessPatch, WebAccessSettings } from "../port/port";
import { Group } from "./Group";
import { Switch } from "./Switch";

type Kind = "search" | "fetch";

const NAME: Record<Kind, Record<string, string>> = {
  search: { searxng: "SearXNG", deepseek_api: "DeepSeek 官方搜索" },
  fetch: { local: "本机抓取", deepseek_api: "DeepSeek 官方抓取" },
};
const NOTE: Record<Kind, Record<string, string>> = {
  search: { searxng: "你自己部署的搜索服务，免费，需要填它的地址。", deepseek_api: "DeepSeek 官方的搜索接口，按次计费。" },
  fetch: { local: "由本机直接下载网页，免费。", deepseek_api: "DeepSeek 官方的网页抓取接口，按次计费。" },
};
const said = (kind: Kind, id: string) => (NAME[kind][id] ? t(NAME[kind][id]) : id);

/** Where the agent looks things up and reads pages from: an ordered list per
 *  kind, free backends first and a paid one only after those fail. Consent to be
 *  billed is its own switch, written only by the person's click and withdrawn the
 *  same way; nothing here ever ticks it. */
export function WebAccess({ port }: { port: AgentPort }) {
  const [state, setState] = useState<WebAccessSettings | null>(null);
  const [url, setUrl] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // A window the server has not granted editing to is read-only; the first refused
  // save is how that is learned, and nothing here asks again.
  const [readOnly, setReadOnly] = useState(false);

  useEffect(() => {
    let alive = true;
    port.webAccessSettings().then((s) => {
      if (!alive) return;
      setState(s);
      setUrl(s.searxng_url);
    }).catch(() => alive && setState(null));
    return () => {
      alive = false;
    };
  }, [port]);

  const apply = useCallback(async (patch: WebAccessPatch) => {
    setBusy(true);
    setError("");
    try {
      const next = await port.saveWebAccessSettings(patch);
      setState(next);
      setUrl(next.searxng_url);
    } catch (e) {
      if (e instanceof HttpError && e.reason?.code === "web_access.editing_disabled") setReadOnly(true);
      setError(reason(e));
    } finally {
      setBusy(false);
    }
  }, [port]);

  if (!state) return null;
  const locked = busy || readOnly;

  const section = (kind: Kind) => {
    const known = kind === "search" ? state.known_search_backends : state.known_fetch_backends;
    const on = kind === "search" ? state.search_backends : state.fetch_backends;
    const allowed = kind === "search" ? state.search_paid_allowed : state.fetch_paid_allowed;
    const order = (next: string[]) => apply(kind === "search" ? { search_backends: next } : { fetch_backends: next });
    const consent = (next: string[]) => apply(kind === "search" ? { search_paid_allowed: next } : { fetch_paid_allowed: next });
    const rows = [...on.filter((id) => known.includes(id)), ...known.filter((id) => !on.includes(id))];
    const title = kind === "search" ? t("网页搜索") : t("网页抓取");
    return (
      <div className="wa-sec" role="group" aria-label={title}>
        <div className="peernote"><b>{title}</b>{" "}{kind === "search" ? t("按顺序试：先用前面的免费后端，都失败了才会用付费的。") : t("按顺序试：先用前面的免费后端，都失败了才会用付费的。")}</div>
        {on.length === 0 && <div className="peernote">{kind === "search" ? t("没有启用任何搜索后端：沿用模型自带的搜索。") : t("没有启用任何抓取后端：只用本机抓取。")}</div>}
        <ul className="wa-list">
          {rows.map((id) => {
            const idx = on.indexOf(id);
            const enabled = idx >= 0;
            const paid = state.paid_backends.includes(id);
            return (
              <li key={id}>
                <span className="tx"><span className="lb">{said(kind, id)}</span><span className="ds">{NOTE[kind][id] ? t(NOTE[kind][id]) : ""}</span></span>
                {enabled && (
                  <span className="acts">
                    <button className="act" data-action="web-access.move" data-value="up" disabled={locked || idx === 0} aria-label={t("上移 {name}", { name: said(kind, id) })}
                      onClick={() => void order(on.map((x, i) => (i === idx - 1 ? id : i === idx ? on[idx - 1] : x)))}>↑</button>
                    <button className="act" data-action="web-access.move" data-value="down" disabled={locked || idx === on.length - 1} aria-label={t("下移 {name}", { name: said(kind, id) })}
                      onClick={() => void order(on.map((x, i) => (i === idx + 1 ? id : i === idx ? on[idx + 1] : x)))}>↓</button>
                  </span>
                )}
                <Switch data-action="web-access.backend" data-value={id} on={enabled} busy={locked}
                  label={t("启用 {name}", { name: said(kind, id) })}
                  onClick={() => void order(enabled ? on.filter((x) => x !== id) : [...on, id])} />
                {paid && (enabled || allowed.includes(id)) && (
                  <div className="wa-consent">
                    <span className="tx">
                      <span className="lb">{kind === "search" ? t("同意网页搜索按次计费") : t("同意网页抓取按次计费")}</span>
                      <span className="ds">{state.deepseek_api_ready || allowed.includes(id) ? t("同意后，前面的后端都失败时才会调用它，费用记在你的 DeepSeek 账户上。不勾选就不会用它。") : t("需要一个带密钥的 DeepSeek 提供方，才能使用并同意这一项。")}</span>
                    </span>
                    <Switch data-action="web-access.consent" data-value={`${kind}:${id}`} on={allowed.includes(id)} busy={locked || (!state.deepseek_api_ready && !allowed.includes(id))}
                      label={kind === "search" ? t("同意网页搜索按次计费") : t("同意网页抓取按次计费")}
                      onClick={() => void consent(allowed.includes(id) ? allowed.filter((x) => x !== id) : [...allowed, id])} />
                  </div>
                )}
              </li>
            );
          })}
        </ul>
        {kind === "search" && on.includes("searxng") && (
          <div className="wa-url">
            <label className="peernote" htmlFor="searxng-url">{t("SearXNG 地址")}</label>
            <input id="searxng-url" className="peerfield" data-action-change="web-access.searxng-url" aria-label={t("SearXNG 地址")} placeholder="http://127.0.0.1:8080"
              value={url} disabled={locked} onChange={(e) => setUrl(e.target.value)}
              onBlur={() => { if (url.trim() !== state.searxng_url) void apply({ searxng_url: url.trim() }); }} />
          </div>
        )}
      </div>
    );
  };

  return (
    <Group id="web-access" title={t("联网：搜索与抓取")} hint={t("agent 查资料和读网页时用哪些后端。免费的先试，付费的只有你明确同意、并且前面的都失败时才会用。")}>
      <div className="box">
        {section("search")}
        {section("fetch")}
        <div className="wa-sec" role="group" aria-label={t("视频地址")}>
          <div className="peernote"><b>{t("视频地址")}</b>{" "}{t("模型可以把你在对话里写的公开视频地址交给模型厂商读取。默认关闭，立即生效。")}</div>
          <ul className="wa-list">
            <li>
              <span className="tx">
                <span className="lb">{t("把视频地址交给模型厂商")}</span>
                <span className="ds">{t("开启后，该网址会被厂商直接访问：由厂商的服务器去打开它，不是本机下载。只有你自己写在对话里的地址才会交出去，且所选模型的厂商要支持。")}</span>
              </span>
              <Switch data-action="web-access.video-url" on={state.video_url_passthrough} busy={locked} label={t("把视频地址交给模型厂商")}
                onClick={() => void apply({ video_url_passthrough: !state.video_url_passthrough })} />
            </li>
          </ul>
        </div>
        <div className="peernote">{t("保存后只对之后新开的窗格生效，已开的窗格沿用旧设置。这些设置写在用户配置文件里：")}<code>{state.path}</code></div>
        {readOnly && <div className="peernote" role="status">{t("这些设置现在是只读的：这个窗口没有被授予编辑权限。")}</div>}
        {error && <div className="why" role="alert">{error}</div>}
      </div>
    </Group>
  );
}
