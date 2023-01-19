package occurrence

import (
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

func GenerateKafkaMessageBatch(occurrences []Occurrence) ([]kafka.Message, error) {
	messages := make([]kafka.Message, 0, len(occurrences))
	for _, o := range occurrences {
		b, err := json.Marshal(o)
		if err != nil {
			return nil, err
		}
		messages = append(messages, kafka.Message{
			Value: b,
		})
	}
	return messages, nil
}
