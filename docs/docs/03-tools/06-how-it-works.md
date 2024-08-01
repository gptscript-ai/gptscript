# How it works

**_GPTScript is fundamentally composed of tools._** Each tool is either a natural language prompt for the LLM, or is
programmatic (i.e. a command, script, or program to be run). Tools that use a natural language prompt can also invoke
other tools, similar to function calls. The LLM decides when a tool needs to be invoked and sets the parameters to pass to it.

## Example

Below are two tool definitions, separated by `---`.
The first tool in the file (often referred to as the "entrypoint tool") does not need a name and description,
but a name is required for all other tools in the file, and a description is recommended.
The entrypoint tool also has the line `Tools: bob` meaning that the tool named `bob` is available to be called if needed.

```yaml
Tools: bob

Ask Bob how he is doing and let me know exactly what he said.

---
Name: bob
Description: I'm Bob, a friendly guy.
Param: question: The question to ask Bob.

When asked how I am doing, respond with "Thanks for asking "${question}", I'm doing great fellow friendly AI tool!"
```

Put the above content in a file named `bob.gpt` and run the following command:

```bash
gptscript bob.gpt
```

```
OUTPUT:

Bob said, "Thanks for asking 'How are you doing?', I'm doing great fellow friendly AI tool!"
```

Tools can be implemented by invoking a program instead of a natural language prompt.
The below example is the same as the previous example but implements Bob using Python.

```yaml
Tools: bob

Ask Bob how he is doing and let me know exactly what he said.

---
Name: bob
Description: I'm Bob, a friendly guy.
Param: question: The question to ask Bob.

#!python3

import os

print(f"Thanks for asking {os.environ['question']}, I'm doing great fellow friendly AI tool!")
```

With these basic building blocks you can create complex scripts with AI interacting with AI, your local system, data, or external services.
