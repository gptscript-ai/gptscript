# FAQs

### I don't have Homebrew, how can I install GPTScript?

On macOS and Linux, you can alternatively install via: `curl https://get.gptscript.ai/install.sh | sh`

On all supported systems, you download and install the archive for your platform and architecture from the [releases page](https://github.com/gptscript-ai/gptscript/releases).

### Does GPTScript have an SDK or API I can program against?

Currently, there are three SDKs being maintained: [Python](https://github.com/gptscript-ai/py-gptscript), [Node](https://github.com/gptscript-ai/node-gptscript), and [Go](https://github.com/gptscript-ai/go-gptscript).
They are under development and are being iterated on relatively rapidly.
The READMEs in each repository contain the most up-to-date documentation for the functionality of each.

### I see there's a --disable-cache flag. How does caching working in GPTScript?

GPTScript leverages caching to speed up execution and reduce LLM costs. There are two areas cached by GPTScript:
- Git commit hash lookups for tools
- LLM responses

Caching is enabled for both of these by default. It can be disabled via the `--disable-cache` flag.
Below is an explanation of how these areas behave when caching is enabled and disabled.

#### Git commit hash lookups for tools

When a remote tool or context is included in your script (like so: `Tools: github.com/gptscript-ai/browser`) and then invoked during script execution,
GPTScript will pull the Git repo for that tool and build it.
The tool's repo and build will be stored in your system's cache directory (at [$XDG_CACHE_HOME](https://pkg.go.dev/os#UserCacheDir)/gptscript/repos).
Subsequent invocations of the tool leverage that cache.
When the cache is enabled, GPTScript will only check for a newer version of the tool once an hour;
if an hour hasn't passed since the last check, it will just use the one it has.
If this is the first invocation and the tool doesn't yet exist in the cache, it will be pulled and built as normal.

When the cache is disabled, GPTScript will check that it has the latest version of the tool (meaning the latest git commit for the repo) on every single invocation of the tool.
If GPTScript determines it already has the latest version, that build will be used as-is.
In other words, disabling the cache DOES NOT force GPTScript to rebuild the tool, it only forces GPTScript to always check if it has the latest version.

#### LLM responses

In regard to LLM responses, when the cache is enabled, GPTScript will cache the LLM's response to a chat completion request.
Each response is stored as a gob-encoded file in $XDG_CACHE_HOME/gptscript, where the file name is a hash of the chat completion request.

It is important to note that all [messages in chat completion request](https://platform.openai.com/docs/api-reference/chat/create#chat-create-messages) are used to generate the hash that is used as the file name.
This means that every message between user and LLM affects the cache lookup.
So, when using GPTScript in chat mode, it is very unlikely you'll receive a cached LLM response.
Conversely, non-chat GPTScript automations are much more likely to be consistent and thus make use of cached LLM responses.

### I see there's a --workspace flag. How do I make use of that?

Every invocation of GPTScript has a workspace directory available to it.
By default, this directory is a one-off temp directory, but you can override this and explicitly set a workspace using the `--workspace` flag, like so:

```
gptscript --workspace . my-script.gpt
```

In the above example, the user's current directory (denoted by `.`) will be set as the workspace. Both absolute and relative paths are supported.

Regardless of whether it is set implicitly or explicitly, the workspace is then made available to the script execution as the `GPTSCRIPT_WORKSPACE_DIR` environment variable.

:::info
GPTScript does not force scripts or tools to write to, read from, or otherwise use the workspace. The tools must decide to make use of the workspace environment variable.
:::

To make prompt-based tools workspace aware, you can reference our workspace context tool, like so:

```
Context: github.com/gptscript-ai/context/workspace
```

This tells the LLM (by way of a [system message](https://platform.openai.com/docs/guides/text-generation/chat-completions-api)) what the workspace directory is, what its initial contents are, and that if it decides to create a file or directory, it should do so in the workspace directory.
This will not, however, have any impact on code-based tools (ie python, bash, or go tools).
Such tools will have the `GPTSCRIPT_WORKSPACE_DIR` environment variable available to them, but they must be written in such a way that they make use of it.

This context also automatically shares the `sys.ls`, `sys.read`, and `sys.write` tools with the tool that is using it as a context.
This is because if a tool intends to interact with the workspace, it minimally needs these tools.

### I'm hitting GitHub's rate limit for unauthenticated requests when using GPTScript.

By default, GPTScript makes unauthenticated requests to GitHub when pulling tools.
Since GitHub's rate limits for unauthenticated requests are fairly low, running into them when developing with GPTScript is a common issue.
To avoid this, you can get GPTScript to make authenticated requests -- which have higher rate limits -- by setting the `GITHUB_AUTH_TOKEN` environment variable to your github account's PAT (Personal Access Token).
If you're already authenticated with the `gh` CLI, you can use its token by running:

```bash
export GITHUB_AUTH_TOKEN="$(gh auth token)"
```

### I'm not able to get GPTScript to work with my OpenAI account which is in "Free Trial" 

When executing GPTScript with OpenAI API key of an account in "Free Trial", user will be presented with the following error message: 
```
status code: 404, message: The model `gpt-4o` does not exist or you do not have access to it
``` 
By default, GPTScript uses `gpt-4o` model which is not available for users in "Free Trial" and this is expected behavior.

User can choose to use other supported models like `gpt-4o-mini`,`gpt-3.5-turbo` by using `--default-model <Model Name>` when executing GPTscripts. But in case of "Free Trial" account with no credits, GPTScript execution will fail with following quota exceeded error messgae which is also the expected behavior since there are no credits available.
```
error, status code: 429, message: You exceeded your current quota, please check your plan and billing details. For more information on this error, read the docs: https://platform.openai.com/docs/guides/error-codes/api-errors.
```
