// model_price.go — a price the user declares for one model on a connection, per
// million tokens. It is stored as the model's entry in the provider's prices
// table, which every cost quote reads before the connection-wide fallback.
package config

import (
	"errors"
	"math"
	"strings"

	"reasonix/internal/contract/pricing"
	"reasonix/internal/contract/provider"
)

// ErrModelPriceInvalid refuses a rate that is negative, not a number, or in a
// currency the price book does not know.
var ErrModelPriceInvalid = errors.New("a model's price must be finite, not negative, and in a known currency")

// SetModelPrice records price as model's own, or removes it with nil so the
// model falls back to the connection-wide price (and, for a vendor's published
// models, to the shipped rate). Zero is a price: a free model is not an unknown one.
func (e *ProviderEntry) SetModelPrice(model string, price *provider.Pricing) error {
	model = strings.TrimSpace(model)
	if e == nil || model == "" {
		return nil
	}
	if price == nil {
		delete(e.Prices, model)
		if len(e.Prices) == 0 {
			e.Prices = nil
		}
		return nil
	}
	for _, v := range []float64{price.Input, price.Output, price.CacheHit} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return ErrModelPriceInvalid
		}
	}
	currency := pricing.NormalizeCurrency(price.Currency)
	if !isCurrencyCode(currency) {
		return ErrModelPriceInvalid
	}
	if e.Prices == nil {
		e.Prices = map[string]*provider.Pricing{}
	}
	e.Prices[model] = &provider.Pricing{Input: price.Input, Output: price.Output, CacheHit: price.CacheHit, Currency: currency}
	return nil
}

// isCurrencyCode accepts only an ISO-style code, so a price states its unit
// instead of leaving a rate to be read in whatever currency is displayed.
func isCurrencyCode(c string) bool {
	if len(c) != 3 {
		return false
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
