// app.js
new Vue({
    el: '#app',
    data: {
        code: '',
        showClues: false,
        cluesMarkdown: '',
        renderedMarkdown: '',
        isLoading: false
    },
    methods: {
        getClues() {
            this.isLoading = true;
            axios.post('/get-clues', { code: this.code })
                .then(response => {
                    this.cluesMarkdown = response.data;
                    this.renderedMarkdown = marked.parse(this.cluesMarkdown)
                    this.showClues = true;
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

