---
title: Overview
slug: /
---
import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

[![Discord](https://img.shields.io/discord/1204558420984864829?label=Discord)](https://discord.gg/9sSf4UyAMC)

![Demo](/img/demo.gif)

GPTScript is a framework that allows Large Language Models (LLMs) to operate and interact with various systems. These systems can range from local executables to complex applications with OpenAPI schemas, SDK libraries, or any RAG-based solutions. GPTScript is designed to easily integrate any system, whether local or remote, with your LLM using just a few lines of prompts.

Here are some sample use cases of GPTScript:
1. Chat with a local CLI - [Try it!](examples/cli)
2. Chat with an OpenAPI compliant endpoint - [Try it!](examples/api)
3. Chat with local files and directories - [Try it!](examples/local-files)
4. Run an automated workflow - [Try it!](examples/workflow)

### Getting Started

<Tabs>
    <TabItem value="MacOS and Linux">
    ```shell
    brew install gptscript-ai/tap/gptscript 
    gptscript github.com/gptscript-ai/llm-basics-demo
    ```
    A few notes:
    - You'll need an [OpenAI API key](https://help.openai.com/en/articles/4936850-where-do-i-find-my-openai-api-key)
    - The above script is a simple chat-based assistant. You can ask it questions and it will answer to the best of its ability.
    </TabItem>
    <TabItem value="Windows">
    ```shell
    winget install gptscript-ai.gptscript
    gptscript github.com/gptscript-ai/llm-basics-demo
    ```
    A few notes:
    - You'll need an [OpenAI API key](https://help.openai.com/en/articles/4936850-where-do-i-find-my-openai-api-key)
    - After installing gptscript you may need to restart your terminal for the changes to take effect
    - The above script is a simple chat-based assistant. You can ask it questions and it will answer to the best of its ability.
    </TabItem>
</Tabs>
