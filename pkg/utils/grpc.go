package utils

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

func IsGrpcPort(target string) bool {
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer dialCancel()

	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return false
	}
	conn.Connect()
	return conn.WaitForStateChange(dialCtx, connectivity.Ready)
}
