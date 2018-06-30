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

	"github.com/Oppodelldog/tig-test/ccount"
)

func main() {

	s, r, err := startServer()
	if err != nil {
		panic(err)
	}

	end := time.NewTimer(time.Second * 2)
	ticker := time.NewTicker(time.Millisecond * 100)
	tickerIncreaseConnections := time.NewTicker(time.Second * 10)
	tickerResetConnections := time.NewTicker(time.Second * 60)
	minConcurrentConnections := 1
	isRunning := true
	for isRunning {
		select {
		case <-tickerResetConnections.C:
			minConcurrentConnections = 1
		case <-tickerIncreaseConnections.C:
			minConcurrentConnections += rand.Intn(10+1) + 1
		case <-ticker.C:

			for i := 0; i < rand.Intn(10+minConcurrentConnections)+minConcurrentConnections; i++ {
				go makeRequest()
			}

		case <-end.C:
			isRunning = false
		}
	}

	r.UnregisterAll()
	stopServer(s)

}

func makeRequest() {
	resp, err := http.Get("http://localhost:10012/")
	if err != nil {
		logrus.Error(err)
	}
	fmt.Println(resp.StatusCode)
	resp.Body.Close()
}

func stopServer(s *http.Server) {
	ctx,_ := context.WithTimeout(context.Background(), time.Second*30)
	err := s.Shutdown(ctx)
	if err != nil {
		panic(err)
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

func startServer() (*http.Server, metrics.Registry, error) {

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

	counter := ccount.NewConcurrentCounter()
	err = metricsRegistry.Register("api-method-1234-concurrent", counter)
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

	return s, metricsRegistry, err
}
