import {useT} from './i18n'
import {isReadOnly, View} from './views'

interface Props {
    view: View
    total: number
    selected: number
    recursive: boolean
    onRecursive: (value: boolean) => void
    hasDuplicates: boolean
    onAlbumFromDir: (dir: string) => void
    onSelectDuplicates: () => void
    onEmptyTrash: () => void
    onMove: () => void
    onTag: () => void
    onAlbum: () => void
    onDate: () => void
    onRemoveFromAlbum: (albumId: number) => void
    onRemoveTag: (tagId: number) => void
    onTrash: () => void
    onRestore: () => void
    onClearSelection: () => void
}

// ActionBar holds the actions of the current view on the left and the
// actions for the selected files on the right.
export default function ActionBar(p: Props) {
    const t = useT()
    const {view} = p

    return (
        <div className="subbar">
            {view.type === 'dir' && (
                <>
                    <label className="check">
                        <input type="checkbox" checked={p.recursive} onChange={e => p.onRecursive(e.target.checked)}/>
                        {t('recursive')}
                    </label>
                    <button onClick={() => p.onAlbumFromDir(view.dir)}>{t('albumFromDir')}</button>
                </>
            )}
            {view.type === 'duplicates' && p.hasDuplicates && (
                <button onClick={p.onSelectDuplicates}>{t('dupSelect')}</button>
            )}
            {view.type === 'trash' && p.total > 0 && (
                <button className="danger" onClick={p.onEmptyTrash}>{t('emptyTrash')}</button>
            )}

            {p.selected > 0 && (
                <div className="selection-bar">
                    <b>{t('selected', {n: p.selected})}</b>
                    {!isReadOnly(view) && (
                        <>
                            <button onClick={p.onMove}>{t('move')}</button>
                            <button onClick={p.onTag}>{t('tagAction')}</button>
                            <button onClick={p.onAlbum}>{t('toAlbum')}</button>
                            <button onClick={p.onDate}>{t('setDate')}</button>
                            {view.type === 'album' && (
                                <button onClick={() => p.onRemoveFromAlbum(view.id)}>{t('removeFromAlbum')}</button>
                            )}
                            {view.type === 'tag' && (
                                <button onClick={() => p.onRemoveTag(view.id)}>{t('removeTag')}</button>
                            )}
                            <button className="danger" onClick={p.onTrash}>{t('toTrash')}</button>
                        </>
                    )}
                    {view.type === 'trash' && <button onClick={p.onRestore}>{t('restore')}</button>}
                    <button className="link" onClick={p.onClearSelection}>{t('clearSelection')}</button>
                </div>
            )}
        </div>
    )
}
