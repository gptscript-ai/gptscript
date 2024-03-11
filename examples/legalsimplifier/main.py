import tiktoken
import sys
from llama_index.readers.file import PyMuPDFReader
from llama_index.core.node_parser import TokenTextSplitter

# Loading the legal document using PyMuPDFReader
index = int(sys.argv[1])
docs = PyMuPDFReader().load("legal.pdf")

# Combining text content from all documents into a single string
combined = ""
for doc in docs:
    combined += doc.text

# Initializing a TokenTextSplitter object with specified parameters
splitter = TokenTextSplitter(
    chunk_size=10000,
    chunk_overlap=10,
    tokenizer=tiktoken.encoding_for_model("gpt-4-turbo-preview").encode)

pieces = splitter.split_text(combined)

# Checking if the specified index is valid
if index >= len(pieces):
    print("No more content")
    sys.exit(0)

print(pieces[index])