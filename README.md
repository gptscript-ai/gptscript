# GPTScript

[![Discord](https://img.shields.io/discord/1204558420984864829?label=Discord)](https://discord.gg/9sSf4UyAMC)

## Overview

GPTScript is a new scripting language to automate your interaction with a Large Language Model (LLM), namely OpenAI. The ultimate goal is to create a natural language programming experience. The syntax of GPTScript is largely natural language, making it very easy to learn and use.
Natural language prompts can be mixed with traditional scripts such as bash and python or even external HTTP service
calls. With GPTScript you can do just about anything, like [plan a vacation](./examples/travel-agent.gpt),
[edit a file](./examples/add-go-mod-dep.gpt), [run some SQL](./examples/sqlite-download.gpt), or [build a mongodb/flask app](./examples/hacker-news-headlines.gpt). Here are some common use cases for GPTScript:

1. [Retrieval-Augmented Generation (RAG)](./docs/README-USECASES.md#retrieval)
2. [Task Automation](./docs/README-USECASES.md#task-automation)
3. [Agents and Assistants](./docs/README-USECASES.md#agents-and-assistants)
4. [Data Analysis](./docs/README-USECASES.md#data-analysis)
5. [Vision, Image, and Audio](./docs/README-USECASES.md#vision-image-and-audio)
6. [Memory Management](./docs/README-USECASES.md#memory-management)
7. [Chatbots](./docs/README-USECASES.md#chatbots)

| :memo: | We are currently exploring options for interacting with local models using GPTScript. |
| ------ | :------------------------------------------------------------------------------------ |

The following example illustrates how GPTScript allows you to accomplish a complex task by writing instructions in English:

```yaml
# example.gpt

Tools: sys.download, sys.exec, sys.remove

Download https://www.sqlitetutorial.net/wp-content/uploads/2018/03/chinook.zip to a
random file. Then expand the archive to a temporary location as there is a sqlite
database in it.

First inspect the schema of the database to understand the table structure.

Form and run a SQL query to find the artist with the most number of albums and output
the result of that.

When done remove the database file and the downloaded content.
```

```shell
$ gptscript ./example.gpt
```

```
OUTPUT:

The artist with the most number of albums in the database is Iron Maiden, with a total
of 21 albums.
```

## Quick Start

### 1. Install the latest release

#### Homebrew (macOS and Linux)

```shell
brew install gptscript-ai/tap/gptscript
```

#### Install Script (macOS and Linux):

```shell
curl https://get.gptscript.ai/install.sh | sh
```

### Scoop (Windows)

```shell
scoop bucket add extras # If 'extras' is not already enabled
scoop install gptscript
```

#### WinGet (Windows)

```shell
winget install gptscript-ai.gptscript
```

#### Manually

Download and install the archive for your platform and architecture from the [releases page](https://github.com/gptscript-ai/gptscript/releases).

### 2. Get an API key from [OpenAI](https://platform.openai.com/api-keys).

#### macOS and Linux

```shell
export OPENAI_API_KEY="your-api-key"
```

Alternatively Azure OpenAI can be utilized. If the [Azure deployment name is different than the model being used](https://learn.microsoft.com/en-us/azure/ai-services/openai/how-to/switching-endpoints#keyword-argument-for-model), be sure to include the `OPENAI_AZURE_DEPLOYMENT` argument.

```shell
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://<your-endpoint>.openai.azure.com/"
export OPENAI_API_TYPE="AZURE"
export OPENAI_AZURE_DEPLOYMENT="<your-deployment-name>"
```

#### Windows

```powershell
$env:OPENAI_API_KEY = 'your-api-key'
```

### 3. Run Hello World

```shell
gptscript https://get.gptscript.ai/echo.gpt --input 'Hello, World!'
```

```
OUTPUT:

Hello, World!
```

The model used by default is `gpt-4-turbo` and you must have access to that model in your OpenAI account.

If using Azure OpenAI, make sure you configure the model to be one of the supported versions with the `--default-model` argument.

### 4. Extra Credit: Examples and Run Debugging UI

Clone examples and run debugging UI

```shell
git clone https://github.com/gptscript-ai/gptscript
cd gptscript/examples

# Run the debugging UI
gptscript --server
```

## How it works

**_GPTScript is composed of tools._** Each tool performs a series of actions similar to a function. Tools have available
to them other tools that can be invoked similar to a function call. While similar to a function, the tools are
primarily implemented with a natural language prompt. **_The interaction of the tools is determined by the AI model_**,
the model determines if the tool needs to be invoked and what arguments to pass. Tools are intended to be implemented
with a natural language prompt but can also be implemented with a command or HTTP call.

### Example

Below are two tool definitions, separated by `---`. The first tool does not require a name or description, but
every tool after name and description are required. The first tool, has the parameter `tools: bob` meaning that the tool named `bob` is available to be called if needed.

```yaml
tools: bob

Ask Bob how he is doing and let me know exactly what he said.

---
name: bob
description: I'm Bob, a friendly guy.
args: question: The question to ask Bob.

When asked how I am doing, respond with "Thanks for asking "${question}", I'm doing great fellow friendly AI tool!"
```

Put the above content in a file named `bob.gpt` and run the following command:

```shell
$ gptscript bob.gpt
```

```
OUTPUT:

Bob said, "Thanks for asking 'How are you doing?', I'm doing great fellow friendly AI tool!"
```

Tools can be implemented by invoking a program instead of a natural language prompt. The below
example is the same as the previous example but implements Bob using python.

```yaml
Tools: bob

Ask Bob how he is doing and let me know exactly what he said.

---
Name: bob
Description: I'm Bob, a friendly guy.
Args: question: The question to ask Bob.

#!python3

import os

print(f"Thanks for asking {os.environ['question']}, I'm doing great fellow friendly AI tool!")
```

With these basic building blocks you can create complex scripts with AI interacting with AI, your local system, data,
or external services.

## GPT File Reference

### Extension

GPTScript files use the `.gpt` extension by convention.

### File Structure

A GPTScript file has one or more tools in the file. Each tool is separated by three dashes `---` alone on a line.

```yaml
Name: tool1
Description: This is tool1

Do sample tool stuff.

---
Name: tool2
Description: This is tool2

Do more sample tool stuff.
```

### Tool Definition

A tool starts with a preamble that defines the tool's name, description, args, available tools and additional parameters.
The preamble is followed by the tool's body, which contains the instructions for the tool. Comments in
the preamble are lines starting with `#` and are ignored by the parser. Comments are not really encouraged
as the text is typically more useful in the description, argument descriptions or instructions.

```yaml
Name: tool-name
# This is a comment in the preamble.
Description: Tool description
# This tool can invoke tool1 or tool2 if needed
Tools: tool1, tool2
Args: arg1: The description of arg1

Tool instructions go here.
```

#### Tool Parameters

Tool parameters are key-value pairs defined at the beginning of a tool block, before any instructional text. They are specified in the format `key: value`. The parser recognizes the following keys (case-insensitive and spaces are ignored):


| Key            | Description                                                                                                                             |
|------------------|-----------------------------------------------------------------------------------------------------------------------------------------|
| `Name`           | The name of the tool.                                                                                                                   |
| `Model Name`     | The OpenAI model to use, by default it uses "gpt-4-turbo"                                                                         |
| `Description`    | The description of the tool. It is important that this properly describes the tool's purpose as the description is used by the LLM.     |
| `Internal Prompt`| Setting this to `false` will disable the built-in system prompt for this tool.                                                          |
| `Tools`          | A comma-separated list of tools that are available to be called by this tool.                                                            |
| `Args`           | Arguments for the tool. Each argument is defined in the format `arg-name: description`.                                                  |
| `Max Tokens`     | Set to a number if you wish to limit the maximum number of tokens that can be generated by the LLM.                                      |
| `JSON Response`  | Setting to `true` will cause the LLM to respond in a JSON format. If you set true you must also include instructions in the tool.        |
| `Temperature`    | A floating-point number representing the temperature parameter. By default, the temperature is 0. Set to a higher number for more creativity. |


#### Tool Body

The tool body contains the instructions for the tool which can be a natural language prompt or
a command to execute. Commands [must start with](https://en.wikipedia.org/wiki/Shebang_(Unix)) `#!` followed by the interpreter (e.g. `#!/bin/bash`, `#!python3`)
a text that will be placed in a file and passed to the interpreter. Arguments can be references in the instructions
using the format `${arg1}`.

```yaml
name: echo-ai
description: A tool that echos the input
args: input: The input

Just return only "${input}"

---
name: echo-command
description: A tool that echos the input
args: input: The input

#!/bin/bash

echo "${input}"
```

## Built in Tools

There are several built in tools to do basic things like read/write files, download http content and execute commands.
Run `gptscript --list-tools` to list all the built-in tools.

## Examples

For more examples check out the [examples](examples) directory.

## Community

Join us on Discord: [![Discord](https://img.shields.io/discord/1204558420984864829?label=Discord)](https://discord.gg/9sSf4UyAMC)

## License

Copyright (c) 2024 [Acorn Labs, Inc.](http://acorn.io)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
