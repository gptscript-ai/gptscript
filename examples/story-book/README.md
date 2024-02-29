# Story Book

Story Book is a GPTScript that can generate a story based on a prompt and the number of pages you want the story to be in. It is generated in HTML format and can then be viewed
by `index.html` which has some JS/CSS to make the story styling consistent and readable.

## How to Use
1. Install the `image-generation` tool.

```shell
git clone https://github.com/gptscript-ai/image-generation
```

2. Bootstrap the `image-generation` tool.

```shell
(
    cd image-generation
    make bootstrap
    source .venv/bin/activate
)
```

3. In the `story-book` directory of `gptscript`, run the `story-book.gpt` script.

```shell
cd examples/story-book
gptscript examples/story-book/story-book.gpt --prompt "Goldilocks" --pages 3
```

4. Open `index.html` in your browser to view the story.