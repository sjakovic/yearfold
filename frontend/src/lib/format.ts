export function formatDate(ts: number): string {
    if (!ts) return ''
    return new Date(ts * 1000).toISOString().slice(0, 16).replace('T', ' ')
}

export function toDateInput(ts: number): string {
    return ts ? new Date(ts * 1000).toISOString().slice(0, 16) : ''
}

export function fromDateInput(value: string): number {
    return Math.floor(Date.parse(`${value}:00Z`) / 1000)
}

export function formatSize(n: number): string {
    if (n < 1024) return `${n} B`
    const units = ['KB', 'MB', 'GB', 'TB']
    let v = n / 1024
    let i = 0
    while (v >= 1024 && i < units.length - 1) {
        v /= 1024
        i++
    }
    return `${v.toFixed(v < 10 ? 1 : 0)} ${units[i]}`
}

export function extOf(name: string): string {
    const i = name.lastIndexOf('.')
    return i > 0 ? name.slice(i + 1).toUpperCase() : 'FILE'
}

export function errorText(e: unknown): string {
    return e instanceof Error ? e.message : String(e)
}
