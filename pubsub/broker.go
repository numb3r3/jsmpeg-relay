package pubsub

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Pubsub Broker
type Broker struct {
	subscribers Subscribers
	slock       sync.RWMutex
	topics      map[string]Subscribers
	tlock       sync.RWMutex
}

// create new broker
func NewBroker() *Broker {
	return &Broker{
		// subscribers: Subscribers{},
		slock:  sync.RWMutex{},
		topics: map[string]Subscribers{},
		tlock:  sync.RWMutex{},
	}
}

// create a new subscriber and register it into the broker
func (b *Broker) Attach() (*Subscriber, error) {
	b.slock.Lock()
	defer b.slock.Unlock()
	id := make([]byte, 50)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}

	s := &Subscriber{
		id:        hex.EncodeToString(id),
		messages:  make(chan *Message, 1),
		createAt:  time.Now().UnixNano(),
		destroyed: false,
		lock:      &sync.RWMutex{},
		topics:    map[string]bool{},
		closing:   make(chan bool, 1),
	}
	// b.subscribers[s.id] = s
	return s, nil
}

// remove the specific subscriber from the broker
func (b *Broker) Detach(s *Subscriber) {
	b.slock.Lock()
	defer b.slock.Unlock()
	s.Destroy()
	b.Unsubscribe(s, s.GetTopics()...)
}

// subscribes the specific subscriber "s" to the specific list of topic(s)
func (b *Broker) Subscribe(s *Subscriber, topics ...string) {
	b.tlock.Lock()
	defer b.tlock.Unlock()
	for _, topic := range topics {
		if nil == b.topics[topic] {
			b.topics[topic] = Subscribers{}
		}
		s.topics[topic] = true
		b.topics[topic][s.id] = s
	}
}

// unsubscribes the specific subscriber from the specific topic(s)
func (b *Broker) Unsubscribe(s *Subscriber, topics ...string) {
	b.tlock.Lock()
	defer b.tlock.Unlock()
	for _, topic := range topics {
		if nil == b.topics[topic] {
			continue
		}
		delete(b.topics[topic], s.id)
		delete(s.topics, topic)
	}
}

// broadcast the specific payload to all the topic(s) subscribers
func (b *Broker) Broadcast(data []byte, topics ...string) {
	// b.tlock.RLock()
	// defer b.tlock.RUnlock()
    now := time.Now().UnixNano()
	for _, topic := range topics {
		if nil == b.topics[topic] {
			continue
		}
        m := &Message{
            topic:    topic,
            data:     data,
            createAt: now,
        }
		for _, s := range b.topics[topic] {
			go (func(s *Subscriber) {
				s.Signal(m)
			})(s)

			// s.Signal(m)
		}
	}
}

// get the subscribers count
func (b *Broker) Subscribers(topic string) int {
	b.tlock.RLock()
	defer b.tlock.RUnlock()
	return len(b.topics[topic])
	return 0
}
