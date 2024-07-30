# GPTReview With Jenkins

This folder contains an example of building and implementing your own code reviewer as part of Jenkins Pipeline. 

Below are the files present here:

- `codereview.gpt`: Contains the GPTScript code and prompts.
- `Jenkinsfile`: Jenkins pipeline file.

## Pre-requisites

- An OpenAI API Key.
- GitHub repository.
- Jenkins.
- [GPTScript](https://github.com/gptscript-ai/gptscript) and [GH](https://github.com/cli/cli) CLI installed on the system running Jenkins.

## How To Run This Example

- Create a new repository in your GitHub account and create a `codereview.gpt` file in the root of that repo based on the contents provided in this file.
- Configure Jenkins:
  - Install required plugins - [GitHub](https://plugins.jenkins.io/github/), [Generic Webhook Trigger Plugin](https://plugins.jenkins.io/generic-webhook-trigger/) & [HTTP Request Plugin](https://plugins.jenkins.io/http_request/).
  - Create a Pipeline
  - Configure the “Open_AI_API” and “GH_TOKEN” environment variables

- Congfigure GitHub:
  - Setup up Webhook by providing your Jenkins pipeline URL: `http://<jenkins-url>/generic-webhook-trigger/invoke?token=<secret-token>`
  - Add `Jenkinsfile` in the root of the repo. *Note: Replace the repository URL with your repo URL in the Jenkinsfile provided.*
  
- Executing the Script:
  - Create a new branch, and add some code file to the repository and open a new pull request.
  - The Jenkins pipeline will trigger and our GPTReview will review your code and provide review comments.
