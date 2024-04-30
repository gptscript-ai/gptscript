# SDKs

Currently, there are three SDKs being maintained: [Python](https://github.com/gptscript-ai/py-gptscript), [Node](https://github.com/gptscript-ai/node-gptscript), and [Go](https://github.com/gptscript-ai/go-gptscript).

The goal with these SDKs is to have the same options and functionality as the `gptscript` CLI through language bindings. If you find that some functionality is missing or different from the CLI, please open an issue in the corresponding SDK repo.

Below are some simple examples for each to get started. However, the individual repos will have more comprehensive examples, and include all currently supported functionality. Additionally, the individual repos contain installation instructions and required dependencies.

### Python

The [Python SDK](https://github.com/gptscript-ai/py-gptscript) includes a `Tool` data structure corresponding to the documented [tool reference](07-gpt-file-reference.md#tool-parameters), which can be used to construct and run tools.

```python
from gptscript.command import exec
from gptscript.tool import Tool

tool = Tool(
    json_response=True,
    instructions="""
Create three short graphic artist descriptions and their muses. 
These should be descriptive and explain their point of view.
Also come up with a made up name, they each should be from different
backgrounds and approach art differently.
the response should be in JSON and match the format:
{
   artists: [{
      name: "name"
      description: "description"
   }]
}
""",
    )


response = exec(tool)
print(response)
```

The SDK also supports running GPTScripts from files.

```python
from gptscript.command import exec_file

response = exec_file("./example.gpt")
print(response)
```

For more functionality, like streaming output and the supported options, visit the [repo](https://github.com/gptscript-ai/py-gptscript).

### Node

The [Node SDK](https://github.com/gptscript-ai/node-gptscript) includes a `Tool` data structure corresponding to the documented [tool reference](07-gpt-file-reference#tool-parameters), which can be used to construct and run tools.

```javascript
const gptscript = require('@gptscript-ai/gptscript');

const t = new gptscript.Tool({
    instructions: "who was the president of the united states in 1928?"
});

try {
    const response = await gptscript.exec(t);
    console.log(response);
} catch (error) {
    console.error(error);
}
```

The SDK also supports running GPTScripts from files.

```javascript
const gptscript = require('@gptscript-ai/gptscript');

const opts = {
    cache: false,
};

async function execFile() {
    try {
        const out = await foo.execFile('./hello.gpt', "", opts);
        console.log(out);
    } catch (e) {
        console.error(e);
    }
}
```

For more functionality, like streaming output and the supported options, visit the [repo](https://github.com/gptscript-ai/node-gptscript).

### Go

The [Go SDK](https://github.com/gptscript-ai/go-gptscript) includes a `Tool` data structure corresponding to the documented [tool reference](07-gpt-file-reference#tool-parameters), which can be used to construct and run tools.

```go
package main

import (
	"context"

	gptscript "github.com/gptscript-ai/go-gptscript"
)

func runTool(ctx context.Context) (string, error) {
	t := gptscript.Tool{
		Instructions: "Who was the president of the united states in 1928?",
	}

	return gptscript.ExecTool(ctx, gptscript.Opts{}, t)
}
```

The SDK also supports running GPTScripts from files.

```go
package main

import (
	"context"

	gptscript "github.com/gptscript-ai/go-gptscript"
)

func execFile(ctx context.Context) (string, error) {
	opts := gptscript.Opts{
		DisableCache: &[]bool{true}[0],
	}

	return gptscript.ExecFile(ctx, "./hello.gpt", "", opts)
}
```

For more functionality, like streaming output and the supported options, visit the [repo](https://github.com/gptscript-ai/go-gptscript).
