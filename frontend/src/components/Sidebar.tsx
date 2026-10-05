import {useMemo, useState} from 'react'
import {useLibrary} from '../context/library'
import {buildTree, DirNode} from '../lib/dirTree'
import {useT} from '../lib/i18n'
import {baseName, View} from '../lib/views'

export default function Sidebar() {
    const t = useT()
    const {overview, view, showView, openDialog, root} = useLibrary()
    const rootName = baseName(root)
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
                     onClick={() => showView({type: 'dir', dir: n.path})} title={n.path || rootName}>
                    <span className="twisty" onClick={e => {
                        e.stopPropagation()
                        toggle(n.path)
                    }}>
                        {n.children.length ? (open.has(n.path) ? '▾' : '▸') : ''}
                    </span>
                    <span className="label">{n.path === '' ? rootName : n.name}</span>
                    {n.path !== '' && (
                        <span className="row-actions">
                            <button className="link" title={t('folderRename')} onClick={e => {
                                e.stopPropagation()
                                openDialog({type: 'renameFolder', dir: n.path})
                            }}>✎</button>
                        </span>
                    )}
                    <span className="count">{n.count}</span>
                </div>
                {open.has(n.path) && n.children.map(c => renderNode(c, depth + 1))}
            </div>
        )
    }

    const item = (label: string, v: View, count?: number) => (
        <div className={`nav-item ${view.type === v.type ? 'active' : ''}`} onClick={() => showView(v)}>
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
                <button className="link" onClick={() => openDialog({type: 'newAlbum'})}>{t('newAlbumLink')}</button>
            </div>
            <div className="nav-section">
                {overview?.albums.map(a => (
                    <div key={a.id} className={`nav-item ${view.type === 'album' && view.id === a.id ? 'active' : ''}`}
                         onClick={() => showView({type: 'album', id: a.id})}>
                        <span className="label">{a.name}</span>
                        <span className="row-actions">
                            <button className="link" title={t('rename')} onClick={e => {
                                e.stopPropagation()
                                openDialog({type: 'renameAlbum', id: a.id, name: a.name})
                            }}>✎</button>
                            <button className="link" title={t('albumDelete')} onClick={e => {
                                e.stopPropagation()
                                openDialog({type: 'deleteAlbum', id: a.id, name: a.name})
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
                         onClick={() => showView({type: 'tag', id: tag.id})}>
                        <span className="label"># {tag.name}</span>
                        <span className="row-actions">
                            <button className="link" title={t('tagDelete')} onClick={e => {
                                e.stopPropagation()
                                openDialog({type: 'deleteTag', id: tag.id, name: tag.name})
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
                         onClick={() => showView({type: 'year', year: y.year})}>
                        <span className="label">{y.year}</span>
                        <span className="count">{y.count}</span>
                    </div>
                ))}
                {!overview?.years.length && <div className="nav-empty">{t('noYears')}</div>}
            </div>
        </aside>
    )
}
