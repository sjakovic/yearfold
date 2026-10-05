import {useState} from 'react'
import {useT} from '../../lib/i18n'
import {Modal} from './Modal'

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
