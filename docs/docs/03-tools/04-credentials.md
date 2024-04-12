# Credentials

GPTScript supports credential provider tools. These tools can be used to fetch credentials from a secure location (or
directly from user input) and conveniently set them in the environment before running a script.

## Writing a Credential Provider Tool

A credential provider tool looks just like any other GPTScript, with the following caveats:
- It cannot call the LLM and must run a command.
- It must print contents to stdout in the format `{"env":{"ENV_VAR_1":"value1","ENV_VAR_2":"value2"}}`.
- Any args defined on the tool will be ignored.

Here is a simple example of a credential provider tool that uses the builtin `sys.prompt` to ask the user for some input:

```yaml
# my-credential-tool.gpt
name: my-credential-tool

#!/usr/bin/env bash

output=$(gptscript -q --disable-cache sys.prompt '{"message":"Please enter your fake credential.","fields":"credential","sensitive":"true"}')
credential=$(echo $output | jq -r '.credential')
echo "{\"env\":{\"MY_ENV_VAR\":\"$credential\"}}"
```

## Using a Credential Provider Tool

Continuing with the above example, this is how you can use it in a script:

```yaml
credentials: my-credential-tool.gpt

#!/usr/bin/env bash

echo "The value of MY_ENV_VAR is $MY_ENV_VAR"
```

When you run the script, GPTScript will call the credential provider tool first, set the environment variables from its
output, and then run the script body. The credential provider tool is called by GPTScript itself. GPTScript does not ask the
LLM about it or even tell the LLM about the tool.

If GPTScript has called the credential provider tool in the same context (more on that later), then it will use the stored
credential instead of fetching it again.

You can also specify multiple credential tools for the same script:

```yaml
credentials: credential-tool-1.gpt, credential-tool-2.gpt

(tool stuff here)
```

## Storing Credentials

By default, credentials are automatically stored in a config file at `$XDG_CONFIG_HOME/gptscript/config.json`.
This config file also has another parameter, `credsStore`, which indicates where the credentials are being stored.

- `file` (default): The credentials are stored directly in the config file.
- `osxkeychain`: The credentials are stored in the macOS Keychain.
- `wincred`: The credentials are stored in the Windows Credential Manager.

In order to use `osxkeychain` or `wincred` as the credsStore, you must have the `gptscript-credential-*` executable
available in your PATH. There will probably be better packaging for this in the future, but for now, you can build them
from the [repo](https://github.com/gptscript-ai/gptscript-credential-helpers). (For wincred, make sure the executable
is called `gptscript-credential-wincred.exe`.)

There will likely be support added for other credential stores in the future.

:::note
Credentials received from credential provider tools that are not on GitHub (such as a local file) will not be stored
in the credentials store.
:::

## Credential Contexts

Each stored credential is uniquely identified by the name of its provider tool and the name of its context. A credential
context is basically a namespace for credentials. If you have multiple credentials from the same provider tool, you can
switch between them by defining them in different credential contexts. The default context is called `default`, and this
is used if none is specified.

You can set the credential context to use with the `--credential-context` flag when running GPTScript. For
example:

```bash
gptscript --credential-context my-azure-workspace my-azure-script.gpt
```

Any credentials fetched for that script will be stored in the `my-azure-workspace` context. If you were to call it again
with a different context, you would be able to give it a different set of credentials.

## Listing and Deleting Stored Credentials

The `gptscript credential` command can be used to list and delete stored credentials. Running the command with no
`--credential-context` set will use the `default` credential context. You can also specify that it should list
credentials in all contexts with `--all-contexts`.

You can delete a credential by running the following command:

```bash
gptscript credential delete --credential-context <credential context> <credential tool name>
```
