APP_NAME := kpatcher
VERSION := 1.0.0

SRC_FILE := cmd/main.go

PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 windows/amd64

OUTPUT_DIR := release

all: build

build: $(PLATFORMS)

$(PLATFORMS):
	@mkdir -p $(OUTPUT_DIR)/$(APP_NAME)-$@
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) go build -o $(OUTPUT_DIR)/$(APP_NAME)-$@/$(APP_NAME)$(if $(filter windows,$(word 1,$(subst /, ,$@))),.exe,) $(SRC_FILE)
	cd $(OUTPUT_DIR) && tar -czvf $(APP_NAME)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@)).tar.gz $(APP_NAME)-$@ && rm -rf $(APP_NAME)-$@

clean:
	rm -rf $(OUTPUT_DIR)

# Phony targets
.PHONY: all build clean
