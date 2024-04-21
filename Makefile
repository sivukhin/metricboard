.PHONY: js
js:
	cd js && bun build index.ts --no-bundle --outfile=index.js

