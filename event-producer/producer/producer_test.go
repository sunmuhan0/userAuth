package producer

import (
	"context"
	"testing"
)

// mockPublisher 模拟IRmqPublisher用于测试
type mockPublisher struct {
	messages []publishRecord
}

type publishRecord struct {
	topic   string
	tag     string
	payload interface{}
}

func (m *mockPublisher) Publish(ctx context.Context, topic string, tag string, payload interface{}) error {
	m.messages = append(m.messages, publishRecord{topic, tag, payload})
	return nil
}

func (m *mockPublisher) PublishBatch(ctx context.Context, topic string, tag string, payloads []interface{}) error {
	for _, p := range payloads {
		m.messages = append(m.messages, publishRecord{topic, tag, p})
	}
	return nil
}

func TestMockPublish(t *testing.T) {
	pub := &mockPublisher{}
	ctx := context.Background()

	if err := pub.Publish(ctx, "TestTopic", "test", "hello"); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if len(pub.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(pub.messages))
	}
	if pub.messages[0].topic != "TestTopic" {
		t.Fatalf("expected topic=TestTopic, got %s", pub.messages[0].topic)
	}
	if pub.messages[0].tag != "test" {
		t.Fatalf("expected tag=test, got %s", pub.messages[0].tag)
	}
}

func TestMockPublishBatch(t *testing.T) {
	pub := &mockPublisher{}
	ctx := context.Background()

	payloads := []interface{}{"msg1", "msg2", "msg3"}
	if err := pub.PublishBatch(ctx, "BatchTopic", "batch", payloads); err != nil {
		t.Fatalf("PublishBatch failed: %v", err)
	}
	if len(pub.messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(pub.messages))
	}
	for i, m := range pub.messages {
		if m.topic != "BatchTopic" {
			t.Fatalf("msg[%d]: expected topic=BatchTopic, got %s", i, m.topic)
		}
		if m.tag != "batch" {
			t.Fatalf("msg[%d]: expected tag=batch, got %s", i, m.tag)
		}
	}
}

func TestMockPublishBatchEmpty(t *testing.T) {
	pub := &mockPublisher{}
	ctx := context.Background()

	if err := pub.PublishBatch(ctx, "Topic", "tag", []interface{}{}); err != nil {
		t.Fatalf("PublishBatch with empty payloads should not error: %v", err)
	}
	if len(pub.messages) != 0 {
		t.Fatalf("expected 0 messages for empty batch, got %d", len(pub.messages))
	}
}

// TestIRmqPublisherInterface 验证接口兼容性
func TestIRmqPublisherInterface(t *testing.T) {
	var pub IRmqPublisher = &mockPublisher{}
	ctx := context.Background()

	if err := pub.Publish(ctx, "t", "g", "data"); err != nil {
		t.Fatalf("Publish via interface failed: %v", err)
	}
	if err := pub.PublishBatch(ctx, "t", "g", []interface{}{"a", "b"}); err != nil {
		t.Fatalf("PublishBatch via interface failed: %v", err)
	}
}
