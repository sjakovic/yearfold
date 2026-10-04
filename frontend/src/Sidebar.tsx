import {useMemo, useState} from 'react'
import {Overview} from './api'
import {useT} from './i18n'
import {View} from './views'

interface DirNode {
    name: string
    path: string
    count: number
    children: DirNode[]
}

function buildTree(dirs: {dir: string; count: number}[]): DirNode {
    const root: DirNode = {name: '', path: '', count: 0, children: []}
    const index = new Map<string, DirNode>([['', root]])
    const ensure = (path: string): DirNode => {
        const hit = index.get(path)
        if (hit) return hit
        const i = path.lastIndexOf('/')
        const parent = ensure(i < 0 ? '' : path.slice(0, i))
        const node: DirNode = {name: path.slice(i + 1), path, count: 0, children: []}
        parent.children.push(node)
        index.set(path, node)
        return node
    }
    for (const d of dirs) {
        let node: DirNode | undefined = ensure(d.dir)
        // Counts are shown recursively, so add to every ancestor as well.
        let path = d.dir
        for (;;) {
            node = index.get(path)!
            node.count += d.count
            if (path === '') break
            const i = path.lastIndexOf('/')
            path = i < 0 ? '' : path.slice(0, i)
        }
    }
    const sort = (n: DirNode) => {
        n.children.sort((a, b) => a.name.localeCompare(b.name, undefined, {numeric: true}))
        n.children.forEach(sort)
    }
    sort(root)
    return root
}

interface Props {
    rootName: string
    overview: Overview | null
    view: View
    onView: (v: View) => void
    onNewAlbum: () => void
    onRenameAlbum: (id: number, name: string) => void
    onDeleteAlbum: (id: number, name: string) => void
    onDeleteTag: (id: number, name: string) => void
}

export default function Sidebar(p: Props) {
    const {overview, view} = p
    const t = useT()
    const tree = useMemo(() => buildTree(overview?.dirs ?? []), [overview?.dirs])
    const [open, setOpen] = useState<Set<string>>(new Set(['']))

    const toggle = (path: string) => {
        const next = new Set(open)
        if (next.has(path)) next.delete(path)
        else next.add(path)
        setOpen(next)
    }

    const renderNode = (n: DirNode, depth: number) => {
        const active = view.type === 'dir' && view.dir === n.path
        return (
            <div key={n.path || '/'}>
                <div className={`nav-item ${active ? 'active' : ''}`} style={{paddingLeft: 8 + depth * 14}}
                     onClick={() => p.onView({type: 'dir', dir: n.path})} title={n.path || p.rootName}>
                    <span className="twisty" onClick={e => {
                        e.stopPropagation()
                        toggle(n.path)
                    }}>
                        {n.children.length ? (open.has(n.path) ? '▾' : '▸') : ''}
                    </span>
                    <span className="label">{n.path === '' ? p.rootName : n.name}</span>
                    <span className="count">{n.count}</span>
                </div>
                {open.has(n.path) && n.children.map(c => renderNode(c, depth + 1))}
            </div>
        )
    }

    const item = (label: string, v: View, count?: number) => (
        <div className={`nav-item ${view.type === v.type ? 'active' : ''}`} onClick={() => p.onView(v)}>
            <span className="label">{label}</span>
            {count !== undefined && <span className="count">{count}</span>}
        </div>
    )

    const s = overview?.stats
    return (
        <aside className="sidebar">
            <div className="nav-section">
                {item(t('viewAll'), {type: 'all'}, s?.media)}
                {item(t('viewNoDate'), {type: 'nodate'}, s?.noDate)}
                {item(t('viewOther'), {type: 'other'}, s?.other)}
                {item(t('viewDuplicates'), {type: 'duplicates'})}
                {item(t('viewTrash'), {type: 'trash'}, s?.trashed)}
                {!!s?.missing && item(t('viewMissing'), {type: 'missing'}, s.missing)}
            </div>

            <div className="nav-title">{t('folders')}</div>
            <div className="nav-section">{renderNode(tree, 0)}</div>

            <div className="nav-title">
                {t('albums')}
                <button className="link" onClick={p.onNewAlbum}>{t('newAlbumLink')}</button>
            </div>
            <div className="nav-section">
                {overview?.albums.map(a => (
                    <div key={a.id} className={`nav-item ${view.type === 'album' && view.id === a.id ? 'active' : ''}`}
                         onClick={() => p.onView({type: 'album', id: a.id})}>
                        <span className="label">{a.name}</span>
                        <span className="row-actions">
                            <button className="link" title={t('rename')} onClick={e => {
                                e.stopPropagation()
                                p.onRenameAlbum(a.id, a.name)
                            }}>✎</button>
                            <button className="link" title={t('albumDelete')} onClick={e => {
                                e.stopPropagation()
                                p.onDeleteAlbum(a.id, a.name)
                            }}>×</button>
                        </span>
                        <span className="count">{a.count}</span>
                    </div>
                ))}
                {!overview?.albums.length && <div className="nav-empty">{t('noAlbums')}</div>}
            </div>

            <div className="nav-title">{t('tags')}</div>
            <div className="nav-section">
                {overview?.tags.map(tag => (
                    <div key={tag.id} className={`nav-item ${view.type === 'tag' && view.id === tag.id ? 'active' : ''}`}
                         onClick={() => p.onView({type: 'tag', id: tag.id})}>
                        <span className="label"># {tag.name}</span>
                        <span className="row-actions">
                            <button className="link" title={t('tagDelete')} onClick={e => {
                                e.stopPropagation()
                                p.onDeleteTag(tag.id, tag.name)
                            }}>×</button>
                        </span>
                        <span className="count">{tag.count}</span>
                    </div>
                ))}
                {!overview?.tags.length && <div className="nav-empty">{t('noTags')}</div>}
            </div>

            <div className="nav-title">{t('years')}</div>
            <div className="nav-section">
                {overview?.years.map(y => (
                    <div key={y.year} className={`nav-item ${view.type === 'year' && view.year === y.year ? 'active' : ''}`}
                         onClick={() => p.onView({type: 'year', year: y.year})}>
                        <span className="label">{y.year}</span>
                        <span className="count">{y.count}</span>
                    </div>
                ))}
                {!overview?.years.length && <div className="nav-empty">{t('noYears')}</div>}
            </div>
        </aside>
    )
}
