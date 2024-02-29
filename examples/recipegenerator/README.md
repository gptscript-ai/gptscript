# Recipe Generator Application

## Overview

This Flask application leverages GPTScript and GPTScript Vision to suggest recipes based on an image uploaded by the user. By analyzing the image, the application can identify ingredients and propose a suitable recipe.

## Features

- Image upload functionality.
- Automatic ingredient recognition from images.
- Recipe suggestion based on identified ingredients.

## Installation

### Prerequisites

- Python 3.8 or later
- Node.js and npm
- Flask
- Other Python and Node.js dependencies listed in `requirements.txt` and `package.json` respectively.

### Steps

1. Clone the repository:

    ``` bash
    git clone https://github.com/gptscript-ai/gptscript.git
    ```

2. Navigate to the `examples/recipegenerator` directory and install the dependencies:

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
2. Use the web interface to upload an image with some grocery items.
3. The application will process the image, identify potential ingredients, and suggest a recipe based on the analysis.
4. View the suggested recipe, try it and let us know how it turned out to be!

## Under the hood

Below are the processes that take place when you execute the application:

- The Python app places the uploaded image as `grocery.png` in the current working directory.
- It then executes `recipegenerator.gpt` which internally calls `tools.gpt` to perform image analysis to identify the items from the uploaded image.
- The identified ingredients from the image will be stored in a `response.json` file.
- The recipegenerator will then read this response file, generate a recipe and add it to a recipe.md file.