# Using Tools
In GPTScript, tools are used to extend the capabilities of a script. The idea behind them is that AI performs better when it has very specific instructions for a given task. Tools are a way to break-up the problem into smaller and more focused pieces where each tool is responsible for a specific task. A typical flow like this is to have a main script that imports a set of tools it can use to accomplish its goal.

GPTScripts can utilize tools in one of three ways:
1. Built-in system tools
2. In-script tools
3. External tools

### System Tools
All GPTScripts have access to system tools, like `sys.read` and `sys.write`, that can be used without any additional configuration.

```yaml
tools: sys.read, sys.write

Read all of the files in my current directory, do not recurse over any subdirectories, and give me a description of this directory's contents.
```

System tools are a set of core tools that come packaged with GPTScript by default.

### In-Script Tools
Things get more interesting when you start to use custom tools.

The most basic example of this is an in-script tool that is defined in the same file as the main script. This is useful for breaking up a large script into smaller, more manageable pieces.

```yaml
tools: random-number

Select a number at random and, based on the results, write a poem about it.

---
name: random-number
description: Generate a random number between 1 and 100.

Select a number at random between 1 and 100 and return only the number.
```

### External Tools
You can refer to GPTScript tool files that are served on the web or stored locally. This is useful for sharing tools across multiple scripts or for using tools that are not part of the core GPTScript distribution.

```yaml
tools: https://get.gptscript.ai/echo.gpt

Echo the phrase "Hello, World!".
```

Or, if the file is stored locally in "echo.gpt":

```yaml
tools: echo.gpt

Echo the phrase "Hello, World!".
```

You can also refer to OpenAPI definition files as though they were GPTScript tool files. GPTScript will treat each operation in the file as a separate tool. For more details, see [OpenAPI Tools](03-openapi.md).

### Packaged Tools on GitHub
GPTScript tools can be packaged and shared on GitHub, and referred to by their GitHub URL. For example:

```yaml
tools: github.com/gptscript-ai/dalle-image-generation, github.com/gptscript-ai/gpt4-v-vision, sys.read

Generate an image of a city skyline at night and write the resulting image to a file called city_skyline.png.

Take this image and write a description of it in the style of pirate.
```

When this script is run, GPTScript will locally clone the referenced GitHub repos and run the tools referenced inside them.
For more info on how this works, see [Authoring Tools](02-authoring.md).
