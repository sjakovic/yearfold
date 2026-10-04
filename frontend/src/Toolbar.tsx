import {ReactNode, useState} from 'react'
import {useT} from './i18n'

interface Props {
    title: string
    total: number
    version: string
    search: string
    onSearch: (value: string) => void
    sort: string
    onSort: (value: string) => void
    // kind narrows the view to photos or videos; '' shows everything.
    kind: string
    onKind: (value: string) => void
    onTakeout: () => void
    onCheckChanges: () => void
    onOrganize: () => void
    onUndo: () => void
    onExport: () => void
    onClearThumbs: () => void
    onOpenFolder: () => void
    languagePicker: ReactNode
}

export default function Toolbar(p: Props) {
    const t = useT()
    const [menuOpen, setMenuOpen] = useState(false)

    return (
        <header className="toolbar">
            <div className="title">
                <h2>{p.title}</h2>
                <span className="muted">{t('files', {n: p.total})}</span>
            </div>
            <input className="search" type="search" placeholder={t('searchPlaceholder')} value={p.search}
                   onChange={e => p.onSearch(e.target.value)}/>
            <select value={p.kind} onChange={e => p.onKind(e.target.value)}>
                <option value="">{t('kindAll')}</option>
                <option value="image">{t('kindImage')}</option>
                <option value="video">{t('kindVideo')}</option>
            </select>
            <select value={p.sort} onChange={e => p.onSort(e.target.value)}>
                <option value="date">{t('sortDate')}</option>
                <option value="name">{t('sortName')}</option>
            </select>
            <button onClick={p.onCheckChanges}>{t('checkChanges')}</button>
            <button onClick={p.onOrganize}>{t('organize')}</button>
            <button onClick={p.onUndo}>{t('undo')}</button>
            <div className="menu">
                <button onClick={() => setMenuOpen(open => !open)}>⋯</button>
                {menuOpen && (
                    <div className="menu-list" onClick={() => setMenuOpen(false)}>
                        <button onClick={p.onTakeout}>{t('menuTakeout')}</button>
                        <button onClick={p.onExport}>{t('menuExport')}</button>
                        <button onClick={p.onClearThumbs}>{t('menuClearThumbs')}</button>
                        <button onClick={p.onOpenFolder}>{t('menuOpen')}</button>
                        {p.languagePicker}
                        <div className="version">Yearfold {p.version}</div>
                    </div>
                )}
            </div>
        </header>
    )
}
