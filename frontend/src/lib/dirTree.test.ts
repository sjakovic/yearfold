import {describe, expect, it} from 'vitest'
import {buildTree} from './dirTree'

describe('buildTree', () => {
    const tree = buildTree([
        {dir: '', count: 2},
        {dir: '2005/Trip', count: 3},
        {dir: '2005', count: 1},
        {dir: '2004/a/b', count: 4},
        {dir: '10', count: 1},
        {dir: '9', count: 1},
    ])

    it('creates the folders in between', () => {
        const y2004 = tree.children.find(n => n.name === '2004')!
        expect(y2004.path).toBe('2004')
        expect(y2004.children[0].children[0].path).toBe('2004/a/b')
    })

    it('counts files in subfolders too', () => {
        const count = (name: string) => tree.children.find(n => n.name === name)!.count
        expect(tree.count).toBe(12)
        expect(count('2005')).toBe(4)
        expect(count('2004')).toBe(4)
    })

    it('sorts names naturally', () => {
        expect(tree.children.map(n => n.name)).toEqual(['9', '10', '2004', '2005'])
    })

    it('is just the root for an empty library', () => {
        expect(buildTree([])).toMatchObject({path: '', count: 0, children: []})
    })
})
