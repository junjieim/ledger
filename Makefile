APP := ledger
DIST_DIR := dist
BUILD_DIR := $(DIST_DIR)/build
SKILL_DIST_DIR := $(DIST_DIR)/skill/$(APP)
PACKAGE_DIR := $(DIST_DIR)/package
RELEASE_DIR := $(DIST_DIR)/release
SKILL_INSTALL_DIR ?= $(HOME)/.claude/skills/$(APP)
MAIN_PKG := ./cmd/ledger
GO ?= go
LDFLAGS := -s -w
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64

.PHONY: build build-cross clean release-package skill-package skill-package-cross skill-install-local verify-package verify-release-package

build:
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP) $(MAIN_PKG)

build-cross:
	@mkdir -p $(BUILD_DIR)
	@for target in $(PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		echo "building $(APP)-$$os-$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP)-$$os-$$arch $(MAIN_PKG); \
	done

skill-package: build
	@rm -rf $(SKILL_DIST_DIR)
	@mkdir -p $(SKILL_DIST_DIR)/script $(SKILL_DIST_DIR)/example
	cp skill/SKILL.md $(SKILL_DIST_DIR)/SKILL.md
	cp -R skill/example/. $(SKILL_DIST_DIR)/example/
	cp $(BUILD_DIR)/$(APP) $(SKILL_DIST_DIR)/script/$(APP)

skill-package-cross: build-cross
	@rm -rf $(PACKAGE_DIR) $(RELEASE_DIR)
	@mkdir -p $(PACKAGE_DIR) $(RELEASE_DIR)
	@for target in $(PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		staging="$(PACKAGE_DIR)/$(APP)-$$os-$$arch"; \
		pkg="$$staging/$(APP)"; \
		echo "packaging $(APP)-$$os-$$arch"; \
		rm -rf "$$staging"; \
		mkdir -p "$$pkg/script" "$$pkg/example"; \
		cp skill/SKILL.md "$$pkg/SKILL.md"; \
		cp -R skill/example/. "$$pkg/example/"; \
		cp "$(BUILD_DIR)/$(APP)-$$os-$$arch" "$$pkg/script/$(APP)"; \
		tar -czf "$(RELEASE_DIR)/$(APP)-$$os-$$arch.tar.gz" -C "$$staging" "$(APP)"; \
	done

release-package: skill-package-cross
	cd $(RELEASE_DIR) && shasum -a 256 *.tar.gz > checksums.txt

verify-package: skill-package
	test -f $(SKILL_DIST_DIR)/SKILL.md
	test -f $(SKILL_DIST_DIR)/script/$(APP)
	test -d $(SKILL_DIST_DIR)/example

verify-release-package: release-package
	test -f $(RELEASE_DIR)/checksums.txt
	@for target in $(PLATFORMS); do \
		os=$${target%/*}; \
		arch=$${target#*/}; \
		test -f "$(RELEASE_DIR)/$(APP)-$$os-$$arch.tar.gz"; \
	done

skill-install-local: skill-package
	rm -rf "$(SKILL_INSTALL_DIR)"
	@mkdir -p "$(SKILL_INSTALL_DIR)/script" "$(SKILL_INSTALL_DIR)/example"
	cp $(SKILL_DIST_DIR)/SKILL.md "$(SKILL_INSTALL_DIR)/SKILL.md"
	rsync -a --delete $(SKILL_DIST_DIR)/example/ "$(SKILL_INSTALL_DIR)/example/"
	cp $(SKILL_DIST_DIR)/script/$(APP) "$(SKILL_INSTALL_DIR)/script/$(APP)"

clean:
	rm -rf $(DIST_DIR)
