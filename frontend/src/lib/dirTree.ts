export interface DirNode {
    name: string
    path: string
    count: number
    children: DirNode[]
    parent?: DirNode
}

export function buildTree(dirs: {dir: string; count: number}[]): DirNode {
    const root: DirNode = {name: '', path: '', count: 0, children: []}
    const index = new Map<string, DirNode>([['', root]])
    const ensure = (path: string): DirNode => {
        const hit = index.get(path)
        if (hit) return hit
        const i = path.lastIndexOf('/')
        const parent = ensure(i < 0 ? '' : path.slice(0, i))
        const node: DirNode = {name: path.slice(i + 1), path, count: 0, children: [], parent}
        parent.children.push(node)
        index.set(path, node)
        return node
    }
    for (const d of dirs) {
        for (let node: DirNode | undefined = ensure(d.dir); node; node = node.parent) {
            node.count += d.count
        }
    }
    const sort = (n: DirNode) => {
        n.children.sort((a, b) => a.name.localeCompare(b.name, undefined, {numeric: true}))
        n.children.forEach(sort)
    }
    sort(root)
    return root
}
