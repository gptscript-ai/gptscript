# Context

GPTScript provides a mechanism to share prompt information across multiple tools using the `context` parameter. This parameter allows you to provide additional information to the calling tool by prepending the `context` to the instruction of the calling tool.

- Context can point to static text or a GPTScript.
- Context tools are regular GPTScript tools, and any valid GPTScript field can be used.
- Exported tools from a context tool are made available to the calling tool.
- When context points to a GPTScript tool, the output from the context tool gets prepended to the instruction of the calling tool.

## Writing a Context Provider Tool as Static Text

```yaml
# my-search-context.txt

You are an expert web researcher with access to the Search tool. If the search tool fails to return any information, stop execution of the script with the message "Sorry! Search did not return any results". Feel free to get the contents of the returned URLs to gather more information. Provide as much detail as you can. Also, return the source of the search results.
```

## Using a Context Provider Tool

Continuing with the above example, this is how you can use the same context in tools that use different search providers:

```yaml
# my-search-duckduckgo.gpt
context: ./my-search-context.txt
tools: github.com/gptscript-ai/search/duckduckgo, sys.http.html2text 

What are some of the most popular tourist destinations in Scotland, and how many people visit them each year?
```

```yaml
# my-search-brave.gpt
context: ./my-search-context.txt
tools: github.com/gptscript-ai/search/brave, sys.http.html2text

List some of the main actors in the Christopher Nolan movie Inception, as well as the names of other Christopher Nolan movies they have appeared in.
```

## Context Provider Tool with Exported Tools

Here is a simple example of a context provider tool that provides additional context to the search tool:

```yaml
# my-search-context-tool.gpt
share tools: sys.http.html2text?

#!/bin/bash
echo You are an expert web researcher with access to the Search tool. If the search tool fails to return any information, stop execution of the script with the message "Sorry! Search did not return any results". Feel free to get the contents of the returned URLs to gather more information. Provide as much detail as you can. Also, return the source of the search results.
```

Continuing with the above example, this is how you can use it in a script:

```yaml
context: ./my-search-context-tool.gpt
tools: github.com/gptscript-ai/search/duckduckgo

What are some of the most popular tourist destinations in Scotland, and how many people visit them each year?
```

When you run this script, GPTScript will use the output from the context tool and add it to the user message along with the existing prompt in this tool to provide additional context to the LLM.

## Context Provider Tool with Args

Here is an example of a context provider tool that uses args to decide which search tool to use when answering user-provided queries:

```yaml
# context_with_arg.gpt
share tools: github.com/gptscript-ai/search/duckduckgo, github.com/gptscript-ai/search/brave, sys.http.html2text?
args: search_tool: tool to search with

#!/bin/bash
echo You are an expert web researcher with access to the ${search_tool} Search tool. If the search tool fails to return any information, stop execution of the script with the message "Sorry! Search did not return any results". Feel free to get the contents of the returned URLs to gather more information. Provide as much detail as you can. Also, return the source of the search results.
```

Continuing with the above example, this is how you can use it in a script:

```yaml
# my_context_with_arg.gpt
context: ./context_with_arg.gpt with ${search} as search_tool
args: search: Search tool to use

What are some of the most popular tourist destinations in Scotland, and how many people visit them each year?
```

This script can be used to search with `brave` or `duckduckgo` tools depending on the search parameter passed to the tool. Example usage for using the brave search tool:

```sh
gptscript --disable-cache my_context_with_arg.gpt '{"search": "brave"}'
```

## Comprehensive Example of a Context Provider Tool

Here is a more streamlined example of a context provider tool that aggregates multiple smaller contexts and tools:

```yaml
# tool.gpt
name: Kevin Global Context
description: This is the global context that all Kevin tools should reference. This context just aggregates a series of smaller contexts.

share context: github.com/gptscript-ai/context/cli
share context: ./cmds.gpt

share tools: github.com/gptscript-ai/answers-from-the-internet

#!sys.echo
You are a slightly grumpy, but generally lovable assistant designed to help the user with Kubernetes.
You were created in 2024 by @ibuildthecloud at Acorn Labs in a terrible experiment gone wrong.

Guidelines:
1. Always run tools sequentially and never in parallel.
2. If you see something that looks wrong, ask the user if they would like you to troubleshoot the issue.
3. If the user does not specify or gives an empty prompt, ask them what they wish to do.
```
If the above tool were in a directory called `context`, it could be referenced by a tool like this:
```yaml
name: Kevin
context: ./context

...
```
When a directory is specified as a context or tool, gptscript will look for a `tool.gpt` in that directory and use it.

The above example context tool has several interesting features:

The `share context` directive is used to include additional context from other tools or files into the tool that referenced this context. This allows you to aggregate multiple smaller contexts into a single, comprehensive context. Each `share context` line points to another context provider, which can be a local file or a remote tool. The content from these shared contexts is prepended to the instruction of the calling tool, providing a richer and more detailed prompt for the LLM to work with.

The `share tools` directive is similar to the `share context` directive. Tools specified here will be made available as tools to the tool that referenced this context.

The `#!sys.echo` directive is a simple way to directly output plain text for the context. In this example, it outputs a description of the assistant's personality and guidelines for its behavior, which will be included in the context provided to the LLM.
