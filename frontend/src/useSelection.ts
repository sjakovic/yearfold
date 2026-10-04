import {useCallback, useMemo, useRef, useState} from 'react'
import {Item} from './api'

export interface SelectModifiers {
    shiftKey: boolean
    metaKey: boolean
    ctrlKey: boolean
}

// useSelection keeps the set of selected file ids with the usual desktop
// behaviour: click selects one, Cmd/Ctrl toggles, Shift extends a range.
export function useSelection(items: Item[]) {
    const [selection, setSelection] = useState<Set<number>>(new Set())
    const anchor = useRef<number | null>(null)

    const select = (index: number, e: SelectModifiers) => {
        const id = items[index].id
        const additive = e.metaKey || e.ctrlKey
        let next: Set<number>
        if (e.shiftKey && anchor.current !== null) {
            const from = Math.min(anchor.current, index)
            const to = Math.max(anchor.current, index)
            next = new Set(additive ? selection : [])
            for (let i = from; i <= to; i++) next.add(items[i].id)
        } else if (additive) {
            next = new Set(selection)
            if (next.has(id)) next.delete(id)
            else next.add(id)
            anchor.current = index
        } else {
            next = new Set([id])
            anchor.current = index
        }
        setSelection(next)
    }

    const toggle = (id: number) => {
        const next = new Set(selection)
        if (next.has(id)) next.delete(id)
        else next.add(id)
        setSelection(next)
    }

    const clear = useCallback(() => {
        anchor.current = null
        setSelection(new Set())
    }, [])

    const ids = useMemo(() => [...selection], [selection])

    return {selection, ids, select, toggle, clear, replace: setSelection}
}
