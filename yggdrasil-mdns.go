package main

import "fmt"
import "net"
import "net/http"
import "io/ioutil"
import "encoding/json"
import "bytes"
import "os/signal"
import "strings"
import "syscall"
import "log"
import "os"
import "github.com/neilalexander/zeroconf"

type service struct {
	instance  string
	name      string
	domain    string
	hostname  string
	port      int
	address   string
}

var servers []*zeroconf.Server

func main() {
	res, err := http.Get("http://y.yakamo.org:3000/nodeinfo")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	response, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var nodeinfos map[string]interface{}
	json.Unmarshal(response, &nodeinfos)

	if _, ok := nodeinfos["yggnodeinfo"]; !ok {
		fmt.Println("No yggnodeinfo found")
		os.Exit(1)
	}

  /*
	var loopback net.Interface
	if intfs, err := net.Interfaces(); err == nil {
		for _, intf := range intfs {
			if intf.Flags&net.FlagLoopback != 0 {
				loopback = intf
				fmt.Println("Found loopback interface", loopback.Name)
				break
			}
		}
	} else {
		panic(err)
	}

	if loopback.Name == "" {
		fmt.Println("No loopback interface found")
		os.Exit(1)
	}
  */

	for key, value := range nodeinfos["yggnodeinfo"].(map[string]interface{}) {
		if services, ok := value.(map[string]interface{})["services"]; ok {
			fmt.Println("Node", key)
			for _, ns := range services.([]interface{}) {
				nodesvc := ns.([]interface{})
				if len(nodesvc) >= 3 {
					s := service{
						instance: nodesvc[0].(string),
						name:     nodesvc[1].(string),
						domain:   "local.",
						port:     int(nodesvc[2].(float64)),
					}

          if len(nodesvc) >= 4 {
            if targetip, ok := nodesvc[3].(string); ok {
  						origin := net.ParseIP(key)
  						target := net.ParseIP(targetip)
  						if target[0] == 0x03 {
  							if bytes.Compare(origin[1:8], target[1:8]) == 0 {
  								s.address = target.String()
  							}
  						} else if target[0] == 0x02 {
  							if bytes.Compare(origin[:16], target[:16]) == 0 {
  								s.address = origin.String()
  							}
  						}
            }
					} else {
            s.address = key
          }

          s.hostname = "yggdrasil-" + strings.Replace(s.address, ":", "", -1)

					if server, err := zeroconf.RegisterProxy(
						s.instance,
						s.name,
						s.domain,
						s.port,
						s.hostname,
						[]string{s.address},
						[]string{"origin=" + key},
						[]net.Interface{},
					); err == nil {
						servers = append(servers, server)
						fmt.Println("- Advertising service:", s.instance)
					} else {
						fmt.Println("- Failed to advertise service:", s.instance)
						fmt.Println("  Error:", err)
					}
				}
			}
		}
	}

	defer func() {
		for _, server := range servers {
			server.Shutdown()
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	select {
	case <-sig:
	}

	log.Println("Shutting down")
}
