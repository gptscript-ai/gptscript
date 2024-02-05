# README.md for GPTscript Project

## Project Description

This project utilizes `GPTscript`, a powerful scripting tool that leverages OpenAI's GPT plugins to execute tasks specified in `.gpt` files. These scripts can combine traditional scripting with GPT's AI capabilities to automate complex workflows, parse information, and more.

## Usage

The primary command used to run GPT scripts is `gptscript`, which can be invoked with various flags to control the script execution. A typical usage pattern looks like this:

```bash
gptscript [flags] PROGRAM_FILE [INPUT...]
```

Where `PROGRAM_FILE` is the script to be executed, optionally followed by `INPUT` arguments to pass into the script.

## GPT File Syntax

GPT script files (`*.gpt`) follow a syntax which allows for defining tools, their descriptions, arguments, and embedded traditional scripting codes. Here's a breakdown of the syntax:

- `tools:` Followed by the tool names to be used.
- `name:` Defines the tool's name.
- `description:` Describes what the tool does.
- `args:` Lists arguments the tool accepts, describing the expected input.
- `tools:` Below the main tool definition, you can specify other tools used by this tool.
- A line containing `---` signifies the separation between tool metadata and the script code itself.

## Example `.gpt` Files and Their Syntax

The repository contains several example `.gpt` files, such as:

- `describe.gpt`: Provides functionalities for working with Go files, with tools like `ls`, `count`, `summarize`, and `compare`.
- `echo.gpt`: A simple script that echoes the provided input.
- `fib.gpt`: An example demonstrating a recursive function `myfunction`.
- `git-commit.gpt`: Automates the creation of a well-formed git commit message.
- `helloworld.gpt`: A basic script outputs the traditional "hello world" message.
- `summarize-syntax.gpt`: Meta-script for generating documentation based on `.gpt` files in a directory.
- `tables.gpt`: Using SQLite, identifies the table in a database with the most rows.

## gptscript Command Help

The help output for `gptscript` provides vital information on how to use the command along with the available flags. Here's an abbreviated listing of the flags:

- `--assemble`: Assemble tool to a single artifact, with output specified by `--output`.
- `--cache`, `--cache-dir`: Controls caching behavior and cache directory location.
- `--debug`: Enable debug logging.
- `--help`: Displays help information for `gptscript`.
- `--input`: Specify an input file or stdin.
- `--list-models`, `--list-tools`: Lists available models or built-in tools.
- `--openai-api-key`: Specifies the OpenAI API key to use.
- `--output`: Saves output to a specified file.
- `--quiet`: Suppresses output logging.
- `--sub-tool`: Selects a specific sub-tool to use from the `.gpt` file.

For a detailed description and more options, refer to the `gptscript --help` command.
