import {MouseEvent, useEffect, useLayoutEffect, useRef, useState} from 'react'
import {useVirtualizer} from '@tanstack/react-virtual'
import {extOf, Item} from './api'
import {useT} from './i18n'

const CELL = 172
const ROW = 196

export function Thumb({item}: {item: Item}) {
    const [failed, setFailed] = useState(false)
    const visual = item.kind === 'image' || item.kind === 'video'
    if (!visual || failed) {
        return <div className="thumb-fallback">{extOf(item.name)}</div>
    }
    return (
        <>
            <img src={`/thumb/${item.id}`} loading="lazy" draggable={false} onError={() => setFailed(true)}/>
            {item.kind === 'video' && <span className="badge">▶</span>}
        </>
    )
}

interface Props {
    items: Item[]
    total: number
    selection: Set<number>
    onSelect: (index: number, e: MouseEvent) => void
    onOpen: (index: number) => void
    onLoadMore: () => void
}

export default function Grid({items, total, selection, onSelect, onOpen, onLoadMore}: Props) {
    const t = useT()
    const scroller = useRef<HTMLDivElement>(null)
    const [cols, setCols] = useState(4)

    useLayoutEffect(() => {
        const el = scroller.current
        if (!el) return
        const update = () => setCols(Math.max(1, Math.floor((el.clientWidth - 16) / CELL)))
        update()
        const ro = new ResizeObserver(update)
        ro.observe(el)
        return () => ro.disconnect()
    }, [])

    const rows = Math.ceil(items.length / cols)
    const virtualizer = useVirtualizer({
        count: rows,
        getScrollElement: () => scroller.current,
        estimateSize: () => ROW,
        overscan: 4,
    })
    const virtualRows = virtualizer.getVirtualItems()
    const lastRow = virtualRows.length ? virtualRows[virtualRows.length - 1].index : 0

    useEffect(() => {
        if (items.length < total && lastRow >= rows - 3) onLoadMore()
    }, [lastRow, rows, items.length, total, onLoadMore])

    return (
        <div className="grid-scroller" ref={scroller}>
            <div style={{height: virtualizer.getTotalSize(), position: 'relative'}}>
                {virtualRows.map(vr => (
                    <div key={vr.key} className="grid-row" style={{transform: `translateY(${vr.start}px)`, height: ROW}}>
                        {items.slice(vr.index * cols, vr.index * cols + cols).map((it, i) => {
                            const index = vr.index * cols + i
                            return (
                                <div key={it.id} className={`cell ${selection.has(it.id) ? 'selected' : ''}`}
                                     style={{width: CELL}} title={it.relPath}
                                     onClick={e => onSelect(index, e)} onDoubleClick={() => onOpen(index)}>
                                    <div className="thumb"><Thumb item={it}/></div>
                                    <div className="caption">{it.name}</div>
                                </div>
                            )
                        })}
                    </div>
                ))}
            </div>
            {items.length === 0 && <div className="empty">{t('gridEmpty')}</div>}
        </div>
    )
}
