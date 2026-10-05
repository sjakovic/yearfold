import {describe, expect, it} from 'vitest'
import {translator} from './i18n'
import {baseName, isReadOnly, viewFilter, viewTitle} from './views'

describe('viewFilter', () => {
    it('maps views to list filters', () => {
        expect(viewFilter({type: 'all'}, false, '')).toEqual({kind: 'media'})
        expect(viewFilter({type: 'trash'}, false, '')).toEqual({status: 'trashed'})
        expect(viewFilter({type: 'dir', dir: 'a/b'}, true, '')).toEqual({inDir: true, dir: 'a/b', recursive: true})
        expect(viewFilter({type: 'year', year: 2005}, false, '')).toEqual({year: 2005, kind: 'media'})
        expect(viewFilter({type: 'duplicates'}, false, '')).toBeNull()
    })

    it('narrows media views by type', () => {
        expect(viewFilter({type: 'all'}, false, 'video')).toEqual({kind: 'video'})
        expect(viewFilter({type: 'album', id: 3}, false, 'image')).toEqual({albumId: 3, kind: 'image'})
    })

    it('leaves the other-files view alone', () => {
        expect(viewFilter({type: 'other'}, false, 'video')).toEqual({kind: 'other'})
    })
})

describe('viewTitle', () => {
    const t = translator('en')

    it('names folders, with the library name for the root', () => {
        expect(viewTitle({type: 'dir', dir: '2005/Trip'}, null, '/home/me/Photos', t)).toBe('2005/Trip')
        expect(viewTitle({type: 'dir', dir: ''}, null, '/home/me/Photos', t)).toBe('Photos')
    })

    it('names years and fixed views', () => {
        expect(viewTitle({type: 'year', year: 2005}, null, '/x', t)).toBe('2005')
        expect(viewTitle({type: 'nodate'}, null, '/x', t)).toBe('No date')
    })
})

describe('helpers', () => {
    it('baseName handles both path separators', () => {
        expect(baseName('/home/me/Photos')).toBe('Photos')
        expect(baseName('C:\\Users\\me\\Photos\\')).toBe('Photos')
    })

    it('trash and missing views are read-only', () => {
        expect(isReadOnly({type: 'trash'})).toBe(true)
        expect(isReadOnly({type: 'missing'})).toBe(true)
        expect(isReadOnly({type: 'all'})).toBe(false)
    })
})
