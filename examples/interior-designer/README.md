# Interior Designer

It allows you to upload a picture of your room and describe how you want it redesigned. The app will first analyze the image and identify the items in it. Then, based on your description, it will suggest changes you can make to redesign your room, including the color scheme, furniture, objects you can add, etc. and search for products on the internet and share their links as well.

## High-level overview

- Users can upload a picture of a room and provide a description of how their room wants to be.
- The Python backend takes this input and uses GPTScript’s Python module to execute the GPTScript.
- The vision tool first analyzes the image, followed by a generation of ideas.
- The output of both is sent to the search tool, which searches the Internet for products you can purchase and add to your room.
- These are saved within a markdown file that is read and displayed on the screen.

## Installation

### Prerequisites

- Python 3.8 or later
- Flask
- Python dependencies listed in `requirements.txt` respectively.

### Steps

1. Clone the repository:

    ``` bash
    git clone https://github.com/gptscript-ai/gptscript.git
    ```

2. Navigate to the `examples/interior-designer` directory and install the dependencies:

    Python:

    ```bash
    pip install -r requirements.txt
    ```

3. Setup `OPENAI_API_KEY` (Eg: `export OPENAI_API_KEY="yourapikey123456"`). You can get your [API key here](https://platform.openai.com/api-keys).

4. Run the Flask application using `flask run` or `python app.py`

## Usage

1. Open your web browser and navigate to `http://127.0.0.1:5000/`.
2. Use the interface to provide upload an image and provide a description.
3. The application will generate ideas to redesing your room.