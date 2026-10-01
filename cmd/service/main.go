package main

import (
	"context"
	app "github.com/laudryfadian/griyo-backend-service-fleet/internal"
	"github.com/laudryfadian/griyo-backend-service-fleet/internal/pb"
	"github.com/laudryfadian/griyo-backend-service-fleet/internal/platform"
	"log"
)

func main() {
	ctx := context.Background()
	db, e := platform.Db(ctx)
	if e != nil {
		log.Fatal(e)
	}
	defer db.Close()
	svc, e := app.New(ctx, db)
	if e != nil {
		log.Fatal(e)
	}
	server := platform.Server()
	pb.RegisterFleetServiceServer(server, svc)
	log.Fatal(platform.ServeGrpc(server, platform.Env("GRPC_ADDR", ":50052")))
}
