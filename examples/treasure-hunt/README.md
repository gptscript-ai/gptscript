# Treasure Hunt

## Overview

This Flask application leverages GPTScript to generate clues for a real world treasure hunt game. The user needs to provide a list of locations within a city and the application will generate fun and interesting clues for them.

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

2. Navigate to the `examples/treasure-hun` directory and install the dependencies:

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
2. Use the interface to provide a comma-separated list of locations within a city. For eg: Statue of Liberty, Hudson Yards, Central Park, Grand Central Station.
3. The application will generate fun and witty clues for all these locations.
