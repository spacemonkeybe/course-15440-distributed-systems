// Implementation of a KeyValueServer. Students should write their code in this file.

package p0

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strconv"
)

type keyValueServer struct {
	listener    net.Listener
	clients     []*Client
	connections chan net.Conn
	queries     chan *Query
	messages    chan []byte
	counts      chan int
}

type Client struct {
	connection net.Conn
	messages   chan []byte
	quitSignal chan int
}

type QueryType string

const (
	PUT   = "PUT"
	GET   = "GET"
	COUNT = "COUNT"
)

type Query struct {
	key       string
	value     []byte
	queryType QueryType
}

const MESSAGE_QUEUE_SIZE_THRESHOLD = 500

// New creates and returns (but does not start) a new KeyValueServer.
func New() KeyValueServer {
	return &keyValueServer{
		nil,
		nil,
		make(chan net.Conn),
		make(chan *Query),
		make(chan []byte),
		make(chan int),
	}
}

func (kvs *keyValueServer) Start(port int) error {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	kvs.listener = listener
	init_db()

	go runLoop(kvs)
	go acceptClients(kvs)

	return nil
}

func (kvs *keyValueServer) Close() {
	// TODO: implement this!
}

func (kvs *keyValueServer) Count() int {
	kvs.queries <- &Query{
		queryType: COUNT,
	}
	return <-kvs.counts
}

func runLoop(kvs *keyValueServer) {
	for {
		select {
		case message := <-kvs.messages:
			fmt.Printf("message\n")
			for _, client := range kvs.clients {
				if len(client.messages) == MESSAGE_QUEUE_SIZE_THRESHOLD {
					<-client.messages
				}
				client.messages <- message
			}
		case connection := <-kvs.connections:
			fmt.Printf("connection\n")
			client := &Client{connection, make(chan []byte, MESSAGE_QUEUE_SIZE_THRESHOLD), make(chan int)}
			kvs.clients = append(kvs.clients, client)
			fmt.Printf("connection count: %v\n", len(kvs.clients))
			go readForClient(kvs, client)
			go writeForClient(client)
		case query := <-kvs.queries:
			fmt.Printf("query\n")
			if query.queryType == PUT {
				put(query.key, query.value)
			} else if query.queryType == GET {
				value := get(query.key)
				fmt.Printf("HAHA1\n")
				// kvs.messages <- append([]byte(query.key+","), value...)
				kvs.messages <- []byte("haha")
				fmt.Printf("HAHA2\n")
			} else if query.queryType == COUNT {
				kvs.counts <- len(kvs.clients)
			}
		}
	}
}

func acceptClients(kvs *keyValueServer) {
	for {
		conn, err := kvs.listener.Accept()
		if err == nil {
			kvs.connections <- conn
		}
	}
}

func readForClient(kvs *keyValueServer, client *Client) {
	reader := bufio.NewReader(client.connection)
	for {
		select {
		case <-client.quitSignal:
			return
		default:
			message, err := reader.ReadBytes('\n')

			fmt.Printf("message: %v\n", message)

			if err != nil {
				return
			} else {
				tokens := bytes.Split(message, []byte(","))
				if string(tokens[0]) == "put" {
					key := string(tokens[1][:])
					kvs.queries <- &Query{
						key:       key,
						value:     tokens[2],
						queryType: PUT,
					}
				} else {
					key := string(tokens[1][:len(tokens[1])-1])
					kvs.queries <- &Query{
						key:       key,
						queryType: GET,
					}
				}
			}
		}
	}
}

func writeForClient(client *Client) {
	for {
		select {
		case <-client.quitSignal:
			return
		case message := <-client.messages:
			client.connection.Write(message)
		}
	}
}
