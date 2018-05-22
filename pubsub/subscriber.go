package pubsub

import (
//"sync"
)

type Subscribers map[string]*Subscriber

type Subscriber struct {
	id        string
	messages  chan *Message
	createAt  int64
	destroyed bool
	//lock      *sync.RWMutex
	topics  map[string]bool
	closing chan bool
}

// to get the subscriber id
func (s *Subscriber) GetID() string {
	return s.id
}

// to get the subscriber topics
func (s *Subscriber) GetTopics() []string {
	topics := []string{}
	for topic, _ := range s.topics {
		topics = append(topics, topic)
	}
	return topics
}

// return a channel of *Message to listen on
func (s *Subscriber) GetMessages() <-chan *Message {
	return s.messages
}

// to send a message to subscriber
func (s *Subscriber) Signal(m *Message) *Subscriber {
	// s.lock.RLock()
	// defer s.lock.RUnlock()
	defer func() {
		// recoverint from panic caused by writing to a closed channel
		if recover() == nil {
			return
		}
	}()

OK:
	for {
		select {
		case <-s.closing:
			break OK
		case s.messages <- m:
			break OK
		default:
		}
		if s.destroyed {
			break
		}
	}

	// if !s.destroyed {
	// 	select {
	// 	case <-s.closing:
	// 		return s
	// 	default:
	// 		s.messages <- m
	// 	}
	// 	// s.messages <- m
	// }
	return s
}

// to close the underlying channels/resources
func (s *Subscriber) Destroy() {
	// s.lock.Lock()
	// defer s.lock.Unlock()
	if !s.destroyed {
		s.destroyed = true
		s.closing <- true
		close(s.messages)
		close(s.closing)
	}
}

func (s *Subscriber) Closing() <-chan bool {
	return s.closing
}
