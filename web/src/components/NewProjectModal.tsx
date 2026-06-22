import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { createProject } from '../api';
import type { Project } from '../api';
import FolderPicker from './FolderPicker';

interface Props {
  onCreated: (project: Project) => void;
  onClose: () => void;
}

type Source = 'local' | 'git';

// Deriva um nome de projeto a partir de uma URL git (basename sem .git).
function nameFromGitUrl(url: string): string {
  const last = url.trim().replace(/\/+$/, '').split(/[/:]/).pop() ?? '';
  return last.replace(/\.git$/i, '');
}

// NewProjectModal cria um projeto a partir de pastas locais OU de uma URL git
// (clonada pelo backend). Reutilizado pela página Projetos e pelo wizard de
// nova sessão.
export default function NewProjectModal({ onCreated, onClose }: Props) {
  const { t } = useTranslation();
  const [source, setSource] = useState<Source>('local');
  const [name, setName] = useState('');
  const [nameTouched, setNameTouched] = useState(false);
  const [description, setDescription] = useState('');
  const [dirs, setDirs] = useState<string[]>([]);
  const [gitUrl, setGitUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  const nameRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    nameRef.current?.focus();
    function onKey(e: KeyboardEvent) { if (e.key === 'Escape') onClose(); }
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose]);

  function onGitUrlChange(v: string) {
    setGitUrl(v);
    if (!nameTouched) setName(nameFromGitUrl(v));
  }

  const canCreate = !!name && (source === 'local' || !!gitUrl.trim());

  async function handleCreate() {
    if (!canCreate || busy) return;
    setBusy(true);
    setError(false);
    try {
      const proj = await createProject(
        name,
        description,
        source === 'local' ? dirs : [],
        source === 'git' ? gitUrl.trim() : undefined,
      );
      onCreated(proj);
    } catch {
      setError(true);
      setBusy(false);
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" role="dialog" aria-modal="true" aria-labelledby="np-title"
        onClick={(e) => e.stopPropagation()}>
        <h2 id="np-title">{t('modal.newProject')}</h2>

        {/* fonte do projeto: pasta local vs repositório git */}
        <div className="np-source" role="tablist" aria-label={t('modal.newProject')}>
          {(['local', 'git'] as Source[]).map((s) => (
            <button key={s} type="button" role="tab" aria-selected={source === s}
              className={`np-source-tab${source === s ? ' on' : ''}`}
              onClick={() => setSource(s)}>
              {t(s === 'local' ? 'modal.sourceLocal' : 'modal.sourceGit')}
            </button>
          ))}
        </div>

        {source === 'git' && (
          <>
            <label htmlFor="np-git" style={{ marginTop: '0.75rem' }}>{t('modal.gitUrl')}</label>
            <input id="np-git" value={gitUrl} placeholder={t('modal.gitUrlPlaceholder')}
              onChange={(e) => onGitUrlChange(e.target.value)} />
          </>
        )}

        <label htmlFor="np-name" style={{ marginTop: '0.75rem' }}>{t('modal.name')}</label>
        <input ref={nameRef} id="np-name" value={name}
          onChange={(e) => { setName(e.target.value); setNameTouched(true); }} />

        <label htmlFor="np-desc" style={{ marginTop: '0.75rem' }}>{t('modal.description')}</label>
        <input id="np-desc" value={description} onChange={(e) => setDescription(e.target.value)} />

        {source === 'local' && (
          <>
            <label style={{ marginTop: '0.75rem', display: 'block' }}>{t('modal.dirs')}</label>
            <FolderPicker value={dirs} onChange={setDirs} />
          </>
        )}

        {error && <p className="error-banner">{t('common.actionFailed')}</p>}

        <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem' }}>
          <button className="btn btn-primary" disabled={busy || !canCreate} onClick={handleCreate}>
            {busy ? t('common.loading') : t('modal.create')}
          </button>
          <button className="btn btn-secondary" onClick={onClose}>{t('modal.cancel')}</button>
        </div>
      </div>
    </div>
  );
}
