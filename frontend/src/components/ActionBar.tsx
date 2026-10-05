import {useLibrary} from '../context/library'
import {api} from '../lib/api'
import {useT} from '../lib/i18n'
import {isReadOnly} from '../lib/views'

export default function ActionBar() {
    const t = useT()
    const lib = useLibrary()
    const {view, selectedIds: ids} = lib
    const files = (n: number) => t('files', {n})

    return (
        <div className="subbar">
            {view.type === 'dir' && (
                <>
                    <label className="check">
                        <input type="checkbox" checked={lib.recursive} onChange={e => lib.setRecursive(e.target.checked)}/>
                        {t('recursive')}
                    </label>
                    <button onClick={() => lib.run(api.AlbumFromDir(view.dir), {
                        message: () => t('albumFromDirDone'),
                        keepSelection: true,
                    })}>
                        {t('albumFromDir')}
                    </button>
                </>
            )}
            {view.type === 'duplicates' && lib.groups.length > 0 && (
                <button onClick={lib.selectDuplicates}>{t('dupSelect')}</button>
            )}
            {view.type === 'trash' && lib.total > 0 && (
                <button className="danger" onClick={() => lib.openDialog({type: 'emptyTrash'})}>{t('emptyTrash')}</button>
            )}

            {ids.length > 0 && (
                <div className="selection-bar">
                    <b>{t('selected', {n: ids.length})}</b>
                    {!isReadOnly(view) && (
                        <>
                            <button onClick={() => lib.openDialog({type: 'move'})}>{t('move')}</button>
                            <button onClick={() => lib.openDialog({type: 'tag'})}>{t('tagAction')}</button>
                            <button onClick={() => lib.openDialog({type: 'album'})}>{t('toAlbum')}</button>
                            <button onClick={() => lib.openDialog({type: 'date', ids, initial: 0})}>{t('setDate')}</button>
                            {view.type === 'album' && (
                                <button onClick={() => lib.run(api.RemoveFromAlbum(view.id, ids), {
                                    message: () => t('removedFromAlbum'),
                                })}>
                                    {t('removeFromAlbum')}
                                </button>
                            )}
                            {view.type === 'tag' && (
                                <button onClick={() => lib.run(api.RemoveTag(ids, view.id), {
                                    message: () => t('tagRemoved'),
                                })}>
                                    {t('removeTag')}
                                </button>
                            )}
                            <button className="danger" onClick={() => lib.openDialog({type: 'trash', ids})}>
                                {t('toTrash')}
                            </button>
                        </>
                    )}
                    {view.type === 'trash' && (
                        <button onClick={() => lib.run(api.RestoreFiles(ids), {
                            message: n => t('restored', {files: files(n)}),
                        })}>
                            {t('restore')}
                        </button>
                    )}
                    <button className="link" onClick={lib.clearSelection}>{t('clearSelection')}</button>
                </div>
            )}
        </div>
    )
}
