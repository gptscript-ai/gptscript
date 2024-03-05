const path = require('path');
const express = require('express');
const bodyParser = require('body-parser');
const gptscript = require('@gptscript-ai/gptscript');
const app = express();
const PORT = 3000;

// Middleware
app.use(bodyParser.json());
app.use(express.static(path.join(__dirname, 'public')));


// JSON data
const artistsData = require('./artists.json');

let newArtistsData = new Map();

function getArtistsData() {
    if (newArtistsData.size === 0) {
        return artistsData;
    } else {
        return newArtistsData;
    }
}

// Route to serve index.html
app.get('/', (req, res) => {
    res.sendFile(__dirname + '/public/index.html');
});

// Route to handle logo generation request
app.post('/generate-logo', async (req, res) => {
    const description = req.body.description;
    const artists = getArtistsData();
    const prompt = `
    tools: ./tool.gpt
    For each of the three amazing and capable artists described here
    ${JSON.stringify(artists)}
    make sure to include the artistName field in the response. 

    Have each one generate a logo that meets these requirements and represents their pov:
    ${description}

    The response format should be json and MUST NOT have other content or formatting.
    the name should be the name of the artist for the submission

    {
        submissions: [{
            name: "artistName",
            url: "imageURL"
        }]
    }
`;
    try {
        const output = await gptscript.exec(prompt);
        const cleanedResp = output.trim().replace(/^```json.*/, '').replace(/```/g, '');
        res.json(JSON.parse(cleanedResp));
    } catch (error) {
        console.error(error);
        res.json({ error: error.message })
    }
});

// Route to request new artists
app.post('/new-artists', async (req, res) => {
    const prompt = `
    tools: sys.write

Create three short graphic artist descriptions and their muses. 
These should be descriptive and explain their point of view.
Also come up with a made up name, they each should be from different
backgrounds and approach art differently.

the response format should be json and MUST NOT have other content or formatting.

{
  artists: [{
     name: "name"
     description: "description"
  }]
}
`;
    try {
        const output = await gptscript.exec(prompt);
        const cleanedResp = output.trim().replace(/^```json.*/, '').replace(/```/g, '');
        newArtistsData = JSON.parse(cleanedResp);
        res.json(newArtistsData);
    } catch (error) {
        console.error(error);
        res.json({ error: error.message })
    }
});

// Route to serve artists data
app.get('/artists', (req, res) => {
    const artistsData = getArtistsData();
    res.json(artistsData);
});

// Start server
app.listen(PORT, '0.0.0.0', () => {
    console.log(`Server is running on http://0.0.0.0:${PORT}`);
});
