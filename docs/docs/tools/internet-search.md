# Internet Search

## Overview

There are a few different Internet search tools for GPTScript:

- `bing`
- `bing-image` (coming soon)
- `brave`
- `brave-image`
- `google`
- `google-image`

Each of these tools use the corresponding search engine's API to perform searches and return results that
the LLM can process. This is the format of the results:

Web search format:
```
Title: the title of the web page
URL: the link to the web page
Description: a short snippet from the web page
```

Image search format:
```
Title: the title of the image
Source: the link to the web page where the image came from
Image URL: the link to the image
```

The tools are developed in the [gptscript-ai/search repo](https://github.com/gptscript-ai/search).

## Setup

:::note
This will become easier in the future, once packaging for remote GPTScript tools has been implemented.
:::

To set up your local environment to use these tools, do the following:

```bash
# Clone the repo
git clone https://github.com/gptscript-ai/search.git

# Build the Go binary
make build
```

Now you can reference the search tools by the relative path to the `tool.gpt` file in the cloned repo, like this:

```yaml
tools: google-image from ./search/tool.gpt, sys.download

Search for images of pandas and download one.
```

:::tip
You must always specify which tool you are specifically importing from the tool.gpt file, as shown
in the example above. If no tool is specified, it will default to the first tool defined in the file,
which currently is `bing`.
:::

Specific instructions for each search engine follow.

## Bing

The `bing` tool returns search results from the [Bing Web Search API](https://www.microsoft.com/en-us/bing/apis/bing-web-search-api).

The environment variable `GPTSCRIPT_BING_SEARCH_TOKEN` must be set to your API key in order for it to work.

## Brave

The `brave` and `brave-image` tools return search results from the [Brave Search API](https://brave.com/search/api/).

The environment variable `GPTSCRIPT_BRAVE_SEARCH_TOKEN` must be set to your API key in order for it to work.

## Google

The `google` and `google-image` tools return search results from the [Google Custom Search JSON API](https://developers.google.com/custom-search/v1/overview).

The environment variable `GPTSCRIPT_GOOGLE_SEARCH_TOKEN` must be set to your API key in order for it to work,
and `GPTSCRIPT_GOOGLE_SEARCH_ENGINE_ID` must be set to your Programmable Search Engine's engine ID.
