package store

// SessionLabel devolve um rótulo curto para a sessão usado nos balões: o nome do
// projeto se houver, senão "sessão <8 chars do id>".
func (s *Store) SessionLabel(sessionID string) string {
	sess, err := s.GetSession(sessionID)
	if err != nil {
		return "sessão"
	}
	if sess.ProjectID != "" {
		if p, err := s.GetProject(sess.ProjectID); err == nil && p.Name != "" {
			return p.Name
		}
	}
	short := sessionID
	if len(short) > 8 {
		short = short[:8]
	}
	return "sessão " + short
}
