import {Item} from '../lib/api'
import {formatSize} from '../lib/format'
import {Thumb} from './Grid'
import {useT} from '../lib/i18n'

const MAX_GROUPS = 300

interface Props {
    groups: Item[][]
    selection: Set<number>
    hashing: boolean
    onToggle: (id: number) => void
    onOpen: (id: number) => void
}

export default function DuplicatesView({groups, selection, hashing, onToggle, onOpen}: Props) {
    const t = useT()
    return (
        <div className="grid-scroller">
            {groups.length === 0 && (
                <div className="empty">{t('dupNone')}{hashing ? t('dupHashing') : ''}</div>
            )}
            {groups.slice(0, MAX_GROUPS).map(group => (
                <div key={group[0].hash} className="dup-group">
                    <div className="muted">{t('dupGroup', {n: group.length})} · {formatSize(group[0].size)}</div>
                    <div className="dup-row">
                        {group.map(it => (
                            <div key={it.id} className={`cell wide ${selection.has(it.id) ? 'selected' : ''}`}
                                 onClick={() => onToggle(it.id)} onDoubleClick={() => onOpen(it.id)}>
                                <div className="thumb"><Thumb item={it}/></div>
                                <div className="caption" title={it.relPath}>{it.relPath}</div>
                            </div>
                        ))}
                    </div>
                </div>
            ))}
            {groups.length > MAX_GROUPS && (
                <div className="empty">{t('dupLimit', {shown: MAX_GROUPS, n: groups.length})}</div>
            )}
        </div>
    )
}
