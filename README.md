# GPTScript

## Overview

GPTScript is a new scripting language to automate your interaction with a Large Language Model (LLM), namely OpenAI.
The syntax of GPTScript is largely natural language, making it very easy to learn and use.
Natural language prompts can be mixed with traditional scripts such as bash and python or even external HTTP service
calls.

```yaml
Tools: sys.download, sys.exec, sys.remove

Download https://www.sqlitetutorial.net/wp-content/uploads/2018/03/chinook.zip to a
random file. Then expand the archive to a temporary location as there is a sqlite
database in it.

First inspect the schema of the database to understand the table structure.

Form and run a SQL query to find the artist with the most number of albums and output
the result of that.

When done remove the database file and the downloaded content.
```

## Quick Start

1. Install with brew (or from [releases](https://github.com/gptscript-ai/gptscript/releases/latest))
```shell
brew install gptscript-ai/tap/gptscript
```
2. Get an API key from [OpenAI](https://platform.openai.com/api-keys).
```shell
export OPENAI_API_KEY="your-api-key"
```
3. Run Hello World:
```shell
gptscript https://gptscript.ai/echo.gpt --input "Hello, World!"
```

## How it works

***GPTScript is composed of tools.*** Each tool performs a series of actions similar to a function. Tools have available
to them other tools that can be invoked similar to a function call. While similar to a function, the tools are
implemented with a natural language and not code. ***The interaction of the tools is determined by the AI model***,
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

OUTPUT:

Bob said, "I'm doing great fellow friendly AI tool!"
```
Tools can be implemented by invoking a program instead of a natural language prompt. The below
example is the same as the previous example but implements Bob using bash.

```yaml
Tools: bob

Ask Bob how he is doing and let me know exactly what he said.

---
Name: bob
Description: I'm Bob, a friendly guy.
Args: question: The question to ask Bob.

#!/bin/bash

echo "Thanks for asking ${question}, I'm doing great fellow friendly AI tool!"
```

With these basic building blocks you can create complex scripts with AI interacting with, your local system, data,
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
# This is a comment in the preamble. Comments don't typically provide much value
Description: Tool description
# This tool can invoke tool1 or tool2 if needed
Tools: tool1, tool2
Args: arg1: The description of arg1
      
Tool instructions go here.
```
#### Tool Parameters

Tool parameters are key-value pairs defined at the beginning of a tool block, before any instructional text. They are specified in the format `key: value`. The parser recognizes the following keys (case-insensitive and spaces are ignored):

`Name`: The name of the tool.

`Model Name`: The OpenAI model to use, by default it uses "gpt-4-turbo-preview"

`Description`: The description of the tool. It is important that the properly describes the tool's purpose as the
description is used by the LLM to determine if the tool is to be invoked.

`Internal Prompt`: Setting this to `false` will disable the built in system prompt for this tool. GPTScript includes a
default system prompt to instruct the AI to behave more like a script engine and not a "helpful assistant."

`Tools`: A comma-separated list of tools that are available to be called by this tool. A tool can only call the tools
that are defined here.

`Args`: Arguments for the tool. Each argument is defined in the format `arg-name: description`. All arguments are essentially
strings. No other type really exists as all input and output to tools is text based.

`Vision`: If set to true this model will use vision capabilities. Vision models currently do not support the `Args` or `Tools` parameters.

`Max Tokens`: Set to a number if you wish to limit the maximum number of tokens that can be generated by the LLM.

`JSON Response`: Setting to `true` will cause the LLM to respond in a JSON format. If you set true you must also include instructions in the tool
to inform the LLM to respond in some JSON structure.

`Temperature`: A floating-point number representing the temperature parameter. By default the temperature is 0. Set to a higher number to make the LLM more creative.

#### Tool Body

The tool body contains the instructions for the tool which can be a natural language prompt or
a command to execute. Commands must start with `#!` followed by the interpreter (e.g. `#!/bin/bash`, `#!python3`)
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

## Examples

For more examples check out the [examples](examples) directory.

## License

GPTScript is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.