# Context

GPTScript provides a mechanism to share prompt information across many tools using the tool parameter `context`. It is used to provide additional information to the calling tool on when to use a specific tool by prepending the `context` to the instruction of the calling tool.

- Context can point to a static text or a GPTScript.
- Context tools are just regular GPTScript tools, and any valid gptscript field can be used.
- Exported tools from a context tool are made available to the calling tool.
- When context points to a GPTScript tool, output from the context tool gets prepended to the instruction of the calling tool.

## Writing a Context Provider Tool as static text

```yaml
# my-search-context.txt

You are an expert web researcher with access to the Search tool.If the search tool fails to return any information stop execution of the script with message "Sorry! Search did not return any results". Feel free to get the contents of the returned URLs in order to get more information. Provide as much detail as you can. Also return the source of the search results.

```

## Using a Context Provider Tool

Continuing with the above example, this is how you can use the same context in tools that uses different search providers:

```yaml
# my-search-duduckgo.gpt
context: ./my-search-context.txt
tools: github.com/gptscript-ai/search/duckduckgo,sys.http.html2text 

What are some of the most popular tourist destinations in Scotland, and how many people visit them each year?

```

```yaml
# my-search-brave.gpt
context: ./my-search-context.txt
tools: github.com/gptscript-ai/search/brave,sys.http.html2text

List out some of the main actors in the Christopher Nolan movie Inception, as well as the names of the other Christopher Nolan movies they have appeared in.

```


## Context Provider Tool with exported tools

Here is a simple example of a context provider tool that provides additional context to search tool:

```yaml
# my-search-context-tool.gpt
export: sys.http.html2text?

#!/bin/bash
echo You are an expert web researcher with access to the Search tool.If the search tool fails to return any information stop execution of the script with message "Sorry! Search did not return any results". Feel free to get the contents of the returned URLs in order to get more information. Provide as much detail as you can. Also return the source of the search results.

```

Continuing with the above example, this is how you can use it in a script:

```yaml
context: ./my-search-context-tool.gpt
tools: github.com/gptscript-ai/search/duckduckgo

What are some of the most popular tourist destinations in Scotland, and how many people visit them each year?

```

When you run this script, GPTScript will use the output from the context tool and add it to the user message along with the 
existing prompt in this tool to provide additional context to LLM.

## Context Provider Tool with args

Here is an example of a context provider tool that uses args to decide which search tool to use when answering the user provided queries:

```yaml
# context_with_arg.gpt
export: github.com/gptscript-ai/search/duckduckgo, github.com/gptscript-ai/search/brave, sys.http.html2text?
args: search_tool: tool to search with

#!/bin/bash
echo You are an expert web researcher with access to the ${search_tool} Search tool.If the search tool fails to return any information stop execution of the script with message "Sorry! Search did not return any results". Feel free to get the contents of the returned URLs in order to get more information. Provide as much detail as you can. Also return the source of the search results.

```

Continuing with the above example, this is how you can use it in a script:

```yaml
# my_context_with_arg.gpt
context: ./context_with_arg.gpt with ${search} as search_tool
Args: search: Search tool to use

What are some of the most popular tourist destinations in Scotland, and how many people visit them each year?

```

This script can be used to search with `brave` or `duckduckdb` tools depending on the search parameter passed to the tool.
Example usage for using brave search tool:
```yaml
gptscript --disable-cache my_context_with_arg.gpt '{"search": "brave"}'
```
