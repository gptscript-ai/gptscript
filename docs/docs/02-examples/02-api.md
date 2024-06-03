# Chat with an API

Interacting with cloud providers through dashboards, APIs, and CLIs is second nature to devops engineers. Using AI chat, the engineer can express a goal, and the AI can generate and execute the calls needed to achieve it. This saves the engineer time from having to look up the API calls needed themselves. GPTScript makes building a chat integration with an existing OpenAPI schema quick and easy.

This guide will walk through the process of using the OpenAPI spec from Digital Ocean to build a chatbot capable of launching droplets and databases. The reader will be able to continue adding Digital Ocean capabilities or build their own chatbot with another OpenAPI schema.

## Too Long; Didn't Read

If you just want to try out the Digital Ocean chatbot first:

Follow the [API credential](#api-access) settings here.

Then you can run the following commands to get started:

```bash
gptscript github.com/gptscript-ai/digital-ocean-agent
```

## Getting started

First we will need to download a copy of the openapi.yaml. This spec technically can be accessed by URL, but initially, it is easier to download a copy and save it as openapi.yaml.

### The Digital Ocean openapi.yaml spec

Getting the openapi.yaml file from Digital Ocean can be done by running the following command in a terminal.

```bash
curl -o openapi.yaml -L https://api-engineering.nyc3.cdn.digitaloceanspaces.com/spec-ci/DigitalOcean-public.v2.yaml
```

This will download a copy of the openapi yaml file to the local directory.

Lets take a look at the spec file a little bit. The integration in GPTScript creates a tool named after each operationId in the OpenAPI spec. You can see what these tools would be by running the following.

```bash
grep operationId openapi.yaml
# …
# operationId: domains_delete_record
# operationId: droplets_list
# operationId: droplets_create
# operationId: droplets_destroy_byTag
# operationId: droplets_get
# operationId: droplets_destroy
# operationId: droplets_list_backups
# operationId: droplets_list_snapshots
# operationId: dropletActions_list
# operationId: dropletActions_post
# operationId: dropletActions_post_byTag
# operationId: dropletActions_get
# operationId: droplets_list_kernels
# operationId: droplets_list_firewalls
# operationId: droplets_list_neighbors
#  …
```

If we look at the operationIds, you’ll notice they are structured around an object like droplet, database, or project. Each object has a collection of verb like list, get, delete, create, etc. Each tool in GPTScript has it’s own set of tools. So we can create agents, tools with chat enabled, that are experts in a specific set of objects and have access to all of the object_verb tools available to them. This allows us to fan out tools from a main entrypoint to multiple experts that can solve the users tasks.

Lets explore this design pattern.

## Creating Main Entrypoint

Lets start by creating our main entrypoint to the Digital Ocean chatbot. The main tool in a GPTScript chat program is usually named agent.gpt. Let’s first setup the agents by giving it a name, the ability to chat, basic instructions, and the main greeting prompt. Create an agent.gpt file with the following contents.

agent.gpt

```
Name: Digital Ocean Bot
Chat: true

You are a helpful DevOps assistant that is an expert in Digital Ocean.
Using only the tools available, do not answer without using a tool, respond to the user task.
Greet the User with: "Hello! How can I help you with Digital Ocean?"
```

This file when run will show the following.

![screenshot](/img/chat-api.png)

In the current form, the chatbot will not be able to do anything since it doesn’t have access to any APIs. Let’s address that now, open our tool.gpt file and add the following.

agent.gpt

```
Name: Digital Ocean Bot
Chat: true
Agents: droplets.gpt

You are a helpful DevOps assistant that is an expert in Digital Ocean
Using only the tools available, do not answer without using a tool, respond to the user task.
Greet the User with: "Hello! How can I help you with Digital Ocean?"
```

Now lets create a droplets.gpt file to bring in the droplet tools.

droplets.gpt

```
Name: Droplet Agent
Chat: true
Tools: droplets* from ./openapi.yaml
Description: Use this tool to work with droplets
Args: request: the task requested by the user

Help the user complete their Droplet operation requests using the tools available.
When creating droplets, always ask if the user would like to access via password or via SSHkey.
```

Here we have defined the Droplet Agent, and enabled chat. We have also brought in an subset of the openapi.yaml tools that relate to droplets. By using droplets* we are making available everything droplet related into the available tools for this agent. We also provided the description to the main agent, and any other agent that has access to it, when to utilize this tool. We also have an argument called “request”, this is used when the LLM decides to call the agent it can smoothly pass off the user request without the Droplet Agent having to ask again.

## Chat with Digital Ocean

### API Access

Now that we have brought in our first tool using the OpenAPI spec, we will need to setup authentication. Defined in the openapi.yaml is how the Digital Ocean API expects authenticated requests to work. If you look in the spec file path of components.securitySchemes you will see that Digital Ocean expects bearer_auth. So you will need to create an API key in the Digital Ocean dashboard with the access you want the LLM to be able to interact with Digital Ocean. For instance, you can do a read only key that will allow you to just query information, or you can provide it full access and the operator can work with the LLM to do anything in the project. It is up to you. For this example, we will be using a full access token, but you can adjust for your needs. You can create your API key by going to this link [Apps & API](https://cloud.digitalocean.com/account/api/tokens) section in your account.

Once you have an API key, you will need to set an environment variable with that value stored.

```bash
export GPTSCRIPT_API_DIGITALOCEAN_COM_BEARER_AUTH=******
```

Where the *** is the API key created in the dashboard.

### Chatting with Digital Ocean APIs

Now you can run gptscript to start your conversation with Digital Ocean.

```bash
gptscript agent.gpt
```

You should now be able to ask how many droplets are running?

And get an output from the chatbot. This is great, but not quite ready to use just yet. Lets keep adding some functionality.

## Adding Database Support

Now that we can do droplets, we can add support for databases just as easy. Lets create a databases.gpt file with the following contents.

Ddtabases.gpt

```
Name: Database Agent
Chat: true
Tools: databases* from ./openapi.yaml
Description: Call this tool to manage databases on digital ocean
Args: request: the task requested by the user

Help the user complete database operation requests with the tools available.
```

Here again, we are essentially scoping our agent to handle database calls with the Digital Ocean API. Now in order for this to be used, we need to add it to our agent list in the main agent.gpt file.

Agent.gpt

```
Name: Digital Ocean Bot
Chat: true
Agents: droplets.gpt, databases.gpt

You are a helpful DevOps assistant that is an expert in Digital Ocean
Using only the tools available, do not answer without using a tool, respond to the user task.
Greet the User with: "Hello! How can I help you with Digital Ocean?"
```

Now when we test it out we can ask how many databases are running? And it should give back the appropriate response.

Now, when it comes to creating a database or droplet, we are missing some APIs to gather the correct information. We don’t have access to size information, regions, SSH Keys, etc. Since these are common tools, it would be a bit of a hassle to add lines to both the databases.gpt and droplets.gpt files. To avoid this, we can make use of the GPTScript Context to provide a common set of tools and instructions.

## Context

Context is a powerful concept in GPTScript that provides information to the system prompt, and provide a mechanism to compose a common set of tools reducing duplication in your GPTScript application. Lets add a context.gpt file to our chatbot here with the following contents.

context.gpt

```
Share Tools: sys.time.now

Share Tools: images* from ./openapi.yaml
Share Tools: regions_list from ./openapi.yaml
Share Tools: tags* from openapi.yaml
Share Tools: sizes_list from ./openapi.yaml
Share Tools: sshKeys_list from ./openapi.yaml


#!sys.echo
Always delegate to the best tool for the users request.
Ask the user for information needed to complete a task.
Provide the user with the exact action you will be taking and get the users confirmation when creating or updating resources.
ALWAYS ask the user to confirm deletions, provide as much detail about the action as possible.
```

There is quite a bit going on here, so lets break it down. Anywhere you see Share Tools it is making that tool available to anything uses the context. In this case, it is providing access to the time now tool so you can ask what was created yesterday and the LLM can get a frame of reference. Additionally, it provides a common set of Digital Ocean APIs that are needed for placement, organization(tags), sizes, and images, etc. Since multiple components in Digital Ocean use these values, it is useful to only need to define it once. Last we are providing a set of common instructions for how we want the chatbot to behave overall. This way, we do not need to  provide this information in each agent. Also, since this is in the system prompt, it is given a higher weight to the instructions in the individual agents.

Now lets add this to our agents. You will need to add the line:

```
Context: context.gpt
```

To each of our agents, so the droplets.gpt, agent.gpt, and databases.gpt will have this line.

## Wrapping up

Provided you have given API access through your token, you should now be able to run the chatbot and create a database or a droplet and be walked through the process. You should also be able to ask quesitons like What VMs were created this week?

You now know how to add additional capabilities through agents to the chatbots. You can follow the same patterns outlined above to add more capabilities or you can checkout the chat bot repository to see additional functionality.

### Use your own OpenAPI schema

If you have your own OpenAPI schema, you can follow the same pattern to build a chatbot for your own APIs. The simplest way to get started is to create a gptscript file with the following contents.

```
Name: {Your API Name} Bot
Chat: true
Tools: openapi.yaml

You are a helpful assistant. Say "Hello, how can I help you with {Your API Name} system today?"
```

You can then run that and the LLM will be able to interact with your API.

#### Note on OpenAI tool limits

As we mentioned before, GPTScript creates a tool for each operationId in the OpenAPI spec. If you have a large OpenAPI spec, you may run into a limit on the number of tools that can be created. OpenAI, the provider of the GPT-4o model only allows a total of 200 tools to be passed in at a single time. If you exceed this limit, you will see an error message from OpenAI. If you run into this issue, you can follow the same pattern we did above to create our Digital Ocean bot.

A quick check to see how many tools total would be created, you can run the following:

```bash
grep operationId openapi.yaml|wc -l
 306
```

In our case, there are 306 tools that would be created in the case of our Digital Ocean spec. This would not fit into a single agent, so breaking it up into multiple agents is the best way to handle this.

## Next Steps

Now that you have seen how to create a chatbot with an OpenAPI schema, checkout our other guides to see how to build other ChatBots and agents.
