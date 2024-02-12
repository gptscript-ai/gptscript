build: clean
	yarn
	yarn generate
	rm -rf ../static/ui/_nuxt
	cp -rp .output/public/* ../static/ui/
	touch ../static/ui/_nuxt/_placeholder

clean:
	yarn clean
