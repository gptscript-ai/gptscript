# Chat with an API

GPTScript makes it easy to create a chatbot interface to interact with an API.

This guide will demonstrate how to build a chatbot that interacts with the DigitalOcean API.

## Getting Started

First, you will need to download a copy of DigitalOcean's OpenAPI definition.
While you can reference it by its URL, it is a bit easier to work with it locally.
You can download the file by running the following command:

```bash
curl -o openapi.yaml -L https://api-engineering.nyc3.cdn.digitaloceanspaces.com/spec-ci/DigitalOcean-public.v2.yaml
```

This will download a copy of the OpenAPI definition to the current directory.

Let's examine this OpenAPI file. GPTScript will create a tool named after each operationId in the file.
You can see the operationIds by running the following command:

```bash
grep operationId openapi.yaml
# ...
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
# ...
```

The operationIds generally follow a pattern of `object_verb`.
This will be helpful for us, because we can use wildcard matching to refer to a subset of the operations.

## Creating the Script

Create a `tool.gpt` file with the following contents:

```
Name: DigitalOcean Bot
Chat: true
Tools: droplets* from openapi.yaml
Tools: databases* from openapi.yaml
Tools: images* from openapi.yaml
Tools: regions_list from openapi.yaml
Tools: tags* from openapi.yaml
Tools: sizes_list from openapi.yaml
Tools: sshKeys_list from openapi.yaml
Tools: sys.time.now

You are a helpful assistant with access to the DigitalOcean API to manage droplets and databases.
Before creating, updating, or deleting anything, tell the user about the exact action you are going to take, and get their confirmation.
Start the conversation by asking the user how you can help.
```

This chatbot has access to several tools that correspond to various operations in the DigitalOcean OpenAPI file.
We give it access to all tools related to droplets and databases, since those are the main things we want it to work with.
In order to support this, we also need to give it access to images, regions, tags, etc. so that it can get the information it needs to create new droplets and databases.
Lastly, the `sys.time.now` tool is a tool that is built-in to GPTScript that provides the current date and time.

:::note
We cannot give the entire `openapi.yaml` file to the tool because it contains too many API operations.
Most LLM providers, such as OpenAI, have a limit on the number of tools that you can provide to the model at one time.
The OpenAPI file contains over 300 operations, which is too many for most LLMs to handle at once.
:::

## Creating an API Token

Before you run this script, you need to have a DigitalOcean API token.

Go to [Applications & API](https://cloud.digitalocean.com/account/api/tokens) in the DigitalOcean dashboard and create a new token.
You can select whichever scopes you want, but you should at least give it the ability to read droplets and databases.

## Running the Script

Let's run the script and start chatting with it:

```bash
gptscript tool.gpt
```

Try asking it to list your current databases or droplets, or to help you create a new one.

The first time the LLM tries to make an API call, it will ask for your API token.
Paste it into the prompt. It will be used for all future API calls as well.
The LLM will never see or store your API token. It is only used client-side, on your computer.

## Next Steps

Feel free to modify the script to add other parts of the DigitalOcean API.
You could also try creating a chatbot for a different API with an OpenAPI definition.

For a more advanced DigitalOcean chatbot, see our [DigitalOcean Agent](https://github.com/gptscript-ai/digital-ocean-agent) tool.

To read more about OpenAPI tools in GPTScript, see the [OpenAPI Tools](../03-tools/03-openapi.md) article.

To read more about credential storage in GPTScript, see the [Credentials](../02-credentials.md) article.
