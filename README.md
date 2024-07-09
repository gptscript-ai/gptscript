# GPTScript

![Demo](docs/static/img/demo.gif)

GPTScript is a framework that allows Large Language Models (LLMs) to operate and interact with various systems. These systems can range from local executables to complex applications with OpenAPI schemas, SDK libraries, or any RAG-based solutions. GPTScript is designed to easily integrate any system, whether local or remote, with your LLM using just a few lines of prompts.

Here are some sample use cases of GPTScript:
1. Chat with a local CLI - [Try it!](https://docs.gptscript.ai/examples/cli)
2. Chat with an OpenAPI compliant endpoint - [Try it!](https://docs.gptscript.ai/examples/api)
3. Chat with local files and directories - [Try it!](https://docs.gptscript.ai/examples/local-files)
4. Run an automated workflow - [Try it!](https://docs.gptscript.ai/examples/workflow)


### Getting started
MacOS and Linux (Homebrew):
```
brew install gptscript 
gptscript github.com/gptscript-ai/llm-basics-demo
```

MacOS and Linux (install.sh):
```
curl https://get.gptscript.ai/install.sh | sh
```

Windows:
```
winget install gptscript-ai.gptscript
gptscript github.com/gptscript-ai/llm-basics-demo
```

A few notes:
- You'll need an [OpenAI API key](https://help.openai.com/en/articles/4936850-where-do-i-find-my-openai-api-key)
- On Windows, after installing gptscript you may need to restart your terminal for the changes to take effect
- The above script is a simple chat-based assistant. You can ask it questions and it will answer to the best of its ability.

## Community

Join us on Discord: [![Discord](https://img.shields.io/discord/1204558420984864829?label=Discord)](https://discord.gg/9sSf4UyAMC)

## License

Copyright (c) 2024 [Acorn Labs, Inc.](http://acorn.io)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
