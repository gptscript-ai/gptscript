# FAQs

### I don't have Homebrew, how can I install GPTScript?

On MacOS and Linux, you can alternatively install via: `curl https://get.gptscript.ai/install.sh | sh`

On all supported systems, you download and install the archive for your platform and architecture from the [releases page](https://github.com/gptscript-ai/gptscript/releases).


### Does GPTScript have an SDK or API I can program against?

Currently, there are three SDKs being maintained: [Python](https://github.com/gptscript-ai/py-gptscript), [Node](https://github.com/gptscript-ai/node-gptscript), and [Go](https://github.com/gptscript-ai/go-gptscript). They are currently under development and are being iterated on relatively rapidly. The READMEs in each repository contain the most up-to-date documentation for the functionality of each.

### I see there's a --disable-cache flag. How does caching working in GPTScript?

GPTScript leverages caching to speed up execution and reduce LLM costs. There are two areas cached by GPTScript:
- Git commit hash lookups for tools
- LLM responses

Caching is enabled for both of these by default. It can be disabled via the `--disable-cache` flag. Below is an explanation of how these areas behave when caching is enabled and disabled.

#### Git commit hash lookups for tools

When a remote tool or context is included in your script (like so: `Tools: github.com/gptscript-ai/browser`) and then invoked during script execution, GPTScript will pull the Git repo for that tool and build it. The tool’s repo and build will be stored in your system’s cache directory (at [$XDG_CACHE_HOME](https://pkg.go.dev/os#UserCacheDir)/gptscript/repos). Subsequent invocations of the tool leverage that cache. When the cache is enabled, GPTScript will only check for a newer version of the tool once an hour; if an hour hasn’t passed since the last check, it will just use the one it has. If this is the first invocation and the tool doesn’t yet exist in the cache, it will be pulled and built as normal.

When the cache is disabled, GPTScript will check that it has the latest version of the tool (meaning the latest git commit for the repo) on every single invocation of the tool. If GPTScript determines it already has the latest version, that build will be used as-is. In other words, disabling the cache DOES NOT force GPTScript to rebuild the tool, it only forces GPTScript to always check if it has the latest version.

#### LLM responses

With regards to LLM responses, when the cache is enabled GPTScript will cache the LLM’s response to a chat completion request. Each response is stored as a gob-encoded file in $XDG_CACHE_HOME/gptscript, where the file name is a hash of the chat completion request.

It is important to note that all [messages in chat completion request](https://platform.openai.com/docs/api-reference/chat/create#chat-create-messages) are used to generate the hash that is used as the file name. This means that every message between user and LLM affects the cache lookup. So, when using GPTScript in chat mode, it is very unlikely you’ll receive a cached LLM response. Conversely, non-chat GPTScript automations are much more likely to be consistent and thus make use of cached LLM responses.