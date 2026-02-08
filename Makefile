.PHONY: help test lint clean build run docker-build swagger install-swag

notify:
	@curl -X POST http://localhost:8088/notify \
	-H "X-Api-Key: secret_key" \
	-H "Content-Type: application/json" \
	-d '{ \
		"title": "Тестовое уведомление", \
		"message": "Привет! Это проверка работы API сервера." \
	}'
