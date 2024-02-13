build: clean
	npm install
	npm run generate
	rm -rf ../static/ui/_nuxt
	cp -rp .output/public/* ../static/ui/
	touch ../static/ui/_nuxt/_placeholder

clean:
	npm run clean
