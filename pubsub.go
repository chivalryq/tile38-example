package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	t38c "github.com/axvq/tile38-client"
)

func main() {
	tile38, err := t38c.New("localhost:9851", t38c.Debug)
	if err != nil {
		log.Fatal(err)
	}
	defer tile38.Close()

	geofenceRequest := tile38.Geofence.Nearby("buses", 33.5123, -112.2693, 200).
		Actions(t38c.Enter, t38c.Exit)
	//和CLI设置一个chan效果一样
	if err := tile38.Channels.SetChan("busstop", geofenceRequest).Do(); err != nil {
		log.Fatal(err)
	}
	//定义回调函数
	handler := func(event *t38c.GeofenceEvent) {
		b, _ := json.Marshal(event)
		fmt.Printf("event: %s\n", b)
	}
	//订阅该chan
	if err := tile38.Channels.Subscribe(context.Background(), handler, "busstop"); err != nil {
		log.Fatal(err)
	}
}
