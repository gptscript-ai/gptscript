# Daemon Tools (Advanced)

One advanced use case that GPTScript supports is daemon tools.
A daemon tool is a tool that starts a long-running HTTP server in the background, that will continue running until GPTScript is done executing.
Other tools can easily send HTTP POST requests to the daemon tool.

## Example

Here is an example of a daemon tool with a simple echo server written in an inline Node.js script:

```
Tools: my-daemon
Param: first: the first parameter
Param: second: the second parameter

#!http://my-daemon.daemon.gptscript.local/myPath

---
Name: my-daemon

#!sys.daemon node

const http = require('http');

const server = http.createServer((req, res) => {
    if (req.method === 'GET' || req.method === 'POST') {
        // Extract the path from the request URL
        const path = req.url;

        let body = '';

        req.on('data', chunk => {
          body += chunk.toString();
        })

        // Respond with the path and body
        req.on('end', () => {
          res.writeHead(200, { 'Content-Type': 'text/plain' });
          res.write(`Body: ${body}\n`);
          res.end(`Path: ${path}`);
        })
    } else {
        res.writeHead(405, { 'Content-Type': 'text/plain' });
        res.end('Method Not Allowed');
    }
});

const PORT = process.env.PORT || 3000;
server.listen(PORT, () => {
    console.log(`Server is listening on port ${PORT}`);
});
```

Let's talk about the daemon tool, called `my-daemon`, first.

### The Daemon Tool

The body of this tool begins with `#!sys.daemon`. This tells GPTScript to take the rest of the body as a command to be
run in the background that will listen for HTTP requests. GPTScript will run this command (in this case, a Node script).
GPTScript will assign a port number for the server and set the `PORT` environment variable to that number, so the
server needs to check that variable and listen on the proper port. The server is required to respond to a GET request at
`/` with 200 OK, so that GPTScript can make sure it is running properly before continuing execution.

### The Entrypoint Tool

The entrypoint tool at the top of this script sends an HTTP request to the daemon tool.
There are a few important things to note here:

- The `Tools: my-daemon` directive is needed to show that this tool requires the `my-daemon` tool to already be running.
  - When the entrypoint tool runs, GPTScript will check if `my-daemon` is already running. If it is not, GPTScript will start it.
- The `#!http://my-daemon.daemon.gptscript.local/myPath` in the body tells GPTScript to send an HTTP request to the daemon tool.
  - The request will be a POST request, with the body of the request being a JSON string of the parameters passed to the entrypoint tool.
    - For example, if the script is run like `gptscript script.gpt '{"first":"hello","second":"world"}'`, then the body of the request will be `{"first":"hello","second":"world"}`.
    - The path of the request will be `/myPath`.
  - The hostname is `my-daemon.daemon.gptscript.local`. When sending a request to a daemon tool, the hostname must always start with the daemon tool's name, followed by `.daemon.gptscript.local`.
    - GPTScript recognizes this hostname and determines the correct port number to send the request to, on localhost.

### Running the Example

Now let's try running it:

```bash
gptscript script.gpt '{"first":"hello","second":"world"}'
```

```
OUTPUT:

Body: {"first":"hello","second":"world"}
Path: /myPath
```

This is exactly what we expected. This is a silly, small example just to demonstrate how this feature works.
A real-world situation would involve several different tools sending different HTTP requests to the daemon tool,
likely with an LLM determining when to call which tool.

### Real-World Example

To see a real-world example of a daemon tool, check out the [GPTScript Browser tool](https://github.com/gptscript-ai/browser).
