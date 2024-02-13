import { useWebSocket, type UseWebSocketReturn, type WebSocketStatus } from '@vueuse/core'
import { defineStore } from 'pinia'

interface SocketState {
  url: string
  interval: number
  isSetup: boolean
  queue: Frame[]
  timer?: NodeJS.Timeout
  sock: UseWebSocketReturn<any>
}

export const useSocket = defineStore('socket', {
  state: () => {
    const url = useRuntimeConfig().public.api.replace(/^http/,'ws')
    const sock = useWebSocket(url, {
      immediate: false,
      autoReconnect: true,
    })

    return {
      url,
      interval: 250,
      isSetup: false,
      queue: [],
      timer: undefined,
      sock
    } as SocketState
  },

  actions: {
    async setup() {
      if ( this.isSetup ) {
        return
      }

      this.isSetup = true

      watch(() => this.sock.data, async (raw: string) => {
        const msg = JSON.parse(raw) as Frame
        this.queue.push(msg)
      })

      this.flush()
      this.sock.open()
    },

    flush() {
      if ( this.queue.length ) {
        const q = this.queue.splice(0, this.queue.length)

        nextTick(() => {
          for (const msg of q ) {
            this.handle(msg)
          }
        })
      }

      this.timer = setTimeout(() => {
        this.flush()
      }, this.interval)
    },

    async handle(f: Frame) {
      const runs = useRuns()
      const run = await runs.find(f.runID)

      if (!run ) {
        console.error('Frame received for unknown run', f.runID, f)
        return
      }

      if (!run.state ) {
        run.state = 'creating'
      }

      if ( f.type === 'runStart' ) {
        run.state = 'running'
        if ( f.program ) {
          run.program = f.program
        }
      } else if ( f.type === 'runFinish' ) {
        if ( f.err ) {
          run.state = 'error'
          run.err = f.err
        } else {
          run.state = 'finished'
          run.output = f.output || ''
        }
      } else if ( f.type.startsWith('call') ) {
        let call = run.calls.find(x => x.id === f.callContext.id)

        if ( !call ) {
          call = {
            id: f.callContext.id,
            parentID: f.callContext.parentID,
            tool: f.callContext.tool,
            messages: [],
            state: 'running',
            chatCompletionId: f.callContext.chatCompletionId,
            chatRequest: f.callContext.chatRequest
          }

          run.calls.push(call)
        }

        if ( f.type === 'callStart' ) {
          call.state = 'creating'
          call.input = f.content || ''
        } else if ( f.type === 'callChat' ) {
          call.state = 'running'

          if ( f.chatRequest ) {
            const more = (f.chatRequest.messages || []).slice(call.messages.length)
            call.messages.push(...more)
          }

          if ( f.chatResponse ) {
            call.messages.push(f.chatResponse)
          }
        } else if ( f.type === 'callContinue' ) {
          call.state = 'running'
        } else if ( f.type === 'callProgress' ) {
          call.state = 'running'

          if ( typeof f.content === 'string' ) {
            call.output = f.content
          }

        } else if ( f.type === 'callFinish' ) {
          call.state = 'finished'
          call.output = f.content
        } 
      }
    }
  }
})
