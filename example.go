package main

import (
	"fmt"
	t38c "github.com/axvq/tile38-client"
)

func main() {
	//client := redis.NewClient(&redis.Options{
	//	Addr: "127.0.0.1:9851",
	//})
	//cmd := redis.NewStringCmd("SET", "fleet", "truck", "POINT", 33.32, 115.423)
	//client.Process(cmd)
	//v, _ := cmd.Result()
	//log.Println(v)
	//cmd1 := redis.NewStringCmd("GET", "fleet", "truck")
	//client.Process(cmd1)
	//v1, _ := cmd1.Result()
	//log.Println(v1)

	//cmd := redis.NewStringCmd("SET", "cab", "cab2:driver", "POINT", 333.5124, -112.269, "STRING", "SOME_STRING")
	//client.Process(cmd)
	//v, _ := cmd.Result()
	//log.Println(v)
	//
	client, err := t38c.New("localhost:9851", t38c.Debug)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	if err := client.Keys.Set("fleet", "truck1").Point(33.5123, -112.2693).Do(); err != nil {
		panic(err)
	}

	if err := client.Keys.Set("fleet", "truck2").Point(33.4626, -112.1695).
		// optional params
		Field("speed", 20).
		Expiration(20).
		Do(); err != nil {
		panic(err)
	}

	// search 6 kilometers around a point. returns one truck.
	response, err := client.Search.Nearby("fleet", 33.462, -112.268, 6000).
		Where("speed", 0, 100).
		Match("truck*").
		Format(t38c.FormatPoints).Do()
	if err != nil {
		panic(err)
	}
	client.Search.Intersects("")

	// truck1 {33.5123 -112.2693}
	fmt.Println(response.Points[0].ID, response.Points[0].Point)
}
