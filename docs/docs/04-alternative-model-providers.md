# Alternative Model Providers


## Usage

You can use GPTScript with a model that is not provided by OpenAI by passing a reference to the model name and the git
repo you want to pull a compatibility shim from.

```bash
gptscript --default-model="mistral-large-latest from github.com/gptscript-ai/mistral-openai-shim" examples/helloworld.gpt"
```

You can also reference this from the `model:` stanza inside a gptscript.
```gptscript
tools: bob
model: mistral-large-latest from github.com/gptscript-ai/mistral-openai-shim

Ask Bob how he is doing and let me know exactly what he said.

---
name: bob
model: mistral-large-latest from github.com/gptscript-ai/mistral-openai-shim
description: I'm Bob, a friendly guy.
args: question: The question to ask Bob.

When asked how I am doing, respond with "Thanks for asking "${question}", I'm doing great fellow friendly AI tool!"
```

Each shim has different requirements for authentication. Please check the readme for the shim you are trying to use.

## Available Model Providers

The following shims are currently available:
* github.com/gptscript-ai/mistral-openai-shim
* github.com/gptscript-ai/claude3-openai-shim
* github.com/gptscript-ai/gemini-openai-shim

For each of these providers you can list available models

```bash
gptscript --list-models github.com/gptscript-ai/claude3-openai-shim
```


## Compatibility

While the shims provide support for using GPTScript with other models, the effectiveness of using a
different model will depend on a combination of prompt engineering and the quality of the model. You may need to change
wording or add more description if you are not getting the results you want. In some cases, the model might not be
capable of intelligently handling the complex function calls.
