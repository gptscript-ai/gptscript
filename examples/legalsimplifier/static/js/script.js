document.addEventListener('DOMContentLoaded', function() {

    const randomMessages = [
      "Brewing up some simplicity.",
      "Decoding legalese.",
      "Simplifying complex texts.",
      "Turning the complicated into the understandable.",
      "Working our magic on your document."
    ];

    // Define uploadFile globally
    window.uploadFile = function() {
      var form = document.getElementById('uploadForm');
      var formData = new FormData(form);
      var summaryBlock = document.getElementById('summaryBlock');
      var summaryOutput = document.getElementById('documentSummary');

      // Display a random message
      var messageDiv = document.getElementById('randomMessage');
      messageDiv.innerHTML = randomMessages[Math.floor(Math.random() * randomMessages.length)]; // Display initial random message
      var messageInterval = setInterval(function() {
        messageDiv.innerHTML = randomMessages[Math.floor(Math.random() * randomMessages.length)];
      }, 5000); // Change message every 5 seconds
  
      fetch('/upload', {
        method: 'POST',
        body: formData,
      })
      .then(response => response.json()) // Parse the JSON response
      .then(data => {
        if(data.summary) {
          console.log(data.summary)
          var converter = new showdown.Converter()
          var parsedHtml = converter.makeHtml(data.summary);
          summaryOutput.innerHTML = parsedHtml; // Display the recipe
          summaryBlock.style.display='block'
          messageDiv.style.display = 'none' // Clear message

           // Scroll to the documentSummary div
          document.getElementById('documentSummary').scrollIntoView({
          behavior: 'smooth', // Smooth scroll
          block: 'start' // Align to the top of the view
          });

        } else if (data.error) {
          summaryOutput.innerHTML = `<p>Error: ${data.error}</p>`;
          messageDiv.style.display = 'none' // Clear message
        }
      })
      .catch(error => {
        console.error('Error:', error);
        summaryOutput.innerHTML = `<p>Error: ${error}</p>`;
        messageDiv.style.display = 'none' // Clear message
      });
    };
  });