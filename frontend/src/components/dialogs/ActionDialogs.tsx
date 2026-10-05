import {useLibrary} from '../../context/library'
import {api} from '../../lib/api'
import {useT} from '../../lib/i18n'
import {baseName} from '../../lib/views'
import {AlbumDialog} from './AlbumDialog'
import {ChangesDialog} from './ChangesDialog'
import {ConfirmDialog} from './ConfirmDialog'
import {DateDialog} from './DateDialog'
import {OrganizeDialog} from './OrganizeDialog'
import {PromptDialog} from './PromptDialog'
import {TakeoutDialog} from './TakeoutDialog'

export default function ActionDialogs() {
    const t = useT()
    const lib = useLibrary()
    const {dialog, overview, view, selectedIds: ids, run, done} = lib
    if (!dialog) return null

    const files = (n: number) => t('files', {n})
    const close = () => lib.openDialog(null)
    const leaveIfShowing = (type: 'album' | 'tag', id: number) => {
        if (view.type === type && view.id === id) lib.showView({type: 'all'})
    }

    switch (dialog.type) {
        case 'move':
            return (
                <PromptDialog title={t('moveTitle', {files: files(ids.length)})} confirmLabel={t('moveConfirm')} allowEmpty
                              label={t('moveLabel')} placeholder={t('movePlaceholder')}
                              suggestions={overview?.dirs.map(d => d.dir).filter(Boolean)} onClose={close}
                              onSubmit={dest => run(api.MoveFiles(ids, dest), {
                                  message: n => t('moved', {files: files(n)}),
                              })}/>
            )
        case 'tag':
            return (
                <PromptDialog title={t('tagTitle', {files: files(ids.length)})} confirmLabel={t('tagConfirm')}
                              placeholder={t('tagPlaceholder')} suggestions={overview?.tags.map(tag => tag.name)}
                              onClose={close}
                              onSubmit={name => run(api.AddTag(ids, name), {
                                  message: () => t('tagAdded', {name}),
                                  keepSelection: true,
                              })}/>
            )
        case 'album':
            return (
                <AlbumDialog albums={overview?.albums ?? []} count={ids.length} onClose={close}
                             onPick={id => run(api.AddToAlbum(id, ids), {
                                 message: () => t('albumAdded'),
                                 keepSelection: true,
                             })}
                             onCreate={name => run(api.CreateAlbum(name).then(id => api.AddToAlbum(id, ids)), {
                                 message: () => t('albumCreated', {name}),
                                 keepSelection: true,
                             })}/>
            )
        case 'newAlbum':
            return (
                <PromptDialog title={t('albumNew')} confirmLabel={t('create')} placeholder={t('albumName')} onClose={close}
                              onSubmit={name => run(api.CreateAlbum(name), {keepSelection: true})}/>
            )
        case 'renameAlbum':
            return (
                <PromptDialog title={t('albumRename')} confirmLabel={t('save')} initial={dialog.name} onClose={close}
                              onSubmit={name => run(api.RenameAlbum(dialog.id, name), {keepSelection: true})}/>
            )
        case 'renameFolder': {
            const {dir} = dialog
            const showRenamed = (renamed: string) => {
                if (view.type === 'dir' && (view.dir === dir || view.dir.startsWith(`${dir}/`))) {
                    lib.showView({type: 'dir', dir: renamed + view.dir.slice(dir.length)})
                }
                return renamed
            }
            return (
                <PromptDialog title={t('folderRename')} confirmLabel={t('save')} initial={baseName(dir)}
                              label={t('folderRenameLabel', {name: baseName(dir)})} onClose={close}
                              onSubmit={name => run(api.RenameFolder(dir, name).then(showRenamed), {
                                  message: renamed => t('folderRenamed', {name: baseName(renamed)}),
                              })}/>
            )
        }
        case 'deleteAlbum':
            return (
                <ConfirmDialog title={t('albumDelete')} danger confirmLabel={t('albumDelete')} onClose={close}
                               message={t('albumDeleteMsg', {name: dialog.name})}
                               onConfirm={() => run(api.DeleteAlbum(dialog.id).then(() => leaveIfShowing('album', dialog.id)))}/>
            )
        case 'deleteTag':
            return (
                <ConfirmDialog title={t('tagDelete')} danger confirmLabel={t('tagDelete')} onClose={close}
                               message={t('tagDeleteMsg', {name: dialog.name})}
                               onConfirm={() => run(api.DeleteTag(dialog.id).then(() => leaveIfShowing('tag', dialog.id)))}/>
            )
        case 'trash':
            return (
                <ConfirmDialog title={t('toTrash')} danger confirmLabel={t('toTrash')} onClose={close}
                               message={t('trashMsg', {files: files(dialog.ids.length)})}
                               onConfirm={() => run(api.TrashFiles(dialog.ids), {
                                   message: n => t('trashDone', {files: files(n)}),
                               })}/>
            )
        case 'date':
            return (
                <DateDialog title={t('dateTitle', {files: files(dialog.ids.length)})} initial={dialog.initial}
                            onClose={close}
                            onSubmit={unix => run(api.SetDate(dialog.ids, unix), {
                                message: res => t('dateDone', {files: files(res.written + res.indexed)}) +
                                    (res.indexed > 0 ? t('dateIndexed', {n: res.indexed}) : ''),
                                keepSelection: true,
                            })}/>
            )
        case 'emptyTrash':
            return (
                <ConfirmDialog title={t('emptyTrash')} danger confirmLabel={t('emptyConfirm')} onClose={close}
                               message={t('emptyMsg', {files: files(lib.total)})}
                               onConfirm={() => run(api.EmptyTrash(), {
                                   message: n => t('emptyDone', {files: files(n)}),
                               })}/>
            )
        case 'changes':
            return <ChangesDialog onClose={close} onApplied={() => done(t('changesApplied'), true)}/>
        case 'takeout':
            return (
                <TakeoutDialog onClose={close}
                               onApplied={res => done(t('takeoutDone', {n: res.trashed, dates: res.dates}) +
                                   (res.skipped > 0 ? t('takeoutSkipped', {n: res.skipped}) : ''))}/>
            )
        case 'organize':
            return <OrganizeDialog onClose={close} onApplied={n => done(t('moved', {files: files(n)}))}/>
    }
}
