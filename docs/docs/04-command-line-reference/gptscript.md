---
title: "gptscript"
---
## gptscript



```
gptscript [flags] PROGRAM_FILE [INPUT...]
```

### Options

```
      --cache-dir string                    Directory to store cache (default: $XDG_CACHE_HOME/gptscript) ($GPTSCRIPT_CACHE_DIR)
      --chat-state string                   The chat state to continue, or null to start a new chat and return the state ($GPTSCRIPT_CHAT_STATE)
  -C, --chdir string                        Change current working directory ($GPTSCRIPT_CHDIR)
      --color                               Use color in output (default true) ($GPTSCRIPT_COLOR)
      --config string                       Path to GPTScript config file ($GPTSCRIPT_CONFIG)
      --confirm                             Prompt before running potentially dangerous commands ($GPTSCRIPT_CONFIRM)
      --credential-context string           Context name in which to store credentials ($GPTSCRIPT_CREDENTIAL_CONTEXT) (default "default")
      --credential-override strings         Credentials to override (ex: --credential-override github.com/example/cred-tool:API_TOKEN=1234) ($GPTSCRIPT_CREDENTIAL_OVERRIDE)
      --debug                               Enable debug logging ($GPTSCRIPT_DEBUG)
      --debug-messages                      Enable logging of chat completion calls ($GPTSCRIPT_DEBUG_MESSAGES)
      --default-model string                Default LLM model to use ($GPTSCRIPT_DEFAULT_MODEL) (default "gpt-4o")
      --default-model-provider string       Default LLM model provider to use, this will override OpenAI settings ($GPTSCRIPT_DEFAULT_MODEL_PROVIDER)
      --disable-cache                       Disable caching of LLM API responses ($GPTSCRIPT_DISABLE_CACHE)
      --disable-tui                         Don't use chat TUI but instead verbose output ($GPTSCRIPT_DISABLE_TUI)
      --dump-state string                   Dump the internal execution state to a file ($GPTSCRIPT_DUMP_STATE)
      --events-stream-to string             Stream events to this location, could be a file descriptor/handle (e.g. fd://2), filename, or named pipe (e.g. \\.\pipe\my-pipe) ($GPTSCRIPT_EVENTS_STREAM_TO)
      --force-chat                          Force an interactive chat session if even the top level tool is not a chat tool ($GPTSCRIPT_FORCE_CHAT)
      --force-sequential                    Force parallel calls to run sequentially ($GPTSCRIPT_FORCE_SEQUENTIAL)
      --github-enterprise-hostname string   The host name for a Github Enterprise instance to enable for remote loading ($GPTSCRIPT_GITHUB_ENTERPRISE_HOSTNAME)
  -h, --help                                help for gptscript
  -f, --input string                        Read input from a file ("-" for stdin) ($GPTSCRIPT_INPUT_FILE)
      --list-models                         List the models available and exit ($GPTSCRIPT_LIST_MODELS)
      --list-tools                          List built-in tools and exit ($GPTSCRIPT_LIST_TOOLS)
      --no-trunc                            Do not truncate long log messages ($GPTSCRIPT_NO_TRUNC)
      --openai-api-key string               OpenAI API KEY ($OPENAI_API_KEY)
      --openai-base-url string              OpenAI base URL ($OPENAI_BASE_URL)
      --openai-org-id string                OpenAI organization ID ($OPENAI_ORG_ID)
  -o, --output string                       Save output to a file, or - for stdout ($GPTSCRIPT_OUTPUT)
  -q, --quiet                               No output logging (set --quiet=false to force on even when there is no TTY) ($GPTSCRIPT_QUIET)
      --save-chat-state-file string         A file to save the chat state to so that a conversation can be resumed with --chat-state ($GPTSCRIPT_SAVE_CHAT_STATE_FILE)
      --sub-tool string                     Use tool of this name, not the first tool in file ($GPTSCRIPT_SUB_TOOL)
      --ui                                  Launch the UI ($GPTSCRIPT_UI)
      --workspace string                    Directory to use for the workspace, if specified it will not be deleted on exit ($GPTSCRIPT_WORKSPACE)
```

### SEE ALSO

* [gptscript credential](gptscript_credential.md)	 - List stored credentials
* [gptscript eval](gptscript_eval.md)	 - 
* [gptscript fmt](gptscript_fmt.md)	 - 
* [gptscript getenv](gptscript_getenv.md)	 - Looks up an environment variable for use in GPTScript tools
* [gptscript parse](gptscript_parse.md)	 - 

