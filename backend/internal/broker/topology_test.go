package broker

import (
	"os"
	"testing"
)

func TestDeclareTopologyRabbitMQ(t *testing.T) {
	url := os.Getenv("TEST_AMQP_URL")
	if url == "" {
		t.Skip("TEST_AMQP_URL is not set")
	}
	for i := 0; i < 2; i++ {
		if err := Initialize(url); err != nil {
			t.Fatal(err)
		}
	}
}
