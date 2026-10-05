import {createContext, useContext} from 'react'
import {en, Entry, Key} from './en'
import {sr} from './sr'

export type {Key}

export type Lang = 'en' | 'sr'

export const LANGUAGES: {code: Lang; label: string}[] = [
    {code: 'en', label: 'English'},
    {code: 'sr', label: 'Српски (ћирилица)'},
]

const dictionaries: Record<Lang, Record<Key, Entry>> = {en, sr}

export type Translate = (key: Key, params?: Record<string, string | number>) => string

export function translator(lang: Lang): Translate {
    const dict = dictionaries[lang] ?? en
    const rules = new Intl.PluralRules(lang)
    return (key, params = {}) => {
        let entry = dict[key]
        if (Array.isArray(entry)) {
            const category = rules.select(Number(params.n ?? 0))
            const index = category === 'one' ? 0 : category === 'few' && entry.length > 2 ? 1 : entry.length - 1
            entry = entry[index]
        }
        return entry.replace(/\{(\w+)\}/g, (_, name) => String(params[name] ?? ''))
    }
}

export function normalizeLang(code: string): Lang {
    return code === 'sr' ? 'sr' : 'en'
}

export const I18n = createContext<Translate>(translator('en'))

export const useT = () => useContext(I18n)
