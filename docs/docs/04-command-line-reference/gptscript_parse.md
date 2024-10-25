---
title: "gptscript parse"
---
## gptscript parse



```
gptscript parse [flags]
```

### Options

```
  -h, --help           help for parse
  -p, --pretty-print   Indent the json output ($GPTSCRIPT_PARSE_PRETTY_PRINT)
```

### Options inherited from parent commands

```
      --cache-dir string                Directory to store cache (default: $XDG_CACHE_HOME/gptscript) ($GPTSCRIPT_CACHE_DIR)
  -C, --chdir string                    Change current working directory ($GPTSCRIPT_CHDIR)
      --color                           Use color in output (default true) ($GPTSCRIPT_COLOR)
      --config string                   Path to GPTScript config file ($GPTSCRIPT_CONFIG)
      --confirm                         Prompt before running potentially dangerous commands ($GPTSCRIPT_CONFIRM)
      --credential-context strings      Context name(s) in which to store credentials ($GPTSCRIPT_CREDENTIAL_CONTEXT)
      --credential-override strings     Credentials to override (ex: --credential-override github.com/example/cred-tool:API_TOKEN=1234) ($GPTSCRIPT_CREDENTIAL_OVERRIDE)
      --debug                           Enable debug logging ($GPTSCRIPT_DEBUG)
      --debug-messages                  Enable logging of chat completion calls ($GPTSCRIPT_DEBUG_MESSAGES)
      --default-model string            Default LLM model to use ($GPTSCRIPT_DEFAULT_MODEL) (default "gpt-4o")
      --default-model-provider string   Default LLM model provider to use, this will override OpenAI settings ($GPTSCRIPT_DEFAULT_MODEL_PROVIDER)
      --disable-cache                   Disable caching of LLM API responses ($GPTSCRIPT_DISABLE_CACHE)
      --dump-state string               Dump the internal execution state to a file ($GPTSCRIPT_DUMP_STATE)
      --events-stream-to string         Stream events to this location, could be a file descriptor/handle (e.g. fd://2), filename, or named pipe (e.g. \\.\pipe\my-pipe) ($GPTSCRIPT_EVENTS_STREAM_TO)
  -f, --input string                    Read input from a file ("-" for stdin) ($GPTSCRIPT_INPUT_FILE)
      --no-trunc                        Do not truncate long log messages ($GPTSCRIPT_NO_TRUNC)
      --openai-api-key string           OpenAI API KEY ($OPENAI_API_KEY)
      --openai-base-url string          OpenAI base URL ($OPENAI_BASE_URL)
      --openai-org-id string            OpenAI organization ID ($OPENAI_ORG_ID)
  -o, --output string                   Save output to a file, or - for stdout ($GPTSCRIPT_OUTPUT)
  -q, --quiet                           No output logging (set --quiet=false to force on even when there is no TTY) ($GPTSCRIPT_QUIET)
      --system-tools-dir string         Directory that contains system managed tool for which GPTScript will not manage the runtime ($GPTSCRIPT_SYSTEM_TOOLS_DIR)
      --workspace string                Directory to use for the workspace, if specified it will not be deleted on exit ($GPTSCRIPT_WORKSPACE)
```

### SEE ALSO

* [gptscript](gptscript.md)	 - 

