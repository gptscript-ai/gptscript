# Image Generation

## Overview

The `image-generation` tool leverages OpenAI's DALL-E model to convert textual descriptions into images. It offers a versatile approach to creating visual content, making it suitable for a wide range of applications, from content creation to design assistance.

For more information, check out the tool [here](https://github.com/gptscript-ai/image-generation)

## Usage
To use the Image Generation tool, you need to make sure that the OPENAI_API_KEY environment variable is set to your OpenAI API key. You can obtain an API key from the OpenAI platform if you don't already have one.

With that setup, you can use it in any GPTScript like so:

:::tip
This script assumes you have the `image-generation` tool repo cloned locally and located at `./image-generation`. The ability to import tools from remote
locations is coming soon.
:::

```
tools: ./image-generation, sys.write

Generate an image of a city skyline at night with a resolution of 512x512 pixels.

Write the resulting image to a file named "city_skyline.png".
```
