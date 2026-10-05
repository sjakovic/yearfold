import {ReactNode, useEffect, useState} from 'react'
import {
    Album, api, ChangeReport, errorText, fromDateInput, OrganizePreview, TakeoutResult, TakeoutSummary, toDateInput,
} from './api'
import {Key, useT} from './i18n'

export function Modal({title, onClose, children, wide}: {title: string; onClose: () => void; children: ReactNode; wide?: boolean}) {
    useEffect(() => {
        const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose()
        window.addEventListener('keydown', onKey)
        return () => window.removeEventListener('keydown', onKey)
    }, [onClose])
    return (
        <div className="overlay" onMouseDown={e => e.target === e.currentTarget && onClose()}>
            <div className={`modal ${wide ? 'wide' : ''}`}>
                <h3>{title}</h3>
                {children}
            </div>
        </div>
    )
}

interface PromptProps {
    title: string
    label?: string
    initial?: string
    placeholder?: string
    suggestions?: string[]
    confirmLabel: string
    allowEmpty?: boolean
    onSubmit: (value: string) => void
    onClose: () => void
}

export function PromptDialog(p: PromptProps) {
    const t = useT()
    const [value, setValue] = useState(p.initial ?? '')
    const ok = p.allowEmpty || value.trim() !== ''
    const submit = () => ok && p.onSubmit(value.trim())
    return (
        <Modal title={p.title} onClose={p.onClose}>
            {p.label && <p className="muted">{p.label}</p>}
            <input autoFocus list="prompt-suggestions" value={value} placeholder={p.placeholder}
                   onChange={e => setValue(e.target.value)} onKeyDown={e => e.key === 'Enter' && submit()}/>
            <datalist id="prompt-suggestions">{p.suggestions?.map(s => <option key={s} value={s}/>)}</datalist>
            <div className="modal-actions">
                <button onClick={p.onClose}>{t('cancel')}</button>
                <button className="primary" disabled={!ok} onClick={submit}>{p.confirmLabel}</button>
            </div>
        </Modal>
    )
}

interface ConfirmProps {
    title: string
    message: string
    confirmLabel: string
    danger?: boolean
    onConfirm: () => void
    onClose: () => void
}

export function ConfirmDialog(p: ConfirmProps) {
    const t = useT()
    return (
        <Modal title={p.title} onClose={p.onClose}>
            <p>{p.message}</p>
            <div className="modal-actions">
                <button onClick={p.onClose}>{t('cancel')}</button>
                <button className={p.danger ? 'danger' : 'primary'} onClick={p.onConfirm}>{p.confirmLabel}</button>
            </div>
        </Modal>
    )
}

export function AlbumDialog({albums, count, onPick, onCreate, onClose}: {
    albums: Album[]
    count: number
    onPick: (id: number) => void
    onCreate: (name: string) => void
    onClose: () => void
}) {
    const t = useT()
    const [name, setName] = useState('')
    return (
        <Modal title={t('albumDialogTitle', {n: count})} onClose={onClose}>
            <p className="muted">{t('albumDialogNote')}</p>
            <div className="pick-list">
                {albums.map(a => (
                    <button key={a.id} onClick={() => onPick(a.id)}>{a.name} <span className="count">{a.count}</span></button>
                ))}
            </div>
            <div className="inline-form">
                <input autoFocus value={name} placeholder={t('albumDialogNew')} onChange={e => setName(e.target.value)}
                       onKeyDown={e => e.key === 'Enter' && name.trim() && onCreate(name.trim())}/>
                <button className="primary" disabled={!name.trim()} onClick={() => onCreate(name.trim())}>{t('albumDialogCreate')}</button>
            </div>
            <div className="modal-actions">
                <button onClick={onClose}>{t('cancel')}</button>
            </div>
        </Modal>
    )
}

export function DateDialog({title, initial, onSubmit, onClose}: {
    title: string
    // Unix seconds of the current date, or 0 when the file has none.
    initial: number
    onSubmit: (unix: number) => void
    onClose: () => void
}) {
    const t = useT()
    const [value, setValue] = useState(toDateInput(initial))
    const unix = value ? fromDateInput(value) : NaN
    const valid = Number.isFinite(unix)
    return (
        <Modal title={title} onClose={onClose}>
            <p className="muted">{t('dateNote')}</p>
            <input autoFocus type="datetime-local" value={value} onChange={e => setValue(e.target.value)}
                   onKeyDown={e => e.key === 'Enter' && valid && onSubmit(unix)}/>
            <div className="modal-actions">
                <button onClick={onClose}>{t('cancel')}</button>
                <button className="primary" disabled={!valid} onClick={() => onSubmit(unix)}>{t('dateConfirm')}</button>
            </div>
        </Modal>
    )
}

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

function PathList({title, paths, total}: {title: string; paths: string[]; total: number}) {
    const t = useT()
    if (total === 0) return null
    return (
        <details open={total <= 20}>
            <summary>{title}: {total}</summary>
            <ul className="path-list">
                {paths.map(p => <li key={p}>{p}</li>)}
                {total > paths.length && <li className="muted">{t('more', {n: total - paths.length})}</li>}
            </ul>
        </details>
    )
}

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
        setPreview(null)
        setError('')
        api.PlanOrganize(layout).then(setPreview).catch(e => setError(errorText(e)))
    }, [layout])

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
                        <input type="radio" name="layout" checked={layout === l.value} onChange={() => setLayout(l.value)}/>
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
