# Credential Tools

GPTScript supports credential provider tools. These tools can be used to fetch credentials from a secure location (or
directly from user input) and conveniently set them in the environment before running a script.

## Writing a Credential Provider Tool

A credential provider tool looks just like any other GPTScript, with the following caveats:
- It cannot call the LLM and must run a command.
- It must print contents to stdout in the format `{"env":{"ENV_VAR_1":"value1","ENV_VAR_2":"value2"}}`.

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

You can also specify multiple credential tools for the same script, but they must be on separate lines:

```yaml
credentials: credential-tool-1.gpt
credentials: credential-tool-2.gpt

(tool stuff here)
```

### Generic Credential Tool

GPTScript also provides a generic credential tool (`github.com/gptscript-ai/credential`) that is ideal for cases
where you only need to set one environment variable. Here is an example of how to use it:

```yaml
credentials: github.com/gptscript-ai/credential as myCredentialName with MY_ENV_VAR as env and "this message will be displayed to the user" as message and key as field

(tool stuff here)
```

When this tool is run, the user will be shown the message and prompted to enter a key. The value entered will be set as
the environment variable `MY_ENV_VAR` and stored in a credential called `myCredentialName`.

See [the repo](https://github.com/gptscript-ai/credential) for more information.

## Credential Tool Arguments

A credential tool may define arguments. Here is an example:

```yaml
name: my-credential-tool
args: env: the environment variable to set
args: val: the value to set it to

#!/usr/bin/env bash

echo "{\"env\":{\"$ENV\":\"$VAL\"}}"
```

When you reference this credential tool in another file, you can use syntax like this to set both arguments:

```yaml
credential: my-credential-tool.gpt with MY_ENV_VAR as env and "my value" as val

(tool stuff here)
```

In this example, the tool's output would be `{"env":{"MY_ENV_VAR":"my value"}}`

## Storing Credentials

By default, credentials are automatically stored in the credential store. Read the [main credentials page](../02-credentials.md)
for more information about the credential store.

:::note
Credentials received from credential provider tools that are not on GitHub (such as a local file) and do not have an alias
will not be stored in the credentials store.
:::

## Credential Aliases

When you reference a credential tool in your script, you can give it an alias using the `as` keyword like this:

```yaml
credentials: my-credential-tool.gpt as myAlias

(tool stuff here)
```

This will store the resulting credential with the name `myAlias`.
This is useful when you want to reference the same credential tool in scripts that need to handle different credentials,
or when you want to store credentials that were provided by a tool that is not on GitHub.

## Credential Contexts

Each stored credential is uniquely identified by the name of its provider tool (or alias, if one was specified) and the name of its context.
A credential context is basically a namespace for credentials. If you have multiple credentials from the same provider tool,
you can switch between them by defining them in different credential contexts. The default context is called `default`,
and this is used if none is specified.

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
gptscript credential delete --credential-context <credential context> <credential name>
```

The `--show-env-vars` argument will also display the names of the environment variables that are set by the credential.
This is useful when working with credential overrides.

## Credential Overrides (Advanced)

You can bypass credential tools and stored credentials by setting the `--credential-override` argument (or the
`GPTSCRIPT_CREDENTIAL_OVERRIDE` environment variable) when running GPTScript. To set up a credential override, you
need to be aware of which environment variables the credential tool sets. You can find this out by running the
`gptscript credential --show-env-vars` command.

To override multiple credentials, specify the `--credential-override` argument multiple times.

### Format

:::info
In the examples that follow, `toolA`, `toolB`, etc. are the names of credentials.
By default, a credential has the same name as the tool that created it.
This can be overridden with a credential alias, i.e. `credential: my-cred-tool.gpt as myAlias`.
If the credential has an alias, use it instead of the tool name when you specify an override.
:::

The `--credential-override` argument must be formatted in one of the following two ways:

#### 1. Key-Value Pairs

`toolA:ENV_VAR_1=value1,ENV_VAR_2=value2`

In this example, both `toolA` provides the variables `ENV_VAR_1` and `ENV_VAR_2`.
This will set the environment variables `ENV_VAR_1` and `ENV_VAR_2` to the specific values `value1` and `value2`.

#### 2. Environment Variables

`toolA:ENV_VAR_1,ENV_VAR_2`

In this example, `toolA` provides the variables `ENV_VAR_1` and `ENV_VAR_2`,
This will read the values of `ENV_VAR_1` through `ENV_VAR_4` from the current environment and set them for the credential.
This is a direct mapping of environment variable names. **This is not recommended when overriding credentials for
multiple tools that use the same environment variable names.**
