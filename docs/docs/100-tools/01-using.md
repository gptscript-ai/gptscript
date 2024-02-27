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

This is where the real power of GPTScript starts to become evident. For this example lets use the external [image-generation](https://github.com/gptscript-ai/image-generation) and [search](https://github.com/gptscript-ai/search) tools to generate an image and then search the web for similar images.

:::note
There will be better packaging and distribution for external tools in the future. For now, this assumes you have the tools cloned locally and are pointing to their repos directly.
:::

```yaml
tools: ./image-generation/tool.gpt, ./vision/tool.gpt, sys.read

Generate an image of a city skyline at night and write the resulting image to a file called city_skyline.png.

Take this image and write a description of it in the style of pirate.
```

External tools are tools that are defined by a `tool.gpt` file in their root directory. They can be imported into a GPTScript by specifying the path to the tool's root directory.
