from flask import Flask, request, render_template, jsonify
import subprocess
import os

app = Flask(__name__)

# Setting the base directory
base_dir = os.path.dirname(os.path.abspath(__file__))
app.config['UPLOAD_FOLDER'] = base_dir

SCRIPT_PATH = os.path.join(base_dir, 'recipegenerator.gpt')
GROCERY_PHOTO_FILE_NAME = 'grocery.png'  # The expected file name
RECIPE_FILE_NAME = 'recipe.md'  # The output file name

@app.route('/', methods=['GET'])
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
        filename = os.path.join(app.config['UPLOAD_FOLDER'], GROCERY_PHOTO_FILE_NAME)
        file.save(filename)
        try:
            # Execute the script to generate the recipe
            subprocess.Popen(f"gptscript {SCRIPT_PATH}", shell=True, stdout=subprocess.PIPE, cwd=base_dir).stdout.read()
            
            # Read recipe.md file
            recipe_file_path = os.path.join(app.config['UPLOAD_FOLDER'], RECIPE_FILE_NAME)
            with open(recipe_file_path, 'r') as recipe_file:
                recipe_content = recipe_file.read()
            
            # Return recipe content
            return jsonify({'recipe': recipe_content})
        except Exception as e:
            return jsonify({'error': str(e)}), 500

if __name__ == '__main__':
    app.run(debug=False)
