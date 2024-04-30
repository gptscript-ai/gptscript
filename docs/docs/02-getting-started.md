# Getting Started

### 1. Install the latest release

#### Homebrew (macOS and Linux)

```shell
brew install gptscript-ai/tap/gptscript
```

#### Install Script (macOS and Linux):

```shell
curl https://get.gptscript.ai/install.sh | sh
```

#### WinGet (Windows)

```shell
winget install gptscript-ai.gptscript
```

#### Manually

Download and install the archive for your platform and architecture from the [releases page](https://github.com/gptscript-ai/gptscript/releases).

### 2. Get an API key from [OpenAI](https://platform.openai.com/api-keys).

#### macOS and Linux

```shell
export OPENAI_API_KEY="your-api-key"
```

Alternatively Azure OpenAI can be utilized. If the [Azure deployment name is different than the model being used](https://learn.microsoft.com/en-us/azure/ai-services/openai/how-to/switching-endpoints#keyword-argument-for-model), be sure to include the `OPENAI_AZURE_DEPLOYMENT` argument.

```shell
export OPENAI_API_KEY="your-api-key"
export OPENAI_BASE_URL="https://<your-endpoint>.openai.azure.com/"
export OPENAI_API_TYPE="AZURE"
export OPENAI_AZURE_DEPLOYMENT="<your-deployment-name>"
```

#### Windows

```powershell
$env:OPENAI_API_KEY = 'your-api-key'
```

### 3. Run Hello World

```shell
gptscript https://get.gptscript.ai/echo.gpt --input 'Hello, World!'
```

```
OUTPUT:

Hello, World!
```

The model used by default is `gpt-4-turbo` and you must have access to that model in your OpenAI account.

If using Azure OpenAI, make sure you configure the model to be one of the supported versions with the `--default-model` argument.

### 4. Extra Credit: Examples and Run Debugging UI

Clone examples and run debugging UI

```shell
git clone https://github.com/gptscript-ai/gptscript
cd gptscript/examples

# Run the debugging UI
gptscript --server
```
