package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	gonats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	trendv1 "github.com/shamhi/top-search/api/gen/trend/v1"
)

var searchQueries = []string{
	"iphone 15 pro max",
	"samsung galaxy s24 ultra",
	"xiaomi redmi note 13",
	"macbook pro m3 2024",
	"airpods pro 2 generation",
	"playstation 5 slim disk",
	"nike air force 1 white",
	"adidas samba og black",
	"лего звездные войны millennium falcon",
	"пазл 2000 деталей пейзаж",
	"рюкзак для ноутбука 17 дюймов",
	"кофемашина delonghi magnifica",
	"пылесос dyson v15 detect",
	"робот-пылесос xiaomi mi robot",
	"кроссовки new balance 574 classic",
	"nintendo switch oled white",
	"apple watch series 9 gps",
	"sony wh-1000xm5 black",
	"kindle paperwhite 2024",
	"монитор 27 дюймов 4k ips",
	"механическая клавиатура 75%",
	"игровое кресло ergohuman",
	"powerbank 20000mah usb-c",
	"usb-c хаб 7 в 1",
	"внешний ssd 1tb samsung",
	"фен dyson supersonic",
	"утюг philips azur",
	"швабра с отжимом vileda",
	"набор отверток профессиональный",
	"дрель аккумуляторная makita",
	"спортивная бутылка 1 литр",
	"коврик для йоги 6мм",
	"гантели разборные 20 кг",
	"умная лампа xiaomi yeelight",
	"видеорегистратор 4k",
	"автокресло детское 9-36 кг",
	"телевизор samsung 55 qled",
	"стиральная машина lg 7kg",
	"холодильник bosch serie 4",
	"микроволновка panasonic inverter",
	"электрочайник стеклянный",
	"блендер стационарный 1500w",
	"мультиварка redmond skycooker",
	"тостер на 4 тоста",
	"весы кухонные электронные",
	"кофе в зернах 1кг арабика",
	"протеин сывороточный 2кг",
	"батончик протеиновый без сахара",
	"витамин d3 5000me",
	"омега 3 рыбий жир",
}

var messySuffixes = []string{
	"!!!",
	"???",
	"!",
	" 2024",
	" скидка",
	" цена",
	" отзывы",
	" купить",
	" недорого",
}

func generateQuery() string {
	q := searchQueries[rand.Intn(len(searchQueries))]

	switch rand.Intn(8) {
	case 0:
		q = gofakeit.ProductName()
	case 1:
		car := gofakeit.Car()
		q = car.Brand + " " + car.Model
	case 2:
		q = gofakeit.BookTitle()
	case 3:
		q = gofakeit.HackerPhrase()
	case 4:
		q = gofakeit.BeerName()
	case 5:
		q = gofakeit.AppName()
	case 6:
		q = gofakeit.Color() + " " + gofakeit.Noun()
	}

	switch rand.Intn(6) {
	case 0:
		q = stringsToUpper(q)
	case 1:
		q = stringsToLower(q)
	case 2:
		q = stringsToTitle(q)
	case 3:
		q = addExtraSpaces(q)
	case 4:
		q += messySuffixes[rand.Intn(len(messySuffixes))]
	}

	return q
}

func stringsToUpper(s string) string {
	runes := []rune(s)
	for i := range runes {
		if rand.Intn(3) == 0 {
			runes[i] = upper(runes[i])
		}
	}
	return string(runes)
}

func stringsToLower(s string) string {
	runes := []rune(s)
	for i := range runes {
		runes[i] = lower(runes[i])
	}
	return string(runes)
}

func stringsToTitle(s string) string {
	runes := []rune(s)
	capacity := true
	for i := range runes {
		if capacity && isLetter(runes[i]) {
			runes[i] = upper(runes[i])
			capacity = false
		} else {
			runes[i] = lower(runes[i])
		}
		if runes[i] == ' ' {
			capacity = true
		}
	}
	return string(runes)
}

func addExtraSpaces(s string) string {
	n := rand.Intn(3) + 1
	for range n {
		s = " " + s + " "
	}
	return s
}

func upper(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return r - 32
	case r >= 'а' && r <= 'я':
		return r - 32
	}
	return r
}

func lower(r rune) rune {
	switch {
	case r >= 'A' && r <= 'Z':
		return r + 32
	case r >= 'А' && r <= 'Я':
		return r + 32
	}
	return r
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= 'а' && r <= 'я') || (r >= 'А' && r <= 'Я')
}

