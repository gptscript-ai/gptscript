from gptscript.command import stream_exec_file
from flask import Flask, render_template, request, jsonify
import os
import uuid
from werkzeug.utils import secure_filename

app = Flask(__name__)

# Setting the base directory
base_dir = os.path.dirname(os.path.abspath(__file__))
app.config['PWD'] = base_dir
SCRIPT_PATH = os.path.join(base_dir, 'designer.gpt')

# The output file name
def print_output(out, err):
    # Error stream has the debug info that is useful to see
    for line in err:
        print(line)
    for line in out:
        print(line)

@app.route('/')
def index():
    return render_template('index.html')

def save_image(image_file, image_file_name, request_id):
    # Save the uploaded image to the current directory
    image_path = os.path.join(app.config['PWD'], image_file_name)
    image_file.save(image_path)

    return image_path

@app.route('/get-ideas', methods=['POST'])
def get_ideas():
    try:
        # Generate a unique request ID
        request_id = str(uuid.uuid4())

        # Get the image file and prompt from the request
        image_file = request.files['image']
        prompt = request.form['prompt']

        # Generate an input image and output file name based on the request ID
        image_file_name = f"{request_id}_room.jpg"
        output_file_name = f"{request_id}_output.md"
        output_file_path = os.path.join(app.config['PWD'], output_file_name)

        # Save the image file to the current directory
        image_path = save_image(image_file, image_file_name, request_id)

        # Execute the script with the prompt, image path and outputfile name
        out, err, wait = stream_exec_file(SCRIPT_PATH, "--prompt " + prompt + " --outputfile "+output_file_name + " --imagefile "+image_file_name)
        print_output(out, err)
        wait()

        # Read the output file
        with open(output_file_path, 'r') as output_file:
            summary = output_file.read()

        # Return the summary content
        return summary
    except Exception as e:
        return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(debug=os.environ.get('FLASK_DEBUG', True), host='0.0.0.0')