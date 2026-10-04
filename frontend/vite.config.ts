import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  // No SPA fallback: unknown paths must 404 so that, in `wails dev`, requests
  // for /thumb/* and /file/* reach the Go asset handler.
  appType: 'mpa',
})
