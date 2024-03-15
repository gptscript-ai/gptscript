# Story Book

Story Book is a GPTScript that can generate a story based on a prompt and the number of pages you want the story to be in. It is generated in HTML format and can then be viewed
by `index.html` which has some JS/CSS to make the story styling consistent and readable.

## Usage Instructions

1. **Run the `story-book.gpt` script.**

    In the same terminal session where the virtual environment (venv) is now activated, navigate to the `story-book` example directory and run the `story-book.gpt` script:

    ```shell
    cd examples/story-book
    gptscript story-book.gpt --prompt "Goldilocks" --pages 3
    ```

2. **View the story.**

    Open `index.html` in your browser to view the generated story.

3. (optional) **Generate a new story.**

    To generate another story, you'll first need to delete the existing `pages` directory. In the `examples/story-book` directory, run the following command:

    ```shell
    rm -rf pages
    ```

    After that, you can generate a new story by running the `story-book.gpt` script again with a different prompt or number of pages.

    ```shell
    gptscript story-book.gpt --prompt "The Three Little Pigs" --pages 5
    ```
