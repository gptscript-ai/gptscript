# Use the official Python image as the base image
FROM python:3.9-slim

# Set the working directory
WORKDIR /app

# Copy the requirements file
COPY requirements.txt .

# Install the Python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the Flask app code
COPY . .

# Expose the port
EXPOSE 5000

# Set the environment variable
ENV FLASK_APP=app.py
ENV FLASK_DEBUG=True  # Set this to True/False to enable/disable debugging.

# Run app
CMD ["flask", "run", "--host=0.0.0.0"]
