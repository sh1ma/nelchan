FROM golang:1.25.5-bookworm

WORKDIR /app


ENV ENV=production



COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -o nelchan cmd/main.go

CMD ["./nelchan"]