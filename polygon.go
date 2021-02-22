package main

import (
	t38c "github.com/axvq/tile38-client"
	"github.com/paulmach/go.geojson"
)

func main() {
	client, err := t38c.New("localhost:9851", t38c.Debug)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	polygon := geojson.NewPolygonGeometry([][][]float64{
		{
			{0, 0},
			{0, 10},
			{10, 10},
			{10, 0},
			{0, 0},
		},
	})
	client.Keys.Set("city", "tempe").Geometry(polygon).Do() // nolint:errcheck
	//我理解在判断点和围栏去比较的时候，其实Within和Intersect没有区别，只有围栏和围栏判断关系的时候才有区别
	//订阅key=city的消息，这个key就是比如一个点或者一个围栏的key
	//先创建一个GeoFenceBuilder
	query := client.Geofence.Intersects("city").Geometry(polygon)
	//第一个参数是webhook的名字,URI中指定了scheme,详细参数,queue等，这个随MQ而变
	err = client.Webhooks.SetHook("withInCity", "amqp://guest:guest@127.0.0.1:5672/city", query).Do()
	//这样所有query所监听到的事件都会被发送到MQ
}
