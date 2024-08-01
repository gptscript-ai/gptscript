# How it works

**_GPTScript is composed of tools._** Each tool performs a series of actions similar to a function. Tools have other tools available to them that can be invoked similar to a function call. While similar to a function, the tools are
primarily implemented with a natural language prompt. **_The interaction of the tools is determined by the AI model_**,
the model determines if the tool needs to be invoked and what arguments to pass. Tools are intended to be implemented
with a natural language prompt but can also be implemented with a command or HTTP call.

## Example

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

