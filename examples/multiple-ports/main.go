package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

func main() {
	var (
		web, tcp, udp string
		wg            sync.WaitGroup
	)

	flag.StringVar(&web, "web", "", "Comma separated list of ports on which to expose a tiny echo web server")
	flag.StringVar(&tcp, "tcp", "", "Comma separated list of ports on which to expose a tiny echo TCP server")
	flag.StringVar(&udp, "udp", "", "Comma separated list of ports on which to expose a tiny echo UDP server")

	flag.Parse()

	exposeHttp(&wg, parsePorts(web)...)
	exposeTCP(&wg, parsePorts(tcp)...)
	exposeUDP(&wg, parsePorts(udp)...)

	wg.Wait()
}

func exposeHttp(wg *sync.WaitGroup, ports ...int) {
	for _, port := range ports {
		server := http.Server{
			Addr: ":" + strconv.Itoa(port),
		}

		server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello from " + server.Addr))
		})

		wg.Add(1)

		go func() {
			defer wg.Done()

			exit := makeExitChannel()

			go func() {
				log.Printf("Starting HTTP server on %s", server.Addr)
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("Failed to start HTTP server on %s: %v", server.Addr, err)
					exit <- syscall.SIGTERM
				}
			}()

			<-exit
			log.Printf("Shutting down HTTP server on %s", server.Addr)
			server.Shutdown(context.Background())
		}()
	}
}

func exposeTCP(wg *sync.WaitGroup, ports ...int) {
	for _, port := range ports {
		addr := net.TCPAddr{Port: port}
		wg.Add(1)

		go func() {
			defer wg.Done()

			var (
				exit     = makeExitChannel()
				listener *net.TCPListener
				err      error
			)

			go func() {
				log.Printf("Starting TCP server on  %s", addr.String())

				listener, err = net.ListenTCP("tcp", &addr)

				if err != nil {
					log.Printf("Failed to start TCP server on %s: %v", addr.String(), err)
					exit <- syscall.SIGTERM
					return
				}

				for {
					conn, err := listener.Accept()

					if errors.Is(err, net.ErrClosed) {
						return
					}

					if err != nil {
						log.Printf("Failed to accept connection on %s: %v", addr.String(), err)
						continue
					}

					conn.Write([]byte("Hello from " + conn.LocalAddr().String()))
					conn.Close()
				}
			}()

			<-exit

			if listener == nil {
				return
			}

			log.Printf("Shutting down TCP server on %s", addr.String())
			listener.Close()
		}()
	}
}

func exposeUDP(wg *sync.WaitGroup, ports ...int) {
	for _, port := range ports {
		addr := net.UDPAddr{Port: port}
		wg.Add(1)

		go func() {
			defer wg.Done()

			var (
				exit = makeExitChannel()
				conn *net.UDPConn
				err  error
			)

			go func() {
				log.Printf("Starting UDP server on  %s", addr.String())

				conn, err = net.ListenUDP("udp", &addr)

				if err != nil {
					log.Printf("Failed to start UDP server on %s: %v", addr.String(), err)
					exit <- syscall.SIGTERM
					return
				}

				buf := make([]byte, 1024)

				for {
					_, addr, err := conn.ReadFrom(buf)

					if errors.Is(err, net.ErrClosed) {
						return
					}

					if err != nil {
						log.Printf("Failed to read from UDP connection on %s: %v", addr.String(), err)
						continue
					}

					conn.WriteTo([]byte("Hello from "+conn.LocalAddr().String()), addr)
				}
			}()

			<-exit

			if conn == nil {
				return
			}

			log.Printf("Shutting down UDP server on %s", addr.String())
			conn.Close()
		}()
	}
}

func makeExitChannel() chan os.Signal {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	return exit
}

func parsePorts(value string) []int {
	strs := strings.Split(value, ",")
	ports := make([]int, 0, len(strs))

	for _, str := range strs {
		str = strings.TrimSpace(str)

		if str == "" {
			continue
		}

		p, err := strconv.Atoi(str)

		if err != nil {
			log.Printf("Failed to parse port %s: %v", str, err)
			continue
		}

		ports = append(ports, p)
	}

	return ports
}
