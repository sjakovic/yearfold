import {useEffect, useMemo, useState} from 'react'
import {api, AppState} from '../lib/api'
import {normalizeLang, translator} from '../lib/i18n'

export function useSession() {
    const [state, setState] = useState<AppState | null>(null)

    useEffect(() => {
        api.GetState().then(initial => {
            if (!initial.root && initial.recent.length > 0) {
                api.OpenLibrary(initial.recent[0]).then(setState).catch(() => setState(initial))
            } else {
                setState(initial)
            }
        })
    }, [])

    const lang = normalizeLang(state?.language ?? 'en')
    const t = useMemo(() => translator(lang), [lang])

    useEffect(() => {
        document.documentElement.lang = lang === 'sr' ? 'sr-Cyrl' : 'en'
    }, [lang])

    return {state, setState, lang, t}
}
