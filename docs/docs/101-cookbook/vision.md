# Vision 

In this example, let's say that we want to create a `.gpt` script that uses vision to describe an image on the internet in a certain style.

1. Clone the vision tool repository from Github

    ```shell
    git clone https://github.com/gptscript-ai/vision
    ```

2. Install the tool's dependencies

    ```shell
    npm install
    ```

3. Create a file called `describe.gpt`.

    ```yaml
    Tools: sys.write, ./vision/tool.gpt
    Args: url: (required) URL of the image to describe. 

    Describe the image at ${url} as if you were the narrator of a Wes Anderson film and write it to a file named description.txt.
    ```

4. Run the script.

    ```shell
    gptscript describe.gpt --url "https://github.com/gptscript-ai/vision/blob/main/examples/eiffel-tower.png?raw=true"
    ```

## Recap

In this example, we have created a GPTScript that leverages the `vision` tool to describe an image at a given URL. We gave it some flexibility by specifying an argument `url` that the user can provide when running the script. We also gave the script a specific style to generate the description with.

Notable things to point out:

#### Tools

The `tools` directive was used here to reference the `vision` and the `sys.write` tools.  GPTScript will know the tools available to it and will use them when it sees fit in the script.

#### Args

We used the `args` directive to specify the `url` argument that the user can provide when running the script. This is a required argument and the user must provide it when running the script.

#### Interpolation

The `${url}` syntax can we used to reference the `url` argument in the script. This is a made-up syntax and can be replaced with any other syntax that you prefer. The LLM should be able to understand the syntax and use the argument in the script.
