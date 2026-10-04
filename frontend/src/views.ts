import {Filter, Overview} from './api'
import {Translate} from './i18n'

// View is what the main area currently shows.
export type View =
    | {type: 'all'}
    | {type: 'nodate'}
    | {type: 'other'}
    | {type: 'duplicates'}
    | {type: 'trash'}
    | {type: 'missing'}
    | {type: 'dir'; dir: string}
    | {type: 'album'; id: number}
    | {type: 'tag'; id: number}
    | {type: 'year'; year: number}

export function baseName(path: string): string {
    const parts = path.split(/[\\/]/).filter(Boolean)
    return parts[parts.length - 1] ?? path
}

// viewFilter returns the list filter of a view, or null for views that are
// not a plain file list (duplicates).
export function viewFilter(view: View, recursive: boolean, kind: string): Filter | null {
    const filter = baseFilter(view, recursive)
    // The type filter narrows any view that can contain photos and videos.
    if (filter && kind && (!filter.kind || filter.kind === 'media')) {
        return {...filter, kind}
    }
    return filter
}

function baseFilter(view: View, recursive: boolean): Filter | null {
    switch (view.type) {
        case 'all': return {kind: 'media'}
        case 'nodate': return {noDate: true}
        case 'other': return {kind: 'other'}
        case 'trash': return {status: 'trashed'}
        case 'missing': return {status: 'missing'}
        case 'dir': return {inDir: true, dir: view.dir, recursive}
        case 'album': return {albumId: view.id}
        case 'tag': return {tagId: view.id}
        case 'year': return {year: view.year, kind: 'media'}
        case 'duplicates': return null
    }
}

export function viewTitle(view: View, overview: Overview | null, root: string, t: Translate): string {
    switch (view.type) {
        case 'all': return t('viewAll')
        case 'nodate': return t('viewNoDate')
        case 'other': return t('viewOther')
        case 'duplicates': return t('viewDuplicates')
        case 'trash': return t('viewTrash')
        case 'missing': return t('viewMissingTitle')
        case 'dir': return view.dir || baseName(root)
        case 'album': return t('viewAlbum', {name: overview?.albums.find(a => a.id === view.id)?.name ?? ''})
        case 'tag': return t('viewTag', {name: overview?.tags.find(x => x.id === view.id)?.name ?? ''})
        case 'year': return String(view.year)
    }
}

// isReadOnly reports whether files in the view cannot be organised.
export function isReadOnly(view: View): boolean {
    return view.type === 'trash' || view.type === 'missing'
}
