PLAYGROUND_GROUP := /tmp/.dolphin_playground

.PHONY: playground clean

# 启动 playground: 创建 Werewolf 群组，两个 Bot 加入并发送 hello world
playground:
	@echo "=== Dolphin Playground ==="
	@echo "确保 Server 已在运行 (make server)"
	@echo ""
	@python3 playground_setup.py

# 清理 playground 临时文件
clean:
	rm -f $(PLAYGROUND_GROUP)
