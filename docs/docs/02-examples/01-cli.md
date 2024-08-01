# Chat with a Local CLI

GPTScript makes it easy to write AI integrations with CLIs and other executables available on your local workstation.
You can describe complex requests in plain English and GPTScript will figure out the best CLI commands to make that happen.
This guide will show you how to build a GPTScript that integrates with the `gh` CLI for GitHub.

:::warning
This script **does not install** or configure `gh`. We assume you've done that already.
You must be logged in via `gh auth login`. [See here for more details](https://docs.github.com/en/github-cli/github-cli/quickstart)
:::

You should have basic familiarity with [tools](../03-tools/01-using.md) before starting this guide.

## Getting Started

First, open up a new file in your favorite editor. We'll call the file `cli-demo.gpt`.

```
vim cli-demo.gpt
```

All edits below are assumed to be in this file.

## Create the entrypoint tool

Let's start by adding the main tool to the file:

```
Context: learn-gh
Context: github.com/gptscript-ai/context/cli
Chat: true

You have the gh cli available to you. Use it to accomplish the tasks that the user asks of you.

```

Let's walk through this tool line by line.

Each `Context` line references a context tool that will be run before the tool itself runs.
Context tools provide helpful output for the LLM to understand its capabilities and what it is supposed to do.
The first, `learn-gh`, we will define later in this file.
The second, `github.com/gptscript-ai/context/cli`, provides information to the LLM about the operating system that GPTScript is running on,
and gives it access to the `sys.exec` built-in tool, which is used to run commands.

`Chat: true` turns this tool into a "chat-able" tool, which we also call an "agent".
This causes the tool to run as an interactive chatbot, asking for user input and providing output.
If `Chat` is set to `false` (or not specified at all), the tool will run once without user interaction and exit.
This is useful for automated tasks, but right now we are working on an agent, so we set it to `true`.

Lastly, there is the **tool body**, which in this case is a simple prompt, letting the LLM know that it should use the `gh` command and follow the user's instructions.
The tool body specifies what the tool should actually do. It can be a prompt or raw code like Python, JavaScript, or bash.
For chat-able tools, the tool body must be a prompt.

### The learn-gh context tool

Next, add this to the file for the `learn-gh` context tool:

```
---
Name: learn-gh

#!/usr/bin/env bash

echo "The following is the help text for the gh cli and some of its sub-commands. Use these when figuring out how to construct new commands. Note that the --search flag is used for filtering and sorting as well; there is no dedicated --sort flag."
gh --help
gh repo --help
gh issue --help
gh issue list --help
gh issue create --help
gh issue comment --help
gh issue delete --help
gh issue edit --help
gh pr --help
gh pr create --help
gh pr checkout --help
gh release --help
gh release create --help
```

The `---` at the top of this tool is a block separator. It's how we delineate tools within a script file.

This tool has a `Name` field. We named this tool `learn-gh` so that it matches the `Context: learn-gh` line from the entrypoint tool.

The body of this tool is a bash script, rather than a prompt.
This context tool will be run by GPTScript automatically at the start of execution, and its output will be provided to the LLM.
We're running a bunch of `--help` commands in the `gh` CLI so that the LLM can understand how to use it.
GPTScript knows that this tool body is a script rather than a prompt because it begins with `#!`.

## Running the tool

Now try running the tool:

```
gptscript cli-demo.gpt
```

Once you're chatting, try asking it do something like "Open an issue in gptscript-ai/gptscript with a title and body that says Hi from me and states how wonderful gptscript is but jazz it up and make it unique".
GPTScript will ask for confirmation before it runs each command, so you can make sure that it only runs the commands you want it to.

## A note on context tools

Context tools are a powerful way to provide additional information to the LLM, but they are not always necessary.
If you are working with a system that the LLM already understands well, you will probably not need to provide additional context.
When writing your own tools, it may take some trial and error to determine whether a context tool is needed.
If the LLM frequently hallucinates subcommands or arguments, it is probably worth adding a context tool to provide more information about the CLI.

## Next Steps

- You can check out some of our other guides available in this section of the docs
- You can dive deeper into the options available when [writing scripts](/tools/gpt-file-reference)
