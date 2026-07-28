.PHONY: air air-api air-term
APP ?= api
ARGS ?=

air:
	@air \
		--build.cmd "go build -o ./bin/$(APP) ./cmd/$(APP)" \
		--build.entrypoint "./bin/$(APP)" \
		--build.args_bin "$(ARGS)" \

air-api:
	@$(MAKE) air APP=api
