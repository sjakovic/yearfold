import {useEffect, useRef, useState} from 'react'
import {EventsOn} from '../../wailsjs/runtime/runtime'
import {Progress} from '../lib/api'

const IDLE: Progress = {phase: '', done: 0, total: 0, seq: 0}

export function useProgress(): Progress {
    const [progress, setProgress] = useState(IDLE)
    useEffect(
        () => EventsOn('progress', (next: Progress) => setProgress(prev => (next.seq > prev.seq ? next : prev))),
        [],
    )
    return progress
}

export function useOnChanged(callback: () => void) {
    const latest = useRef(callback)
    useEffect(() => {
        latest.current = callback
    }, [callback])
    useEffect(() => EventsOn('changed', () => latest.current()), [])
}
