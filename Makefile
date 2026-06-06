.PHONY: server server-stop macos macos-stop

# 后端服务
server:
	cd server && go build -o /tmp/im-server ./cmd/im-server/
	/tmp/im-server --config server/config/config.yaml &
	@echo "Server started (PID: $$!)"

server-stop:
	@pkill -f "/tmp/im-server" || true
	@echo "Server stopped"

APP_BUNDLE = client/.build/debug/IMApp-macOS.app

# macOS 客户端
macos:
	cd client && swift build --target IMApp-macOS
	mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	cp client/.build/debug/IMApp-macOS "$(APP_BUNDLE)/Contents/MacOS/"
	plutil -replace CFBundleExecutable -string IMApp-macOS "$(APP_BUNDLE)/Contents/Info.plist" 2>/dev/null; \
	plutil -replace CFBundleIdentifier -string com.dolphinz.imapp "$(APP_BUNDLE)/Contents/Info.plist" 2>/dev/null; \
	plutil -replace CFBundleName -string "DolphinZ IM" "$(APP_BUNDLE)/Contents/Info.plist" 2>/dev/null; \
	plutil -replace CFBundleVersion -string 1 "$(APP_BUNDLE)/Contents/Info.plist" 2>/dev/null; \
	plutil -replace CFBundlePackageType -string APPL "$(APP_BUNDLE)/Contents/Info.plist" 2>/dev/null; \
	plutil -replace LSMinimumSystemVersion -string "15.0" "$(APP_BUNDLE)/Contents/Info.plist" 2>/dev/null; \
	true
	open "$(APP_BUNDLE)"

macos-stop:
	pkill -f "IMApp-macOS" || true
	@echo "App closed"
