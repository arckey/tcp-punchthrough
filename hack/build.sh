# build client
# go build -v -o ./client/client ./client/.

# build server
# go build -v -o ./server/server ./server/.
# compile flatbuffers
flatc -o types -I flatbuffers --go flatbuffers/*.fbs

# build peer
go build -v -o ./peer/peer ./peer/.

# build negotiator
go build -v -o ./negotiator/negotiator ./negotiator/.

