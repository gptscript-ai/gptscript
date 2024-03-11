# Image Generator

In this example, let's say that we want to create an image generator tool that will take a prompt from the user and generate an image based on that prompt but
in a specific style that we tell the tool to use.

1. Clone the image-generation tool repository from Github.

    ```shell
    git clone https://github.com/gptscript-ai/image-generation
    ```

2. Bootstrap the tool.

    ```shell
    make bootstrap
    ```

3. Create a file called `painter.gpt`.

    ```yaml
    tools: ./image-generation/tool.gpt, sys.download, sys.write
    args: prompt: (required) The prompt to use for generating the image.

    Generate the ${prompt} in the style of a 17th century impressionist oil painting.

    Write the resulting image to a file named "painting.png".
    ```

4. Run the script.

    ```shell
    gptscript painter.gpt --prompt "a city skyline at night"
    ```

This will generate an image of a city skyline at night in the style of a 17th century impressionist oil painting and write the resulting image to a file named `painting.png`.

## Recap
In this example, we have created a GPTScript that leverages the `image-generation` tool to generate an image based on a prompt. We gave it some flexibility by specifying an argument `prompt` that the user can provide when running the script. We also specified gave the script a specific style of the image that we want to generate, that being a 17th century impressionist oil painter.

Notable things to point out:
#### Tools
The `tools` directive was used here to reference the `image-generation` tool and the `sys.download` and `sys.write` system tools. GPTScript will know the tools available to it and will use them when it sees fit in the script.

#### Args
We used the `args` directive to specify the `prompt` argument that the user can provide when running the script. This is a required argument and the user must provide it when running the script.

#### Interpolation
The `${prompt}` syntax can we used to reference the `prompt` argument in the script. This is a made-up syntax and can be replaced with any other syntax that you prefer. The LLM should be able to understand the syntax and use the argument in the script.
