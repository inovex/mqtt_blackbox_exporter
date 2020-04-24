appname := mqtt_blackbox_exporter

GO:=go
GO111MODULE=on

sources = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
artifact_version = $(shell cat VERSION | tr -d '\n')
build_version = $(artifact_version)-$(shell date +%Y%m%d-%H%M%S)+$(shell git rev-parse --short HEAD)

build = GOOS=$(1) GOARCH=$(2) GOARM=$(4) $(GO) build -ldflags "-X=main.build=$(build_version)" -o build/$(appname)$(3)
tar = cd build && tar -cvzf $(appname)-$(artifact_version).$(1)-$(2).tar.gz $(appname)$(3) && rm $(appname)$(3)
zip = cd build && zip $(appname)-$(artifact_version).$(1)-$(2).zip $(appname)$(3) && rm $(appname)$(3)

.PHONY: all test clean fmt vendor-deps vendor windows darwin linux

all: windows darwin linux

build/$(appname): $(sources)
	$(GO) build -ldflags "-X=main.build=$(build_version)" -o build/$(appname)

test: build/$(appname)
	./test/run-integration-tests.sh

clean:
	rm -rf build/
	rm -rf mqtt_blackbox_exporter

fmt:
	@gofmt -l -w $(sources)

tidy:
	go mod tidy

##### LINUX #####
linux: build/$(appname)-$(artifact_version).linux-amd64.tar.gz build/$(appname)-$(artifact_version).linux-arm5.tar.gz build/$(appname)-$(artifact_version).linux-arm6.tar.gz build/$(appname)-$(artifact_version).linux-arm7.tar.gz

build/$(appname)-$(artifact_version).linux-amd64.tar.gz: $(sources)
	$(call build,linux,amd64,)
	$(call tar,linux,amd64)

build/$(appname)-$(artifact_version).linux-arm5.tar.gz: $(sources)
	$(call build,linux,arm,,5)
	$(call tar,linux,arm5)

build/$(appname)-$(artifact_version).linux-arm6.tar.gz: $(sources)
	$(call build,linux,arm,,6)
	$(call tar,linux,arm6)

build/$(appname)-$(artifact_version).linux-arm7.tar.gz: $(sources)
	$(call build,linux,arm,,7)
	$(call tar,linux,arm7)


##### DARWIN (MAC) #####
darwin: build/$(appname)-$(artifact_version).darwin-amd64.tar.gz

build/$(appname)-$(artifact_version).darwin-amd64.tar.gz: $(sources)
	$(call build,darwin,amd64,)
	$(call tar,darwin,amd64)

##### WINDOWS #####
windows: build/$(appname)-$(artifact_version).windows-amd64.zip

build/$(appname)-$(artifact_version).windows-amd64.zip: $(sources)
	$(call build,windows,amd64,.exe)
	$(call zip,windows,amd64,.exe)
