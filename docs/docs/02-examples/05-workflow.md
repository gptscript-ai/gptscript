# Run an Automated Workflow

Automating a sequence of tasks that integrate with one or more systems is a ubiquitous engineering problem that typically requires some degree of domain-specific knowledge up-front.
However, workflows written with GPTScript all but eliminate this prerequisite, enabling developers to build their workflows by describing the high-level steps it should perform.

This guide will show you how to build a GPTScript that encapsulates a workflow consisting of the following steps:
1. Get a selection of X (Twitter) posts 
2. Summarize their content 
3. Summarize the content of any links they directly reference
4. Write the results to a Markdown document

## Too long; didn't read

Want to start using this script now? Just run:

```
gptscript github.com/gptscript-ai/workflow-demo
```

Or if you want to skip ahead and just grab the full script so that you can start hacking on it, jump to the [Putting it all together section](workflow#putting-it-all-together).

## Getting Started

First, open up a new GPTScript file in your favorite editor. We'll call the file `workflow-demo.gpt`

```
vim workflow-demo.gpt
```

All edits below are assumed to be in this file. At the end, we'll share the entire script as one cohesive file, but along the way we'll just be adding tools one-by-one.

## Workflow Entrypoint 

To get started, let's define the first tool in our file. This will act as the entrypoint for our workflow.

Add the following tool definition to the file: 

```
Tools: sys.write, summarize-tweet
Description: Summarize tweets

Always summarize tweets synchronously in the order they appear.
Never summarize tweets in parallel.
Write all the summaries for the tweets at the following URLs to `tweets.md`:
- https://x.com/acornlabs/status/1798063732394000559
- https://x.com/acornlabs/status/1797998244447900084
```

Let's walk through the important bits. 

This tool:
- imports two other tools
  - `sys.write` is a built-in tool which enables the entrypoint tool to write files to your system.
  - `summarize-tweet` is a custom tool that encapsulates how each tweet gets summarized. We'll define this tool in the next step.
- ensures tweets are never summarized in parallel so that they are summarized in the correct order
- defines the tweet URLs to summarize and the file to write them to

At a high-level, it's getting the summaries for two tweets and storing them in the `tweets.md` file.

## Tweet Summarization Tool

The next step is to define the `summarize-tweet` tool used by the entrypoint to summarize each tweet. 

Add the following tool definition below the entrypoint tool:

```
---
Name: summarize-tweet
Description: Returns a brief summary of a tweet in markdown
Tools: get-hyperlinks, summarize-hyperlink, github.com/gptscript-ai/browser
Parameters: url: URL of the tweet to summarize

Return a markdown formatted summary of the tweet at ${url} using the following format:

## <tweet-date> [<tweet-subject>](${url})

Short summary of the tweet.

### References

<list of hyperlinks found in the tweet's text and their respective summaries>
```

This tool
- takes the `url` of a tweet as an argument
- imports three other tools to solve summarization sub-problems
  - `github.com/gptscript-ai/browser` is an external tool that is used to open the tweet URL in the browser and extract the page content
  - `get-hyperlinks` and `summarize-hyperlinks` are custom helper tools we'll define momentarily that extract hyperlinks from tweet text and summarize them
- describes the Markdown document this tool should produce, leaving it up to the LLM to decide which of the available tools to call to make this happen

## Hyperlink Summarization Tools

To complete our workflow, we just need to define the `get-hyperlinks` and `summarize-hyperlinks` tools.

Add the following tool definitions below the `summarize-tweet` definition:

```
---
Name: get-hyperlinks
Description: Returns the list of hyperlinks in a given text
Parameters: text: Text to extract hyperlinks from

Return the list of hyperlinks found in ${text}. If ${text} contains no hyperlinks, return an empty list.

---
Name: summarize-hyperlink
Description: Returns a summary of the page at a given hyperlink URL
Tools: github.com/gptscript-ai/browser
Parameters: url: HTTPS URL of a page to summarize

Briefly summarize the page at ${url}.
```

As we can see above, `get-hyperlinks` takes the raw text of the tweet and returns any hyperlinks it finds, while `summarize-hyperlink`
uses the `github.com/gptscript-ai/browser` tool to get content at a URL and returns a summary of it.

## Putting it all together

Let's take a look at this script as one cohesive file:

```
Description: Summarize tweets
Tools: sys.write, summarize-tweet

Always summarize tweets synchronously in the order they appear.
Never summarize tweets in parallel.
Write all the summaries for the tweets at the following URLs to `tweets.md`:
- https://x.com/acornlabs/status/1798063732394000559
- https://x.com/acornlabs/status/1797998244447900084

---
Name: summarize-tweet
Description: Returns a brief summary of a tweet in markdown
Tools: get-hyperlinks, summarize-hyperlink, github.com/gptscript-ai/browser
Parameters: url: Absolute URL of the tweet to summarize

Return a markdown formatted summary of the tweet at ${url} using the following format:

## <tweet-date> [<tweet-subject>](${url})

Short summary of the tweet.

### References

<list of hyperlinks found in the tweet's text and their respective summaries>

---
Name: get-hyperlinks
Description: Returns the list of hyperlinks in a given text
Parameters: text: Text to extract hyperlinks from

Return the list of hyperlinks found in ${text}. If ${text} contains no hyperlinks, return an empty list.

---
Name: summarize-hyperlink
Description: Returns a summary of the page at a given hyperlink URL
Tools: github.com/gptscript-ai/browser
Parameters: url: HTTPS URL of a page to summarize

Briefly summarize the page at ${url}.
```

You can launch it via:

```
gpscript workflow-demo.gpt
```

Which should produce a `tweets.md` file in the current directory:

```markdown
## Jun 4, 2024 [Acorn Labs Tweet](https://x.com/acornlabs/status/1798063732394000559)

Our latest release of #GPTScript v0.7 is available now. Key updates include OpenAPI v2 support, workspace tools replacement, and a chat history context tool.

### References

- [Acorn | Introducing GPTScript v0.7](https://buff.ly/3V1KiWw): The page is a blog post introducing GPTScript v0.7, highlighting its new features and fixes. Key updates include support for OpenAPI v2, removal and replacement of workspace tools, introduction of a chat history context tool, and several housekeeping changes like defaulting to the gpt-4o model and logging token usage. The post also mentions upcoming features in v0.8 and provides links to related articles and resources.

## Jun 4, 2024 [Acorn Labs Tweet](https://x.com/acornlabs/status/1797998244447900084)

Use #GPTScript with #Python to build the front end of this AI-powered YouTube generator. Part of this series walks you through it all.

### References

- [Tutorial on building an AI-powered YouTube title and thumbnail generator (Part 1)](https://buff.ly/4aGJYCD): Covers setting up the front end with Flask, modifying the script, and creating templates for the front page and result page.
- [Tutorial on building an AI-powered YouTube title and thumbnail generator (Part 2)](https://buff.ly/49MSK1k): Covers the installation and setup of GPTScript, creating a basic script, handling command-line arguments, generating a YouTube title, and generating thumbnails using DALL-E 3.
```
