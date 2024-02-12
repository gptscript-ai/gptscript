import { useContext } from '@/stores/context'
import { addObject } from '@/utils/array'
import { get, set } from '@/utils/object'

export default defineNuxtPlugin(({ $pinia }) => {
  const w: any = window
  const ctx = useContext($pinia)

  w.ctx = ctx
  w.gpts = useGpts($pinia)
  w.runs = useRuns($pinia)
  w.socket = useSocket($pinia)
  w.prefs = usePrefs($pinia)

  w.get = get
  w.set = set
  w.ref = ref
  w.computed = computed
  w.reactive = reactive
  w.watchEffect = watchEffect
  w.addObject = addObject
  w.route = useRoute()
  w.router = useRouter()

  console.info('# Welcome to warp zone')

  return {
    provide: {
      get,
      set,
    },
  }
})
