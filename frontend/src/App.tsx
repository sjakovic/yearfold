import LanguagePicker from './components/LanguagePicker'
import LibraryScreen from './components/LibraryScreen'
import Welcome from './components/Welcome'
import {LibraryProvider} from './context/library'
import {ToastProvider, useToast} from './context/toast'
import {useSession} from './hooks/useSession'
import {api, AppState} from './lib/api'
import {I18n, Lang, useT} from './lib/i18n'

export default function App() {
    const {state, setState, lang, t} = useSession()
    return (
        <I18n.Provider value={t}>
            <ToastProvider>
                <Shell state={state} lang={lang} onState={setState}/>
            </ToastProvider>
        </I18n.Provider>
    )
}

function Shell({state, lang, onState}: {state: AppState | null; lang: Lang; onState: (state: AppState) => void}) {
    const t = useT()
    const {fail} = useToast()

    if (!state) {
        return <div className="welcome"><p className="muted">{t('loading')}</p></div>
    }

    const open = (opening: Promise<AppState>) => opening.then(onState).catch(fail)
    const languagePicker = (
        <LanguagePicker lang={lang} onChange={code => api.SetLanguage(code).then(onState).catch(fail)}/>
    )

    if (!state.root) {
        return (
            <Welcome recent={state.recent} version={state.version} languagePicker={languagePicker}
                     onPick={() => open(api.PickFolder())} onOpen={root => open(api.OpenLibrary(root))}/>
        )
    }
    return (
        <LibraryProvider key={state.root} root={state.root}>
            <LibraryScreen version={state.version} languagePicker={languagePicker}
                           onOpenFolder={() => open(api.PickFolder())}/>
        </LibraryProvider>
    )
}
