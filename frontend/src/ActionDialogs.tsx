import {api, Overview} from './api'
import {
    AlbumDialog, ChangesDialog, ConfirmDialog, DateDialog, OrganizeDialog, PromptDialog, TakeoutDialog,
} from './Dialogs'
import {useT} from './i18n'

// Dialog is the modal currently open, with the data it acts on.
export type Dialog =
    | {type: 'move'}
    | {type: 'tag'}
    | {type: 'album'}
    | {type: 'newAlbum'}
    | {type: 'renameAlbum'; id: number; name: string}
    | {type: 'deleteAlbum'; id: number; name: string}
    | {type: 'deleteTag'; id: number; name: string}
    | {type: 'trash'; ids: number[]}
    | {type: 'date'; ids: number[]; initial: number}
    | {type: 'emptyTrash'}
    | {type: 'changes'}
    | {type: 'organize'}
    | {type: 'takeout'}

interface Props {
    dialog: Dialog
    // ids of the selected files the bulk dialogs act on.
    ids: number[]
    total: number
    overview: Overview | null
    onClose: () => void
    // onDone closes the dialog and refreshes; keepSelection is for actions
    // that leave the selected files in the current view.
    onDone: (message?: string, keepSelection?: boolean) => void
    onError: (e: unknown) => void
    onAlbumDeleted: (id: number) => void
    onTagDeleted: (id: number) => void
}

export default function ActionDialogs(p: Props) {
    const t = useT()
    const {dialog, ids, overview, onClose, onDone, onError} = p
    const files = (n: number) => t('files', {n})

    switch (dialog.type) {
        case 'move':
            return (
                <PromptDialog title={t('moveTitle', {files: files(ids.length)})} confirmLabel={t('moveConfirm')} allowEmpty
                              label={t('moveLabel')} placeholder={t('movePlaceholder')}
                              suggestions={overview?.dirs.map(d => d.dir).filter(Boolean)} onClose={onClose}
                              onSubmit={dest => api.MoveFiles(ids, dest)
                                  .then(n => onDone(t('moved', {files: files(n)}))).catch(onError)}/>
            )
        case 'tag':
            return (
                <PromptDialog title={t('tagTitle', {files: files(ids.length)})} confirmLabel={t('tagConfirm')}
                              placeholder={t('tagPlaceholder')} suggestions={overview?.tags.map(tag => tag.name)}
                              onClose={onClose}
                              onSubmit={name => api.AddTag(ids, name)
                                  .then(() => onDone(t('tagAdded', {name}), true)).catch(onError)}/>
            )
        case 'album':
            return (
                <AlbumDialog albums={overview?.albums ?? []} count={ids.length} onClose={onClose}
                             onPick={id => api.AddToAlbum(id, ids)
                                 .then(() => onDone(t('albumAdded'), true)).catch(onError)}
                             onCreate={name => api.CreateAlbum(name).then(id => api.AddToAlbum(id, ids))
                                 .then(() => onDone(t('albumCreated', {name}), true)).catch(onError)}/>
            )
        case 'newAlbum':
            return (
                <PromptDialog title={t('albumNew')} confirmLabel={t('create')} placeholder={t('albumName')} onClose={onClose}
                              onSubmit={name => api.CreateAlbum(name).then(() => onDone(undefined, true)).catch(onError)}/>
            )
        case 'renameAlbum':
            return (
                <PromptDialog title={t('albumRename')} confirmLabel={t('save')} initial={dialog.name} onClose={onClose}
                              onSubmit={name => api.RenameAlbum(dialog.id, name)
                                  .then(() => onDone(undefined, true)).catch(onError)}/>
            )
        case 'deleteAlbum':
            return (
                <ConfirmDialog title={t('albumDelete')} danger confirmLabel={t('albumDelete')} onClose={onClose}
                               message={t('albumDeleteMsg', {name: dialog.name})}
                               onConfirm={() => api.DeleteAlbum(dialog.id).then(() => {
                                   p.onAlbumDeleted(dialog.id)
                                   onDone()
                               }).catch(onError)}/>
            )
        case 'deleteTag':
            return (
                <ConfirmDialog title={t('tagDelete')} danger confirmLabel={t('tagDelete')} onClose={onClose}
                               message={t('tagDeleteMsg', {name: dialog.name})}
                               onConfirm={() => api.DeleteTag(dialog.id).then(() => {
                                   p.onTagDeleted(dialog.id)
                                   onDone()
                               }).catch(onError)}/>
            )
        case 'trash':
            return (
                <ConfirmDialog title={t('toTrash')} danger confirmLabel={t('toTrash')} onClose={onClose}
                               message={t('trashMsg', {files: files(dialog.ids.length)})}
                               onConfirm={() => api.TrashFiles(dialog.ids)
                                   .then(n => onDone(t('trashDone', {files: files(n)}))).catch(onError)}/>
            )
        case 'date':
            return (
                <DateDialog title={t('dateTitle', {files: files(dialog.ids.length)})} initial={dialog.initial}
                            onClose={onClose}
                            onSubmit={unix => api.SetDate(dialog.ids, unix)
                                .then(res => onDone(t('dateDone', {files: files(res.written + res.indexed)}) +
                                    (res.indexed > 0 ? t('dateIndexed', {n: res.indexed}) : ''), true)).catch(onError)}/>
            )
        case 'emptyTrash':
            return (
                <ConfirmDialog title={t('emptyTrash')} danger confirmLabel={t('emptyConfirm')} onClose={onClose}
                               message={t('emptyMsg', {files: files(p.total)})}
                               onConfirm={() => api.EmptyTrash()
                                   .then(n => onDone(t('emptyDone', {files: files(n)}))).catch(onError)}/>
            )
        case 'changes':
            return <ChangesDialog onClose={onClose} onApplied={() => onDone(t('changesApplied'), true)}/>
        case 'takeout':
            return (
                <TakeoutDialog onClose={onClose}
                               onApplied={res => onDone(t('takeoutDone', {n: res.trashed, dates: res.dates}) +
                                   (res.skipped > 0 ? t('takeoutSkipped', {n: res.skipped}) : ''))}/>
            )
        case 'organize':
            return <OrganizeDialog onClose={onClose} onApplied={n => onDone(t('moved', {files: files(n)}))}/>
    }
}
