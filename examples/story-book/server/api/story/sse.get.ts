import { commands } from '@/server/api/story/index.post';

export default defineEventHandler(async (event) => {
    const { prompt } = getQuery(event);
    if (!prompt) {
        throw createError({
            statusCode: 400,
            statusMessage: 'prompt is required'
        });
    }

    const child = commands[prompt as string];
    if (!child) {
        throw createError({
            statusCode: 404,
            statusMessage: 'command not found'
        });
    }

    setResponseStatus(event, 200);
    setHeaders(event, {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Credentials': 'true',
        'Connection': 'keep-alive',
        'Content-Type': 'text/event-stream',
    });
    
    child.stdout?.on('data', (data) => {
        event.node.res.write(`data: ${data}\n\n`);
    });

    child.stderr?.on('data', (data) => {
        event.node.res.write(`data: ${data}\n\n`);
    });

    child.on('close', (code) => {
        event.node.res.write(`data: Command exited with code ${code}\n\n`);
        event.node.res.end();
    });

    event._handled = true; 
});