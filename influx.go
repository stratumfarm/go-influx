package influx

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

const forceChanLen = 30

//Config represents config values stored in json
type Config struct {
	Endpoint      string `json:"endpoint"`
	Database      string `json:"database"`
	User          string `json:"user"`
	Password      string `json:"password"`
	Host          string `json:"host"`
	Label         string `json:"label"`
	BatchInterval string `json:"batch_interval"`
	BatchCount    int    `json:"batch_count"`
	WorkerCount   int    `json:"worker_count"`
	Precision     string `json:"precision"`
}

//Writer accept messages and write them to influx in the background
type Writer struct {
	wg            sync.WaitGroup
	client        client.Client
	label         string
	database      string
	host          string
	messageCh     chan interface{}
	BatchInterval time.Duration
	BatchCount    int
	Precision     string
}

//NewWriter creates a new writer from config
func NewWriter(cfg Config) (*Writer, error) {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     cfg.Endpoint,
		Username: cfg.User,
		Password: cfg.Password,
	})
	if err != nil {
		return nil, err
	}
	//We should check precision, because it's the only reason to fail for newBatch
	if _, err := time.ParseDuration("1" + cfg.Precision); err != nil {
		log.Panicf("Can't parse Precision `%s`: %v", cfg.Precision, err)
	}

	w := &Writer{
		client:        c,
		database:      cfg.Database,
		label:         cfg.Label,
		host:          cfg.Host,
		messageCh:     make(chan interface{}, cfg.BatchCount+100), //TODO 100?
		BatchInterval: mustParseDuration(cfg.BatchInterval),
		BatchCount:    cfg.BatchCount,
		Precision:     cfg.Precision,
	}
	if cfg.WorkerCount < 1 {
		cfg.WorkerCount = 1
	}
	w.wg.Add(cfg.WorkerCount)
	for i := 0; i < cfg.WorkerCount; i++ {
		go w.worker()
	}
	return w, nil
}

//Close sends the rest of the messages and closes client
func (s *Writer) Close() error {
	close(s.messageCh)
	s.wg.Wait() //let's send the rest
	return s.client.Close()
}

//WriteSample writes with given probability
func (s *Writer) WriteSample(p interface{}, prob float64) {
	if rand.Float64() < prob {
		s.Write(p)
	}
}

//Write accepts metric and put it to the queue to write
func (s *Writer) Write(p interface{}) {
	if p == nil {
		return
	}
	if len(s.messageCh) >= cap(s.messageCh) {
		log.Printf("[WARN] Discarded influx message, queue is full %d", len(s.messageCh))
		return
	}

	select {
	case s.messageCh <- p:
	default:
		log.Printf("[ERROR] Discarded influx message, len didn't protect, queue is full %d", len(s.messageCh))

	}
}

func (s *Writer) worker() {
	defer s.wg.Done()
	batch := newBatch(s.database, s.Precision)

	tags := map[string]string{
		"label": s.label,
		"host":  s.host,
	}

	forceWriteChan := make(chan bool, forceChanLen)
	go func() {
		for {
			time.Sleep(s.BatchInterval)
			forceWriteChan <- true
		}
	}()

	count := 0

	for {
		select {
		case m, ok := <-s.messageCh:
			if !ok {
				if err := s.client.Write(batch); err != nil {
					log.Printf("[ERROR] Can't write to influx %v", err)
				}
				return
			}
			count += s.processMessage(m, batch, tags)
			if count > s.BatchCount {
				select {
				case forceWriteChan <- true:
				default:
				}
			}
		case <-forceWriteChan:
			if count == 0 {
				continue
			}

			if err := s.client.Write(batch); err != nil {
				log.Printf("[ERROR] Can't write to influx %v", err)
			}
			count = 0
			batch = newBatch(s.database, s.Precision) // Error is impossible here, because it's only if parsing is failed, but we already did it
		}
	}
}

func (s *Writer) processMessage(msg interface{}, batch client.BatchPoints, tags map[string]string) int {
	ret := 0

	switch d := msg.(type) {
	case *Metric:
		newPoint(tags, batch, *d)
		ret++
	case Metric:
		newPoint(tags, batch, d)
		ret++
	case []Metric:
		for _, m := range d {
			newPoint(tags, batch, m)
			ret++
		}
	default:
		log.Printf("[NEVER] Don't know how to cast metric, type: %T", msg)
	}
	return ret
}

func newPoint(commonTags map[string]string, batch client.BatchPoints, m Metric) {
	point, err := client.NewPoint(m.Measurement(), mergeTags(m.Tags(), commonTags), m.Values(), m.Time())
	if err != nil {
		log.Printf("[ERROR] Can't create new point %v %v", m, err)
		return
	}
	batch.AddPoint(point)
}

func newBatch(database, precision string) client.BatchPoints {
	c, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: precision,
	})
	return c
}
func mergeTags(tags, commonTags map[string]string) map[string]string {
	if tags == nil {
		return commonTags
	}
	for k, v := range commonTags {
		tags[k] = v
	}
	return tags
}
