import {useCallback, useRef, useState} from 'react'
import {api, Filter, Item, PAGE_SIZE} from './api'

// useItems loads the files of the current view page by page. A null filter
// means the duplicates view, which loads groups instead of a flat list.
export function useItems(filter: Filter | null, onError: (e: unknown) => void) {
    const [items, setItems] = useState<Item[]>([])
    const [total, setTotal] = useState(0)
    const [groups, setGroups] = useState<Item[][]>([])

    // Guards against responses that arrive after the view has changed.
    const request = useRef(0)
    const loading = useRef(false)
    const count = useRef(0)
    count.current = items.length

    const reset = useCallback(() => {
        count.current = 0
        setItems([])
    }, [])

    const reload = useCallback(() => {
        const id = ++request.current
        if (!filter) {
            api.Duplicates().then(found => {
                if (id !== request.current) return
                const flat = found.flat()
                setGroups(found)
                setItems(flat)
                setTotal(flat.length)
            }).catch(onError)
            return
        }
        // Reload as many rows as are already shown so the scroll position holds.
        const limit = Math.max(PAGE_SIZE, count.current)
        api.List({...filter, offset: 0, limit}).then(page => {
            if (id !== request.current) return
            setItems(page.items)
            setTotal(page.total)
        }).catch(onError)
    }, [filter, onError])

    const loadMore = useCallback(() => {
        if (!filter || loading.current) return
        loading.current = true
        const id = request.current
        api.List({...filter, offset: count.current, limit: PAGE_SIZE}).then(page => {
            if (id === request.current) setItems(prev => [...prev, ...page.items])
        }).catch(onError).finally(() => {
            loading.current = false
        })
    }, [filter, onError])

    return {items, total, groups, reload, loadMore, reset}
}
