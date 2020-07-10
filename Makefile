# Makefile to build the command lines and tests in Scdo project.
# This Makefile doesn't consider Windows Environment. If you use it in Windows, please be careful.
gopath="$(shell go env GOPATH)"
goarch="$(shell go env GOARCH)"
gohostos="$(shell go env GOHOSTOS)"
all: discovery node client light tool vm
discovery:
	go build -o ./build/discovery ./cmd/discovery
	@echo "Done discovery building"

node:
	$(shell mkdir $(gopath)/pkg/$(gohostos)_$(goarch)/github.com)
	$(shell mkdir $(gopath)/pkg/$(gohostos)_$(goarch)/github.com/scdoproject)
	$(shell mkdir $(gopath)/pkg/$(gohostos)_$(goarch)/github.com/scdoproject/go-scdo)
	$(shell mkdir $(gopath)/pkg/$(gohostos)_$(goarch)/github.com/scdoproject/go-scdo/consensus)
ifeq ($(gohostos),"darwin") 
	$(shell cp ./consensus/scdorand/scdorand_darwin_amd64.a $(gopath)/pkg/$(gohostos)_$(goarch)/github.com/scdoproject/go-scdo/consensus/scdorand.a)
else
	$(shell cp ./consensus/scdorand/scdorand_linux_amd64.a $(gopath)/pkg/$(gohostos)_$(goarch)/github.com/scdoproject/go-scdo/consensus/scdorand.a)
endif
	go build -o ./build/node ./cmd/node 
	@echo "Done node building"

client:
	go build -o ./build/client ./cmd/client
	@echo "Done full node client building"

light:
	go build -o ./build/light ./cmd/client/light
	@echo "Done light node client building"

tool:
	go build -o ./build/tool ./cmd/tool
	@echo "Done tool building"

vm:
	go build -o ./build/vm ./cmd/vm
	@echo "Done vm building"

.PHONY: discovery node client light tool vm
