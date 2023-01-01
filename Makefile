# ==============================================================================
# Tools

proto_update:
	buf mod update network/proto && buf mod update node/proto

proto_gen:
	buf generate

proto_lint:
	buf lint