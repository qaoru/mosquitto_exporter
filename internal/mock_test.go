package internal

import (
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// mockToken implements mqtt.Token for testing. Wait() always reports the
// operation as complete; Error() returns the configured error (nil by default
// for the success path).
type mockToken struct {
	err error
}

func (t *mockToken) Wait() bool                       { return true }
func (t *mockToken) WaitTimeout(_ time.Duration) bool { return true }
func (t *mockToken) Done() <-chan struct{}            { return nil }
func (t *mockToken) Error() error                     { return t.err }

// mockClient implements mqtt.Client for exercising the collectors' Subscribe
// methods without a real broker. It records the topics subscribed and the
// handler registered for each, and can be configured to return an error from
// Subscribe to test the subscription-error counting path.
type mockClient struct {
	mu            sync.Mutex
	subscriptions map[string]mqtt.MessageHandler
	subscribeErr  error
}

func newMockClient() *mockClient {
	return &mockClient{subscriptions: make(map[string]mqtt.MessageHandler)}
}

// withSubscribeError configures the next Subscribe calls to fail with err.
func (c *mockClient) withSubscribeError(err error) *mockClient {
	c.subscribeErr = err
	return c
}

// subscribedTopics returns a snapshot of the topics Subscribe was called with.
func (c *mockClient) subscribedTopics() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	topics := make([]string, 0, len(c.subscriptions))
	for t := range c.subscriptions {
		topics = append(topics, t)
	}
	return topics
}

// handlerFor returns the message handler registered for a topic, or nil.
func (c *mockClient) handlerFor(topic string) mqtt.MessageHandler {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.subscriptions[topic]
}

func (c *mockClient) Subscribe(topic string, _ byte, callback mqtt.MessageHandler) mqtt.Token {
	c.mu.Lock()
	c.subscriptions[topic] = callback
	c.mu.Unlock()
	return &mockToken{err: c.subscribeErr}
}

func (c *mockClient) IsConnected() bool      { return true }
func (c *mockClient) IsConnectionOpen() bool { return true }
func (c *mockClient) Connect() mqtt.Token    { return &mockToken{} }
func (c *mockClient) Disconnect(_ uint)      {}
func (c *mockClient) Publish(_ string, _ byte, _ bool, _ interface{}) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) SubscribeMultiple(_ map[string]byte, _ mqtt.MessageHandler) mqtt.Token {
	return &mockToken{}
}
func (c *mockClient) Unsubscribe(_ ...string) mqtt.Token       { return &mockToken{} }
func (c *mockClient) AddRoute(_ string, _ mqtt.MessageHandler) {}
func (c *mockClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.ClientOptionsReader{}
}
