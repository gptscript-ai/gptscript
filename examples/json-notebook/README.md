# JSON Query tool and Jupyter Notebook

This is a simple example of how to use the JSON Query tool in Jupyter Notebook, along with the gptscript python module.

## Prerequisites

You will need to set the OPENAI_API_KEY environment variable with your OpenAI key in the shell you start the notebook from.

## Running the notebook

Open the notebook, it will install both gptscript when possible and download a dataset and query it using the json-query tool.

## Example

```bash
# Create a Python venv
python3 -m venv venv
. ./venv/bin/activate
pip install jupyter ipykernel
python -m ipykernel install --user --name=venv
jupyter notebook
```

Once the notebook is open, make sure you are using the venv kernel and run the cells.
