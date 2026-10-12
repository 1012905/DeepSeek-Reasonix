package checkpoint

// OnUndoChange installs the owner's content-free notification callback.
func (s *Store) OnUndoChange(notify func()) {
	s.mu.Lock()
	s.undoChanged = notify
	s.mu.Unlock()
}

// AvailableUndo reads the saved offer without disk I/O or file verification.
func (s *Store) AvailableUndo() (*RewindUndo, error) {
	if s == nil || s.barrier.HasWriters() {
		return nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tx := s.lastUndo
	if tx == nil || tx.State != TxCommitted {
		return nil, nil
	}
	return &RewindUndo{TransactionID: tx.ID, Turn: tx.Turn, Files: len(tx.Targets)}, nil
}

// Callbacks may read the offer, so release the lock before notifying.
func (s *Store) unlockUndoChange(previous *TransactionManifest) {
	before, after := "", ""
	if previous != nil {
		before = previous.ID
	}
	if s.lastUndo != nil {
		after = s.lastUndo.ID
	}
	notify := s.undoChanged
	s.mu.Unlock()
	if before != after && notify != nil {
		notify()
	}
}

// LastUndoTransactionID returns the committed transaction id available for undo.
func (s *Store) LastUndoTransactionID() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastUndo == nil || s.lastUndo.State != TxCommitted {
		return ""
	}
	return s.lastUndo.ID
}

func (s *Store) restoreUndoOffer(tx TransactionManifest) {
	s.mu.Lock()
	defer s.unlockUndoChange(s.lastUndo)
	if s.lastUndo == nil || s.lastUndo.UpdatedAt.Before(tx.UpdatedAt) {
		s.lastUndo = &tx
	}
}
