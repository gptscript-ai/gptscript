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
    
    let stdoutBuffer = '';
    runningScript.stdout.on('data', (data) => {
        stdoutBuffer += data;
        if (data.includes('\n')) {
            event.node.res.write(`data: ${stdoutBuffer}\n\n`);
            stdoutBuffer = '';
        }
    });

    let stderrBuffer = '';
    runningScript.stderr.on('data', (data) => {
        stderrBuffer += data;
        if (data.includes('\n')) {
            event.node.res.write(`data: ${stderrBuffer}\n\n`);
            stderrBuffer = '';
        }
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