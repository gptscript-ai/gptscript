# GPTReview

This folder contains an example of building and implementing your own code reviewer as part of GitHub Actions. 

Below are the files present here:

- `codereview.gpt`: Contains the GPTScript code and prompts.
- `workflow.yaml`: The workflow file for the GitHub action.

## Pre-requisites

- GitHub Account
- OpenAI API Key

## How To Run This Example

- Create a new repository in your GitHub account and create a `codereview.gpt` file in the root of that repo based on the contents provided in this file.
- Congfigure a GitHub Action for that repository. To do so, navigate to the "Actions" tab and then click on "setup a workflow yourself" link. This will create a new `main.yaml` inside `.github/workflows` path. Copy the contents from `workflow.yaml` to your `main.yaml`.
- Configure your `OPENAI_API_KEY` and `GH_TOKEN` as environment variables in your GitHub repo. Refer to [these steps](https://docs.github.com/en/actions/learn-github-actions/variables#creating-configuration-variables-for-a-repository) to create environment variables for your repository.
- Create a new branch, and add some code file to the repository and open a new pull request.
- The GitHub Action will trigger and our GPTReview will review your code and provide review comments.
