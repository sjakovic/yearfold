import * as go from '../../wailsjs/go/app/App'
import {app, library, store} from '../../wailsjs/go/models'

export type Item = store.Item
export type FileRow = store.File
export type Tag = store.Tag
export type Album = store.Album
export type Overview = app.Overview
export type Detail = app.Detail
export type AppState = app.State
export type ChangeReport = app.ChangeReport
export type OrganizePreview = app.OrganizePreview
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
