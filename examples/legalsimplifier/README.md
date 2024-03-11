# Legal Simplifier

Legal Simplifier is an application that allows you to simplify legal documents - from terms and conditions of an insrance policy or a business contract. This example also shows how we can summarize contents of a large documents in chunks.

Based on the document you upload, the size can vary and hence mgight find the need to split larger documents into chunks of 10,000 tokens to fit within GTP-4's context window.

## Design

The script consists of three tools: a top-level tool that orchestrates everything, a summarizer that
will summarize one chunk of text at a time, and a Python script that ingests the PDF and splits it into
chunks and provides a specific chunk based on an index.

The summarizer tool looks at the entire summary up to the current chunk and then summarizes the current
chunk and adds it onto the end. In the case of models with very small context windows, or extremely large
documents, this approach may still exceed the context window, in which case another tool could be added to
only give the summarizer the previous few chunk summaries instead of all of them.

## Installation

### Prerequisites

- Python 3.8 or later
- Flask
- Other Python dependencies listed in `requirements.txt`.

### Steps

1. Clone the repository:

    ``` bash
    git clone https://github.com/gptscript-ai/gptscript.git
    ```

2. Navigate to the `examples/legalsimplifier` directory and install the dependencies:

    Python:

    ```bash
    pip install -r requirements.txt
    ```

    Node:

    ```bash
    npm install
    ```

3. Setup `OPENAI_API_KEY` (Eg: `export OPENAI_API_KEY="yourapikey123456"`). You can get your [API key here](https://platform.openai.com/api-keys).

4. Run the Flask application using `flask run` or `python app.py`

## Usage

1. Open your web browser and navigate to `http://127.0.0.1:5000/`.
2. Use the web interface to upload an a legal document in .pdf format.
3. The application will analyze the document and show a summary.

## Under the hood

Below are the processes that take place when you execute the application:

- The Python app places the uploaded image as `legal.pdf` in the current working directory.
- It then executes `legalsimplifier.gpt` which internally calls `main.py` to split the large document in chunks so that they fit within the token limit of GPT-4's context.
- The analysis will be stored in a `summary.md` document.
- The app will then read this summary file and show the summary on the UI.
