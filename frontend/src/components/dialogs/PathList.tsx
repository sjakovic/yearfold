import {useT} from '../../lib/i18n'

export function PathList({title, paths, total}: {title: string; paths: string[]; total: number}) {
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
