# build client
go build -v -o ./client/client ./client/.

# build server
go build -v -o ./server/server ./server/.

# build negotiator
go build -v -o ./negotiator/negotiator ./negotiator/.