func main() {
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS URL")
	subject := flag.String("subject", "search.query.created", "NATS subject")
	rate := flag.Int("rate", 1000, "events per second total (0 = unlimited)")
	duration := flag.Duration("duration", 30*time.Second, "run duration (0 = forever)")
	workers := flag.Int("workers", 4, "concurrent publisher goroutines")
	batch := flag.Int("batch", 5, "events per publish call")
	silent := flag.Bool("silent", false, "suppress per-second stats")

	flag.Parse()

	if *rate < 0 {
		log.Fatal("rate must be >= 0")
	}
	if *workers <= 0 {
		log.Fatal("workers must be > 0")
	}
	if *batch <= 0 {
		log.Fatal("batch must be > 0")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *duration > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), *duration)
		defer cancel()
	}

	nc, err := gonats.Connect(
		*natsURL,
		gonats.Name("top-search-producer"),
		gonats.MaxReconnects(-1),
		gonats.ReconnectWait(time.Second),
	)
	if err != nil {
		log.Fatalf("connect nats: %v", err)
	}
	defer func() { _ = nc.Drain() }()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("create jetstream: %v", err)
	}

	var total atomic.Int64
	var errors atomic.Int64
	var wg sync.WaitGroup

	startTime := time.Now()

	var ticker *time.Ticker
	var rateLimiter <-chan time.Time
	if *rate > 0 {
		perWorkerRate := *rate / *workers
		if perWorkerRate == 0 {
			perWorkerRate = 1
		}
		interval := time.Second / time.Duration(perWorkerRate)
		ticker = time.NewTicker(interval)
		defer ticker.Stop()
	}

	for w := range *workers {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			workerC := tickerCh(ticker)

			for {
				select {
				case <-ctx.Done():
					return
				case <-workerC:
				}

				if *rate > 0 && *batch > 1 {
					<-workerC
				}

				for i := range *batch {
					if i > 0 && *rate > 0 {
						select {
						case <-ctx.Done():
							return
						case <-workerC:
						}
					}

					query := generateQuery()
					event := buildEvent(query, workerID, int(total.Load()))

					payload, err := proto.Marshal(event)
					if err != nil {
						errors.Add(1)
						continue
					}

					if _, err := js.Publish(ctx, *subject, payload); err != nil {
						errors.Add(1)
						if errors.Load()%100 == 0 {
							log.Printf("publish error (#%d): %v", errors.Load(), err)
						}
						continue
					}

					total.Add(1)
				}
			}
		}(w)
	}

	_ = rateLimiter

	if !*silent {
		wg.Add(1)
		go func() {
			defer wg.Done()

			tk := time.NewTicker(time.Second)
			defer tk.Stop()

			var prev int64
			for {
				select {
				case <-ctx.Done():
					return
				case <-tk.C:
					cur := total.Load()
					elapsed := time.Since(startTime).Seconds()
					log.Printf("sent: %d total | +%d/s | rate %.0f/s avg | errors: %d",
						cur, cur-prev, float64(cur)/elapsed, errors.Load())
					prev = cur
				}
			}
		}()
	}

	wg.Wait()

	elapsed := time.Since(startTime)
	final := total.Load()
	log.Printf("done: %d events in %v | avg %.0f/s | errors %d",
		final, elapsed.Round(time.Millisecond), float64(final)/elapsed.Seconds(), errors.Load())
}

func tickerCh(t *time.Ticker) <-chan time.Time {
	if t == nil {
		return nil
	}
	return t.C
}

func buildEvent(query string, workerID, seq int) *trendv1.SearchEvent {
	sessions := []string{"mobile-app", "web", "mobile-web"}
	platforms := []string{"ios", "android", "web"}
	locales := []string{"ru-RU", "en-US", "de-DE", "fr-FR"}
	devices := []string{"iphone-15", "pixel-8", "galaxy-s24", "macbook-m3", "desktop"}

	userID := fmt.Sprintf("usr-%d-%d", workerID, rand.Intn(1000))
	sessionID := sessions[rand.Intn(len(sessions))]
	platform := platforms[rand.Intn(len(platforms))]
	locale := locales[rand.Intn(len(locales))]
	deviceID := devices[rand.Intn(len(devices))]

	if rand.Intn(20) == 0 {
		userID = fmt.Sprintf("bot-%d", rand.Intn(5))
	}

	return &trendv1.SearchEvent{
		EventId:   fmt.Sprintf("evt-%d-%d-%d", workerID, seq, time.Now().UnixNano()),
		Query:     query,
		CreatedAt: timestamppb.Now(),
		UserId:    userID,
		SessionId: sessionID,
		DeviceId:  deviceID,
		Locale:    locale,
		Platform:  platform,
	}
}
