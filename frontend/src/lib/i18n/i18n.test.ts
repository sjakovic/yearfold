import {describe, expect, it} from 'vitest'
import {en} from './en'
import {sr} from './sr'
import {normalizeLang, translator} from './index'

describe('translator', () => {
    it('fills in parameters', () => {
        const t = translator('en')
        expect(t('viewAlbum', {name: 'Trip'})).toBe('Album: Trip')
        expect(t('exportSaved', {path: '/tmp/x.json'})).toBe('Saved: /tmp/x.json')
    })

    it('uses English plural forms', () => {
        const t = translator('en')
        expect(t('files', {n: 1})).toBe('1 file')
        expect(t('files', {n: 0})).toBe('0 files')
        expect(t('files', {n: 5})).toBe('5 files')
    })

    it('uses Serbian plural forms', () => {
        const t = translator('sr')
        expect(t('files', {n: 1})).toBe('1 фајл')
        expect(t('files', {n: 3})).toBe('3 фајла')
        expect(t('files', {n: 5})).toBe('5 фајлова')
        expect(t('files', {n: 21})).toBe('21 фајл')
        expect(t('files', {n: 22})).toBe('22 фајла')
        expect(t('files', {n: 11})).toBe('11 фајлова')
    })

    it('falls back to English for unknown languages', () => {
        expect(normalizeLang('de')).toBe('en')
        expect(normalizeLang('sr')).toBe('sr')
    })
})

describe('dictionaries', () => {
    const placeholders = (entry: string | string[]) =>
        [...new Set([entry].flat().flatMap(s => s.match(/\{\w+\}/g) ?? []))].sort()

    it('have the same keys', () => {
        expect(Object.keys(sr).sort()).toEqual(Object.keys(en).sort())
    })

    it('use the same placeholders in every translation', () => {
        for (const key of Object.keys(en) as (keyof typeof en)[]) {
            expect(placeholders(sr[key]), key).toEqual(placeholders(en[key]))
        }
    })

    it('have no empty strings', () => {
        for (const dict of [en, sr]) {
            for (const [key, entry] of Object.entries(dict)) {
                for (const text of [entry].flat()) {
                    expect(text.trim(), key).not.toBe('')
                }
            }
        }
    })
})
