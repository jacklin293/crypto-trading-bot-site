run:
	go build && ./crypto-trading-bot-api
deploy:
	env GOOS=linux GOARCH=amd64 go build -o prod-api
	rsync -av -e ssh prod-api fomobot:/home/fomobot/app/fomobot-api/
	rsync -avr --delete -e ssh view fomobot:/home/fomobot/app/fomobot-api/
	rsync -avr --delete -e ssh assets fomobot:/home/fomobot/app/fomobot-api/
	rm prod-api
	ssh -t fomobot "sudo systemctl restart fomobot-api"
