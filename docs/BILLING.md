# Billing, display currency, and cost quotes

Reasonix keeps three facts separate:

1. `original`: an estimate from the selected public/custom rate card in its
   pricing-table currency. It is not an invoice or a provider debit.
2. `valuations`: occurrence-time `identity` and, when available, an
   `official_table` estimate for the same model in the other official region.
3. Wallet balances: the exact original-currency values returned by a provider.

Reasonix has no runtime FX download, cache, refresh goroutine, or wallet
conversion. Old `fx`/`rateSnapshot` fields remain readable for history only;
new quotes never generate them.

```toml
[billing]
display_currency = "auto"   # auto | CNY | USD

[[providers]]
billing_currency = "USD"    # pricing-table basis, not settlement currency
billing_mode = "payg"       # payg | subscription_equivalent
```

Legacy `[desktop].currency` remains readable and migrates to
`[billing].display_currency`. `auto` is intentionally unresolved in config:
one valid wallet currency may become a tab/session hint; otherwise a single
original currency is selected or mixed currencies are shown as buckets. A
language, browser locale, or host region never changes a rate card.

## CostQuote

`usage.costQuote` is the canonical host-side usage payload:

| Field | Meaning |
| --- | --- |
| `original` | Original-currency rate-card estimate |
| `originalTotals[]` | ISO-sorted original buckets for mixed aggregates |
| `valuations.*.basis` | `identity` or `official_table` for new quotes |
| `selected` | A single amount only when a display total exists |
| `costComplete` | Usage and pricing facts are complete |
| `displayComplete` | A requested single-currency total exists |
| `complete` | Compatibility alias mirroring `displayComplete` |
| `displayStatus` | `matched`, `fallback_original`, `bucketed`, or `unavailable` |
| `aggregateMode` | `single_currency`, `common_valuation`, or `currency_buckets` |

If a requested currency is unavailable but every original is the same, the
original amount is shown with `fallback_original`. Mixed originals produce
`originalTotals` and no scalar zero. `—` is reserved for missing usage or
pricing (`unavailable`). Legacy scalar aliases (`cost`, `costUsd`,
`total_cost`) are written only when `selected` exists.

## Price sources

A price you enter for a model can carry where it was read and when.

- `source_url` is an `http` or `https` link without credentials, control characters or blanks, at most 500 characters.
- `checked` is a calendar date, `YYYY-MM-DD`, not later than today in UTC.
- Both describe the price and take no part in quoting it; changing them does not rebuild a running conversation.
- The editor shows the host of the link and opens it only when you click it. Nothing fetches it, checks that it answers, or sends anything to it.
- Reasonix 1.x rewrites the shared config file without keys it does not know.
- The next start puts `source_url`, `checked` and `billing_mode` back from `provider-keys.json` beside the config.
- A price you changed does not inherit the old source, and a key you cleared in the editor is not put back.

## Wallets and diagnostics

Wallets are never converted or cross-added. An explicit target uses the exact
matching wallet; if it is absent, the real currency is shown with an ISO
prefix. Automatic mode uses a single valid wallet currency only as a runtime
hint. Multiple/unknown/error responses do not affect the cost facts.

```sh
reasonix doctor billing
reasonix doctor billing --json
```

The compatible `fx` report is always `enabled=false` and has no cache. The
report also lists the automatic selection policy, pricing-table currencies,
and official-catalog matches.
