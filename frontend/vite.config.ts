import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // unknown paths must 404 so wails dev can serve /thumb and /file
  appType: 'mpa',
})
