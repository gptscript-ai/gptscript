import dotenv from 'dotenv'
import pkg from './package.json'

dotenv.config()

const port = 9091
const isDev = process.env.NODE_ENV === 'development'
const api = process.env.NUXT_PUBLIC_API || (isDev ? 'http://localhost:9090/' : '/')

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  app: {
    baseURL: '/ui',
  },
  build: {
    analyze: {
      filename: '.nuxt/stats/{name}.html',
      template: 'treemap',
      brotliSize: true,
      gzipSize: true,
    },
  },
  colorMode: { classSuffix: '' },
  css: [
    '@/assets/styles/app.scss',
  ],
  components: true,
  devServer: {
    port,
  },
  devtools: { enabled: true },
  modules: [
    '@nuxt/ui',
    '@pinia/nuxt',
    '@nuxtjs/tailwindcss'
  ],
  nitro: { sourceMap: true },
  runtimeConfig: {
    public: {
      api: api.replace(/\/+$/,'')+'/',
    },
  },
  sourcemap: true,
  ssr: false,
  typescript: { strict: true },
  vite: {
    build: {
      manifest: true,
      rollupOptions: {
        output: {
          banner: `/* GPTScript ${pkg.version} */`,
        },
      },
      sourcemap: true,
      ssrManifest: true,
    },

    css: { devSourcemap: true },
  },
})
