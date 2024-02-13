import type { Component } from 'vue'
import type { JsonDict, JsonValue } from '@/utils/object'

declare global {
  type MapBool = Record<string,boolean>

  interface SelectOption {
    label: string
    value: string
  }

  interface Gpt {
    name: string
    entryToolId: string
    toolSet: Record<string, Tool>
  }

  interface Property {
    type: "string"
    description: string
    default?: string
  }

  interface ArgSchema {
    type: 'object'
    properties: Record<string,Property>
    required: string[]
  }

  type Args = string | Record<string, string>

  interface Tool {
    id: string
    name?: string
    description: string
    arguments: ArgSchema
    instructions: string
    tools: string[]
    localTools: Record<string,string>
    toolMapping: Record<string,string>
    modelName: string
    source: {
      file: string
      lineNo: number
    }
  }

  type State = 'creating'|'running'|'finished'|'error'

  interface Run {
    id: string
    state: State
    calls: Call[]
    err?: string
    output?: string
    program? : Gpt
  }

  interface Call {
    id: string
    parentID?: string
    chatCompletionId?: string
    state: State
    messages: ChatMessage[]
    tool?: Tool
    chatRequest?: JsonDict
    input?: Args
    output?: string
    showSystemMessages?: boolean
  }

  interface BaseFrame {
    type: string
    time: string
    runID: string
  }

  interface RunStartFrame extends BaseFrame {
    type: "runStart"
    program: Gpt
  }

  interface RunFinishFrame extends BaseFrame {
    type: "runFinish"
    input: Args

    err?: string
    output?: string
  }

  interface CallFrame extends BaseFrame {
    callContext: Call
    input: Args
  }

  interface CallStartFrame extends CallFrame {
    type: "callStart"
  }

  interface CallChatFrame extends CallFrame {
    type: "callChat"
    chatCompletionId: string
    chatRequest?: ChatRequest
    chatResponse?: ChatMessage
    chatResponseCached?: boolean
  }

  interface CallProgressFrame extends CallFrame {
    type: "callProgress"
    chatCompletionId: string
    content: string
  }

  interface CallContinueFrame extends CallFrame {
    type: "callContinue"
    toolResults: number
  }

  interface CallFinishFrame extends CallFrame {
    type: "callFinish"
    content: string
  }

  type Frame = RunStartFrame | RunFinishFrame | CallStartFrame | CallChatFrame | CallProgressFrame | CallContinueFrame | CallFinishFrame

  interface ChatRequest {
    max_tokens: number
    messages: ChatMessage[]
    model: string
    temperature: string
    tools?: ChatTool[]
  }

  interface ChatText {
    text: string
  }

  interface ChatToolCall {
    toolCall: {
      id: string
      index: number
      type: "function"
      function: {
        name: string,
        arguments: string
      }
    }
  }

  interface ChatToolMessage {
    role: "system"|"assistant"|"user"|"tool"
    content: string | (ChatToolCall|ChatText)[]
  }

  interface ChatErrorMessage {
    err: string
  }

  interface ChatOutputMessage {
    output: string
  }

  type ChatMessage = ChatToolMessage | ChatOutputMessage | ChatErrorMessage

  interface ChatToolFunction {
    type: "function"
    function: {
      name: string
      description: string
      parameters: {
        type: "object"
        properties: Record<string, ChatProperty>
      }
    }
  }

  type ChatTool = ChatToolFunction

  interface ChatProperty {
    type: string
    description: string
  }
}
