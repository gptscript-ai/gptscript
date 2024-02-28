# Use Cases of GPTScript

## Retrieval

Retrieval-Augmented Generation (RAG) leverages a knowledge base outside of the training data before consulting the LLM to generate a response.
The following GPTScript implements RAG:

```yaml
name: rag
description: implements retrieval-augmented generation
args: prompt: a string
tools: query

First query the ${prompt} in the knowledge base, then construsts an answer to ${prompt} based on the result of the query.

---

name: query
description: queries the prompt in the knowledge base
args: prompt: a string

... (implementation of knowledge base query follows) ...
```

The idea behind RAG is simple. Its core logic can be implemented in one GPTScript statement: `First query the ${prompt} in the knowledge base, then construsts an answer to ${prompt} based on the result of the query.` The real work of building RAG lies in the tool that queries your knowledge base.

You construct the appropriate query tool based on the type of knowledge base you have.

| Knowledge Base | Query Tool |
|------|------|
| A vector database storing common document types such as text, HTML, PDF, and Word | Integrate with LlamaIndex for vector database support [Link to example]|
| Use the public or private internet to supply up-to-date knowledge | Implement query tools using a search engine, as shown in [`search.gpt`](../examples/search.gpt)|
| A structured database supporting SQL such as sqlite, MySQL, PostgreSQL, and Oracle DB | Implement query tools using database command line tools such as `sqlite` and `mysql` [Link to example]|
| An ElasticSearch/OpenSearch database storing logs or other text files | Implement query tools using database command line tools [Link to example]|
| Other databases such as graph or time series databases | Implement query tools using database command line tools [Link to example]|

## Agents and Assistants

Agents and assistants are synonyms. They are software programs that leverage LLM to carry out tasks.

In GPTScript, agents and assistants are implemented using tools. Tools can use other tools. Tools can be implemented using natural language prompts or using traditional programming languages such as Python or JavaScript. You can therefore build arbitrarily complex agents and assistants in GPTScript.

## Data Analysis

Depending on the context window supported by the LLM, you can either send a large amount of data to the LLM to analyze in one shot or supply data in batches.

### Summerization

Here is a GPTScript that sends a large document in batches to the LLM and produces a summary of the entire document. [Link to example here]

Here is a GPTScript that reads the content of a large SQL database and produces a summary of the entire database. [Link to example here]

### Tagging

Here is a GPTScript that performs sentiment analysis on the input text. [Link to example here]

### CSV Files

Here is a GPTScript that summarizes the content of a CSV file. [Link to example here]

### Understanding Code

Here is a GPTScript that summarizes the the code stored in a given directory. [Link to example here]

## Task Automation

### Planning

Here is a GPTScript that produces a detailed travel itenirary based on inputs from a user. [Link to example here]

### Web UI Automation

Here is a GPTScript that automates data gathering and analysis of web sites by driving a web browser. [Link to example here]

### CLI Automation

Here is a GPTScript that automates Kubernetes operations by driving the `kubectl` command line. [Link to example here]

## Audio, Image, and Vision

[More details to come]

## Memory Management

GPTScript provides a means to manage memory that persists across multiple LLM invocations. The relevant information can be extracted and passed into the LLM as part of the context. In addition, LLM can utilize tools to obtain additional data from the memory maintained in GPTScript.

[More details to follow.]

## Chatbots

GPTScript in combination with Rubra UI provide you with a turn-key implementation of chatbots. [More details to follow]
