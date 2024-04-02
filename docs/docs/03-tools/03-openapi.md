# OpenAPI Tools

:::note
This is a new feature and might be buggy.
:::

GPTScript can treat OpenAPI v3 definition files as though they were tool files.
Each operation (a path and HTTP method) in the file will become a simple tool that makes an HTTP request.
GPTScript will automatically and internally generate the necessary code to make the request and parse the response.

Here is an example that uses the OpenAPI [Petstore Example](https://github.com/OAI/OpenAPI-Specification/blob/main/examples/v3.0/petstore.yaml):

```yaml
Tools: https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml

List all the pets. After you get a response, create a new pet named Mark. He is a lizard.
```

You can also use a local file path instead of a URL.

## Servers

GPTScript will look at the top-level `servers` array in the file and choose the first HTTPS server it finds.
If no HTTPS server exists, it will choose the first HTTP server. Other protocols (such as WSS) are not yet supported.

GPTScript will also handle path- and operation-level server overrides, following the same logic of choosing the first HTTPS server it finds,
or the first HTTP server if no HTTPS server exists in the array.

Additionally, GPTScript can handle variables in the server name. For example, this:

```yaml
servers:
  - url: '{server}/v1'
    variables:
      server:
        default: https://api.example.com
```

Will be resolved as `https://api.example.com/v1`.

## Authentication

:::warning
All authentication options will be completely ignored if the server uses HTTP and not HTTPS.
This is to protect users from accidentally sending credentials in plain text.
:::

### 1. Security Schemes

GPTScript will read the defined [security schemes](https://swagger.io/docs/specification/authentication/) in the OpenAPI definition. The currently supported types are `apiKey` and `http`.
OAuth and OIDC schemes will be ignored.

GPTScript will look at the `security` defined on the operation (or defined globally, if it is not defined on the operation) before it makes the request.
It will set the necessary headers, cookies, or query parameters based on the corresponding security scheme.

Environment variables must be set for each security scheme that will be used by the operation.
`<HOSTNAME>` is the hostname of the server, but all caps, and with dashes (`-`) and dots (`.`) replaced with underscores (`_`).
`<SCHEME NAME>` is the name of the security scheme, but all caps, and with dashes (`-`) and dots (`.`) replaced with underscores (`_`).

- For `apiKey`-type and `http`-type with `bearer` scheme, the environment variable is `GPTSCRIPT_<HOSTNAME>_<SCHEME NAME>`
- For `http`-type with `basic` scheme, the environment variables are `GPTSCRIPT_<HOSTNAME>_<SCHEME NAME>_USERNAME` and `GPTSCRIPT_<HOSTNAME>_<SCHEME NAME>_PASSWORD`

#### Example

To explain this better, let's use this example:

```yaml
servers:
  - url: https://api.example.com/v1
components:
  securitySchemes:
    MyBasic:
      type: http
      scheme: basic

    MyAPIKey:
      type: apiKey
      in: header
      name: X-API-Key
security:
  - MyBasic: []
  - MyAPIKey: []
# The rest of the document defines paths, etc.
```

In this example, we have two security schemes, and both are defined as the defaults on the global level.
They are separate entries in the global `security` array, so they are treated as a logical OR, and GPTScript will only
need the environment variable for one or the other to make the request.

When put into the same entry, they would be a logical AND, and the environment variables for both would be required.
It would look like this:

```yaml
security:
  - MyBasic: []
    MyAPIKey: []
```

The environment variable names are as follows:

- `GPTSCRIPT_API_EXAMPLE_COM_MYBASIC_USERNAME` and `GPTSCRIPT_API_EXAMPLE_COM_MYBASIC_PASSWORD` for basic auth
- `GPTSCRIPT_API_EXAMPLE_COM_MYAPIKEY` for the API key

### 2. Bearer token for server

GPTScript can also use a bearer token for all requests to a particular server that don't already have an `Authorization` header.
To do this, set the environment variable `GPTSCRIPT_<HOSTNAME>_BEARER_TOKEN`.
If a request to the server already has an `Authorization` header, the bearer token will not be added.

This can be useful in cases of unsupported auth types. For example, GPTScript does not have built-in support for OAuth,
but you can go through an OAuth flow, get the access token, and set it to the environment variable as a bearer token
for the server and use it that way.

## MIME Types and Request Bodies

In OpenAPI definitions, request bodies are described with a MIME type. Currently, GPTScript supports these MIME types:
- `application/json`
- `text/plain`
- `multipart/form-data`

GPTScript will ignore any operations that have a request body without a supported MIME type.
