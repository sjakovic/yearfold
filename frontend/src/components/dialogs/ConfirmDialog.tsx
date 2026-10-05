import {useT} from '../../lib/i18n'
import {Modal} from './Modal'

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
