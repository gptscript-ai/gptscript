import dotenv from 'dotenv'

dotenv.config()

export const api = cleanUrl(process.env.NUXT_PUBLIC_API || process.env.API || 'http://localhost:9090')
export const isDev = process.env.NODE_ENV === 'development'

export function cleanUrl(url: string) {
  return (url || '').trim().replace(/\/+$/, '')
}
