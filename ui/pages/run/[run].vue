<script lang="ts" setup>
  const route = useRoute()
  const prefs = usePrefs()

  let id = fromArray(route.params.run) as string

  definePageMeta({
    middleware: defineNuxtRouteMiddleware(async (to/*, from*/) => {
      const toId = fromArray(to.params.run) as string
      const toRun = (await useRuns().find(toId))
      if ( !toRun ) {
        window.location.href = '/'
      }
    })
  })

  const run = (await useRuns().find(id))!

  const callMap = reactive<Record<string, Call>>({})
  const referredTo = reactive<MapBool>({})

  watch(run.calls, (neu) => {
    for ( const call of neu ) {
      callMap[call.id] = call
      if ( prefs.allExpanded ) {
        prefs.expand(call.id)
      }

      for ( const msg of call.messages ) {
        if ( 'role' in msg &&  msg.role === 'assistant' && typeof msg.content !== 'string' && isArray(msg.content) ) {
          for ( const line of msg.content ) {
            if ( 'toolCall' in line ) {
              referredTo[line.toolCall.id] = true
            }
          }
        }
      }
    }
  }, {immediate: true})

  const topLevel = computed(() => {
    return run.calls.filter((x) => {
      if ( x.parentID ) {
        if ( referredTo[x.id] ) {
          return false
        }

        if ( callMap[x.parentID] ) {
          return false
        }
      }

      return true
    })
  })

  if ( !run ) {
    navigateTo('/')
  }

  function toggleAll() {
    if ( prefs.allExpanded ) {
      prefs.collapseAll()
    } else {
      prefs.expandAll(Object.keys(callMap))
    }
  }

  function edit() {
    const p = run.program!

    if (!p?.name || !p?.toolSet ) {
      return
    }

    const gpt = p.name
    const tools = p.toolSet[p.entryToolId].localTools
    let tool = ''
    let input = run.calls?.[0]?.input

    for ( const k in tools ) {
      if ( tools[k] === p.entryToolId ) {
        tool = k
      }
    }

    navigateTo({
      name: 'gpt-gpt',
      params: {
        gpt,
      },
      query: {
        tool,
        input,
      },
    })
  }
</script>

<template>
  <div v-if="run">
    <div class="grid lg:grid-cols-[1fr,375px] grid-cols-1">
      <div>
        <h1 class="text-xl">
          {{run.program?.name}} #{{id}}
        </h1>
      </div>
      <div class="mt-2 lg:mt-0 lg:text-right">
        <UButton
          size="sm"
          :icon="prefs.allExpanded ? 'i-heroicons-minus' : 'i-heroicons-plus'"
          :label="prefs.allExpanded ? 'Collapse All' : 'Expand All'"
          @click="toggleAll()"
        />

        <UButton size="sm" icon="i-heroicons-pencil" label="Run Again…" @click="edit" class="ml-2"/>

        <UBadge :color="colorForState(run.state)" size="lg" class="align-top ml-2" variant="subtle">
          <i :class="iconForState(run.state)"/>&nbsp;{{ucFirst(run.state)}}
        </UBadge>
      </div>
    </div>

    <UAlert
      v-if="run?.err"
      icon="i-heroicons-exclamation-triangle"
      color="red"
      class="my-2"
      title="Error"
      variant="solid"
      :description="run.err"
    />

    <UDivider class="my-4" />
    <UCard class="p-2">
      <Content v-if="run.output || run.state === 'finished'" v-model="run.output"/>
      <UProgress v-else animation="swing" />
    </UCard>

    <UDivider class="my-4" />

    <template v-for="c in topLevel" :key="c.id">
      <Call
        :call="c"
        :run="run"
        :referred-to="referredTo"
        :call-map="callMap"
        :depth="0"
      />
    </template>
  </div>
</template>
