package config

import (
	"crypto/tls"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"os"
	"strings"
)

var c client.Client

// MustGetClient returns a client using the singleton pattern pulling configuration from environment variables. Panics
// if the client cannot be acquired. Clients are heavyweight and ideally created once per process so this is a
// convenient way to access the client consistently and enforce orthogonal concerns such as codecs, logging, etc.
func MustGetClient() client.Client {
	if c != nil {
		return c
	}
	clientKeyPath := os.Getenv("TEMPORAL_CLIENT_KEY_PATH")
	clientCertPath := os.Getenv("TEMPORAL_CLIENT_CERT_PATH")
	hostPort := os.Getenv("TEMPORAL_CLIENT_HOSTPORT")
	namespace := os.Getenv("TEMPORAL_CLIENT_NAMESPACE")
	if strings.TrimSpace(clientKeyPath) == "" {
		panic("TEMPORAL_CLIENT_KEY_PATH required and missing")
	}
	if strings.TrimSpace(clientCertPath) == "" {
		panic("TEMPORAL_CLIENT_CERT_PATH required and missing")
	}
	if strings.TrimSpace(hostPort) == "" {
		panic("TEMPORAL_CLIENT_HOSTPORT required and missing")
	}
	if strings.TrimSpace(namespace) == "" {
		panic("TEMPORAL_CLIENT_NAMESPACE required and missing")
	}
	// Use the crypto/tls package to create a cert object
	cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
	if err != nil {
		panic(err)
	}
	clientOptions := client.Options{
		HostPort:  hostPort,
		Namespace: namespace,
		ConnectionOptions: client.ConnectionOptions{
			TLS: &tls.Config{Certificates: []tls.Certificate{cert}},
		},
		DataConverter: converter.GetDefaultDataConverter(),
	}
	c, err := client.Dial(clientOptions)
	if err != nil {
		panic(err)
	}
	return c
}
