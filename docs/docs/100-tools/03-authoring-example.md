# Authoring Tools Example Guide

This is a guide for writing portable tools for GPTScript.
The supported languages currently are Python, NodeJS, and Go. This guide uses Python.

## 1. Write the code

Create a file called `tool.py` with the following contents:

```python
import os
import requests

print(requests.get(os.getenv("url")).text)
```

Create a file called `requirements.txt` with the following contents:

```
requests
```

## 2. Create the tool

Create a file called `tool.gpt` with the following contents:

```
description: Returns the contents of a webpage.
args: url: The URL of the webpage.

#!/usr/bin/env python3 ${GPTSCRIPT_TOOL_DIR}/tool.py
```

The `GPTSCRIPT_TOOL_DIR` environment variable is automatically populated by GPTScript so that the tool
will be able to find the `tool.py` file no matter what the user's current working directory is.

If you make the tool available in a public GitHub repo, then you will be able to refer to it by
the URL, i.e. `github.com/<user>/<repo name>`. GPTScript will automatically set up a Python virtual
environment, install the required packages, and execute the tool.
