FROM golang:latest
WORKDIR /app
COPY . .
RUN chmod +x goose_migrations_up.sh
RUN go mod download
RUN git clone https://github.com/vishnubob/wait-for-it.git