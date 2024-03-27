import { exec } from 'child_process'
import type { ChildProcess } from 'child_process'

type Request = {
    prompt: string;
    pages: number;
}

export const commands: Record<string, ChildProcess>= {}

export default defineEventHandler(async (event) => {
    const request = await readBody(event) as Request

    if (!request.prompt) {
        throw createError({
            statusCode: 400,
            statusMessage: 'prompt is required'
        });
    }

    if (!request.pages) {
        throw createError({
            statusCode: 400,
            statusMessage: 'pages are required'
        });
    }

    const command = `gptscript story-book.gpt --story ${request.prompt}, '--pages', ${`${request.pages}`}`

    setHeaders(event,{
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Credentials': 'true',
        'Connection': 'keep-alive',
        'Content-Type': 'text/event-stream',
    })

    setResponseStatus(event, 202)

    const child = exec(command)
    commands[request.prompt] = child
})