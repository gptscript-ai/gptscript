# Hamlet Summarizer

This is an example tool that summarizes the contents of a large documents in chunks.

The example document we are using is the Shakespeare play Hamlet. It is about 51000 tokens
(according to OpenAI's tokenizer for GPT-4), so it can fit within GPT-4's context window,
but this serves as an example of how larger documents can be split up and summarized.
This example splits it into chunks of 10000 tokens.

Hamlet PDF is from https://nosweatshakespeare.com/hamlet-play/pdf/.

## Design

The script consists of three tools: a top-level tool that orchestrates everything, a summarizer that
will summarize one chunk of text at a time, and a Python script that ingests the PDF and splits it into
chunks and provides a specific chunk based on an index.

The summarizer tool looks at the entire summary up to the current chunk and then summarizes the current
chunk and adds it onto the end. In the case of models with very small context windows, or extremely large
documents, this approach may still exceed the context window, in which case another tool could be added to
only give the summarizer the previous few chunk summaries instead of all of them.

## Run the Example

```bash
# Create a Python venv
python3 -m venv venv

# Source it
source venv/bin/activate

# Install the packages
pip install -r requirements.txt

# Set your OpenAI key
export OPENAI_API_KEY=your-api-key

# Run the example
gptscript --disable-cache hamlet-summarizer.gpt
```
