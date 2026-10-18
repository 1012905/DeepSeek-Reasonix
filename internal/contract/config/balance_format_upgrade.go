package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"reasonix/internal/base/fileutil"
	"reasonix/internal/contract/walletformat"
)

// balanceFormatsFile records, beside the user config, the balance formats a
// relay entry was given. The 1.x line shares the config file and rewrites it
// whole from the keys it knows, so a key it does not know is gone after its
// next save; the record is what puts it back. It is not config_version, which
// 7 through 12 leave to the 1.x line's own upgrades.
const balanceFormatsFile = "balance-formats.json"

type balanceFormatRecord struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

type balanceFormatLedger struct {
	Entries map[string]balanceFormatRecord `json:"entries"`
}

func readBalanceFormatLedger(dir string) (balanceFormatLedger, bool) {
	var l balanceFormatLedger
	data, err := os.ReadFile(filepath.Join(dir, balanceFormatsFile))
	if err != nil || json.Unmarshal(data, &l) != nil {
		return balanceFormatLedger{Entries: map[string]balanceFormatRecord{}}, err == nil
	}
	if l.Entries == nil {
		l.Entries = map[string]balanceFormatRecord{}
	}
	return l, true
}

func writeBalanceFormatLedger(dir string, l balanceFormatLedger) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return fileutil.AtomicWriteFile(filepath.Join(dir, balanceFormatsFile), append(data, '\n'), 0o600)
}

// relayBalanceEntry is an entry whose wallet address is not a vendor's own host,
// the only kind that needs a declared format.
func relayBalanceEntry(p *ProviderEntry) bool {
	return strings.TrimSpace(p.BalanceURL) != "" && walletformat.Implied(p.BalanceURL) == ""
}

// reconcileBalanceFormats keeps the formats of relay entries across a rewrite by
// the 1.x line. The first run declares "deepseek" where none is named, which is
// how such an address was always read; later runs put back a format the file lost
// for an unchanged address. A removal on purpose goes through ForgetBalanceFormat.
func reconcileBalanceFormats(path string) (bool, error) {
	dir := filepath.Dir(path)
	ledger, existed := readBalanceFormatLedger(dir)
	cfg := LoadForEdit(path)
	changed := false
	for i := range cfg.Providers {
		p := &cfg.Providers[i]
		if !relayBalanceEntry(p) {
			continue
		}
		rec, known := ledger.Entries[p.Name]
		switch {
		case strings.TrimSpace(p.BalanceFormat) != "":
			ledger.Entries[p.Name] = balanceFormatRecord{URL: strings.TrimSpace(p.BalanceURL), Format: p.BalanceFormat}
		case known && rec.URL == strings.TrimSpace(p.BalanceURL) && walletformat.Format(rec.Format).Valid():
			p.BalanceFormat, changed = rec.Format, true
		case !existed:
			p.BalanceFormat, changed = string(walletformat.DeepSeek), true
			ledger.Entries[p.Name] = balanceFormatRecord{URL: strings.TrimSpace(p.BalanceURL), Format: p.BalanceFormat}
		}
	}
	if changed {
		if err := cfg.SaveTo(path); err != nil {
			return false, err
		}
	}
	return changed, writeBalanceFormatLedger(dir, ledger)
}

// ForgetBalanceFormat drops the record of an entry's format, so a person who
// removed it is not given it back at the next start.
func ForgetBalanceFormat(path, provider string) error {
	dir := filepath.Dir(path)
	ledger, _ := readBalanceFormatLedger(dir)
	if _, ok := ledger.Entries[provider]; !ok {
		return nil
	}
	delete(ledger.Entries, provider)
	return writeBalanceFormatLedger(dir, ledger)
}

// NoteBalanceFormat records what an editor save left on an entry: its format
// when it is a relay's, nothing otherwise, so a removal is not undone.
func NoteBalanceFormat(path string, p *ProviderEntry) error {
	if p == nil {
		return nil
	}
	if !relayBalanceEntry(p) || strings.TrimSpace(p.BalanceFormat) == "" {
		return ForgetBalanceFormat(path, p.Name)
	}
	dir := filepath.Dir(path)
	ledger, _ := readBalanceFormatLedger(dir)
	ledger.Entries[p.Name] = balanceFormatRecord{URL: strings.TrimSpace(p.BalanceURL), Format: p.BalanceFormat}
	return writeBalanceFormatLedger(dir, ledger)
}
