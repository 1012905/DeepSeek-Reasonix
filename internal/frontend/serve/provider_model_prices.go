// provider_model_prices.go — the price per million tokens a connection declares
// for each of its models, so a model the price book does not know can be quoted.
package serve

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"reasonix/internal/contract/config"
	"reasonix/internal/contract/pricing"
	"reasonix/internal/contract/provider"
)

// modelPriceView is one model's own price. A model with none stays out of the
// map: an absent entry is "unknown", which is not the same answer as zero.
type modelPriceView struct {
	Input    float64 `json:"input"`
	Output   float64 `json:"output"`
	CacheHit float64 `json:"cacheHit"`
	Currency string  `json:"currency,omitempty"`
}

func modelPricesOf(p *config.ProviderEntry) map[string]modelPriceView {
	out := map[string]modelPriceView{}
	for _, model := range p.ChatModelList() {
		if price := p.Prices[model]; price != nil {
			out[model] = modelPriceView{Input: price.Input, Output: price.Output, CacheHit: price.CacheHit, Currency: pricing.NormalizeCurrency(price.Currency)}
		}
	}
	return orNilMap(out)
}

// applyModelPrices makes the form's answer the whole answer for the models it
// lists: one sent with a price declares it, one sent null or left out has none of
// its own again. Models the form does not list keep what they have.
func applyModelPrices(entry *config.ProviderEntry, models []string, sent *map[string]*modelPriceView) *editRefusal {
	if sent == nil {
		return nil
	}
	for model := range *sent {
		if !slices.Contains(models, strings.TrimSpace(model)) {
			return &editRefusal{"provider.model_prices_unlisted",
				fmt.Sprintf("%q is not one of the selected models", model), map[string]any{"model": model}}
		}
	}
	for _, model := range models {
		var want *modelPriceView
		for k, v := range *sent {
			if strings.TrimSpace(k) == model {
				want = v
			}
		}
		var price *provider.Pricing
		if want != nil {
			price = &provider.Pricing{Input: want.Input, Output: want.Output, CacheHit: want.CacheHit, Currency: want.Currency}
		}
		if err := entry.SetModelPrice(model, price); errors.Is(err, config.ErrModelPriceInvalid) {
			return &editRefusal{"provider.bad_model_price",
				fmt.Sprintf("the price for %s must be a non-negative number in a known currency", model), map[string]any{"model": model}}
		}
	}
	return nil
}

// applyModelDeclarations applies what the form declares per model beyond the
// connection's own values: limits, then prices, each with the whole-answer rule
// for the models it lists.
func applyModelDeclarations(entry *config.ProviderEntry, models []string, limits *map[string]modelLimitsView, prices *map[string]*modelPriceView) *editRefusal {
	if bad := applyModelLimits(entry, models, limits); bad != nil {
		return bad
	}
	return applyModelPrices(entry, models, prices)
}
