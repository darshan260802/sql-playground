import react from '@vitejs/plugin-react'
import { defineConfig, type Plugin } from 'vite'

// Cloudflare's Rocket Loader rewrites <script> tags and breaks module scripts,
// leaving a blank page. data-cfasync="false" tells it to leave a script alone.
const skipRocketLoader: Plugin = {
  name: 'skip-rocket-loader',
  transformIndexHtml: {
    order: 'post',
    handler: (html) => html.replace(/<script\b(?![^>]*data-cfasync)/g, '<script data-cfasync="false"'),
  },
}

// The Go API serves /api; in development Vite proxies to it.
export default defineConfig({
  plugins: [react(), skipRocketLoader],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:8080',
    },
  },
  build: {
    outDir: 'dist',
    rolldownOptions: {
      output: {
        // libraries change rarely: separate chunks stay cached across deploys
        codeSplitting: {
          groups: [
            { name: 'codemirror', test: /node_modules[\\/](@codemirror|@lezer|codemirror|crelt|style-mod|w3c-keyname)/ },
            { name: 'react', test: /node_modules[\\/](react|react-dom|react-router|react-router-dom|scheduler)[\\/]/ },
            { name: 'ui', test: /node_modules[\\/](motion|motion-dom|motion-utils|framer-motion|sonner|lucide-react|@tanstack|marked|zustand)/ },
          ],
        },
      },
    },
  },
})
