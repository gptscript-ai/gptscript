from flask import Flask, jsonify, render_template, request
import subprocess
import os

app = Flask(__name__)

# Setting the base directory
base_dir = os.path.dirname(os.path.abspath(__file__))
app.config['UPLOAD_FOLDER'] = base_dir

SCRIPT_PATH = os.path.join(base_dir, 'legalsimplifier.gpt')
LEGAL_FILE_NAME = 'legal.pdf' # Uploaded document name
SUMMARY_FILE_NAME = 'summary.md'  # The output file name

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/upload', methods=['POST'])
def upload_file():
    if 'file' not in request.files:
        return jsonify({'error': 'No file part'}), 400
    file = request.files['file']
    if file.filename == '':
        return jsonify({'error': 'No selected file'}), 400
    if file:
        # Process the file here to generate the summary
        filename = os.path.join(app.config['UPLOAD_FOLDER'], LEGAL_FILE_NAME)
        file.save(filename)
        summary = process_file(file)
        return jsonify({'summary': summary})

def process_file(file):
    try:
        # Execute the script to generate the recipe
        subprocess.run(f"gptscript {SCRIPT_PATH}", shell=True, check=True)  
            
        # Read summary.md file
        summary_file_path = os.path.join(app.config['UPLOAD_FOLDER'], SUMMARY_FILE_NAME)
        with open(summary_file_path, 'r') as summary_file:
            summary = summary_file.read()
        
        # Return summary content
        return summary
    except Exception as e:
        return jsonify({'error': str(e)}), 500
    
if __name__ == '__main__':
    app.run(debug=False)