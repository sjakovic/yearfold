import {useState} from 'react'
import {Album} from '../../lib/api'
import {useT} from '../../lib/i18n'
import {Modal} from './Modal'

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
