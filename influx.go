package influx

import (
	"log"
	"sync"
	"time"

	"github.com/influxdata/influxdb/client/v2"
)

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

	w := &Writer{
		client:        c,
		database:      cfg.Database,
		label:         cfg.Label,
		host:          cfg.Host,
		messageCh:     make(chan interface{}, 1024),
		BatchInterval: mustParseDuration(cfg.BatchInterval),
		BatchCount:    cfg.BatchCount,
		Precision:     cfg.Precision,
	}
	batch, err := newBatch(w.database, w.Precision)
	if err != nil {
		return nil, err
	}
	w.wg.Add(1) //TODO more workers?
	go w.worker(batch)
	return w, nil
}

//Close sends the rest of the messages and closes client
func (s *Writer) Close() error {
	close(s.messageCh)
	s.wg.Wait() //let's send the rest
	return s.client.Close()
}

//Write accepts metric and put it to the queue to write
func (s *Writer) Write(p interface{}) {
	if p != nil {
		s.messageCh <- p
	}
}

func (s *Writer) worker(batch client.BatchPoints) {

	defer s.wg.Done()

	tags := map[string]string{
		"label": s.label,
		"host":  s.host,
	}

	forceWriteChan := make(chan bool, 3)
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
				forceWriteChan <- true
			}
		case <-forceWriteChan:
			if count == 0 {
				continue
			}

			if err := s.client.Write(batch); err != nil {
				log.Printf("[ERROR] Can't write to influx %v", err)
			}
			count = 0
			batch, _ = newBatch(s.database, s.Precision) // Error is impossible here, because it's only if parsing is failed, but we already did it
		}
	}
}

func (s *Writer) processMessage(msg interface{}, batch client.BatchPoints, tags map[string]string) int {
	ret := 0

	switch msg.(type) {
	case *Metric:
		newPoint(tags, batch, msg.(*Metric))
		ret++
	case []Metric:
		slice := msg.([]Metric)
		for _, m := range slice {
			newPoint(tags, batch, &m)
			ret++
		}
	default:
		log.Printf("[NEVER] Don't know how to cast metric, type: %T", msg)
	}
	return ret
}

func newPoint(commonTags map[string]string, batch client.BatchPoints, pm *Metric) {
	m := *pm
	point, err := client.NewPoint(m.Measurement(), mergeTags(m.Tags(), commonTags), m.Values(), m.Time())
	if err != nil {
		log.Printf("[ERROR] Can't create new point %v %v", m, err)
		return
	}
	batch.AddPoint(point)
}

func newBatch(database, precision string) (client.BatchPoints, error) {
	return client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: precision,
	})
}
func mergeTags(tags, commonTags map[string]string) map[string]string {
	for k, v := range commonTags {
		tags[k] = v
	}
	return tags
}
