package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

type MetricSummary struct {
	Min   float64
	Max   float64
	Sum   float64
	Count uint64
}

func createKafkaRoundTripper(e ServiceConfig) kafka.RoundTripper {
	var saslMechanism sasl.Mechanism
	var tlsConfig *tls.Config

	switch strings.ToUpper(e.KafkaSaslMechanism) {
	case "PLAIN":
		saslMechanism = plain.Mechanism{
			Username: e.KafkaSaslUsername,
			Password: e.KafkaSaslPassword,
		}
	case "SCRAM-SHA-256":
		mechanism, err := scram.Mechanism(scram.SHA256, e.KafkaSaslUsername, e.KafkaSaslPassword)
		if err != nil {
			log.Fatal("unable to create scram-sha-256 mechanism", err)
			return nil
		}

		saslMechanism = mechanism
	case "SCRAM-SHA-512":
		mechanism, err := scram.Mechanism(scram.SHA512, e.KafkaSaslUsername, e.KafkaSaslPassword)
		if err != nil {
			log.Fatal("unable to create scram-sha-512 mechanism", err)
			return nil
		}

		saslMechanism = mechanism
	}

	if e.KafkaSslCertPath != "" && e.KafkaSslKeyPath != "" {
		certs, err := tls.LoadX509KeyPair(e.KafkaSslCertPath, e.KafkaSslKeyPath)
		if err != nil {
			log.Fatal("unable to load certificate key pair", err)
			return nil
		}

		caCertificatePool, err := x509.SystemCertPool()
		if err != nil {
			caCertificatePool = x509.NewCertPool()
		}
		if e.KafkaSslCaPath != "" {
			caFile, err := os.ReadFile(e.KafkaSslCaPath)
			if err != nil {
				log.Fatal("unable to read ca file", err)
				return nil
			}

			if ok := caCertificatePool.AppendCertsFromPEM(caFile); !ok {
				log.Fatal("unable to append ca certificate to pool")
				return nil
			}
		}

		tlsConfig = &tls.Config{
			RootCAs:      caCertificatePool,
			Certificates: []tls.Certificate{certs},
		}
	}

	return &kafka.Transport{
		SASL: saslMechanism,
		TLS:  tlsConfig,
		Dial: (&net.Dialer{
			Timeout:   3 * time.Second,
			DualStack: true,
		}).DialContext,
	}
}
