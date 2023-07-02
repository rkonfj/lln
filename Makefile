version ?= latest
git_hash := $(shell git rev-parse --short HEAD)
gomod := github.com/rkonfj/lln

GOBUILD := CGO_ENABLED=0 go build -ldflags "-s -w -X '${gomod}/tools.Version=${version}' -X '${gomod}/tools.Commit=${git_hash}'"

all: linux darwin windows

linuxamd64:
	GOOS=linux GOARCH=amd64 ${GOBUILD} -o lln-${version}-linux-amd64
linuxarm64:
	GOOS=linux GOARCH=arm64 ${GOBUILD} -o lln-${version}-linux-arm64
linux: linuxamd64 linuxarm64
darwinamd64:
	GOOS=darwin GOARCH=amd64 ${GOBUILD} -o lln-${version}-darwin-amd64
darwinarm64:
	GOOS=darwin GOARCH=arm64 ${GOBUILD} -o lln-${version}-darwin-arm64
darwin: darwinamd64 darwinarm64
windows:
	GOOS=windows GOARCH=amd64 ${GOBUILD} -o lln-${version}-windows-amd64.exe
image:
	docker build . -t rkonfj/lln:${version} --build-arg version=${version} --build-arg githash=${git_hash} --build-arg gomod=${gomod}
dockerhub: image
	docker push rkonfj/lln:${version}
github: clean all
	gzip lln-${version}*
	git tag -d ${version} 2>/dev/null || true
	gh release delete ${version} -R rkonfj/lln -y --cleanup-tag 2>/dev/null || true
	gh release create ${version} -R rkonfj/lln --generate-notes --title "lln ${version}" lln-${version}*.gz
dist: github dockerhub
clean:
	rm lln* 2>/dev/null || true