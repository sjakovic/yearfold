import {ReactNode, useEffect} from 'react'

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
