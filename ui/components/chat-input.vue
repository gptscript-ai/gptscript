<script setup lang="ts">

declare global {
  interface ChatEvent {
    message: string,
    cb: () => void
  }
}

// eslint-disable-next-line func-call-spacing
const emit = defineEmits<{
  (e: 'message', value: ChatEvent): void
}>()

const waiting = ref(false)
const message = ref('')

async function send() {
  const ev: ChatEvent = {
    message: message.value,
    cb: () => {
      waiting.value = false
    }
  }

  waiting.value = true
  message.value = ''
  console.log('Send', ev)
  emit('message', ev)
}

function keypress(e: KeyboardEvent) {
  if ( e.code === 'Enter' && !e.shiftKey ) {
    e.preventDefault()
    e.stopPropagation()
    send()
  }
}
</script>

<template>
  <div class="input">
    <UTextarea v-if="waiting" placeholder="Thinking… Please wait a surprisingly long time"/>
    <UTextarea v-else placeholder="Say something…" v-model="message" @keypress="keypress" />
    <!-- <UButton :disabled="waiting || !message" @click="send">
      Send
    </UButton> -->
  </div>
</template>

<style lang="scss" scoped>
.input {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 2rem;

}
</style>
