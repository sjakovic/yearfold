import {ReactNode} from 'react'
import {useLibrary} from '../context/library'
import {useToast} from '../context/toast'
import ActionBar from './ActionBar'
import DetailView from './DetailView'
import ActionDialogs from './dialogs/ActionDialogs'
import DuplicatesView from './DuplicatesView'
import Grid from './Grid'
import Sidebar from './Sidebar'
import StatusBar from './StatusBar'
import Toolbar from './Toolbar'

interface Props {
    version: string
    languagePicker: ReactNode
    onOpenFolder: () => void
}

export default function LibraryScreen(props: Props) {
    const lib = useLibrary()
    const {notify} = useToast()
    const {items, detailIndex} = lib
    const detailItem = detailIndex !== null ? items[detailIndex] : undefined

    return (
        <div className="app">
            <Sidebar/>

            <main className="main">
                <Toolbar {...props}/>
                <ActionBar/>
                {lib.view.type === 'duplicates' ? (
                    <DuplicatesView groups={lib.groups} selection={lib.selection} hashing={lib.progress.phase !== ''}
                                    onToggle={lib.toggle}
                                    onOpen={id => lib.openDetail(items.findIndex(item => item.id === id))}/>
                ) : (
                    <Grid items={items} total={lib.total} selection={lib.selection} onSelect={lib.select}
                          onOpen={lib.openDetail} onLoadMore={lib.loadMore}/>
                )}
                <StatusBar progress={lib.progress}/>
            </main>

            {detailItem && detailIndex !== null && (
                <DetailView key={detailItem.id} item={detailItem} albums={lib.overview?.albums ?? []}
                            tagNames={lib.overview?.tags.map(tag => tag.name) ?? []}
                            hasPrev={detailIndex > 0} hasNext={detailIndex < items.length - 1}
                            onPrev={() => lib.openDetail(detailIndex - 1)}
                            onNext={() => lib.openDetail(detailIndex + 1)}
                            onClose={() => lib.openDetail(null)}
                            onTrash={id => lib.openDialog({type: 'trash', ids: [id]})}
                            onSetDate={(id, initial) => lib.openDialog({type: 'date', ids: [id], initial})}
                            onChanged={lib.refreshOverview} onError={notify}/>
            )}

            <ActionDialogs/>
        </div>
    )
}
