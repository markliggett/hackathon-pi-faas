package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"github.com/uber/jaeger-client-go/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const topLevelSpanName = "heavyLifting"
const isPrimeArrSize = 50000000
const bvHeader = "\n\n" +
	"______                                      _          \n" +
	"| ___ \\                                    (_)         \n" +
	"| |_/ / __ _ ______ _  __ _ _ ____   _____  _  ___ ___ \n" +
	"| ___ \\/ _` |_  / _` |/ _` | '__\\ \\ / / _ \\| |/ __/ _ \\\n" +
	"| |_/ / (_| |/ / (_| | (_| | |   \\ V / (_) | | (_|  __/\n" +
	"\\____/ \\__,_/___\\__,_|\\__,_|_|    \\_/ \\___/|_|\\___\\___|\n\n"

const bvHeaderHtml = "<br><br><pre>" +
	"______                                      _          <br>" +
	"| ___ \\                                    (_)         <br>" +
	"| |_/ / __ _ ______ _  __ _ _ ____   _____  _  ___ ___ <br>" +
	"| ___ \\/ _` |_  / _` |/ _` | '__\\ \\ / / _ \\| |/ __/ _ \\<br>" +
	"| |_/ / (_| |/ / (_| | (_| | |   \\ V / (_) | | (_|  __/<br>" +
	"\\____/ \\__,_/___\\__,_|\\__,_|_|    \\_/ \\___/|_|\\___\\___|</pre><br><br>"

var (
	tracer          opentracing.Tracer
	colour          string
	heavyLiftingOps = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myfirstserverlessapp_heavyLifting_calls",
		Help: "The total number of heavyLifting calls",
	})
	heavyweightMethodAOps = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myfirstserverlessapp_heavyweightMethodA_calls",
		Help: "The total number of heavyweightMethodA calls",
	})
	heavyweightMethodBOps = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myfirstserverlessapp_heavyweightMethodB_calls",
		Help: "The total number of heavyweightMethodB calls",
	})
	heavyweightMethodCOps = promauto.NewCounter(prometheus.CounterOpts{
		Name: "myfirstserverlessapp_heavyweightMethodC_calls",
		Help: "The total number of heavyweightMethodC calls",
	})
)

func initJaeger(service string) (opentracing.Tracer, io.Closer) {
	cfg := &config.Configuration{
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
		},
	}
	tracer, closer, err := cfg.New(service, config.Logger(jaeger.StdLogger))
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	return tracer, closer
}

func handler(w http.ResponseWriter, r *http.Request) {

	fmt.Fprintf(w, "<html><body><p><font color=%s>", colour)
	fmt.Fprintf(w, bvHeaderHtml)
	fmt.Fprintf(w, "</font></p>")

	bigPrime, err := parseIn(r)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	nap, err := parseNap(r)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	processes, err := parseProcesses(r)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}

	a, b, c, err := heavyLifting(bigPrime, nap, processes)

	fmt.Fprintf(w, "<p>Method A calculated the highest prime below %d as %d<br>", bigPrime, a)
	fmt.Fprintf(w, "<p>Method B napped for %d seconds<br>", b)
	fmt.Fprintf(w, "<p>Method C calculated pi as %f using %d processes<br><br></p></body><html>", c, processes)
}

/*
 * HTTP in query param `in` - should be an integer
 */
func parseIn(r *http.Request) (int, error) {
	i, err := strconv.Atoi(r.URL.Query().Get("in"))
	if err != nil {
		return 1, errors.New("ERROR: Don't understand the given `in` value")
	}
	return i, nil
}

/*
 * HTTP in query param `nap` - should be an integer < 10
 */
func parseNap(r *http.Request) (int, error) {

	givenPause := r.URL.Query().Get("nap")
	if givenPause == "" {
		return -1, errors.New("ERROR: Can't live without my nap...")
	}

	i, err := strconv.Atoi(givenPause)
	if err != nil {
		return -1, err
	}
	if i > 10 {
		return -1, errors.New("ERROR: Can't nap longer than 10 seconds...")
	}
	return i, nil
}

