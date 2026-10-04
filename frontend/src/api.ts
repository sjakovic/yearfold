import * as go from '../wailsjs/go/main/App'
import {library, main, store} from '../wailsjs/go/models'

export type Item = store.Item
export type FileRow = store.File
export type Tag = store.Tag
export type Album = store.Album
export type Overview = main.Overview
export type Detail = main.Detail
export type AppState = main.AppState
export type ChangeReport = main.ChangeReport
export type OrganizePreview = main.OrganizePreview
export type TakeoutSummary = library.TakeoutSummary
export type TakeoutResult = library.TakeoutResult

export interface Filter {
    inDir?: boolean
    dir?: string
    recursive?: boolean
    kind?: string
    tagId?: number
    albumId?: number
    search?: string
    status?: string
    noDate?: boolean
    year?: number
    sort?: string
    offset?: number
    limit?: number
}

export interface Progress {
    phase: string
    done: number
    total: number
    seq: number
}

export const PAGE_SIZE = 400

export const api = {
    ...go,
    List: (f: Filter) => go.List(f as store.Filter),
    Duplicates: () => go.Duplicates() as Promise<Item[][]>,
}

export function formatDate(ts: number): string {
    if (!ts) return ''
    return new Date(ts * 1000).toISOString().slice(0, 16).replace('T', ' ')
}

// The index keeps dates as wall-clock time expressed in UTC; these convert
// to and from the value of an <input type="datetime-local">.
export function toDateInput(ts: number): string {
    return ts ? new Date(ts * 1000).toISOString().slice(0, 16) : ''
}

export function fromDateInput(value: string): number {
    return Math.floor(Date.parse(`${value}:00Z`) / 1000)
}

export function formatSize(n: number): string {
    if (n < 1024) return `${n} B`
    const units = ['KB', 'MB', 'GB', 'TB']
    let v = n / 1024
    let i = 0
    while (v >= 1024 && i < units.length - 1) {
        v /= 1024
        i++
    }
    return `${v.toFixed(v < 10 ? 1 : 0)} ${units[i]}`
}

export function extOf(name: string): string {
    const i = name.lastIndexOf('.')
    return i > 0 ? name.slice(i + 1).toUpperCase() : 'FILE'
}

export function errorText(e: unknown): string {
    return e instanceof Error ? e.message : String(e)
}
