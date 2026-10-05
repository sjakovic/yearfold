import {Progress} from '../lib/api'
import {Key, useT} from '../lib/i18n'

const PHASES: Record<string, Key> = {
    scan: 'phaseScan',
    meta: 'phaseMeta',
    hash: 'phaseHash',
}

export default function StatusBar({progress}: {progress: Progress}) {
    const t = useT()
    if (!progress.phase) return null

    const label = PHASES[progress.phase] ? t(PHASES[progress.phase]) : progress.phase
    const known = progress.total > 0
    return (
        <footer className="status">
            {label} {known ? `${progress.done} / ${progress.total}` : progress.done}
            {known && <progress value={progress.done} max={progress.total}/>}
        </footer>
    )
}