/*
 * HTTP in query param `processes` - should be an integer
 */
func parseProcesses(r *http.Request) (int, error) {
	i, err := strconv.Atoi(r.URL.Query().Get("processes"))
	if err != nil {
		return -1, errors.New("ERROR: Don't understand the given `processes` value")
	}
	return i, nil
}

/*
 * Represents some computing work
 */
func heavyLifting(primesTo int, nap int, processes int) (int, int, float64, error) {

	heavyLiftingOps.Inc()
	span := tracer.StartSpan(topLevelSpanName)
	highestPrime, calculatedPi, err := heavyweightMethodA(span, primesTo, nap, processes)
	if err != nil {
		return -1, -1, -1, errors.New("error in heavyweightMethodA")
	}
	span.Finish()

	return highestPrime, nap, calculatedPi, nil
}

/*
 * Method A, does some work then delegates to method B
 */
func heavyweightMethodA(s opentracing.Span, i int, pause int, processes int) (int, float64, error) {

	heavyweightMethodAOps.Inc()
	t := tracer.StartSpan("heavyweightMethodA", opentracing.ChildOf(s.Context()))

	var x, y, n int
	var in = i - 1
	nsqrt := math.Sqrt(float64(in))

	is_prime := [isPrimeArrSize]bool{}

	for x = 1; float64(x) <= nsqrt; x++ {
		for y = 1; float64(y) <= nsqrt; y++ {
			n = 4*(x*x) + y*y
			if n <= in && (n%12 == 1 || n%12 == 5) {
				is_prime[n] = !is_prime[n]
			}
			n = 3*(x*x) + y*y
			if n <= in && n%12 == 7 {
				is_prime[n] = !is_prime[n]
			}
			n = 3*(x*x) - y*y
			if x > y && n <= in && n%12 == 11 {
				is_prime[n] = !is_prime[n]
			}
		}
	}

	for n = 5; float64(n) <= nsqrt; n++ {
		if is_prime[n] {
			for y = n * n; y < in; y += n * n {
				is_prime[y] = false
			}
		}
	}

	is_prime[2] = true
	is_prime[3] = true

	primes := make([]int, 0, 1270606)
	for x = 0; x < len(is_prime)-1; x++ {
		if is_prime[x] {
			primes = append(primes, x)
		}
	}

	// Delegtate to method B
	f := heavyweightMethodB(t, pause, processes)
	t.Finish()
	return primes[len(primes)-1], f, nil
}

/*
*  Naps for `nap` * seconds before calling method C with param `p`
 */
func heavyweightMethodB(s opentracing.Span, i int, p int) float64 {
	heavyweightMethodBOps.Inc()
	bSpan := tracer.StartSpan("heavyweightMethodB", opentracing.ChildOf(s.Context()))
	time.Sleep(time.Duration(i) * time.Second)
	// Delegate to methodC
	cResult := heavyweightMethodC(bSpan, p)
	bSpan.Finish()
	return cResult
}

/*
*  Called from method B - calculate pi using `i` concurrent processes
 */
func heavyweightMethodC(s opentracing.Span, i int) float64 {
	heavyweightMethodCOps.Inc()
	cSpan := tracer.StartSpan("heavyweightMethodC", opentracing.ChildOf(s.Context()))
	out := pi(i)
	cSpan.Finish()
	return out
}

func pi(n int) float64 {
	ch := make(chan float64)
	for k := 0; k <= n; k++ {
		go term(ch, float64(k))
	}
	f := 0.0
	for k := 0; k <= n; k++ {
		f += <-ch
	}
	return f
}

func term(ch chan float64, k float64) {
	ch <- 4 * math.Pow(-1, k) / (2*k + 1)
}

func main() {
	log.Print(bvHeader)
	log.Print("Starting MyFirstServerlessApp...")

	t, closer := initJaeger("MyFirstServerlessApp")
	tracer = t
	defer closer.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	colour = os.Getenv("COLOUR")
	if colour == "" {
		colour = "blue"
	}

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", handler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}
