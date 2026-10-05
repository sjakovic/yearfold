import {useEffect, useState} from 'react'
import {api, TakeoutResult, TakeoutSummary} from '../../lib/api'
import {errorText} from '../../lib/format'
import {useT} from '../../lib/i18n'
import {Modal} from './Modal'

export function TakeoutDialog({onApplied, onClose}: {onApplied: (result: TakeoutResult) => void; onClose: () => void}) {
    const t = useT()
    const [summary, setSummary] = useState<TakeoutSummary | null>(null)
    const [error, setError] = useState('')
    const [busy, setBusy] = useState(false)

    useEffect(() => {
        api.TakeoutSummary().then(setSummary).catch(e => setError(errorText(e)))
    }, [])

    const apply = () => {
        setBusy(true)
        api.MergeTakeout().then(onApplied).catch(e => {
            setError(errorText(e))
            setBusy(false)
        })
    }

    return (
        <Modal title={t('takeoutTitle')} onClose={onClose}>
            <p className="muted">{t('takeoutText')}</p>
            {error && <p className="error">{error}</p>}
            {!summary && !error && <p className="muted">{t('loading')}</p>}
            {summary && (
                <p>{summary.sidecars > 0 ? t('takeoutCounts', {n: summary.sidecars, dates: summary.dates}) : t('takeoutNone')}</p>
            )}
            <div className="modal-actions">
                <button onClick={onClose}>{t('cancel')}</button>
                <button className="primary" disabled={busy || !summary || summary.sidecars === 0} onClick={apply}>
                    {t('takeoutConfirm')}
                </button>
            </div>
        </Modal>
    )
}
