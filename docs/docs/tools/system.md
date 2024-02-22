# System Tools
GPTScript comes with a set of system tools that provide various core functionalities.

## sys.abort
Aborts the operation and provides an error message.
#### Arguments
- `message`: The description of the error or unexpected result that caused abort to be called.

## sys.download
Downloads a file from a specified URL to an optional location on disk with an option to override existing files.
#### Arguments
- `location` (optional): The on-disk location to store the downloaded file.
- `override`: If true, allows overwriting of an existing file. Default is false.
- `url`: The HTTP or HTTPS URL of the file to be downloaded.

## sys.exec
Executes a command with the ability to specify command arguments and the working directory.
#### Arguments
- `command`: The full command to run, including all arguments.
- `directory`: The working directory for the command. Defaults to the current directory ".".

## sys.find
Searches for files within a directory that match a given pattern using Unix glob format.
#### Arguments
- `directory`: The directory to perform the search in. Defaults to the current directory ".".
- `pattern`: The pattern to match against filenames.

## sys.getenv
Retrieves the value of an environment variable.
#### Arguments
- `name`: The name of the environment variable to retrieve.

## sys.http.get
Performs an HTTP GET request to the specified URL.
#### Arguments
- `url`: The URL to perform the GET request.

## sys.http.html2text
Converts the HTML content from a given URL to plain text.
#### Arguments
- `url`: The URL of the HTML content to be converted.

## sys.http.post
Sends an HTTP POST request with given content to a specified URL.
#### Arguments
- `content`: The content to be posted.
- `contentType`: The MIME type of the content being posted.
- `url`: The URL to which the POST request should be sent.

## sys.read
Reads the content from a specified file.
#### Arguments
- `filename`: The name of the file from which to read content.

## sys.remove
Removes a file from the specified location.
#### Arguments
- `location`: The path to the file that needs to be removed.

## sys.write
Writes content to a specified file.
#### Arguments
- `content`: The content to be written to the file.
- `filename`: The filename where the content should be written.
