# Tile38指北

## 简介

Tile38是一个专用的地理点、范围相关的高性能空间索引引擎，支持多种类型的对象，包括经纬度、边界框（经纬度矩形），**xyz tile**，**geohash**，**geojson**。我理解大概就是一个有相关地理功能的Redis，可以起到简化程序结构的作用。

可以进行多种操作，例如判定交叉、判定在内、判定附近、静态和动态的围栏。

在通信协议上支持HTTP、websockets、telnet、RESP 调用，其中HTTP和Websockets使用JSON通信。Telnet和RESP clients 使用RESP通信。Go调用可以使用go-redis作为client进行调用或者专用的client包Tile Client

https://github.com/zweihander/tile38-client

本文示例代码在repo:https://github.com/chivalryq/tile38-example

## 对象和CLI命令

#### 对象

对象有上述类型，并且一个对象有两级索引key和id。可以在一个对象上附加z字段（例如海拔或者时间戳，可以由使用者约定这个字段的意义）。可以附加其他字段，用FIELD关键字

#### set

如下命令set了一个对象

```shell
SET fleet truck1 point 33.5123 -112.2693     # plain lat/lon
```

fleet是一级 truck是二级，在文档中fleet称作key，truck1称作id，在这个命令里，point及之后的部分表示这是一个点，这部分可以改变，例如如果这个对象用GeoJSON格式表示，见下面这个例子

```shell
SET city tempe OBJECT {"type":"Polygon","coordinates":[[[-111.9787,33.4411],[-111.8902,33.4377],[-111.8950,33.2892],[-111.9739,33.2932],[-111.9787,33.4411]]]}
```

附加其他字段，例子中附加了俩字段age和speed
```sh
SET fleet truck1 FIELD speed 90 FIELD age 21 POINT 33.5123 -112.2693
```


#### scan

```shell
SCAN fleet
```

得到所有key为fleet的对象（一个集合）

#### get

```
GET fleet truck1
```

得到key、id对应的对象

#### BOUNDS

```shell
BOUNDS fleet
```
返回key=fleet所有对象的外接矩形，以GeoJSON格式表示

#### 订阅发布模型的地理围栏 SETCHAN/SUBSCRIBE

可以通过如下命令设置和订阅一个围栏

```shell
localhost:9851> SETCHAN warehouse NEARBY fleet FENCE POINT 33.462 -112.268 6000
{"ok":true,"elapsed":"21.712µs"}
localhost:9851> SUBSCRIBE warehouse
{"ok":true,"command":"subscribe","channel":"warehouse","num":1,"elapsed":"7.361µs"}
```

如果一个点进入了这个围栏（另一个客户端）

```shell
localhost:9851> SET fleet bus POINT 33.460 -112.260
{"ok":true,"elapsed":"12.988µs"}
```

则订阅者收到

```shell
{
    "command":"set",
    "group":"602f7813619b821ae0bb2b12",
    "detect":"enter",
    "hook":"warehouse",
    "key":"fleet",
    "time":"2021-02-19T16:34:27.9758986+08:00",
    "id":"bus",
    "object":{
        "type":"Point",
        "coordinates":[
            -112.26,
            33.46
        ]
    }
}
{
    "command":"set",
    "group":"602f7813619b821ae0bb2b12",
    "detect":"inside",
    "hook":"warehouse",
    "key":"fleet",
    "time":"2021-02-19T16:34:27.9758986+08:00",
    "id":"bus",
    "object":{
        "type":"Point",
        "coordinates":[
            -112.26,
            33.46
        ]
    }
}
```

表示监听到了一个enter和一个inside事件，总共可以监听五种事件**`inside`**，**`outside`** ，**`enter`** ，**`exit`** ，**`cross`** 

另外两种监听`INTERSECTS`,`WITHIN`可以监听更多表示形式的围栏，例如用GeoJSON表示的多边形围栏

使用`DELCHAN`,`PDELCHAN`删除和按照字符串匹配删除channels

使用`PSUBSCRIBE`按照字符串模式匹配订阅channels


## Go & Tile38 实践

### PUB/SUB

```go
func main() {
	tile38, err := t38c.New("localhost:9851", t38c.Debug)
	if err != nil {
		log.Fatal(err)
	}
	defer tile38.Close()
    //第一步定义监听哪个围栏的请求，这里由于是圆形围栏，所以用Nearby方法一步到位得到了请求
	//若要用矩形围栏，见下一个例子，比圆形多一步定义
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
```
### 订阅信息发送到MQ

用RabbitMQ做一个例子，在Go程序中，如下给设置webhook（到MQ）

```go
func main() {
	client, err := t38c.New("localhost:9851", t38c.Debug)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	//第一步，定义一个矩形的围栏（经纬度）
	polygon := geojson.NewPolygonGeometry([][][]float64{
		{
			{0, 0},
		    {0, 10},
			{10, 10},
			{10, 0},
			{0, 0},
		},
	})

	//第二步，定义要监听的围栏的query
	//对Within这个方法，我的理解是在判断点和围栏去比较的时候，其实Within和Intersect没有区别，只有围栏和围栏判断关系的时候才有区别
    //订阅key=city的消息，这个key就是比如一个点或者一个围栏的key
	query := client.Geofence.Within("city").Geometry(polygon)

	//第三步，设置webhook
	//第一个参数是webhook的名字,URI中指定了scheme,详细参数,queue等，这个随MQ而变，这是一个MQ的例子
	client.Webhooks.SetHook("withInCity", "amqp://guest:guest@127.0.0.1:5672/endpoint", query).Do()
    //这样所有query所监听到的事件都会被发送到MQ
}

```

下面我们启动一个监听程序

```go
package main

import (
	"github.com/streadway/amqp"
	"log"
)

func fail(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	fail(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	fail(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"city", // name
		true,   // durable,注意这里是tail38的默认值，声明错误会导致RabbitMQ内部except，就收不到消息，但是RabbitMQ示例代码是false（真不戳）
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	fail(err, "Failed to declare a queue")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	fail(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

```

下面在这个围栏里投放一个点

```shell
set city car1 POINT 5 5
```

上述接受程序会接受并在控制台打印：

```
2021/02/20 17:07:59 Received a message: {"command":"set","group":"6030d16f619b821ae0bb2b28","detect":"inside","hook":"withInCity","key":"city","time":"2021-02-20T17:07:59.9593229+08:00","id":"car1","object":{"type":"Point","coordinates":[5,5]}}
```

## 注解

XYZ tile:一种web常用的地图图片计算和传输的表示方法，在浏览器看到的整张地图是根据地图范围，缩放级别，分辨率、地图中心点等参数，计算出的一片一片的瓦片组成的，XY表示横纵索引，Z是缩放级别

geohash:把一块地理区域映射为一个字符串

geoJson:用json表示地图上点和多边形的方式

RESP:Redis Protocol specification，Redis所使用的应用层协议。实现简单、解析快、可读性好。
