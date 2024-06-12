# Credentials

Some GPTScript tools will use [credential tools](03-tools/04-credential-tools.md) to get sensitive information like API keys from the user.
These credentials will be stored in a credential store and are used to set environment variables before executing a tool.

GPTScript itself will also prompt you for your OpenAI API key and save it in the credential store if the
`OPENAI_API_KEY` environment variable is not set. The environment variable always overrides the value stored in the
credential store.

## Credential Store

There are different options available for credential stores, depending on your operating system.
When you first run GPTScript, the default credential store for your operating system will be selected.

You can change the credential store by modifying the `credsStore` field in your GPTScript configuration file.
The configuration file is located in the following location based on your operating system:
- Windows: `%APPDATA%\Local\gptscript\config.json`
- macOS: `$HOME/Library/Application Support/gptscript/config.json`
- Linux: `$XDG_CONFIG_HOME/gptscript/config.json`

(Note: if you set the `XDG_CONFIG_HOME` environment variable on macOS, then the same path as Linux will be used.)

The configured credential store will be automatically downloaded and compiled from the [gptscript-ai/gptscript-credential-helpers](https://github.com/gptscript-ai/gptscript-credential-helpers)
repository, other than the `file` store, which is built-in to GPTScript itself.
The `wincred` and `osxkeychain` stores do not require any external dependencies in order to compile correctly.
The `secretservice` store on Linux may require some extra packages to be installed, depending on your distribution.

## Credential Store Options

### Wincred (Windows)

Wincred, or the Windows Credential Manager, is the default credential store for Windows.
This is Windows' built-in credential manager that securely stores credentials for Windows applications.
This credential store is called `wincred` in GPTScript's configuration.

### macOS Keychain (macOS)

The macOS Keychain is the default credential store for macOS.
This is macOS' built-in password manager that securely stores credentials for macOS applications.
This credential store is called `osxkeychain` in GPTScript's configuration.

### File (all operating systems)

"File" is the default credential store for every other operating system besides Windows and macOS, but it
can also be configured on Windows and macOS. This will store credentials **unencrypted** inside GPTScript's
configuration file.
This credential store is called `file` in GPTScript's configuration.

### D-Bus Secret Service (Linux)

The D-Bus Secret Service can be used as the credential store for Linux systems with a desktop environment that supports it.
This credential store is called `secretservice` in GPTScript's configuration.

### Pass (Linux)

Pass can be used as the credential store for Linux systems. This requires the `pass` package to be installed
and configured. See [this guide](https://www.howtogeek.com/devops/how-to-use-pass-a-command-line-password-manager-for-linux-systems/)
for information about how to set it up.
This credential store is called `pass` in GPTScript's configuration.

## GPTScript `credential` Command

The `gptscript credential` command can be used to interact with your stored credentials.
`gptscript credential` without any arguments will list all stored credentials.
`gptscript credential delete <credential name>` will delete the specified credential, and you will be
prompted to enter it again the next time a tool that requires it is run.

## See Also

For more advanced credential usage, including credential contexts, writing credential tools, and using
credential tools, see [the credential tools documentation](03-tools/04-credential-tools.md).
