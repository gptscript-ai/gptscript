import { useContext } from '@/stores/context'

export default defineNuxtRouteMiddleware(async (to/* , from */) => {
  const sock = useSocket()
  await sock.setup()

  return true
})
