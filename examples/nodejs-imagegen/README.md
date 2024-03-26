# AI Logo Design (Nodejs image generation example)

## Overview

This nodejs app uses the gptscript node module and the image generation tool (leverages DALL-E) to create a simple AI logo design site. The app uses an existing three AI generated artists, or can create an additional set of three artists, to render a logo based on the description provided by the user. The resulting logo images are then presented to the user.

## Features

- Three built in AI artists.
- Ability to generate three additional AI artists.
- Provide a description of the logo to be generated.
- View three rendered logo options.

## Installation

1. Clone the repository.

```bash
git clone https://github.com/gptscript-ai/gptscript.git
```

1. Navigate to the 'examples/nodejs-imagegen' directory and install dependencies.

Node:

```bash
npm install
```

This is for the application itself.

1. Setup OPENAI_API_KEY environment variable.

```bash
export OPENAI_API_KEY=your-api-key
```

1. Run the application.

```bash
npm run dev
```

## Usage

Once the application is started, open <http://localhost:3000> in your browser. You will be presented with a form to enter a description of the logo you would like to generate. After submitting the form, you will be presented with three logo options.

The backend models take a bit of time to render the images, so it might seem like the application is hanging. This is normal, and the images will be presented once they are ready.

## Diving in

The application is a simple nodejs app that uses the gptscript node module to interact with the image generation tool. The gptscript module is just running the CLI under the hood, and will pick up the standard configuration environment variables, the most important of which is the OPENAI_API_KEY.

Sometimes the OpenAI LLM model will return an error, which can be seen in the terminal running the server app. Sometimes, the model will return a response that doesn't quite match the expected output format and throw an error.
