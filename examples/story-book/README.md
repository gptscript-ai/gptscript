# Story Book

Story Book is a GPTScript that can generate a story based on a prompt and the number of pages you want the story to be in. It is generated in HTML format and can then be viewed
by `index.html` which has some JS/CSS to make the story styling consistent and readable.

## Usage Instructions

1. **Install and bootstrap the `image-generation` tool.**

    This tool uses the `image-generation` tool, which is in a separate repository. To install and bootstrap the `image-generation` tool, starting at the root of `gptscript` run the following commands:

    > Note: We'll soon have package management that handles tools installation for you, but until then, you have to manually clone and boostrap tools.

    ```shell
    cd .. # This assumes you're starting at the root of the gptscript project. We want image-generation to be a sibling of gptscript.
    git clone https://github.com/gptscript-ai/image-generation
    cd image-generation
    make bootstrap
    source .venv/bin/activate
    cd ../gptscript # returns you back to your original directory
    ```

    > Note: You can install the python dependencies manually by running `pip install -r requirements.txt` in the root of the cloned repository. This prevents the need to run `make bootstrap` or activate the virtual environment.

2. **Run the `story-book.gpt` script.**

    In the same terminal session where the virtual environment (venv) is now activated, navigate to the `story-book` example directory and run the `story-book.gpt` script:

    ```shell
    cd examples/story-book
    gptscript story-book.gpt --prompt "Goldilocks" --pages 3
    ```

3. **View the story.**

    Open `index.html` in your browser to view the generated story.
