import {ReactNode, useState} from 'react'
import {useLibrary} from '../context/library'
import {useToast} from '../context/toast'
import {api} from '../lib/api'
import {useT} from '../lib/i18n'
import {viewTitle} from '../lib/views'

interface Props {
    version: string
    languagePicker: ReactNode
    onOpenFolder: () => void
}

export default function Toolbar({version, languagePicker, onOpenFolder}: Props) {
    const t = useT()
    const lib = useLibrary()
    const {notify, fail} = useToast()
    const [menuOpen, setMenuOpen] = useState(false)

    const undo = () => lib.run(api.Undo(), {
        message: n => (n ? t('undoDone', {files: t('files', {n})}) : t('undoNone')),
    })
    const exportJSON = () => api.ExportJSON().then(path => path && notify(t('exportSaved', {path}))).catch(fail)
    const clearThumbs = () => api.ClearThumbs().then(() => notify(t('thumbsCleared'))).catch(fail)

    return (
        <header className="toolbar">
            <div className="title">
                <h2>{viewTitle(lib.view, lib.overview, lib.root, t)}</h2>
                <span className="muted">{t('files', {n: lib.total})}</span>
            </div>
            <input className="search" type="search" placeholder={t('searchPlaceholder')} value={lib.search}
                   onChange={e => lib.setSearch(e.target.value)}/>
            <select value={lib.kind} onChange={e => lib.setKind(e.target.value)}>
                <option value="">{t('kindAll')}</option>
                <option value="image">{t('kindImage')}</option>
                <option value="video">{t('kindVideo')}</option>
            </select>
            <select value={lib.sort} onChange={e => lib.setSort(e.target.value)}>
                <option value="date">{t('sortDate')}</option>
                <option value="name">{t('sortName')}</option>
            </select>
            <button onClick={() => lib.openDialog({type: 'changes'})}>{t('checkChanges')}</button>
            <button onClick={() => lib.openDialog({type: 'organize'})}>{t('organize')}</button>
            <button onClick={undo}>{t('undo')}</button>
            <div className="menu">
                <button onClick={() => setMenuOpen(open => !open)}>⋯</button>
                {menuOpen && (
                    <div className="menu-list" onClick={() => setMenuOpen(false)}>
                        <button onClick={() => lib.openDialog({type: 'takeout'})}>{t('menuTakeout')}</button>
                        <button onClick={exportJSON}>{t('menuExport')}</button>
                        <button onClick={clearThumbs}>{t('menuClearThumbs')}</button>
                        <button onClick={onOpenFolder}>{t('menuOpen')}</button>
                        {languagePicker}
                        <div className="version">Yearfold {version}</div>
                    </div>
                )}
            </div>
        </header>
    )
}
