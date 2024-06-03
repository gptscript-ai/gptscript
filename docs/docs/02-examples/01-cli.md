# Chat with a Local CLI

GPTScript makes it easy to write AI integrations with CLIs and other executable available on your local workstation. This is powerful because it allows you to work AI to solve complex problems using your available CLIs. You can describe complex requests in plain English and GPTScript will figure out the best CLI commands to make that happen. This guide will show you how to build a GPTScript that integrates with two CLIs:

- [gh](https://cli.github.com/) - the GitHub CLI
- [kubectl](https://kubernetes.io/docs/reference/kubectl/) - the Kubernetes CLI

:::warning
This script **does not install** or configure gh or kubectl. We assume you've done that already.

- For gh, you must be logged in via `gh auth login`. [See here for more details](https://docs.github.com/en/github-cli/github-cli/quickstart)
- For kubectl, you must have a proper `kubeconfig`. [See here for more details](https://kubernetes.io/docs/tasks/tools/)

:::

## Too Long; Didn't Read

Want to start using this script now? Just run:

```
gptscript github.com/gptscript-ai/cli-demo
```

Or if you want to skip ahead and just grab the full script so that you can start hacking on it, jump to the [Putting it all together section](cli#putting-it-all-together).

## Getting Started

The rest of this guide will walk you through building a script that can serve as an assistant for GitHub and Kubernetes tasks. We'll be explaining the how, what, and why along the way.

First, open up a new gptscript file in your favorite editor. We'll call the file cli-demo.gpt

```
vim cli-demo.gpt
```

All edits below are assumed to be in this file. At the end, we'll share the entire script as one cohesive file, but along the way we'll just be adding tools one-by-one.

## Create the Kubernetes Agent

Let's start by adding the Kubernetes agent. In our script, add the following:

```
---
Name: k8s-agent
Description: An agent that can help you with your Kubernetes cluster by executing kubectl commands
Context: shared-context
Tools: sys.exec
Parameter: task: The kubectl related task to accomplish
Chat: true

You have the kubectl cli available to you. Use it to accomplish the tasks that the user asks of you.

```

Now, let's walk through this tool line-by-line.

**---** is a block separator. It's how we delineate tools in a script.

**Name and Description** help the LLM understand the purpose of this tool. You should always have meaningful names and descriptions.

**Tools: sys.exec** makes the built-in `sys.exec` tool available to this agent. This gives the agent the ability to execute arbitrary commands. Based on our prompt, it will be used for kubectl commands. GPTScript's authorization system will prompt for approval whenever it's going to run a `sys.exec` command.

**Parameter: task:** defines a parameter named "task" for this tool. This will be important later on when other tools need to hand-off to this tool - they'll pass the task to it as this parameter. As with the name and description fields, it's important to provide a good description so that the LLM knows how to use this parameter.

**Chat: true** turns this tool into a "chat-able" tool, which we also call an "agent". This is important for open-ended tasks that might take some iteration.

Finally, we have the **tool body**, which in this case is a prompt:

```
You have the kubectl cli available to you. Use it to accomplish the tasks that the user asks of you.
```

This is what the tool will actually do. Tool bodies can be prompts or raw code like python, javascript, or the [world's best programming language](https://x.com/ibuildthecloud/status/1796227491943637125) - bash. For chat-able tools, your tool body should always be a prompt.

That's all there is to the Kubernetes agent. You can try it out now. One nice thing about GPTScript is that tools are composable. So, you can get this tool working well and then move onto the next tool without affecting this one. To launch this tool, run:

```
gptscript --sub-tool k8s-agent cli-demo.gpt
```

Once you're chatting, try asking it do something like list all the pods in your cluster or even to launch an new deployment in the cluster.

## Create the GitHub Agent

Now let's add the GitHub Agent. Drop the following into the file below the tool we just added.

```
---
Name: github-agent
Description: An agent to help you with GitHub related tasks using the gh cli
Context: learn-gh
Tools: sys.exec
Parameter: task: The GitHub task to accomplish
Chat: true

You have the gh cli available to you. Use it to accomplish the tasks that the user asks of you.

```

This tool is very similar to the Kubernetes agent. There are just a few key differences:

1. Names and descriptions have been changed to reference GitHub and gh as appropriate.
2. We've introduced the `learn-gh` context. We'll explore this next.

### The learn-gh context tool

Add this for the learn-gh context tool:

```
---
Name: learn-gh

#!/usr/bin/env bash

echo "The following is the help text for the gh cli and some of its sub-commands. Use these when figuring out how to construct new commands.  Note that the --search flag is used for filtering and sorting as well; there is no dedicate --sort flag."
gh --help
gh repo --help
gh issue --help
gh issue list --help
gh issue create --help
gh issue comment --help
gh issue delete --help
gh issue edit --help
gh pr --help
gh pr create --help
gh pr checkout --help
gh release --help
gh release create --help
```

As we saw, this tool is used as the context for the github-agent. Why did we add this and what does it do?

To answer that, let's first understand what the Context stanza does. Any tools referenced in the Context stanza will be called and their output will be added to the chat context. As the name suggests, this gives the LLM additional context for subsequent messages. Sometimes, an LLM needs extra instructions or context in order to achieve the desired results. There's no hard or fast rule here for when you should include context; it's best discovered through trial-and-error.

 We didn't need extra context for the Kubernetes tool because we found our default LLM knows kubectl (and Kubernetes) quite well. However, our same testing showed that our default LLM doesn't know the gh cli as well. Specifically, the LLM would sometimes hallucinate invalid combinations of flags and parameters.  Without this context, the LLM often takes several tries to get the gh command correct.

:::tip
Did you catch that "takes several tries to get the command correct" part? One useful feature of GPTScript is that it will feed error messages back to the LLM, which allows the LLM to learn from its mistake and try again.
:::

And that's the GitHub Agent. You can try it out now:

```
gptscript --sub-tool github-agent cli-demo.gpt
```

Once you're chatting, try asking it do something like "Open an issue in gptscript-ai/gptscript with a title and body that says Hi from me and states how wonderful gptscript is but jazz it up and make it unique"

## Your CLI Assistant

Right now if you were to launch this script, you'd be dropped right into the Kubernetes agent. Let's create a new entrypoint whose job it is to handle your initial conversation and route to the appropriate agent. Add this to the **TOP** of your file:

```
Name: Your CLI Assistant
Description: An assistant to help you with local cli-based tasks for GitHub and Kubernetes
Agents: k8s-agent, github-agent
Context: shared-context
Chat: true

Help the user acomplish their tasks using the tools you have. When the user starts this chat, just say hello and ask what you can help with. You donlt need to start off by guiding them.
```

By being at the top of the file, this tool will serve as the script's entrypoint. Here are the parts of this tool that are worth additional explanation:

**Agents: k8s-agent, github-agent** puts these two agents into a group that can hand-off to each other.  So, you can ask a GitHub question, then a Kubernetes question, and then a GitHub question again and the chat conversation will get transferred to the proper agent each time.

Next is **Context: shared-context**. You're already familiar with contexts, but in the next section we'll explain what's unique about this one.

### The shared-context tool

Drop the shared-context tool in at the very bottom of the page:

```
---
Name: shared-context
Share Context: github.com/gptscript-ai/context/history

#!sys.echo
Always delegate to the best tool for the users request.
Ask the user for information needed to complete a task.
Provide the user with the exact action you will be taking and get the users confirmation when creating or updating resources.
ALWAYS ask the user to confirm deletions, provide as much detail about the action as possible.
```

and do one more thing: add it as a context tool to both the k8s-agent and github-agent. For k8s-agent, that means adding this line: `Context: shared-context` and for github-agent, it means modifying the existing Context line to: `Context: learn-gh, shared-context`.

**Share Context: github.com/gptscript-ai/context/history** - In this line, "Share Context" means that the specified tool(s) will be part of the context for any tools that references this tool in their Context stanza. It's a way to compose and aggregate contexts.

 The specific tool referenced here - github.com/gptscript-ai/context/history - makes it so that when you transition from one agent to the next, your chat history is carried across. Using this file as an example, this would allow you to have a history of all the Kubernetes information you gathered available when talking to the GitHub tool.

The **#!sys.echo** body is a simple way to directly output whatever text follows it. This is useful if you just have a static set of instructions you need to inject into the context.  The actual text should make sense if you read it. We're telling the agents how we want them to behave and interact.

## Putting it all together

Let's take a look at this script as one cohesive file:

```
Name: Your CLI Assistant
Description: An assistant to help you with local cli-based dev tasks
Context: shared-context
Agents: k8s-agent, github-agent
Chat: true

Help the user acomplish their tasks using the tools you have. When the user starts this chat, just say hello and ask what you can help with. You donlt need to start off by guiding them.

---
Name: k8s-agent
Description: An agent that can help you with your Kubernetes cluster by executing kubectl commands
Context: shared-context
Tools: sys.exec
Parameter: task: The kubectl releated task to accomplish
Chat: true

You have the kubectl cli available to you. Use it to accomplish the tasks that the user asks of you.

---
Name: github-agent
Description: An agent to help you with GitHub related tasks using the gh cli
Context: learn-gh, shared-context
Tools: sys.exec
Parameter: task: The GitHub task to accomplish
Chat: true

You have the gh cli available to you. Use it to accomplish the tasks that the user asks of you.

---
Name: learn-gh

#!/usr/bin/env bash

echo "The following is the help text for the gh cli and some of its sub-commands. Use these when figuring out how to construct new commands.  Note that the --search flag is used for filtering and sorting as well; there is no dedicate --sort flag."
gh --help
gh repo --help
gh issue --help
gh issue list --help
gh issue create --help
gh issue comment --help
gh issue delete --help
gh issue edit --help
gh pr --help
gh pr create --help
gh pr checkout --help
gh release --help
gh release create --help


---
Name: shared-context
Share Context: github.com/gptscript-ai/context/history

#!sys.echo
Always delegate to the best tool for the users request.
Ask the user for information needed to complete a task.
Provide the user with the exact action you will be taking and get the users confirmation when creating or updating resources.
ALWAYS ask the user to confirm deletions, provide as much detail about the action as possible.
```

There isn't anything new to cover in this file, we just wanted you to get a holistic view of it. This script is now fully functional. You can launch it via:

```
gpscript cli-demo.gpt
```

### Adding your own CLI

By now you should notice a simple pattern emerging that you can follow to add your own CLI-powered agents to a script. Here are the basics of what you need:

```
Name: {your cli}-agent
Description: An agent to help you with {your taks} related tasks using the gh cli
Context: {here's your biggest decsion to make}, shared-context
Tools: sys.exec
Parameter: task: The {your task}The GitHub task to accomplish
Chat: true

You have the {your cli} cli available to you. Use it to accomplish the tasks that the user asks of you.
```

You can drop in your task and CLI and have a fairly functional CLI-based chat agent. The biggest decision you'll need to make is what and how much context to give your agent. For well-known for CLIs/technologies like kubectl and Kubernetes, you probably won't need a custom context. For custom CLIs, you'll definitely need to help the LLM out. The best approach is to experiment and see what works best.

## Next steps

Hopefully you've found this guide helpful. From here, you have several options:

- You can checkout out some of our other guides available in this section of the docs
- You can dive deeper into the options available when [writing script](/tools/gpt-file-reference)
