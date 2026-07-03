package store

import (
	"github.com/google/uuid"
)

// UserIntent é a intenção destilada de uma mensagem `user/text`, persistida como
// cache de extração (por (session_id, seq)) e como base de recorrência
// cross-session (por intent_key). É um tipo do próprio store — os detectores do
// pacote user o produzem, mas não há import de `user` aqui (evita ciclo).
type UserIntent struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	SessionID string `json:"session_id"`
	Seq       int64  `json:"seq"`
	Summary   string `json:"summary"`
	Category  string `json:"category"`
	IntentKey string `json:"intent_key"`
	CreatedAt int64  `json:"created_at"`
}

// SaveUserIntent grava (ou ignora se já existe) uma intenção. O UNIQUE
// (session_id, seq) torna a chamada idempotente: re-executar o motor sobre a
// mesma sessão não duplica linhas nem re-chama o LLM (ver ListSessionIntents).
func (s *Store) SaveUserIntent(ui *UserIntent) error {
	if ui.ID == "" {
		ui.ID = uuid.NewString()
	}
	if ui.CreatedAt == 0 {
		ui.CreatedAt = now()
	}
	var pid any
	if ui.ProjectID != "" {
		pid = ui.ProjectID
	}
	_, err := s.db.Exec(`INSERT INTO user_intents
		(id, project_id, session_id, seq, summary, category, intent_key, created_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(session_id, seq) DO NOTHING`,
		ui.ID, pid, ui.SessionID, ui.Seq, ui.Summary, ui.Category, ui.IntentKey, ui.CreatedAt)
	return err
}

// ListSessionIntents devolve as intenções já extraídas de uma sessão — usado
// como cache: o motor só chama o LLM para os seqs ausentes deste conjunto.
func (s *Store) ListSessionIntents(sessionID string) ([]*UserIntent, error) {
	rows, err := s.db.Query(`SELECT id, COALESCE(project_id,''), session_id, seq, summary, category, intent_key, created_at
		FROM user_intents WHERE session_id=? ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserIntents(rows)
}

// FindSimilarIntents devolve intenções do mesmo projeto com a MESMA intent_key
// (chave canônica). É a base da recorrência cross-session: mesma chave em
// sessões distintas = mesma tarefa recorrente do usuário.
func (s *Store) FindSimilarIntents(projectID, intentKey string) ([]*UserIntent, error) {
	if intentKey == "" {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT id, COALESCE(project_id,''), session_id, seq, summary, category, intent_key, created_at
		FROM user_intents WHERE COALESCE(project_id,'')=? AND intent_key=? ORDER BY created_at`, projectID, intentKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserIntents(rows)
}

func scanUserIntents(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]*UserIntent, error) {
	out := []*UserIntent{}
	for rows.Next() {
		u := &UserIntent{}
		if err := rows.Scan(&u.ID, &u.ProjectID, &u.SessionID, &u.Seq, &u.Summary, &u.Category, &u.IntentKey, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// SkillsUsedInSession devolve os skill_ids que registraram uso nesta sessão.
// Fonte do TargetSkill para roteamento de correções do usuário: se o usuário
// corrigiu o agente numa sessão que usou exatamente uma skill, a correção
// provavelmente se refere a ela.
func (s *Store) SkillsUsedInSession(sessionID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT DISTINCT skill_id FROM skill_usage WHERE session_id=?`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
