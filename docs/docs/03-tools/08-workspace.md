# Workspace

One concept in GPTScript is the workspace directory.
This is a directory meant to be used by tools that need to interact with the local file system.
By default, the workspace directory is a one-off temporary directory.
The workspace directory can be set with the `--workspace` argument when running GPTScript, like this:

```bash
gptscript --workspace . my-script.gpt
```

In the above example, the user’s current directory (denoted by `.`) will be set as the workspace.
Both absolute and relative paths are supported.

Regardless of whether it is set implicitly or explicitly, the workspace is then made available to the script execution as the `GPTSCRIPT_WORKSPACE_DIR` environment variable.

:::info
GPTScript does not force scripts or tools to write to, read from, or otherwise use the workspace.
The tools must decide to make use of the workspace environment variable.
:::

## The Workspace Context Tool

To make prompt-based tools aware of the workspace, you can reference the workspace context tool:

```
Context: github.com/gptscript-ai/context/workspace
```

This tells the LLM (by way of a [system message](https://platform.openai.com/docs/guides/text-generation/chat-completions-api)) what the workspace directory is,
what its initial contents are, and that if it decides to create a file or directory, it should do so in the workspace directory.
This will not, however, have any impact on code-based tools (i.e. Python, Bash, or Go tools).
Such tools will have the `GPTSCRIPT_WORKSPACE_DIR` environment variable available to them, but they must be written in such a way that they make use of it.

This context tool also automatically shares the `sys.ls`, `sys.read`, and `sys.write` tools with the tool that is using it as a context.
This is because if a tool intends to interact with the workspace, it minimally needs these tools.
