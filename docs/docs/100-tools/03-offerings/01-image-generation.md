# Image Generation

The `image-generation` tool leverages OpenAI's DALL-E model to convert textual descriptions into images. It offers a versatile approach to creating visual content, making it suitable for a wide range of applications, from content creation to design assistance.

For more information, check out the tool [here](https://github.com/gptscript-ai/image-generation)

## Installation
We are working on an process where installation of tools will be as simple as
referencing the tool's link in your GPTScript. For now, you can follow the steps below to install the image-generation tool.

1. Clone the image-generation tool repository [from GitHub](https://github.com/gptscript-ai/image-generation).

```shell
git clone https://github.com/gptscript-ai/image-generation
```

2. Run `make bootstrap` to setup the python venv and install the required dependencies.

```shell
make bootstrap
```

3. Reference `<path-to-the-repo>/tool.gpt` in your GPTScript.

## Usage
To use the Image Generation tool, you need to make sure that the OPENAI_API_KEY environment variable is set to your OpenAI API key. You can obtain an API key from the OpenAI platform if you don't already have one.

With that setup, you can use it in any GPTScript like so:

```
tools: ./image-generation/tool.gpt, sys.download, sys.write

Generate an image of a city skyline at night with a resolution of 512x512 pixels.

Write the resulting image to a file named "city_skyline.png".
```
