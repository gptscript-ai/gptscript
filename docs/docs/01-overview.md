---
title: Overview
slug: /
---

[![Discord](https://img.shields.io/discord/1204558420984864829?label=Discord)](https://discord.gg/9sSf4UyAMC)

GPTScript is a new scripting language to automate your interaction with a Large Language Model (LLM), namely OpenAI. The ultimate goal is to create a natural language programming experience. The syntax of GPTScript is largely natural language, making it very easy to learn and use. Natural language prompts can be mixed with traditional scripts such as bash and python or even external HTTP service calls. With GPTScript you can do just about anything, like [plan a vacation](https://github.com/gptscript-ai/gptscript/blob/main/examples/travel-agent.gpt), [edit a file](https://github.com/gptscript-ai/gptscript/blob/main/examples/add-go-mod-dep.gpt), [run some SQL](https://github.com/gptscript-ai/gptscript/blob/main/examples/sqlite-download.gpt), or [build a mongodb/flask app](https://github.com/gptscript-ai/gptscript/blob/main/examples/hacker-news-headlines.gpt).

:::note
We are currently exploring options for interacting with local models using GPTScript.
:::

```yaml
# example.gpt

Tools: sys.download, sys.exec, sys.remove

Download https://www.sqlitetutorial.net/wp-content/uploads/2018/03/chinook.zip to a
random file. Then expand the archive to a temporary location as there is a sqlite
database in it.

First inspect the schema of the database to understand the table structure.

Form and run a SQL query to find the artist with the most number of albums and output
the result of that.

When done remove the database file and the downloaded content.
```
```
$ gptscript ./example.gpt

OUTPUT:

The artist with the most number of albums in the database is Iron Maiden, with a total
of 21 albums.
```

For more examples check out the [examples](https://github.com/gptscript-ai/gptscript/blob/main/examples) directory.
