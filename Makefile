.PHONY: server server-stop macos macos-stop ios-deploy xcodegen

# 后端服务
server: server-stop
	cd server && go build -o /tmp/im-server ./cmd/im-server/
	cd server && /tmp/im-server --config config/config.yaml &
	@echo "Server started"

server-stop:
	@pkill -f "/tmp/im-server" 2>/dev/null || true
	@echo "Server stopped"

APP_BUNDLE = client/.build/debug/IMApp-macOS.app

# macOS 客户端
macos: macos-stop
	cd client && swift build --product IMApp-macOS
	mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	cp client/.build/debug/IMApp-macOS "$(APP_BUNDLE)/Contents/MacOS/"
	if [ ! -f "$(APP_BUNDLE)/Contents/Info.plist" ]; then \
		echo '<?xml version="1.0" encoding="UTF-8"?>' > "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
		echo '<plist version="1.0"><dict></dict></plist>' >> "$(APP_BUNDLE)/Contents/Info.plist"; \
	fi
	plutil -replace CFBundleExecutable -string IMApp-macOS "$(APP_BUNDLE)/Contents/Info.plist" && \
	plutil -replace CFBundleIdentifier -string com.dolphinz.imapp "$(APP_BUNDLE)/Contents/Info.plist" && \
	plutil -replace CFBundleName -string "Dolphin" "$(APP_BUNDLE)/Contents/Info.plist" && \
	plutil -replace CFBundleVersion -string 1 "$(APP_BUNDLE)/Contents/Info.plist" && \
	plutil -replace CFBundlePackageType -string APPL "$(APP_BUNDLE)/Contents/Info.plist" && \
	plutil -replace LSMinimumSystemVersion -string "15.0" "$(APP_BUNDLE)/Contents/Info.plist"
	open "$(APP_BUNDLE)"

macos-stop:
	pkill -f "IMApp-macOS" 2>/dev/null || true
	@echo "App closed"

IOS_DEVICE ?= 荼靡花开

# iOS 客户端 — 一键编译部署到真机
ios-deploy: xcodegen
	cd client && xcodebuild build \
		-project IMApp.xcodeproj \
		-scheme IMApp-iOS \
		-destination 'platform=iOS,name=$(IOS_DEVICE)' \
		-allowProvisioningUpdates \
		CODE_SIGN_STYLE=Automatic
	@echo "Build succeeded for $(IOS_DEVICE)"
	# 查找编译产物并安装到设备
	DERIVED=$$(xcodebuild -project client/IMApp.xcodeproj -showBuildSettings -scheme IMApp-iOS 2>/dev/null | grep BUILD_DIR | head -1 | awk '{print $$NF}'); \
	APP="$${DERIVED}/Debug-iphoneos/IMApp-iOS.app"; \
	if [ -d "$$APP" ]; then \
		if command -v ios-deploy >/dev/null 2>&1; then \
			ios-deploy -b "$$APP"; \
		elif xcrun devicectl list devices 2>/dev/null | grep -q "$(IOS_DEVICE)"; then \
			xcrun devicectl install app --device "$(IOS_DEVICE)" "$$APP"; \
		else \
			echo "App built at: $$APP"; \
			echo "Install manually via Xcode Devices window (Cmd+Shift+2)"; \
		fi \
	else \
		echo "App not found at expected path, checking DerivedData..."; \
	fi

xcodegen:
	cd client && xcodegen generate
