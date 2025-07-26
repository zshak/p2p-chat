import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],

  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: false,
    minify: 'esbuild',
    target: 'es2015',

    rollupOptions: {
      output: {
        manualChunks: {
          vendor: ['react', 'react-dom'],
          mui: ['@mui/material', '@mui/icons-material'],
          router: ['react-router-dom']
        }
      }
    }
  },

  // Dev
  server: {
    port: 5173,
    host: true,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:59579',
        changeOrigin: true,
      },
      '/ws': {
        target: 'ws://127.0.0.1:59579',
        ws: true,
        changeOrigin: true
      }
    }
  },

  base: '/',

  define: {
    'import.meta.env.VITE_BACKEND_API_BASE_URL': JSON.stringify(process.env.NODE_ENV === 'production' ? '/api' : undefined)
  }
})