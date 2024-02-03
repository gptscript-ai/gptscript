# README for GPT File Syntax

## Overview
GPT files found in this project use a custom syntax to describe various shell tools and their respective functionalities. Each file contains definitions for tools including their names, descriptions, arguments, and executable code. Below is an explanation of the syntax based on the contents of the GPT files reviewed (`./describe.gpt`, `./summarize-syntax.gpt`, `./tables.gpt`).

## Syntax Description

### General Format
GPT files follow a structured format that includes a preamble (indicating available tools), followed by metadata (descriptions of each tool), and Bash script examples for the tool's operation. 

Here's the general layout of a GPT file definition:
```
Tools: <tool_list>
<empty_line>
--- (3 hyphens to separate each tool definition)
name: <tool_name>
description: <description_of_tool>
arg: <argument_description> (Optional and re-occurring)
<empty_line>
<executable_code_block>
```

### Tool Definition Header
A tool definition always starts with three hyphens (`---`). This is a separator used to distinguish between different tool descriptions within the file.

### Name
The `name` field defines the name of the tool. It is used to invoke a specific functionality.

### Description
The `description` field provides a concise explanation of what the tool does.

### Arguments
The `arg` fields describe the arguments that the tool accepts. Each argument is paired with a brief description. Argument definitions are optional and can be repeated if a tool accepts multiple arguments.

### Executable Code Block
Following the metadata, an executable code block is provided. It is written in Bash and contains the script to execute the tool's functionality.

### Examples
Here are examples extracted from the GPT files which follow the syntax described:

#### From `./describe.gpt`
```
---
name: ls
description: Recursively lists all go files

#!/bin/bash

find . -name '*.go'
```
#### From `./summarize-syntax.gpt`
```
---
name: cat
description: Reads the contents of a file
arg: file: The filename to read

#!/bin/bash

cat ${FILE} </dev/null
```
#### From `./tables.gpt`
```
---
Name: sqlite
Description: The sqlite command line program ... to run sqlite command or SQL statements against a database.
Arg: databaseFile: The filename of the database to open
Arg: cmd: The sqlite command or sql statement to run.

#!/bin/bash

sqlite3 ${DATABASEFILE} -cmd "${CMD}" </dev/null
```

## Additional Observations
- The fields and code block are separated by a new line.
- The arguments are indicated as `arg: argument_name: argument_description`.
- The executable code is introduced with a `#!/bin/bash` shebang line and is expected to replace the argument placeholders (`${NAME}`) with actual values during execution.

Please note that this description of the syntax is based on the provided examples and may not cover variations which are not included in those files. Additional GPT files may include diverse structures and should be reviewed to check for consistency with this syntax.

