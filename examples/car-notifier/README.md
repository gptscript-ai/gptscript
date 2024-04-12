# Car Notifier

This is an example tool that fetches information about recently posted Toyota 4Runners
on Phoenix Craigslist and sends an email with information about them.

This tool is meant to be run as a cron job, daily, and requires a PostgreSQL database.
It uses SendGrid to send the email.

## Design

This tool does not rely on any code besides a Dockerfile and a GPTScript. The Dockerfile
is only used to containerize the environment that the GPTScript will run in.

## Run the Example

Prerequisites:
- A PostgreSQL database, and a connection URL to it
- A SendGrid API key
- `psql` CLI client for PostgreSQL
- `curl`

Before running the script (or building the Dockerfile), be sure to edit it and fill in
every occurrence  of `<email address>` and `<name>`. Unfortunately, there is no
straightforward way to provide this information through environment variables, due to
issues with escaping quotes in the cURL command that the LLM will run.

```bash
# Set up the environment variables
export PGURL=your-postgres-connection-url
export SENDGRID_API_KEY=your-sendgrid-api-key

# Run the script
gptscript --disable-cache car-notifier.gpt
```
