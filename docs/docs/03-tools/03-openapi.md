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

:::tip
The file must have extension `.json`, `.yaml`, or `.yml`.
:::

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

GPTScript currently ignores any security schemes and authentication/authorization information in the OpenAPI definition file. This might change in the future.

For now, the only supported type of authentication is bearer tokens. GPTScript will look for a special environment variable based
on the hostname of the server. It looks for the format `GPTSCRIPT_<HOST>_BEARER_TOKEN`, where `<HOST>` is the hostname, but in all caps and
dots are replaced by underscores. For example, if the server is `https://api.example.com`, GPTScript will look for an environment variable
called `GPTSCRIPT_API_EXAMPLE_COM_BEARER_TOKEN`. If it finds one, it will use it as the bearer token for all requests to that server.

:::note
GPTScript will not look for bearer tokens if the server uses HTTP instead of HTTPS.
:::

## MIME Types and Request Bodies

In OpenAPI definitions, request bodies are described with a MIME type. Currently, GPTScript supports these MIME types:
- `application/json`
- `text/plain`
- `multipart/form-data`

GPTScript will return an error when parsing the OpenAPI definition if it finds a request body that does not specify
at least one of these supported MIME types.
