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
Name: my-credential-tool

#!/usr/bin/env bash

output=$(gptscript -q --disable-cache sys.prompt '{"message":"Please enter your fake credential.","fields":"credential","sensitive":"true"}')
credential=$(echo $output | jq -r '.credential')
echo "{\"env\":{\"MY_ENV_VAR\":\"$credential\"}}"
```

## Using a Credential Provider Tool

Continuing with the above example, this is how you can use it in a script:

```yaml
Credentials: my-credential-tool.gpt as myCred

#!/usr/bin/env bash

echo "The value of MY_ENV_VAR is $MY_ENV_VAR"
```

:::note
GPTScript accepts `Cred:`, `Creds:`, `Credential:`, and `Credentials:` as valid directives.
:::

When you run the script, GPTScript will call the credential provider tool first, set the environment variables from its
output, and then run the script body. The credential provider tool is called by GPTScript itself. GPTScript does not ask the
LLM about it or even tell the LLM about the tool.

If GPTScript has called the credential provider tool in the same context (more on that later), then it will use the stored
credential instead of fetching it again.

To delete the credential that just got stored, run `gptscript credential delete myCred`.

You can also specify multiple credential tools for the same script, but they must be on separate lines:

```yaml
Credentials: credential-tool-1.gpt
Credentials: credential-tool-2.gpt

(tool stuff here)
```

### Generic Credential Tool

GPTScript also provides a generic credential tool (`github.com/gptscript-ai/credential`) that is ideal for cases
where you only need to set one environment variable. Here is an example of how to use it:

```yaml
Credentials: github.com/gptscript-ai/credential as myCredentialName with MY_ENV_VAR as env and "this message will be displayed to the user" as message and key as field

(tool stuff here)
```

When this tool is run, the user will be shown the message and prompted to enter a key. The value entered will be set as
the environment variable `MY_ENV_VAR` and stored in a credential called `myCredentialName`.

See [the repo](https://github.com/gptscript-ai/credential) for more information.

## Credential Tool Parameters

A credential tool may define parameters. Here is an example:

```yaml
Name: my-credential-tool
Parameter: env: the environment variable to set
Parameter: val: the value to set it to

#!/usr/bin/env bash

echo "{\"env\":{\"$ENV\":\"$VAL\"}}"
```

When you reference this credential tool in another file, you can use syntax like this to set both parameters:

```yaml
Credential: my-credential-tool.gpt with MY_ENV_VAR as env and "my value" as val

(tool stuff here)
```

In this example, the tool's output would be `{"env":{"MY_ENV_VAR":"my value"}}`

## Storing Credentials

By default, credentials are automatically stored in the credential store. Read the [main credentials page](../06-credentials.md)
for more information about the credential store.

:::note
Credentials received from credential provider tools that are not on GitHub (such as a local file) and do not have an alias
will not be stored in the credentials store.
:::

## Credential Aliases

When you reference a credential tool in your script, you can give it an alias using the `as` keyword like this:

```yaml
Credentials: my-credential-tool.gpt as myAlias

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

You can set the credential context to use with the `--credential-context` flag when running GPTScript. For example:

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

## Credential Refresh (Advanced)

Some use cases (such as OAuth) may involve the need to refresh expired credentials.
To support this, your credential tool can return other fields besides `env` in its JSON output.
This is the full list of supported fields in the credential tool output:

- `env` (type: object) - The environment variables to set.
- `expiresAt` (type: string, timestamp in RFC3339 format) - The time when the credential expires.
- `refreshToken` (type: string) - The refresh token to use to refresh the credential.

When GPTScript tries to use a credential that has a defined `expiresAt` time, it will check if the credential has expired.
If the credential has expired, it will run the credential tool again, and the current value of the credential will be
set to the environment variable `GPTSCRIPT_EXISTING_CREDENTIAL` as a JSON string. This way, the credential tool can check for
that environment variable, and if it is set, get the refresh token from the existing credential and use it to refresh and return a new credential,
typically without user interaction.

For an example of a tool that uses the refresh feature, see the [Gateway OAuth2 tool](https://github.com/gptscript-ai/gateway-oauth2).

### GPTSCRIPT_CREDENTIAL_EXPIRATION environment variable

When a tool references a credential tool, GPTScript will add the environment variables from the credential to the tool's
environment before executing the tool. If at least one of the credentials has an `expiresAt` field, GPTScript will also
set the environment variable `GPTSCRIPT_CREDENTIAL_EXPIRATION`, which contains the nearest expiration time out of all
credentials referenced by the tool, in RFC 3339 format. That way, it can be referenced in the tool body if needed.
Here is an example:

```
Credential: my-credential-tool.gpt as myCred

#!python3

import os

print("myCred expires at " + os.getenv("GPTSCRIPT_CREDENTIAL_EXPIRATION", ""))
```

## Stacked Credential Contexts (Advanced)

When setting the `--credential-context` argument in GPTScript, you can specify multiple contexts separated by commas.
We refer to this as "stacked credential contexts", or just stacked contexts for short. This allows you to specify an order
of priority for credential contexts. This is best explained by example.

### Example: stacked contexts when running a script that uses a credential

Let's say you have two contexts, `one` and `two`, and you specify them like this:

```bash
gptscript --credential-context one,two my-script.gpt
```

```
Credential: my-credential-tool.gpt as myCred

<tool stuff here>
```

When GPTScript runs, it will first look for a credential called `myCred` in the `one` context.
If it doesn't find it there, it will look for it in the `two` context. If it also doesn't find it there,
it will run the `my-credential-tool.gpt` tool to get the credential. It will then store the new credential into the `one`
context, since that has the highest priority.

### Example: stacked contexts when listing credentials

```bash
gptscript --credential-context one,two credentials
```

When you list credentials like this, GPTScript will print out the information for all credentials in contexts one and two,
with one exception. If there is a credential name that exists in both contexts, GPTScript will only print the information
for the credential in the context with the highest priority, which in this case is `one`.

(To see all credentials in all contexts, you can still use the `--all-contexts` flag, and it will show all credentials,
regardless of whether the same name appears in another context.)

### Example: stacked contexts when showing credentials

```bash
gptscript --credential-context one,two credential show myCred
```

When you show a credential like this, GPTScript will first look for `myCred` in the `one` context. If it doesn't find it
there, it will look for it in the `two` context. If it doesn't find it in either context, it will print an error message.

:::note
You cannot specify stacked contexts when doing `gptscript credential delete`. GPTScript will return an error if
more than one context is specified for this command.
:::
