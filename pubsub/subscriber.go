package pubsub

import (
    "sync"
)

type Subscribers map[string]*Subscriber

type Subscriber struct {
    id        string
    messages  chan *Message
    createAt  int64
    destroyed bool
    lock      *sync.RWMutex
    topics    map[string]bool
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
    s.lock.RLock()
    defer s.lock.RUnlock()
    if !s.destroyed {
        s.messages <- m
    }
    return s
}

// to close the underlying channels/resources
func (s *Subscriber) Destroy() {
    s.lock.Lock()
    defer s.lock.Unlock()
    if !s.destroyed {
        s.destroyed = true
        close(s.messages)
    }
    
}
