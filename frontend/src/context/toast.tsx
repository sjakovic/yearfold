import {createContext, ReactNode, useCallback, useContext, useMemo, useState} from 'react'
import {errorText} from '../lib/format'
import {useT} from '../lib/i18n'

const TOAST_MS = 5000

interface Toast {
    notify: (message: string) => void
    fail: (error: unknown) => void
}

const ToastContext = createContext<Toast>({notify: () => {}, fail: () => {}})

export function ToastProvider({children}: {children: ReactNode}) {
    const t = useT()
    const [message, setMessage] = useState('')

    const notify = useCallback((text: string) => {
        setMessage(text)
        window.setTimeout(() => setMessage(current => (current === text ? '' : current)), TOAST_MS)
    }, [])
    const fail = useCallback((error: unknown) => notify(t('error', {msg: errorText(error)})), [notify, t])
    const value = useMemo(() => ({notify, fail}), [notify, fail])

    return (
        <ToastContext.Provider value={value}>
            {children}
            {message && <div className="toast">{message}</div>}
        </ToastContext.Provider>
    )
}

export const useToast = () => useContext(ToastContext)
