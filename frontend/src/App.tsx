import {useCallback, useEffect, useMemo, useRef, useState} from 'react'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {api, AppState, errorText, Filter, Overview, Progress} from './api'
import ActionBar from './ActionBar'
import ActionDialogs, {Dialog} from './ActionDialogs'
import DetailView from './DetailView'
import DuplicatesView from './DuplicatesView'
import Grid from './Grid'
import {I18n, Key, Lang, normalizeLang, translator} from './i18n'
import LanguagePicker from './LanguagePicker'
import Sidebar from './Sidebar'
import Toolbar from './Toolbar'
import {useItems} from './useItems'
import {useSelection} from './useSelection'
import {baseName, View, viewFilter, viewTitle} from './views'
import Welcome from './Welcome'

const TOAST_MS = 5000

const PHASES: Record<string, Key> = {
    scan: 'phaseScan',
    meta: 'phaseMeta',
    hash: 'phaseHash',
}

export default function App() {
    const [state, setState] = useState<AppState | null>(null)
    const [overview, setOverview] = useState<Overview | null>(null)
    const [view, setView] = useState<View>({type: 'all'})
    const [search, setSearch] = useState('')
    const [sort, setSort] = useState('date')
    const [kind, setKind] = useState('')
    const [recursive, setRecursive] = useState(false)
    const [detailIndex, setDetailIndex] = useState<number | null>(null)
    const [dialog, setDialog] = useState<Dialog | null>(null)
    const [progress, setProgress] = useState<Progress>({phase: '', done: 0, total: 0, seq: 0})
    const [toast, setToast] = useState('')

    const root = state?.root ?? ''
    const lang = normalizeLang(state?.language ?? 'en')
    const t = useMemo(() => translator(lang), [lang])
    const files = (n: number) => t('files', {n})

    const notify = useCallback((message: string) => {
        setToast(message)
        window.setTimeout(() => setToast(current => (current === message ? '' : current)), TOAST_MS)
    }, [])
    const fail = useCallback((e: unknown) => notify(t('error', {msg: errorText(e)})), [notify, t])

    const filter = useMemo<Filter | null>(() => {
        const base = viewFilter(view, recursive, kind)
        return base && {...base, search, sort}
    }, [view, recursive, kind, search, sort])

    const {items, total, groups, reload, loadMore, reset} = useItems(filter, fail)
    const {selection, ids, select, toggle, clear, replace} = useSelection(items)

    const refreshOverview = useCallback(() => {
        api.GetOverview().then(setOverview).catch(fail)
    }, [fail])

    const refresh = useCallback(() => {
        if (!root) return
        refreshOverview()
        reload()
    }, [root, refreshOverview, reload])

    useEffect(() => {
        document.documentElement.lang = lang === 'sr' ? 'sr-Cyrl' : 'en'
    }, [lang])

    // Reopen the last library on start.
    useEffect(() => {
        api.GetState().then(st => {
            if (!st.root && st.recent.length > 0) {
                api.OpenLibrary(st.recent[0]).then(setState).catch(() => setState(st))
            } else {
                setState(st)
            }
        })
    }, [])

    // A different view starts from the top with a clean selection. This must
    // run before the refresh effect below so the reload starts from page one.
    useEffect(() => {
        reset()
        clear()
        setDetailIndex(null)
    }, [view, root, reset, clear])

    useEffect(refresh, [refresh])

    const refreshRef = useRef(refresh)
    refreshRef.current = refresh
    useEffect(() => {
        const offProgress = EventsOn('progress', (p: Progress) => setProgress(prev => (p.seq > prev.seq ? p : prev)))
        const offChanged = EventsOn('changed', () => refreshRef.current())
        return () => {
            offProgress()
            offChanged()
        }
    }, [])

    // done runs after an action that changed data: close, refresh and report.
    const done = (message?: string, keepSelection = false) => {
        setDialog(null)
        if (!keepSelection) {
            clear()
            setDetailIndex(null)
        }
        if (message) notify(message)
        refresh()
    }

    const openLibrary = (opening: Promise<AppState>) => {
        opening.then(st => {
            setState(st)
            setOverview(null)
            setView({type: 'all'})
        }).catch(fail)
    }

    // Keeps the first file of every duplicate group and selects the rest.
    const selectDuplicates = () => replace(new Set(groups.flatMap(group => group.slice(1).map(it => it.id))))

    const leaveView = (type: 'album' | 'tag', id: number) => {
        if (view.type === type && view.id === id) setView({type: 'all'})
    }

    const languagePicker = (
        <LanguagePicker lang={lang} onChange={(code: Lang) => api.SetLanguage(code).then(setState).catch(fail)}/>
    )

    if (!state) {
        return <div className="welcome"><p className="muted">{t('loading')}</p></div>
    }

    if (!root) {
        return (
            <I18n.Provider value={t}>
                <Welcome recent={state.recent} version={state.version} languagePicker={languagePicker}
                         onPick={() => openLibrary(api.PickFolder())} onOpen={r => openLibrary(api.OpenLibrary(r))}/>
                {toast && <div className="toast">{toast}</div>}
            </I18n.Provider>
        )
    }

    const detailItem = detailIndex !== null ? items[detailIndex] : undefined

    return (
        <I18n.Provider value={t}>
            <div className="app">
                <Sidebar rootName={baseName(root)} overview={overview} view={view} onView={setView}
                         onNewAlbum={() => setDialog({type: 'newAlbum'})}
                         onRenameAlbum={(id, name) => setDialog({type: 'renameAlbum', id, name})}
                         onDeleteAlbum={(id, name) => setDialog({type: 'deleteAlbum', id, name})}
                         onDeleteTag={(id, name) => setDialog({type: 'deleteTag', id, name})}/>

                <main className="main">
                    <Toolbar title={viewTitle(view, overview, root, t)} total={total} version={state.version}
                             search={search} onSearch={setSearch} sort={sort} onSort={setSort}
                             kind={kind} onKind={setKind} onTakeout={() => setDialog({type: 'takeout'})}
                             languagePicker={languagePicker}
                             onCheckChanges={() => setDialog({type: 'changes'})}
                             onOrganize={() => setDialog({type: 'organize'})}
                             onUndo={() => api.Undo()
                                 .then(n => done(n ? t('undoDone', {files: files(n)}) : t('undoNone'))).catch(fail)}
                             onExport={() => api.ExportJSON()
                                 .then(path => path && notify(t('exportSaved', {path}))).catch(fail)}
                             onClearThumbs={() => api.ClearThumbs().then(() => notify(t('thumbsCleared'))).catch(fail)}
                             onOpenFolder={() => openLibrary(api.PickFolder())}/>

                    <ActionBar view={view} total={total} selected={ids.length}
                               recursive={recursive} onRecursive={setRecursive}
                               hasDuplicates={groups.length > 0}
                               onAlbumFromDir={dir => api.AlbumFromDir(dir)
                                   .then(() => done(t('albumFromDirDone'), true)).catch(fail)}
                               onSelectDuplicates={selectDuplicates}
                               onEmptyTrash={() => setDialog({type: 'emptyTrash'})}
                               onMove={() => setDialog({type: 'move'})}
                               onTag={() => setDialog({type: 'tag'})}
                               onAlbum={() => setDialog({type: 'album'})}
                               onDate={() => setDialog({type: 'date', ids, initial: 0})}
                               onRemoveFromAlbum={albumId => api.RemoveFromAlbum(albumId, ids)
                                   .then(() => done(t('removedFromAlbum'))).catch(fail)}
                               onRemoveTag={tagId => api.RemoveTag(ids, tagId)
                                   .then(() => done(t('tagRemoved'))).catch(fail)}
                               onTrash={() => setDialog({type: 'trash', ids})}
                               onRestore={() => api.RestoreFiles(ids)
                                   .then(n => done(t('restored', {files: files(n)}))).catch(fail)}
                               onClearSelection={clear}/>

                    {view.type === 'duplicates' ? (
                        <DuplicatesView groups={groups} selection={selection} hashing={progress.phase !== ''}
                                        onToggle={toggle}
                                        onOpen={id => setDetailIndex(items.findIndex(it => it.id === id))}/>
                    ) : (
                        <Grid items={items} total={total} selection={selection} onSelect={select}
                              onOpen={setDetailIndex} onLoadMore={loadMore}/>
                    )}

                    {progress.phase && (
                        <footer className="status">
                            {PHASES[progress.phase] ? t(PHASES[progress.phase]) : progress.phase}
                            {progress.total > 0 ? ` ${progress.done} / ${progress.total}` : ` ${progress.done}`}
                            {progress.total > 0 && <progress value={progress.done} max={progress.total}/>}
                        </footer>
                    )}
                </main>

                {detailItem && detailIndex !== null && (
                    <DetailView item={detailItem} albums={overview?.albums ?? []}
                                tagNames={overview?.tags.map(tag => tag.name) ?? []}
                                hasPrev={detailIndex > 0} hasNext={detailIndex < items.length - 1}
                                onPrev={() => setDetailIndex(detailIndex - 1)}
                                onNext={() => setDetailIndex(detailIndex + 1)}
                                onClose={() => setDetailIndex(null)}
                                onTrash={id => setDialog({type: 'trash', ids: [id]})}
                                onSetDate={(id, initial) => setDialog({type: 'date', ids: [id], initial})}
                                onChanged={refreshOverview} onError={notify}/>
                )}

                {dialog && (
                    <ActionDialogs dialog={dialog} ids={ids} total={total} overview={overview}
                                   onClose={() => setDialog(null)} onDone={done} onError={fail}
                                   onAlbumDeleted={id => leaveView('album', id)}
                                   onTagDeleted={id => leaveView('tag', id)}/>
                )}

                {toast && <div className="toast">{toast}</div>}
            </div>
        </I18n.Provider>
    )
}
