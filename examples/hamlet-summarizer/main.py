import tiktoken
import sys
from llama_index.readers.file import PyMuPDFReader
from llama_index.core.node_parser import TokenTextSplitter

index = int(sys.argv[1])
docs = PyMuPDFReader().load("Hamlet.pdf")

combined = ""
for doc in docs:
    combined += doc.text

splitter = TokenTextSplitter(
    chunk_size=10000,
    chunk_overlap=10,
    tokenizer=tiktoken.encoding_for_model("gpt-4").encode)

pieces = splitter.split_text(combined)

if index >= len(pieces):
    print("No more content")
    sys.exit(0)

print(pieces[index])
