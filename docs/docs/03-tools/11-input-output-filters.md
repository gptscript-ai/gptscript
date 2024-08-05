# Input and Output Filters (Advanced)

GPTScript supports input and output filters, which are tools that can modify the input to a tool or the output from a tool.
These are best explained with examples.

## Input Filter Example

In this example, the entrypoint tool uses an input filter to modify the `message` parameter, before calling the subtool.
Then, the subtool uses another input filter to modify the message, then writes it to a file.

```
# File name: script.gpt
Param: message: the message from the user
Tools: subtool
Input Filter: appleToOrange

Take the message and give it to the subtool. Then say "Done".

---
Name: subtool
Param: message: the message from the user
Input Filter: orangeToBanana

#!python3

import os

message = os.getenv("message", "")
with open("gptscript_output.txt", "w") as f:
    f.write(message)

---
Name: appleToOrange

#!python3

import os

def output(input: str):
    return input.replace("apple", "orange")

print(output(os.getenv("INPUT", "")))

---
Name: orangeToBanana

#!python3

import os

def output(input: str):
    return input.replace("orange", "banana")
    
print(output(os.getenv("INPUT", "")))
```

Try running this tool with the following command:

```bash
gptscript script.gpt '{"message":"apple is great"}'

# Then view the output:
cat gptscript_output.txt
```

The output should say "banana is great".
This matches what we expect, because the input filter `appleToOrange` changes "apple" to "orange",
and the input filter `orangeToBanana` changes "orange" to "banana".
If we run the tool again with a different message, like "hello world", the final message will be unmodified,
since it did not include the words "apple" or "orange".

The input filter tools both read the input from the environment variable `INPUT`.
They write their modified input to stdout.
This variable is set by GPTScript before running the input filter tool.

### Input Filter Real-World Example

For a real-world example of an input filter tool, check out the [gptscript-ai/context/at-syntax](https://github.com/gptscript-ai/context/tree/main/at-syntax) tool.

## Output Filter Example

In this example, the tool is asked to write a poem about apples.
The output filter then replaces all references to apples with oranges.

```
Output Filter: applesToOranges

Write a poem about apples.

---
Name: applesToOranges

#!python3

import os

replacements = {
    "Apples": "Oranges",
    "apples": "oranges",
    "apple": "orange",
    "Apple": "Orange",            
}

def applesToOranges(input: str) -> str:
    for key, value in replacements.items():
        if input.startswith(key):
            # This approach doesn't maintain whitespace, but it's good enough for this example
            input = input.replace(key, value)
    return input

output: str = os.getenv("OUTPUT", "")
new_output: str = ""
for i in output.split():
    new_output += applesToOranges(i) + " "
print(new_output.strip())
```

```
OUTPUT:

In orchards where the sunlight gleams, Among the leaves, in golden beams, The oranges hang on branches high, A feast for both the heart and eye.
Their skins, a palette rich and bright, In hues of red and green delight, With every bite, a crisp surprise, A taste of autumn, pure and wise.
From pies to cider, sweet and bold, Their stories through the seasons told, In every crunch, a memory, Of nature's gift, so wild and free.
Oh, oranges, treasures of the earth, In every form, you bring us mirth, A simple fruit, yet so profound, In you, a world of joy is found.
```

The output tool reads the output from the environment variable `OUTPUT`.
It can then modify the output as needed, and print the new output to stdout.

Output filter tools can also access the following environment variables if needed:

- `CHAT` (boolean): indicates whether the current script is being run in chat mode or not
- `CONTINUATION` (boolean): if `CHAT` is true, indicates whether the current chat will continue executing, or if this is the final message

### Output Filter Real-World Example

For a real-world example of an output filter tool, check out the [gptscript-ai/context/chat-summary](https://github.com/gptscript-ai/context/tree/main/chat-summary) tool.
