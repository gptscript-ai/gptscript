# Authoring Tools

You can author your own tools for your use or to share with others.
The process for authoring a tool is as simple as creating a `tool.gpt` file in the root directory of your project.
This file is itself a GPTScript that defines the tool's name, description, and what it should do.

## Quickstart

This is a guide for writing portable tools for GPTScript. The supported languages currently are Python, NodeJS, and Go. This guide uses Python but you can see documentation for the other language below.

### 1. Write the code

Create a file called `tool.py` with the following contents:

```python
import os
import requests

print(requests.get(os.getenv("url")).text)
```

Create a file called `requirements.txt` with the following contents:

```
requests
```

### 2. Create the tool

Create a file called `tool.gpt` with the following contents:

```
Description: Returns the contents of a webpage.
Param: url: The URL of the webpage.

#!/usr/bin/env python3 ${GPTSCRIPT_TOOL_DIR}/tool.py
```

:::tip
Every param becomes an environment variable when the tool is invoked. So instead of accepting params using flags like `--size="${size}"`, your program can just read the `size` environment variable.
:::

The `GPTSCRIPT_TOOL_DIR` environment variable is automatically populated by GPTScript so that the tool
will be able to find the `tool.py` file no matter what the user's current working directory is.

If you make the tool available in a public GitHub repo, then you will be able to refer to it by
the URL, i.e. `github.com/<user>/<repo name>`. GPTScript will automatically set up a Python virtual
environment, install the required packages, and execute the tool.

### 3. Use the tool

Here is an example of how you can use the tool once it is on GitHub:

```
Tools: github.com/<user>/<repo name>

Get the contents of https://github.com
```

You can also run the tool directly and set the parameter from the command line using a JSON string:

```bash
gptscript github.com/<user>/<repo name> '{"url": "https://github.com"}'
```

## Sharing Tools

GPTScript is designed to easily export and import tools. Doing this is currently based entirely around the use of GitHub repositories. You can export a tool by creating a GitHub repository and ensuring you have the `tool.gpt` file in the root of the repository. You can then import the tool into a GPTScript by specifying the URL of the repository in the `tools` section of the script. For example, we can leverage the `image-generation` tool by adding the following line to a GPTScript:

```yaml
tools: github.com/gptscript-ai/dalle-image-generation

Generate an image of a city skyline at night.
```

### Supported Languages

GPTScript can execute any binary that you ask it to. However, it can also manage the installation of a language runtime and dependencies for you. Currently this is only supported for a few languages. Here are the supported languages and examples of tools written in those languages:

| Language  | Example                                                                                                        |
|-----------|----------------------------------------------------------------------------------------------------------------|
| `Python`  | [Image Generation](https://github.com/gptscript-ai/dalle-image-generation) - Generate images based on a prompt |
| `Node.js` | [Vision](https://github.com/gptscript-ai/gpt4-v-vision) - Analyze and interpret images                         |
| `Golang`  | [Search](https://github.com/gptscript-ai/search) - Use various providers to search the internet                |


### Automatic Documentation

Each GPTScript tool is self-documented using the `tool.gpt` file. You can automatically generate documentation for your tools by visiting `tools.gptscript.ai/<github repo url>`. This documentation site allows others to easily search and explore the tools that have been created. 

You can add more information about how to use your tool by adding an `examples` directory to your repository and adding a collection of `.gpt` files that demonstrate how to use your tool. These examples will be automatically included in the documentation.

For more information and to explore existing tools, visit [tools.gptscript.ai](https://tools.gptscript.ai).
