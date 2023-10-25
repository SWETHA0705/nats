package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/nats-io/stan.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/resolver"
)

const (
	exampleScheme      = "example"
	exampleServiceName = "lb.example.grpc.io"
)

var (
	addrs []string
	addr  string
)

func healthCheck(addrs []string) {
	fmt.Println("******************\nHeathcheck invoked")
	for i:=1;i<=3;i++{
		for _, addr := range addrs {
			err := ping(addr)
			if err != nil {
				log.Printf("Server at %s is down: %s", addr, err)
			} else {
				log.Printf("Server at %s is up and running.", addr)
			}
			time.Sleep(2 * time.Second)
		}
	}
}

func ping(address string) error {
	fmt.Println("Ping invoked")
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func main() {
	// NATS Streaming cluster ID, client ID, and subject
	clusterID := "test-cluster"   // Replace with your NATS Streaming cluster ID.
	clientID := "test-publish"    // Replace with your NATS Streaming client ID.
	const SER_REG_SUB = "ServiceRegistery" // Replace with the subject you want to subscribe to.

	// Connect to the NATS Streaming server
	sc, err := stan.Connect(clusterID, clientID, stan.NatsURL("nats://192.168.1.31:4222"))
	if err != nil {
		log.Fatalf("Error connecting to NATS Streaming: %v", err)
	}
	defer sc.Close()

	//{"Application":"TestServer","InstanceIP":"192.168.1.31","InstancePort":"9080"}
	// Subscribe to the subject
	_, err = sc.Subscribe(SER_REG_SUB, func(msg *stan.Msg) {
		// Assuming the data is in JSON format, you can unmarshal it into a map
		var data map[string]interface{}
		err := json.Unmarshal(msg.Data, &data)
		if err != nil {
			log.Printf("Error unmarshalling data: %s", err)
			return
		}

		// Now you can work with the map data
		fmt.Println("******************\nReceived a message:")
		for i:=1;i <=1;i++ {
			if _, ok := data["application"].(string); ok {
				if ip, ok := data["InstanceIP"].(string); ok {
					if port, ok := data["InstancePort"].(string); ok {
						// Construct the address string and append it to the slice
						addr = ip + port
						addrs = append(addrs, addr)
					}
				}
			}
			fmt.Println(data)
			//fmt.Printf("%s: %v\n", key, value)
		}
		fmt.Printf("Address added: %s\n", addr)
		//build invoke		
		roundrobinConn, err := grpc.Dial(
			fmt.Sprintf("%s:///%s", exampleScheme, exampleServiceName),
			grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin":{}}]}`), // This sets the initial balancing policy.
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		// health(ip, port)
		defer roundrobinConn.Close()
	})
	// , stan.DurableName("Swe"))
	if err != nil {
		log.Fatalf("Error subscribing to subject: %v", err)
	}

	fmt.Printf("Subscribed to subject %s. Waiting for messages...\n", SER_REG_SUB)
	select {}

}

type exampleResolverBuilder struct{}

func (*exampleResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &exampleResolver{
		target: target,
		cc:     cc,
		addrsStore: map[string][]string{
			exampleServiceName: addrs,
		},
	}
	fmt.Println("******************\nResolver invoked\nbuild")
	r.start()
	return r, nil
}
func (*exampleResolverBuilder) Scheme() string { return exampleScheme }

type exampleResolver struct {
	target     resolver.Target
	cc         resolver.ClientConn
	addrsStore map[string][]string
}

func (r *exampleResolver) start() {
	fmt.Println("start")
	addrStrs := r.addrsStore[r.target.Endpoint()]
	
	addrs := make([]resolver.Address, len(addrStrs))
	for i, s := range addrStrs {
		addrs[i] = resolver.Address{Addr: s}
	}

	// Extract IP and port from the resolver format and create a slice of addresses
	var extractedAddrs []string
	for _, addr := range addrs {
		ip, port, err := net.SplitHostPort(addr.Addr)
		if err != nil {
			log.Printf("Error extracting IP and port: %v", err)
			continue
		}
		extractedAddrs = append(extractedAddrs, net.JoinHostPort(ip, port))
	}
	
	r.cc.UpdateState(resolver.State{Addresses: addrs})
	//  healthCheck(extractedAddrs)
	go func() { 
        go healthCheck(extractedAddrs)
    }()
}

func (*exampleResolver) ResolveNow(o resolver.ResolveNowOptions) {}
func (*exampleResolver) Close()                                  {}

func init() {
	resolver.Register(&exampleResolverBuilder{})
}
