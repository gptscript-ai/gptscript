build: clean
	yarn
	yarn generate
	rm -rf ../static/ui/_nuxt
	cp -rp .output/public/* ../static/ui/

clean:
	yarn clean
