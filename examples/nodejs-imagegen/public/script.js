document.addEventListener('DOMContentLoaded', function () {
    // Function to fetch and display artists
    function fetchArtists() {
        fetch('/artists')
            .then(response => response.json())
            .then(data => {
                const artistsSection = document.getElementById('artistsSection');
                artistsSection.innerHTML = ""; // reset

                data.artists.forEach(artist => {
                    const artistCard = `
            <div class="max-w-xs rounded overflow-hidden shadow-lg bg-white m-4">
              <div class="px-6 py-4">
                <div class="font-bold text-xl mb-2">${artist.name}</div>
                <p class="text-gray-700 text-base">${artist.description}</p>
              </div>
            </div>
          `;
                    artistsSection.insertAdjacentHTML('beforeend', artistCard);
                });
            })
            .catch(error => {
                console.error('Error:', error);
            });
    }

    // Function to reattach event listeners
    function attachEventListeners() {
        // Event listener for fetching new artists
        const newArtistsLink = document.getElementById('newArtistsLink');

        newArtistsLink.addEventListener('click', function (event) {
            event.preventDefault();

            // Clear previous submissions
            resetSubmissions();

            // Fetch and display new artists
            fetch('/new-artists', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            })
                .then(response => response.json())
                .then(data => {
                    fetchArtists();
                })
                .catch(error => {
                    console.error('Error:', error);
                });
        });

        // Event listener for submitting logo description
        const submitBtn = document.getElementById('submitBtn');

        submitBtn.addEventListener('click', function () {
            resetSubmissions();
            // Show spinner and waiting message for each response area
            const spinner = document.querySelector('.spinner-border');
            spinner.classList.remove('hidden');

            // Send POST request to backend for logo generation
            fetch('/generate-logo', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({
                    description: document.getElementById('logoDescription').value
                })
            })
                .then(response => response.json())
                .then(data => {
                    const artistResponse = document.getElementById('artistResponse');
                    // Hide spinner and display response message for each response area
                    document.querySelectorAll('.spinner-border').forEach(spinner => {
                        spinner.classList.add('hidden');
                    });
                    spinner.classList.add('hidden');
                    artistResponse.innerHTML = "";

                    // Display generated logos for each artist
                    data.submissions.forEach(submission => {
                        // Create HTML elements
                        const artistCard = document.createElement('div');
                        artistCard.classList.add('max-w-xs', 'rounded', 'overflow-hidden', 'shadow-lg', 'bg-white', 'm-4');

                        const innerDiv = document.createElement('div');
                        innerDiv.classList.add('px-6', 'py-4');

                        const artistName = document.createElement('div');
                        artistName.classList.add('font-bold', 'text-xl', 'mb-2');
                        artistName.textContent = submission.name;

                        const image = document.createElement('img');
                        image.setAttribute('src', submission.url);
                        image.setAttribute('alt', `${submission.name} Logo`);
                        image.classList.add('w-full', 'h-auto');

                        // Append elements to each other
                        innerDiv.appendChild(artistName);
                        innerDiv.appendChild(image);
                        artistCard.appendChild(innerDiv);

                        // Append the generated HTML to the container
                        artistResponse.appendChild(artistCard);
                    });
                })
                .catch(error => {
                    console.error('Error:', error);
                });

        });
    }

    function resetSubmissions() {
        // Clear previous submissions
        document.getElementById('artistResponse').innerHTML = `
            <div class="spinner-border hidden text-blue-500" role="status">
             <span class="sr-only">Waiting for submissions...</span>
            </div>`;
        const spinner = document.querySelector('.spinner-border');
        spinner.classList.add('hidden');
    }

    // Initial setup
    fetchArtists(); // Fetch and display artists
    attachEventListeners(); // Attach event listeners
});
