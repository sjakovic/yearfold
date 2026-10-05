import {createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState} from 'react'
import {useOnChanged, useProgress} from '../hooks/useBackendEvents'
import {useItems} from '../hooks/useItems'
import {SelectModifiers, useSelection} from '../hooks/useSelection'
import {api, Filter, Item, Overview, Progress} from '../lib/api'
import {View, viewFilter} from '../lib/views'
import {useToast} from './toast'

export type Dialog =
    | {type: 'move'}
    | {type: 'tag'}
    | {type: 'album'}
    | {type: 'newAlbum'}
    | {type: 'renameAlbum'; id: number; name: string}
    | {type: 'renameFolder'; dir: string}
    | {type: 'deleteAlbum'; id: number; name: string}
    | {type: 'deleteTag'; id: number; name: string}
    | {type: 'trash'; ids: number[]}
    | {type: 'date'; ids: number[]; initial: number}
    | {type: 'emptyTrash'}
    | {type: 'changes'}
    | {type: 'organize'}
    | {type: 'takeout'}

interface RunOptions<T> {
    message?: (result: T) => string
    keepSelection?: boolean
}

interface Library {
    root: string
    overview: Overview | null
    progress: Progress

    view: View
    showView: (view: View) => void
    search: string
    setSearch: (value: string) => void
    sort: string
    setSort: (value: string) => void
    kind: string
    setKind: (value: string) => void
    recursive: boolean
    setRecursive: (value: boolean) => void

    items: Item[]
    total: number
    groups: Item[][]
    loadMore: () => void

    selection: Set<number>
    selectedIds: number[]
    select: (index: number, modifiers: SelectModifiers) => void
    toggle: (id: number) => void
    clearSelection: () => void
    selectDuplicates: () => void

    detailIndex: number | null
    openDetail: (index: number | null) => void
    dialog: Dialog | null
    openDialog: (dialog: Dialog | null) => void

    refreshOverview: () => void
    done: (message?: string, keepSelection?: boolean) => void
    run: <T>(action: Promise<T>, options?: RunOptions<T>) => void
}

const LibraryContext = createContext<Library | null>(null)

export function useLibrary(): Library {
    const library = useContext(LibraryContext)
    if (!library) throw new Error('useLibrary must be used inside LibraryProvider')
    return library
}

export function LibraryProvider({root, children}: {root: string; children: ReactNode}) {
    const {notify, fail} = useToast()
    const progress = useProgress()

    const [overview, setOverview] = useState<Overview | null>(null)
    const [view, setView] = useState<View>({type: 'all'})
    const [search, setSearch] = useState('')
    const [sort, setSort] = useState('date')
    const [kind, setKind] = useState('')
    const [recursive, setRecursive] = useState(false)
    const [detailIndex, openDetail] = useState<number | null>(null)
    const [dialog, openDialog] = useState<Dialog | null>(null)

    const filter = useMemo<Filter | null>(() => {
        const base = viewFilter(view, recursive, kind)
        return base && {...base, search, sort}
    }, [view, recursive, kind, search, sort])

    const {items, total, groups, reload, loadMore, reset} = useItems(filter, fail)
    const {selection, ids: selectedIds, select, toggle, clear: clearSelection, replace} = useSelection(items)

    const refreshOverview = useCallback(() => {
        api.GetOverview().then(setOverview).catch(fail)
    }, [fail])

    const refresh = useCallback(() => {
        refreshOverview()
        reload()
    }, [refreshOverview, reload])

    useEffect(refresh, [refresh])
    useOnChanged(refresh)

    const showView = useCallback((next: View) => {
        reset()
        clearSelection()
        openDetail(null)
        setView(next)
    }, [reset, clearSelection])

    const done = useCallback((message?: string, keepSelection = false) => {
        openDialog(null)
        if (!keepSelection) {
            clearSelection()
            openDetail(null)
        }
        if (message) notify(message)
        refresh()
    }, [clearSelection, notify, refresh])

    const run = useCallback(<T, >(action: Promise<T>, options: RunOptions<T> = {}) => {
        action.then(result => done(options.message?.(result), options.keepSelection)).catch(fail)
    }, [done, fail])

    const selectDuplicates = useCallback(() => {
        replace(new Set(groups.flatMap(group => group.slice(1).map(item => item.id))))
    }, [groups, replace])

    const value: Library = {
        root, overview, progress,
        view, showView, search, setSearch, sort, setSort, kind, setKind, recursive, setRecursive,
        items, total, groups, loadMore,
        selection, selectedIds, select, toggle, clearSelection, selectDuplicates,
        detailIndex, openDetail, dialog, openDialog,
        refreshOverview, done, run,
    }
    return <LibraryContext.Provider value={value}>{children}</LibraryContext.Provider>
}
