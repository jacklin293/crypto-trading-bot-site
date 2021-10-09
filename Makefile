run:
	go build && ./crypto-trading-bot-api
deploy:
	env GOOS=linux GOARCH=amd64 go build -o prod-api
	scp prod-api fomobot:~/app/api/
	rm prod-api
	rsync -av -e ssh view fomobot:/home/fomobot/app/api/
	rsync -av -e ssh assets fomobot:/home/fomobot/app/api/
