# Context

GPTScript provides a mechanism to share prompt information across many tools using the tool directive `context`.
It is used to provide additional information to the calling tool on when to use a specific tool by prepending the `context` to the instruction of the calling tool.

- Context can point to a static text or a GPTScript.
- Context tools are just regular GPTScript tools, and any valid GPTScript fields can be used in them.
- Shared tools from a context tool are made available to the calling tool.
- When context points to a GPTScript tool, output from the context tool gets prepended to the instruction of the calling tool.

## Writing a Context Provider Tool as static text

```yaml
# my-context.txt

You have access to run commands on the user's system. Please ask for confirmation from the user before running a command.

```

## Using a Context Tool

Continuing with the above example, this is how you can use the same context in different tools:

```yaml
Context: ./my-context.txt
Tools: sys.exec, sys.write

Which processes on my system are using the most memory? Write their PIDs to a file called pids.txt.
```

```yaml
Context: ./my-context.txt
Tools: sys.exec

Which file in my current directory is the largest?
```

## Context Provider Tool with exported tools

Here is a simple example of a context provider tool that provides additional context to search tool:

```yaml
# my-context-tool.gpt
Share Tools: sys.exec

#!sys.echo
You have access to run commands on the user's system. Please ask for confirmation from the user before running a command.

```

The `#!sys.echo` at the start of the tool body tells GPTScript to return everything after it as the output of the tool. 

Continuing with the above example, this is how you can use it in a script:

```yaml
Context: ./my-context-tool.gpt
Tools: sys.write

Which processes on my system are using the most memory? Write their PIDs to a file called pids.txt.
```

When you run this script, GPTScript will use the output from the context tool and add it to the user message along with the 
existing prompt in this tool to provide additional context to LLM.

## Context Provider Tool with Parameters

Here is an example of a context provider tool that takes a parameter:

```yaml
# context_with_param.gpt
Param: tone: the tone to use when responding to the user's request

#!/bin/bash
echo "Respond to the user's request in a ${tone} tone."

```

Continuing with the above example, this is how you can use it in a script:

```yaml
# tool.gpt
Context: ./context_with_param.gpt with ${tone} as tone
Param: tone: the tone to use when responding to the user's request
Tools: sys.http.html2text

What are the top stories on Hacker News right now?

```

Here's how you can run the script and define the tone parameter:

```yaml
gptscript tool.gpt '{"tone": "obnoxious"}'
```
