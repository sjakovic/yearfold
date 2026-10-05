import {useCallback, useEffect, useMemo, useState} from 'react'
import {Album, api, Detail, Item} from '../lib/api'
import {errorText, extOf, formatDate, formatSize} from '../lib/format'
import {Key, useT} from '../lib/i18n'

const SOURCE_LABEL: Record<string, Key> = {
    exif: 'srcExif',
    takeout: 'srcTakeout',
    mtime: 'srcMtime',
    manual: 'srcManual',
}

interface Props {
    item: Item
    albums: Album[]
    tagNames: string[]
    hasPrev: boolean
    hasNext: boolean
    onPrev: () => void
    onNext: () => void
    onClose: () => void
    onTrash: (id: number) => void
    onSetDate: (id: number, current: number) => void
    onChanged: () => void
    onError: (msg: string) => void
}

export default function DetailView(p: Props) {
    const {item, onError} = p
    const t = useT()
    const [detail, setDetail] = useState<Detail | null>(null)
    const [tagInput, setTagInput] = useState('')
    const [mediaFailed, setMediaFailed] = useState(false)

    const reload = useCallback(() => {
        api.GetDetail(item.id).then(setDetail).catch(e => onError(errorText(e)))
    }, [item.id, onError])

    useEffect(reload, [reload, item.relPath, item.takenAt, item.takenSrc])

    useEffect(() => {
        const onKey = (e: KeyboardEvent) => {
            if ((e.target as HTMLElement).tagName === 'INPUT') return
            if (e.key === 'Escape') p.onClose()
            if (e.key === 'ArrowLeft' && p.hasPrev) p.onPrev()
            if (e.key === 'ArrowRight' && p.hasNext) p.onNext()
        }
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    })

    const meta = useMemo(() => {
        if (!detail?.file.metaJson) return {tags: {}, takeout: null}
        try {
            const doc = JSON.parse(detail.file.metaJson)
            return {tags: (doc.tags ?? {}) as Record<string, string>, takeout: doc.takeout ?? null}
        } catch {
            return {tags: {}, takeout: null}
        }
    }, [detail])

    const run = (action: Promise<unknown>) =>
        action.then(() => {
            reload()
            p.onChanged()
        }).catch(e => onError(errorText(e)))

    const addTag = () => {
        const name = tagInput.trim()
        if (!name) return
        setTagInput('')
        run(api.AddTag([item.id], name))
    }

    const f = detail?.file
    const otherAlbums = p.albums.filter(a => !detail?.albums.some(x => x.id === a.id))

    return (
        <div className="detail">
            <div className="detail-media">
                <button className="detail-close" onClick={p.onClose} title={t('detailClose')}>×</button>
                {p.hasPrev && <button className="detail-nav prev" onClick={p.onPrev}>‹</button>}
                {p.hasNext && <button className="detail-nav next" onClick={p.onNext}>›</button>}
                {mediaFailed || (item.kind !== 'image' && item.kind !== 'video') ? (
                    <div className="detail-fallback">
                        <div className="thumb-fallback big">{extOf(item.name)}</div>
                        <div>{t('detailCannotShow')}</div>
                    </div>
                ) : item.kind === 'video' ? (
                    <video key={item.id} src={`/file/${item.id}`} controls onError={() => setMediaFailed(true)}/>
                ) : (
                    <img key={item.id} src={`/file/${item.id}`} onError={() => setMediaFailed(true)}/>
                )}
            </div>

            <div className="detail-panel">
                <h3>{item.name}</h3>
                {f && (
                    <>
                        <table className="props">
                            <tbody>
                            <tr><th>{t('propPath')}</th><td>{f.relPath}</td></tr>
                            <tr><th>{t('propSize')}</th><td>{formatSize(f.size)}</td></tr>
                            {f.width > 0 && <tr><th>{t('propDimensions')}</th><td>{f.width} × {f.height}</td></tr>}
                            <tr><th>{t('propDate')}</th><td>{formatDate(f.takenAt)} <small>({t(SOURCE_LABEL[f.takenSrc] ?? 'srcUnknown')})</small></td></tr>
                            {f.camera && <tr><th>{t('propCamera')}</th><td>{f.camera}</td></tr>}
                            {f.hasGps && <tr><th>{t('propLocation')}</th><td>{f.lat.toFixed(5)}, {f.lon.toFixed(5)}</td></tr>}
                            {meta.takeout?.description && <tr><th>{t('propDescription')}</th><td>{meta.takeout.description}</td></tr>}
                            {meta.takeout?.people?.length > 0 && <tr><th>{t('propPeople')}</th><td>{meta.takeout.people.join(', ')}</td></tr>}
                            {detail.sidecars.length > 0 && <tr><th>{t('propSidecar')}</th><td>{detail.sidecars.join(', ')}</td></tr>}
                            {f.hash && <tr><th>SHA-256</th><td className="mono">{f.hash.slice(0, 16)}…</td></tr>}
                            </tbody>
                        </table>

                        <div className="detail-actions">
                            <button onClick={() => api.Reveal(item.id).catch(e => onError(errorText(e)))}>{t('reveal')}</button>
                            {f.status === 'present' && (f.kind === 'image' || f.kind === 'video') && (
                                <button onClick={() => p.onSetDate(item.id, f.takenSrc === 'mtime' ? 0 : f.takenAt)}>
                                    {t('setDate')}
                                </button>
                            )}
                            {f.status === 'present' && <button className="danger" onClick={() => p.onTrash(item.id)}>{t('toTrash')}</button>}
                        </div>

                        <h4>{t('tags')}</h4>
                        <div className="chips">
                            {detail.tags.map(t => (
                                <span key={t.id} className="chip">
                                    {t.name}
                                    <button onClick={() => run(api.RemoveTag([item.id], t.id))}>×</button>
                                </span>
                            ))}
                        </div>
                        <div className="inline-form">
                            <input list="detail-tags" value={tagInput} placeholder={t('addTagPlaceholder')}
                                   onChange={e => setTagInput(e.target.value)}
                                   onKeyDown={e => e.key === 'Enter' && addTag()}/>
                            <datalist id="detail-tags">{p.tagNames.map(n => <option key={n} value={n}/>)}</datalist>
                            <button onClick={addTag}>{t('add')}</button>
                        </div>

                        <h4>{t('albums')}</h4>
                        <div className="chips">
                            {detail.albums.map(a => (
                                <span key={a.id} className="chip">
                                    {a.name}
                                    <button onClick={() => run(api.RemoveFromAlbum(a.id, [item.id]))}>×</button>
                                </span>
                            ))}
                        </div>
                        {otherAlbums.length > 0 && (
                            <select value="" onChange={e => run(api.AddToAlbum(Number(e.target.value), [item.id]))}>
                                <option value="" disabled>{t('toAlbum')}</option>
                                {otherAlbums.map(a => <option key={a.id} value={a.id}>{a.name}</option>)}
                            </select>
                        )}

                        <h4>{t('allMeta')}</h4>
                        {Object.keys(meta.tags).length === 0 ? (
                            <div className="muted">{t('noMeta')}</div>
                        ) : (
                            <table className="props small">
                                <tbody>
                                {Object.entries(meta.tags).sort(([a], [b]) => a.localeCompare(b)).map(([k, v]) => (
                                    <tr key={k}><th>{k}</th><td>{v}</td></tr>
                                ))}
                                </tbody>
                            </table>
                        )}
                    </>
                )}
            </div>
        </div>
    )
}
