# GPTReview

This folder contains an example of building and implementing your own code reviewer as part of GitHub Actions. 

Below are the files present here:

- `codereview.gpt`: Contains the GPTScript code and prompts.
- `workflow.yaml`: The workflow file for the GitHub action.

## Pre-requisites

- GitHub Account
- OpenAI API Key

## How To Run This Example

- Create a new repository in your GitHub account and create a `codereview.gpt` file in that repo based on the contents provided in this file.
- Congfigure a GitHub Action for that repository and copy the contents from `workflow.yaml` to your `main.yaml`.
- Configure your `OPENAI_API_KEY` and `GH_TOKEN` as environment variables.
- Add some code file to the repository and open a new pull request.
- The GitHub Action will trigger and our GPTReview will review your code and provide review comments.
