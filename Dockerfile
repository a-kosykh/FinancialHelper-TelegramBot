FROM golang:alpine
WORKDIR /app
COPY . .
RUN go mod download
RUN mkdir bin
RUN go build -o ./bin/bot gitlab.ozon.dev/akosykh114/telegram-bot/cmd/bot
CMD ["/app/bin/bot"]