package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/twmb/franz-go/pkg/kgo"
	awsiam "github.com/twmb/franz-go/pkg/sasl/aws"
)

// KafkaConfig holds configuration for Kafka operations.
type KafkaConfig struct {
	Brokers       []string
	Topic         string
	Auth          string // "iam" or "none"
	UseTLS        bool
	Acks          int16
	Group         string
	FromBeginning bool
}

// Record represents a Kafka record.
type Record struct {
	Topic     string
	Key       []byte
	Value     []byte
	Partition int32
	Offset    int64
}

// KafkaService provides methods to interact with Kafka.
type KafkaService struct {
	logger *slog.Logger
	cfg    aws.Config
}

// NewKafkaService creates a new KafkaService.
func NewKafkaService(cfg aws.Config, logger *slog.Logger) *KafkaService {
	return &KafkaService{
		cfg:    cfg,
		logger: logger,
	}
}

// Produce sends a message to the specified Kafka topic.
func (s *KafkaService) Produce(ctx context.Context, kcfg KafkaConfig, key, value []byte) error {
	opts, err := s.getClientOptions(ctx, kcfg)
	if err != nil {
		return err
	}

	if kcfg.Acks != 0 {
		var acks kgo.Acks

		switch kcfg.Acks {
		case 1:
			acks = kgo.LeaderAck()
		case 0:
			acks = kgo.NoAck()
		default:
			// Fallback to default if unknown value is provided
			acks = kgo.AllISRAcks()
		}

		opts = append(opts, kgo.RequiredAcks(acks))
	}

	opts = append(opts, kgo.WithLogger(&kgoLogger{s: s}))

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return fmt.Errorf("creating kafka client: %w", err)
	}
	defer client.Close()

	record := &kgo.Record{
		Topic: kcfg.Topic,
		Key:   key,
		Value: value,
	}

	if err := client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("producing message: %w", err)
	}

	s.logger.Info("message produced", "topic", kcfg.Topic, "partition", record.Partition, "offset", record.Offset)

	return nil
}

type kgoLogger struct {
	s *KafkaService
}

func (*kgoLogger) Level() kgo.LogLevel {
	return kgo.LogLevelDebug
}

func (l *kgoLogger) Log(level kgo.LogLevel, msg string, keyvals ...any) {
	switch level {
	case kgo.LogLevelError:
		l.s.logger.Error(msg, keyvals...)
	case kgo.LogLevelWarn:
		l.s.logger.Warn(msg, keyvals...)
	case kgo.LogLevelInfo:
		l.s.logger.Info(msg, keyvals...)
	default:
		l.s.logger.Debug(msg, keyvals...)
	}
}

// Consume reads messages from the specified Kafka topic.
func (s *KafkaService) Consume(
	ctx context.Context,
	kcfg KafkaConfig,
	callback func(*Record),
) error {
	opts, err := s.getClientOptions(ctx, kcfg)
	if err != nil {
		return err
	}

	opts = append(opts, kgo.ConsumeTopics(kcfg.Topic))

	if kcfg.Group != "" {
		opts = append(opts, kgo.ConsumerGroup(kcfg.Group))
	}

	if kcfg.FromBeginning {
		opts = append(opts, kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return fmt.Errorf("creating kafka client: %w", err)
	}
	defer client.Close()

	for {
		fetches := client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			// Check if context was cancelled
			if ctx.Err() != nil {
				return nil
			}

			return fmt.Errorf("polling fetches: %v", errs)
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			callback(&Record{
				Topic:     record.Topic,
				Key:       record.Key,
				Value:     record.Value,
				Partition: record.Partition,
				Offset:    record.Offset,
			})
		}
	}
}

func (s *KafkaService) getClientOptions(ctx context.Context, kcfg KafkaConfig) ([]kgo.Opt, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(kcfg.Brokers...),
	}

	if kcfg.Auth == "iam" {
		creds, err := s.cfg.Credentials.Retrieve(ctx)
		if err != nil {
			return nil, fmt.Errorf("retrieving aws credentials: %w", err)
		}

		auth := awsiam.Auth{
			AccessKey:    creds.AccessKeyID,
			SecretKey:    creds.SecretAccessKey,
			SessionToken: creds.SessionToken,
		}
		opts = append(opts, kgo.SASL(auth.AsManagedStreamingIAMMechanism()))
	}

	if kcfg.UseTLS || kcfg.Auth == "iam" {
		// MSK IAM requires TLS
		opts = append(opts, kgo.DialTLSConfig(new(tls.Config)))
	}

	return opts, nil
}
