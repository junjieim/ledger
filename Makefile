APP := ledger
DIST_DIR := dist
BUILD_DIR := $(DIST_DIR)/build
SKILL_DIST_DIR := $(DIST_DIR)/skill/$(APP)
MAIN_PKG := ./cmd/ledger

.PHONY: build build-cross clean skill-package verify-package

build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP) $(MAIN_PKG)

build-cross:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP)-darwin-arm64 $(MAIN_PKG)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP)-darwin-amd64 $(MAIN_PKG)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APP)-linux-amd64 $(MAIN_PKG)

skill-package: build
	@rm -rf $(SKILL_DIST_DIR)
	@mkdir -p $(SKILL_DIST_DIR)/script $(SKILL_DIST_DIR)/example $(SKILL_DIST_DIR)/data
	cp skill/SKILL.md $(SKILL_DIST_DIR)/SKILL.md
	cp -R skill/example/. $(SKILL_DIST_DIR)/example/
	cp $(BUILD_DIR)/$(APP) $(SKILL_DIST_DIR)/script/$(APP)
	touch $(SKILL_DIST_DIR)/data/.gitkeep

verify-package: skill-package
	test -f $(SKILL_DIST_DIR)/SKILL.md
	test -f $(SKILL_DIST_DIR)/script/$(APP)
	test -d $(SKILL_DIST_DIR)/example
	test -d $(SKILL_DIST_DIR)/data

clean:
	rm -rf $(DIST_DIR)
