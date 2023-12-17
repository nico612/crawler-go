package main

import (
	"context"
	"fmt"
	etcdReg "github.com/go-micro/plugins/v4/registry/etcd"
	gs "github.com/go-micro/plugins/v4/server/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/nico612/crawler-go/collect"
	"github.com/nico612/crawler-go/engine"
	"github.com/nico612/crawler-go/limiter"
	"github.com/nico612/crawler-go/log"
	pb "github.com/nico612/crawler-go/proto/greeter"
	"github.com/nico612/crawler-go/storage"
	"github.com/nico612/crawler-go/storage/sqlstorage"
	"go-micro.dev/v4"
	"go-micro.dev/v4/logger"
	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"

	"net/http"
)

type Greeter struct {
}

func (g *Greeter) Hello(ctx context.Context, req *pb.Request, rsp *pb.Response) error {
	rsp.Greeting = "Hello " + req.Name
	return nil
}

func main() {
	// log
	plugin := log.NewStdoutPlugin(zapcore.DebugLevel)
	logger := log.NewLogger(plugin)
	logger.Info("log init end")
	// set zap global logger
	zap.ReplaceGlobals(logger)

	// proxy
	//proxyURLs := []string{"http://127.0.0.1:8888", "http://127.0.0.1:8888"}
	//var p proxy.ProxyFunc
	//var err error
	//if p, err = proxy.RoundRobinProxySwitcher(proxyURLs...); err != nil {
	//	logger.Error("RoundRobinProxySwitcher failed")
	//	return
	//}

	// fetcher
	var f collect.Fetcher = &collect.BrowserFetch{
		Timeout: 3000 * time.Millisecond,
		Logger:  logger,
		//Proxy:   p,
	}

	// storage
	var storage storage.Storage
	var err error
	if storage, err = sqlstorage.New(
		sqlstorage.WithSqlUrl("root:123456@tcp(127.0.0.1:3326)/crawler?charset=utf8"),
		sqlstorage.WithLogger(logger.Named("sqlDB")),
		sqlstorage.WithBatchCount(2),
	); err != nil {
		logger.Error("create sqlstorage failed")
		return
	}

	// speed limiter
	secondLimit := rate.NewLimiter(limiter.Per(1, 2*time.Second), 1)
	minuteLimit := rate.NewLimiter(limiter.Per(20, 1*time.Minute), 20)
	multiLimiter := limiter.MultiLimiter(secondLimit, minuteLimit)

	// init tasks
	seeds := make([]*collect.Task, 0, 1000)
	seeds = append(seeds, &collect.Task{
		Property: collect.Property{
			Name: "douban_book_list",
		},
		Fetcher: f,
		Storage: storage,
		Limit:   multiLimiter,
	})

	s := engine.NewEngine(
		engine.WithFetcher(f),
		engine.WithLogger(logger),
		engine.WithWorkCount(5),
		engine.WithSeeds(seeds),
		engine.WithScheduler(engine.NewSchedule()),
	)

	// worker start
	go s.Run()

	go HandleHttp()

	// start grpc server
	HandleGrpc()
}

func HandleGrpc() {

	// etcd 注册中心
	reg := etcdReg.NewRegistry(
		registry.Addrs(":2379"),
	)

	// grpc server
	service := micro.NewService(
		micro.Server(gs.NewServer()), // 生成grpc服务器
		micro.Address(":9090"),       // 指定服务监听地址
		micro.Registry(reg),
		micro.Name("go.micro.server.worker"), // 服务器名字
	)
	service.Init()
	pb.RegisterGreeterHandler(service.Server(), new(Greeter))
	if err := service.Run(); err != nil {
		logger.Fatal(err)
	}
}

func HandleHttp() {
	ctx, cancle := context.WithCancel(context.Background())
	defer cancle()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// 指定了要转发到哪一个 GRPC 服务器
	if err := pb.RegisterGreeterGwFromEndpoint(ctx, mux, "localhost:9090", opts); err != nil {
		fmt.Println(err)
	}

	http.ListenAndServe(":8080", mux)

}
