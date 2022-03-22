all:

release:
	@set -e; \
	echo -n 'enter a version number for this release: '; \
	read -r version; \
	test ! -z "$$version"; \
	git tag -a v$$version -m "v$$version"; \
	git push origin v$$version

.PHONY: release
