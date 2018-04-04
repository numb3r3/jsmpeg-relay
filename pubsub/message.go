package pubsub

type Message struct {
	topic    string
	data     []byte
	createAt int64
}

// to return the topic of the current message
func (m *Message) GetTopic() string {
	return m.topic
}

// to get the payload of the current message
func (m *Message) GetData() []byte {
	return m.data
}

// to get the creation time of the current message
func (m *Message) GetCreatedAt() int64 {
	return m.createAt
}
