import {describe, expect, it} from 'vitest'
import {extOf, formatDate, formatSize, fromDateInput, toDateInput} from './format'

describe('formatSize', () => {
    it.each([
        [0, '0 B'],
        [1023, '1023 B'],
        [1024, '1.0 KB'],
        [1536, '1.5 KB'],
        [10 * 1024, '10 KB'],
        [5 * 1024 * 1024, '5.0 MB'],
        [3 * 1024 ** 3, '3.0 GB'],
    ])('%d bytes is %s', (bytes, text) => {
        expect(formatSize(bytes)).toBe(text)
    })
})

describe('dates', () => {
    const noon = Date.UTC(2004, 5, 15, 12, 30) / 1000

    it('shows wall-clock time regardless of the local zone', () => {
        expect(formatDate(noon)).toBe('2004-06-15 12:30')
        expect(formatDate(0)).toBe('')
    })

    it('round-trips through a datetime-local input', () => {
        expect(toDateInput(noon)).toBe('2004-06-15T12:30')
        expect(fromDateInput(toDateInput(noon))).toBe(noon)
        expect(toDateInput(0)).toBe('')
    })
})

describe('extOf', () => {
    it.each([
        ['IMG_1.jpg', 'JPG'],
        ['archive.tar.gz', 'GZ'],
        ['README', 'FILE'],
        ['.hidden', 'FILE'],
    ])('%s -> %s', (name, ext) => {
        expect(extOf(name)).toBe(ext)
    })
})
