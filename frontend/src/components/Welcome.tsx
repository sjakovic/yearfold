import {ReactNode} from 'react'
import {useT} from '../lib/i18n'

interface Props {
    recent: string[]
    version: string
    onPick: () => void
    onOpen: (root: string) => void
    languagePicker: ReactNode
}

export default function Welcome({recent, version, onPick, onOpen, languagePicker}: Props) {
    const t = useT()
    return (
        <div className="welcome">
            <h1>Yearfold</h1>
            <p className="muted">{t('welcomeText')}</p>
            <button className="primary big" onClick={onPick}>{t('welcomePick')}</button>
            {recent.length > 0 && (
                <div className="recent">
                    <div className="nav-title">{t('recent')}</div>
                    {recent.map(root => (
                        <button key={root} className="link" onClick={() => onOpen(root)}>{root}</button>
                    ))}
                </div>
            )}
            {languagePicker}
            <div className="version">{version}</div>
        </div>
    )
}
