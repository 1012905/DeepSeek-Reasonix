package control

import "reasonix/internal/state/memory"

// refreshRecalled reloads the snapshot when a fact the last recall returned no
// longer reads the same on disk, so an edit made outside this session is what
// the next recall offers. It compares the facts, not a timestamp, and reads only
// those: no full scan, so a fact never recalled is not seen until a reload.
func (m *memoryManager) refreshRecalled() {
	m.mu.Lock()
	recalled := m.recalled
	m.mu.Unlock()
	if len(recalled) == 0 {
		return
	}
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	mem := m.current()
	if mem == nil {
		return
	}
	for _, held := range recalled {
		now, ok, err := mem.Store.Read(held.ID)
		if err == nil && ok && sameFact(held, now) {
			continue
		}
		// Changed, gone, or unreadable: the reload settles which, and an
		// unreadable store reaches the recall as its StoreError.
		m.applyWrite(mem, "")
		return
	}
}

func sameFact(a, b memory.Memory) bool {
	return a.ID == b.ID && a.Revision == b.Revision && a.Name == b.Name && a.Title == b.Title &&
		a.Description == b.Description && a.Type == b.Type && a.Scope == b.Scope &&
		a.Activation == b.Activation && a.Volatility == b.Volatility && a.SubjectKey == b.SubjectKey &&
		a.ExpiresAt.Equal(b.ExpiresAt) && a.LastVerifiedAt.Equal(b.LastVerifiedAt) &&
		a.Keywords == b.Keywords && a.Body == b.Body
}
