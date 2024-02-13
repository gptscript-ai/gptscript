<script setup lang="ts">

const prefs = usePrefs()

interface Props {
  run: Run
  call: Call
  msg: ChatMessage
  referredTo: MapBool
  callMap: Record<string, Call>
  depth: number
}

const { run, /*call,*/ msg, referredTo, depth=0 } = defineProps<Props>()
</script>

<template>
  <div>
    <i v-if="prefs.debug" class="text-blue-400">{{msg}}</i>

    <UCard v-if="'output' in msg">
      {{msg.output}}
    </UCard>
    <UAlert
      v-else-if="'err' in msg"
      icon="i-heroicons-exclamation-triangle"
      color="red"
      class="my-2"
      title="Error"
      variant="solid"
      :description="msg.err"
    />
    <template v-else>
      <UCard v-if="typeof msg.content === 'string'" class="mt-2">
        <b>{{ucFirst(msg.role)}}:&nbsp;</b>
        <span class="whitespace-pre-wrap">{{`${msg.content}`.trim()}}</span>
      </UCard>
      <template v-else>
        <UCard v-for="(c, idx) in msg.content" :key="idx">
          <template v-if="'text' in c">
            <b>{{ucFirst(msg.role)}}:&nbsp;</b>
            {{c.text}}
          </template>
          <template v-else>
            <b>Tool Call: </b>
            <i>{{c.toolCall.function?.name || '<?>'}}({{c.toolCall.function?.arguments || '{}'}})</i>
            <i class="i-heroicons-arrow-right align-text-bottom"/> {{prefs.mapCall(c.toolCall.id)}}

            <Call
              v-if="c.toolCall.id && callMap[c.toolCall.id]"
              :run="run"
              :call="callMap[c.toolCall.id]"
              :referred-to="referredTo"
              :call-map="callMap"
              :depth="depth+1"
              class="mt-5"
            />
            <UCard v-else>
              <i>Waiting for call ({{c.toolCall.index}})…</i>
            </UCard>
          </template>
        </UCard>
      </template>
    </template>
  </div>
</template>
