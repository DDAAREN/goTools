package data

import (
	"github.com/astaxie/beego"
	cluster "github.com/bsm/sarama-cluster"
)

type Kconsumer struct {
	consumer *cluster.Consumer
	messages chan []byte
	signal   chan int
}

func NewKconsumer(topic, groupName string) (*Kconsumer, error) {
	brokers := beego.AppConfig.Strings("kfkHost")
	topics := []string{topic}
	config := cluster.NewConfig()
	config.Consumer.Return.Errors = true
	config.Group.Mode = cluster.ConsumerModePartitions
	//config.Consumer.Offsets.Retention = time.Minute * 5 // 设置offset信息的有效期
	consumer, err := cluster.NewConsumer(brokers, groupName, topics, config)
	if err != nil {
		return nil, err
	}
	messages := make(chan []byte, 10)
	signal := make(chan int)
	kconsumer := &Kconsumer{
		consumer: consumer,
		messages: messages,
		signal:   signal,
	}
	// join message loop
	go func(k *Kconsumer) {
		for {
			select {
			case part, ok := <-k.consumer.Partitions():
				if !ok {
					return
				}
				go func(pc cluster.PartitionConsumer) {
					for msg := range pc.Messages() {
						k.messages <- msg.Value
						k.consumer.MarkOffset(msg, "")
					}
				}(part)
			case <-k.signal:
				k.consumer.Close()
				return
			}
		}
	}(kconsumer)

	return kconsumer, nil
}

func (this *Kconsumer) Messages() <-chan []byte {
	return this.messages
}

func (this *Kconsumer) Close() {
	this.signal <- 1
}
