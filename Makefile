.PHONY: air air-server air-term
APP ?= server
ARGS ?=

air:
	@air \
		--build.cmd "go build -o ./bin/$(APP) ./cmd/$(APP)" \
		--build.entrypoint "./bin/$(APP)" \
		--build.args_bin "$(ARGS)" \

air-server:
	@$(MAKE) air APP=server
