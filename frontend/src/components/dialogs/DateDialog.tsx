import {useState} from 'react'
import {fromDateInput, toDateInput} from '../../lib/format'
import {useT} from '../../lib/i18n'
import {Modal} from './Modal'

export function DateDialog({title, initial, onSubmit, onClose}: {
    title: string
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
