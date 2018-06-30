package main

import (
	"time"
	"net/http"
	"github.com/sirupsen/logrus"
	"github.com/rcrowley/go-metrics"
	"github.com/vrischmann/go-metrics-influxdb"
	"context"
	"fmt"
	"math/rand"
)

func main() {

	s, err := startServer()
	if err != nil {
		panic(err)
	}

	end := time.NewTimer(time.Second * 600)
	ticker := time.NewTicker(time.Millisecond * 100)
	isRunning := true
	for isRunning {
		select {
		case <-ticker.C:
			resp, err := http.Get("http://localhost:10012/")
			if err != nil {
				logrus.Error(err)
			}
			fmt.Println(resp.StatusCode)
			resp.Body.Close()

		case <-end.C:
			isRunning = false
		}
	}

	stopServer(s)

}

func stopServer(s *http.Server) {
	ctx := context.Background()
	err := s.Shutdown(ctx)
	if err != nil {
		panic(err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func startMonitoring() metrics.Registry {

	r := metrics.NewRegistry()

	go influxdb.InfluxDB(
		r,
		time.Second*3,
		"http://localhost:8086",
		"mydb",
		"",
		"",
	)

	return r
}

func startServer() (*http.Server, error) {

	metricsRegistry := startMonitoring()
	serverMux := http.NewServeMux()

	s := &http.Server{Addr: "0.0.0.0:10012", Handler: serverMux}

	metrics.RegisterDebugGCStats(metricsRegistry)
	metrics.RegisterRuntimeMemStats(metricsRegistry)

	go metrics.CaptureDebugGCStats(metricsRegistry, time.Second*5)
	go metrics.CaptureRuntimeMemStats(metricsRegistry, time.Second*5)

	as := metrics.NewExpDecaySample(100, 0.99)
	//s := metrics.NewUniformSample(1028)
	sample := metrics.NewHistogram(as)

	//sample := metrics.NewTimer()
	//sample := metrics.NewGauge()

	err := metricsRegistry.Register("api-method-1234-histogramm", sample)
	if err != nil {
		panic(err)
	}

	counter := metrics.NewCounter()
	err = metricsRegistry.Register("api-method-1234-counter", counter)
	if err != nil {
		panic(err)
	}

	serverMux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Inc(1)
		timeStart := time.Now()

		time.Sleep(time.Nanosecond * time.Duration(rand.Intn(10000)+100))

		w.WriteHeader(204)

		duration := time.Since(timeStart)
		sample.Update(duration.Nanoseconds())
		counter.Dec(1)
	}))

	go s.ListenAndServe()

	return s, err
}
