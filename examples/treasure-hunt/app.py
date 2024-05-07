from gptscript.command import stream_exec_file
from flask import Flask, render_template, request, jsonify
import os
import uuid

app = Flask(__name__)

# Setting the base directory
base_dir = os.path.dirname(os.path.abspath(__file__))
app.config['PWD'] = base_dir
SCRIPT_PATH = os.path.join(base_dir, 'treasure-hunt.gpt')

def print_output(out, err):
    # Error stream has the debug info that is useful to see
    for line in err:
        print(line)
    for line in out:
        print(line)

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/get-clues', methods=['POST'])
def get_clues():
    try:
        code = request.json['code']

        # Generate a unique request ID
        request_id = str(uuid.uuid4())

        # Generate an output file name based on the request ID
        output_file_name = f"{request_id}_treasure-hunt.md"
        output_file_path = os.path.join(app.config['PWD'], output_file_name)

        # Execute the script to generate the clues
        out, err, wait = stream_exec_file(SCRIPT_PATH, "--locations " + code + " --outputfile "+output_file_name)
        print_output(out, err)
        wait()

        # Read the output file
        with open(output_file_path, 'r') as output_file:
            summary = output_file.read()

        # Return clues
        return summary
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(debug=False)