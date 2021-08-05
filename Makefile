GO         = go
GOGET      = $(GO) get -u
GOFLAGS   ?= -mod=vendor
GOPATH    ?= $(HOME)/go

OAPI_CODEGEN = $(GOPATH)/bin/oapi-codegen

GIN        = $(GOPATH)/bin/gin
GIN_ARG_PORT  ?= 3200
GIN_ARG_APORT ?= 3201
GIN_ARG_LADDR ?= localhost
GIN_ARGS      ?= --laddr $(GIN_ARG_LADDR) --port $(GIN_ARG_PORT) --appPort $(GIN_ARG_APORT) --immediate


CODEGEN_API = api/gen.go

watch: $(GIN)
	$(GIN) $(GIN_ARGS) run -- serve
