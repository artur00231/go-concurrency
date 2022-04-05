package main

import (
	"fmt"
	"math/rand"
	"time"
	"os"
	"strconv"
)

var printer chan string

func main() {
	n := 15
	d := 30

	rand.Seed(time.Now().Unix())

	if len(os.Args) != 3 {
		fmt.Print("Invalid argumants!")
		return
	}

	n, _ = strconv.Atoi(os.Args[1])
	d, _ = strconv.Atoi(os.Args[2])
	/*k, _ = strconv.Atoi(os.Args[3])
	b, _ = strconv.Atoi(os.Args[4])
	h, _ = strconv.Atoi(os.Args[5])*/

	g := generateGraph(n, d)

	g.print()
	g.printf()

	shortestPathsCosts := calculateShortestPaths(g)

	for i := 0; i < g.size(); i++ {
		for j := 0; j < g.size(); j++ {
			fmt.Printf("%5d", shortestPathsCosts[i][j])
		}

		fmt.Println("")
	}

	fmt.Println("")
	fmt.Println("")

	end_chan := make(chan bool, 0)

	printer = make(chan string)
	go print_loop(printer)

	for i := 0; i < g.size(); i++ {
		go routingTableIO(&g, i)
		go reciver(&g, i)
		go sender(&g, i, 1500, 500, shortestPathsCosts, end_chan)
	}

	for i := 0; i < g.size(); i++ {
		<-end_chan
	}

	for i := 0; i < g.size(); i++ {
		for j := 0; j < g.size(); j++ {
			fmt.Printf("%5d", g.vertices[j].nexthop_cost[i])
		}

		fmt.Println("")
	}
}

func print_loop(printer_in chan string) {
	for {
		message := <-printer_in

		fmt.Println(message)
	}
}

func reciver(g *Graph, vertex_id int) {
	for {
		message := <-g.vertices[vertex_id].in_channel

		g.vertices[vertex_id].routing_table_in <- RoutingTableMessage{0, 0, message.to, 0, false}
		info := <-g.vertices[vertex_id].routing_table_out

		printer <- fmt.Sprint("Wieszchołek: ", vertex_id, " otrzymal od: ", message.from, " oferte do ", message.to, " za ", message.cost)

		if message.cost+g.vertices[vertex_id].costs[message.from] < info.cost {
			routing_message := RoutingTableMessage{1, message.from, message.to, message.cost + g.vertices[vertex_id].costs[message.from], false}
			g.vertices[vertex_id].routing_table_in <- routing_message

			printer <- fmt.Sprint("Wieszchołek: ", vertex_id, " akceputuje oferte od: ", message.from, " poprawa z ", info.cost, " do ", message.cost+g.vertices[vertex_id].costs[message.from])
		}

	}
}

func sender(g *Graph, vertex_id int, sleep_max, sleep_min int, shortestPathsCosts [][]int, end chan bool) {
	for {
		to_sleep := rand.Intn(sleep_max-sleep_min) + sleep_min
		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		send := false
		var message Message

		g.vertices[vertex_id].routing_table_in <- RoutingTableMessage{4, 0, 0, 0, false}
		info := <-g.vertices[vertex_id].routing_table_out

		if info.changed && info.to != vertex_id {
			send = true
			message = Message{vertex_id, info.to, info.cost}
		}

		if send {
			for id := range g.vertices[vertex_id].edges {
				g.vertices[id].in_channel <- message

				printer <- fmt.Sprint("Wieszchołek: ", vertex_id, " wysyła do: ", id, " oferte do ", message.to, " za ", message.cost)
			}
		} else {
			all_optimal := true
			for i := 0; i < g.size(); i++ {
				g.vertices[vertex_id].routing_table_in <- RoutingTableMessage{0, 0, i, 0, false}
				info := <-g.vertices[vertex_id].routing_table_out

				if info.cost != shortestPathsCosts[vertex_id][i] || info.changed {
					all_optimal = false
				}
			}

			if all_optimal {
				end <- true

				printer <- fmt.Sprint("Wieszchołek: ", vertex_id, " kończy prace, sciezki zoptymalizowane")
				return
			}
		}
	}
}

func routingTableIO(g *Graph, vertex_id int) {
	for {
		message := <-g.vertices[vertex_id].routing_table_in
		return_message := RoutingTableMessage{0, 0, 0, 0, false}

		if message.op == 0 {
			return_message.from = g.vertices[vertex_id].nexthop[message.to]
			return_message.to = message.to
			return_message.cost = g.vertices[vertex_id].nexthop_cost[message.to]
			return_message.changed = g.vertices[vertex_id].nexthop_changed[message.to]

			g.vertices[vertex_id].routing_table_out <- return_message
		} else if message.op == 1 {
			g.vertices[vertex_id].nexthop[message.to] = message.from
			g.vertices[vertex_id].nexthop_cost[message.to] = message.cost
			g.vertices[vertex_id].nexthop_changed[message.to] = true
		} else if message.op == 2 {
			g.vertices[vertex_id].nexthop_changed[message.to] = false
		} else if message.op == 3 {
			return_message.changed = g.vertices[vertex_id].nexthop_changed[message.to]

			g.vertices[vertex_id].routing_table_out <- return_message
		} else if message.op == 4 {
			//Find changed routing

			found := false
			for i := 0; i < g.size(); i++ {
				if g.vertices[vertex_id].nexthop_changed[i] {
					return_message.changed = true
					return_message.cost = g.vertices[vertex_id].nexthop_cost[i]
					return_message.from = vertex_id
					return_message.to = i

					g.vertices[vertex_id].routing_table_out <- return_message
					g.vertices[vertex_id].nexthop_changed[i] = false

					found = true
					break
				}
			}

			if !found {
				return_message.changed = false
				g.vertices[vertex_id].routing_table_out <- return_message
			}
		}
	}
}

func generateGraph(n, d int) Graph {
	g := MakeGraph()

	for i := 0; i < n; i++ {
		g.addVertex()
	}

	for i := 0; i < n-1; i++ {
		g.addEdge(i, i+1 /*rand.Intn(100)+1*/, 1)
	}

	for ; d > 0; d-- {
		from, to := 0, 0

		for done := false; !done; done = g.addEdge(from, to /*rand.Intn(100)+1*/, 1) {
			from = rand.Intn(g.size() - 1)
			to = rand.Intn(g.size()-from-1) + from + 1
		}
	}

	g.resetHopTables()

	return g
}

func calculateShortestPaths(g Graph) [][]int {
	shortestPathsCosts := make([][]int, g.size())
	for i := 0; i < g.size(); i++ {
		shortestPathsCosts[i] = make([]int, g.size())
	}

	for i := 0; i < g.size(); i++ {
		for j := 0; j < g.size(); j++ {
			shortestPathsCosts[i][j] = g.vertices[i].nexthop_cost[j]
		}
	}

	for i := 0; i < g.size(); i++ {
		for j := 0; j < g.size(); j++ {
			fmt.Printf("%5d", shortestPathsCosts[i][j])
		}

		fmt.Println("")
	}
	fmt.Println("")
	fmt.Println("")

	for k := 0; k < g.size(); k++ {
		for i := 0; i < g.size(); i++ {
			for j := 0; j < g.size(); j++ {
				if shortestPathsCosts[i][j] > shortestPathsCosts[i][k]+shortestPathsCosts[k][j] {
					shortestPathsCosts[i][j] = shortestPathsCosts[i][k] + shortestPathsCosts[k][j]
				}
			}
		}
	}

	return shortestPathsCosts
}
