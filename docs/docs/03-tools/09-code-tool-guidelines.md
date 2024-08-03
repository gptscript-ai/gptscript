# Code Tool Guidelines

GPTScript can handle the packaging and distribution of code-based tools via GitHub repos.
For more information on how this works, see the [authoring guide](02-authoring.md#sharing-tools).

This guide provides guidelines for setting up GitHub repos for proper tool distribution.

## Common Guidelines

### `tool.gpt` or `agent.gpt` file

Every repo should have a `tool.gpt` or `agent.gpt` file. This is the main logic of the tool.
If both files exist, GPTScript will use the `agent.gpt` file and ignore the `tool.gpt` file.
Your repo can have other `.gpt` files that are referenced by the main file, but there must be a `tool.gpt` or `agent.gpt` file present.

Under most circumstances, this file should live in the root of the repo.
If you are using a single repo for the distribution of multiple tools (see [gptscript-ai/context](https://github.com/gptscript-ai/context) for an example),
then you can have the `tool.gpt`/`agent.gpt` file in a subdirectory, and the tool will now be able to be referenced as `github.com/<user>/<repo>/<subdirectory>`.

### Name and Description directives

We recommend including a `Name` and `Description` directive for your tool.
This is useful for both people and LLMs to understand what the tool will do and when to use it.

### Parameters

Any parameters specified in the tool will be available as environment variables in your code.
We recommend handling parameters that way, rather than using command-line arguments.

## Python Guidelines

### Calling Python in the tool body

The body of the `tool.gpt`/`agent.gpt` file needs to call Python. This can be done as an inline script like this:

```
Name: my-python-tool

#!python3

print('hello world')
```

An inline script like this is only recommended for simple use cases that don't need external dependencies.

If your use case is more complex or requires external dependencies, you can reference a Python script in your repo, like this:

```
Name: my-python-tool

#!/usr/bin/env python3 ${GPTSCRIPT_TOOL_DIR}/tool.py
```

(This example assumes that your entrypoint to your Python program is in a file called `tool.py`. You can call it what you want.)

### `requirements.txt` file

If your Python program needs any external dependencies, you can create a `requirements.txt` file at the same level as
your `tool.gpt`/`agent.gpt` file. GPTScript will handle downloading the dependencies before it runs the tool.

The file structure should look something like this:

```
.
├── requirements.txt
├── tool.py
└── tool.gpt
```

## JavaScript (Node.js) Guidelines

### Calling Node.js in the tool body

The body of the `tool.gpt`/`agent.gpt` file needs to call Node. This can be done as an inline script like this:

```
Name: my-node-tool

#!node

console.log('hello world')
```

An inline script like this is only recommended for simple use cases that don't need external dependencies.

If your use case is more complex or requires external dependencies, you can reference a Node script in your repo, like this:

```
Name: my-node-tool

#!/usr/bin/env node ${GPTSCRIPT_TOOL_DIR}/tool.js
```

(This example assumes that your entrypoint to your Node program is in a file called `tool.js`. You can call it what you want.)

### `package.json` file

If your Node program needs any external dependencies, you can create a `package.json` file at the same level as
your `tool.gpt`/`agent.gpt` file. GPTScript will handle downloading the dependencies before it runs the tool.

The file structure should look something like this:

```
.
├── package.json
├── tool.js
└── tool.gpt
```

## Go Guidelines

GPTScript does not support inline code for Go, so you must call to an external program from the tool body like this:

```
Name: my-go-tool

#!${GPTSCRIPT_TOOL_DIR}/bin/gptscript-go-tool
```

:::important
Unlike the Python and Node cases above where you can name the file anything you want, Go tools must be `#!${GPTSCRIPT_TOOL_DIR}/bin/gptscript-go-tool`.
:::

GPTScript will build the Go program located at `./main.go` to a file called `./bin/gptscript-go-tool` before running the tool.
All of your dependencies need to be properly specified in a `go.mod` file.

The file structure should look something like this:

```
.
├── go.mod
├── go.sum
├── main.go
└── tool.gpt
```
