package grpcconnection

import (
	"crypto/tls"
	"fmt"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func CreateGRPCConn(host, port string) (*grpc.ClientConn, error) {
	addr := fmt.Sprintf("%s:%s", host, port)

	var opts grpc.DialOption
	if host == "" || host == "localhost" || host == "127.0.0.1" {
		// Локальное соединение — без TLS
		opts = grpc.WithTransportCredentials(insecure.NewCredentials())
		log.Println("gRPC: using insecure connection (localhost)")
	} else {
		// Удаленное соединение — через TLS
		tlsConfig := &tls.Config{} // По умолчанию система использует доверенные CA
		creds := credentials.NewTLS(tlsConfig)
		opts = grpc.WithTransportCredentials(creds)
		log.Println("gRPC: using secure TLS connection")
	}

	return grpc.Dial(addr, opts)
}
