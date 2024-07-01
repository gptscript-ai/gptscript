// app.js
new Vue({
    el: '#app',
    data: {
        prompt: '',
        imageFile: null,
        imageUrl: null,
        showIdeas: false,
        ideasMarkdown: '',
        renderedMarkdown: '',
        isLoading: false
    },
    methods: {
        handleImageUpload(event) {
            this.imageFile = event.target.files[0];
            this.imageUrl = URL.createObjectURL(event.target.files[0]);
        },
        getIdeas() {
            this.isLoading = true;
            const formData = new FormData();
            formData.append('image', this.imageFile);
            formData.append('prompt', this.prompt);

            axios.post('/get-ideas', formData, {
                headers: {
                    'Content-Type': 'multipart/form-data'
                }
            })
            .then(response => {
                this.ideasMarkdown = response.data;
                this.renderedMarkdown = marked.parse(this.ideasMarkdown);
                this.showIdeas = true;
            })
            .catch(error => {
                if (error.response && error.response.data && error.response.data.error) {
                    alert('Error: ' + error.response.data.error);
                } else {
                    alert('An unexpected error occurred. Please try again later.');
                }
            })
            .finally(() => {
                this.isLoading = false;
            });
        }
    }
});

// Initialize the marked library
marked.setOptions({
    renderer: new marked.Renderer(),
    highlight: function(code, language) {
        const hljs = require('highlight.js');
        const validLanguage = hljs.getLanguage(language) ? language : 'plaintext';
        return hljs.highlight(validLanguage, code).value;
    },
    pedantic: false,
    gfm: true,
    breaks: false,
    sanitize: false,
    smartLists: true,
    smartypants: false,
    xhtml: false
});