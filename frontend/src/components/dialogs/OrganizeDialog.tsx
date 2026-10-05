import {useEffect, useState} from 'react'
import {api, OrganizePreview} from '../../lib/api'
import {errorText} from '../../lib/format'
import {Key, useT} from '../../lib/i18n'
import {Modal} from './Modal'

const LAYOUTS: {value: string; label: Key}[] = [
    {value: 'folder', label: 'orgLayoutFolder'},
    {value: 'year', label: 'orgLayoutYear'},
    {value: 'month', label: 'orgLayoutMonth'},
]

export function OrganizeDialog({onApplied, onClose}: {onApplied: (moved: number) => void; onClose: () => void}) {
    const t = useT()
    const [layout, setLayout] = useState(LAYOUTS[0].value)
    const [preview, setPreview] = useState<OrganizePreview | null>(null)
    const [error, setError] = useState('')
    const [busy, setBusy] = useState(false)

    useEffect(() => {
        api.PlanOrganize(layout).then(setPreview).catch(e => setError(errorText(e)))
    }, [layout])

    const choose = (value: string) => {
        setPreview(null)
        setError('')
        setLayout(value)
    }

    const apply = () => {
        setBusy(true)
        api.ApplyOrganize(layout).then(onApplied).catch(e => {
            setError(errorText(e))
            setBusy(false)
        })
    }

    return (
        <Modal title={t('orgTitle')} onClose={onClose} wide>
            <p className="muted">
                {t('orgText')}
            </p>
            <div className="options">
                {LAYOUTS.map(l => (
                    <label key={l.value} className="check">
                        <input type="radio" name="layout" checked={layout === l.value} onChange={() => choose(l.value)}/>
                        {t(l.label)}
                    </label>
                ))}
            </div>
            {layout === 'folder' && <p className="muted">{t('orgLayoutFolderHint')}</p>}
            {error && <p className="error">{error}</p>}
            {!preview && !error && <p className="muted">{t('orgPlanning')}</p>}
            {preview && (
                <div className="report">
                    <p>
                        {t('orgToMove')} <b>{preview.count}</b>.
                        {preview.noDate > 0 && <> {t('orgSkipped')} <b>{preview.noDate}</b>.</>}
                    </p>
                    <ul className="path-list">
                        {preview.moves.map(m => <li key={m.id}>{m.from}  →  {m.to}/</li>)}
                        {preview.count > preview.moves.length &&
                            <li className="muted">{t('more', {n: preview.count - preview.moves.length})}</li>}
                    </ul>
                </div>
            )}
            <div className="modal-actions">
                <button onClick={onClose}>{t('cancel')}</button>
                <button className="primary" disabled={busy || !preview || preview.count === 0} onClick={apply}>
                    {t('moveTitle', {files: t('files', {n: preview?.count ?? 0})})}
                </button>
            </div>
        </Modal>
    )
}
