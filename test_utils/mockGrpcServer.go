package test_utils

import (
	"log"

	bankv1beta1 "cosmossdk.io/api/cosmos/bank/v1beta1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func SetupMockGrpcServer() *grpc.Server {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	bankv1beta1.RegisterQueryServer(s, &mockBankEndpoints{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Printf("Mock Server exited with error: %v", err)
		}
	}()
	return s
}
