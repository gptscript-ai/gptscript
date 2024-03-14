# Authoring Tools

You can author your own tools for your use or to share with others.
The process for authoring a tool is as simple as creating a `tool.gpt` file in the root directory of your project.
This file is itself a GPTScript that defines the tool's name, description, and what it should do.

Here's an example of the `tool.gpt` file for the `image-generation` tool:

```
description: Generates images based on the specified parameters and returns a list of URLs to the generated images.
args: prompt: (required) The text prompt based on which the GPT model will generate a response
args: size: (optional) The size of the image to generate, format WxH (e.g. 1024x1024). Defaults to 1024x1024.
args: quality: (optional) The quality of the generated image. Allowed values are "standard" or "hd". Default is "standard".
args: number: (optional) The number of images to generate. Defaults to 1.

#!/usr/bin/env python3 ${GPTSCRIPT_TOOL_DIR}/cli.py --prompt="${prompt}" --size="${size}" --quality="${quality}" --number="${number}"
```

At the bottom you'll notice a shebang (`#!/usr/bin/env ...`) line that specifies the command to run when the tool is invoked. This is the exact command that will be executed when the tool is used in a GPTScript.
If there is no shebang line, then it will be treated as natural language for the LLM to process.

:::tip
Every arg becomes an environment variable when the tool is invoked. So instead of accepting args using flags like `--size="${size}", your program can just read the `size` environment variable.
:::

## Binary Tools
GPTScript can call binaries and scripts located on your system or externally. This is done by defining a `tool.gpt` file in the same directory that the binary or script is located. In that `tool.gpt` file, you call it directly using the `#!` directive.

For example:

```yaml
description: This is a tool that calls a binary.
args: arg1: The first argument.

#!/usr/bin/env ./path/to/binary --arg1="${arg1}"
```

## Shell
You can call shell scripts directly using the `#!` directive. For example:

```yaml
description: This is a tool that calls a shell script.
args: arg1: The first argument.

#!/usr/bin/env sh
echo "Hello, world!"
./path/to/script.sh --arg1="${arg1}"
```

## Python
You can call Python scripts directly, for example:

```yaml
description: This is a tool that calls a Python script.
args: arg1: The first argument.

#!/usr/bin/env python3 ./path/to/script.py --arg1="${arg1}"
```

If you need to bootstrap a Python environment, you can use a virtual environment. For example:

```yaml
description: This is a tool that calls a Python script in a virtual environment.
args: arg1: The first argument.

#!/usr/bin/env sh
python3 -m venv .venv
pip3 install -r requirements.txt
python3 ./path/to/script.py --arg1="${arg1}"
```

## Node

You can call Node.js scripts directly, for example:

```yaml
description: This is a tool that calls a Node.js script.
args: arg1: The first argument.

#!/usr/bin/env node ./path/to/script.js --arg1="${arg1}"
```

If you need to bootstrap a Node.js environment, you can use call npm in the `tool.gpt` file. For example:

```yaml
description: This is a tool that calls a Node.js script with a package manager.
args: arg1: The first argument.

#!/usr/bin/env sh
npm install
node ./path/to/script.js --arg1="${arg1}"
```

## Golang

You can call Golang binaries directly, for example:

```yaml
description: This is a tool that calls a Golang binary.
args: arg1: The first argument.

#!/usr/bin/env ./path/to/binary --arg1="${arg1}"
```

If you need to bootstrap a Golang binary, you can the go CLI to build it before running it. For example:

```yaml
description: This is a tool that calls a Golang binary.
args: arg1: The first argument.

#!/usr/bin/env sh
go build -o ./path/to/binary
./path/to/binary --arg1="${arg1}"
```
