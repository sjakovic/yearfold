import {Lang, LANGUAGES} from '../lib/i18n'

export default function LanguagePicker({lang, onChange}: {lang: Lang; onChange: (lang: Lang) => void}) {
    return (
        <div className="languages">
            {LANGUAGES.map(l => (
                <button key={l.code} className={`link ${l.code === lang ? 'current' : ''}`} onClick={() => onChange(l.code)}>
                    {l.label}
                </button>
            ))}
        </div>
    )
}
