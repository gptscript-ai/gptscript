import { runningScripts } from '@/server/api/story/index.post';

export default defineEventHandler(async (event) => {
    const { prompt } = getQuery(event);
    if (!prompt) {
        throw createError({
            statusCode: 400,
            statusMessage: 'prompt is required'
        });
    }

    const runningScript = runningScripts[prompt as string];
    if (!runningScript) {
        throw createError({
            statusCode: 404,
            statusMessage: 'running script not found'
        });
    }

    setResponseStatus(event, 200);
    setHeaders(event, {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Credentials': 'true',
        'Connection': 'keep-alive',
        'Content-Type': 'text/event-stream',
    });
    
    runningScript.stdout.on('data', (data) => {
        event.node.res.write(`data: ${data}\n\n`);
    });

    runningScript.stderr.on('data', (data) => {
        event.node.res.write(`data: ${data}\n\n`);
    });

    event._handled = true; 
    await runningScript.promise.then(() => {
        event.node.res.write('data: done\n\n');
        event.node.res.end();
    }).catch((error) => {
        setResponseStatus(event, 500);
        event.node.res.write(`data: error: ${error}\n\n`);
        event.node.res.end();
    });
});