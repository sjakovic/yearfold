import {useEffect, useState} from 'react'
import {api, ChangeReport} from '../../lib/api'
import {errorText} from '../../lib/format'
import {useT} from '../../lib/i18n'
import {Modal} from './Modal'
import {PathList} from './PathList'

export function ChangesDialog({onApplied, onClose}: {onApplied: () => void; onClose: () => void}) {
    const t = useT()
    const [report, setReport] = useState<ChangeReport | null>(null)
    const [error, setError] = useState('')
    const [busy, setBusy] = useState(false)

    useEffect(() => {
        api.CheckChanges().then(setReport).catch(e => setError(errorText(e)))
    }, [])

    const total = report ? report.newCount + report.modifiedCount + report.missingCount + report.movedCount : 0
    const apply = () => {
        setBusy(true)
        api.ApplyChanges().then(onApplied).catch(e => {
            setError(errorText(e))
            setBusy(false)
        })
    }

    return (
        <Modal title={t('checkChanges')} onClose={onClose} wide>
            {error && <p className="error">{error}</p>}
            {!report && !error && <p className="muted">{t('changesScanning')}</p>}
            {report && total === 0 && <p>{t('changesNone')}</p>}
            {report && total > 0 && (
                <div className="report">
                    <PathList title={t('changesNew')} paths={report.new} total={report.newCount}/>
                    <PathList title={t('changesModified')} paths={report.modified} total={report.modifiedCount}/>
                    <PathList title={t('changesMissing')} paths={report.missing} total={report.missingCount}/>
                    <PathList title={t('changesMoved')} total={report.movedCount}
                              paths={report.moved.map(m => `${m.from}  →  ${m.to}`)}/>
                </div>
            )}
            <div className="modal-actions">
                <button onClick={onClose}>{total > 0 ? t('cancel') : t('close')}</button>
                {total > 0 && <button className="primary" disabled={busy} onClick={apply}>{t('changesApply')}</button>}
            </div>
        </Modal>
    )
}
