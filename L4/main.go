package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

var printer chan string

func main() {
	n := 15
	d := 30

	rand.Seed(time.Now().Unix())

	if len(os.Args) != 3 {
		fmt.Println("Invalid argumants!")
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
		go forwarderR_loop(&g, i)
		go forwarderS_loop(&g, i)
		go sender(&g, i, 1500, 500, shortestPathsCosts, end_chan)
	}

	for i := 0; i < g.size(); i++ {
		for j := 0; j < len(g.vertices[i].hosts); j++ {
			go host(&g, i, j, 800, 400)
		}
	}

	for i := 0; i < g.size(); i++ {
		<-end_chan
	}

	time.Sleep(5 * time.Second)
	printer <- "__END__"

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

		if message == "__END__" {
			break
		}

		fmt.Println(message)
	}
}

func forwarderR_loop(g *Graph, vertex_id int) {
	for {
		message := <-g.vertices[vertex_id].forwarder_chan
		message.via = append(message.via, vertex_id)

		g.vertices[vertex_id].forwarder_lock.Lock()
		g.vertices[vertex_id].forwarder_queue = append(g.vertices[vertex_id].forwarder_queue, message)
		g.vertices[vertex_id].forwarder_lock.Unlock()
	}
}

func forwarderS_loop(g *Graph, vertex_id int) {
	for {
		g.vertices[vertex_id].forwarder_lock.Lock()
		if len(g.vertices[vertex_id].forwarder_queue) > 0 {
			message := g.vertices[vertex_id].forwarder_queue[0]
			g.vertices[vertex_id].forwarder_queue = g.vertices[vertex_id].forwarder_queue[1:len(g.vertices[vertex_id].forwarder_queue)]
			g.vertices[vertex_id].forwarder_lock.Unlock()

			if message.to.node == vertex_id {
				g.vertices[vertex_id].hosts[message.to.id].in_channel <- message
			} else {
				g.vertices[vertex_id].routing_table_in <- RoutingTableMessage{0, 0, message.to.node, 0, false}
				info := <-g.vertices[vertex_id].routing_table_out
				g.vertices[info.from].forwarder_chan <- message
			}
		} else {
			g.vertices[vertex_id].forwarder_lock.Unlock()
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func host(g *Graph, vertex_id int, id int, sleep_max, sleep_min int) {
	message := DataMessage{Address{vertex_id, id}, Address{g.vertices[vertex_id].hosts[id].reciver.node, g.vertices[vertex_id].hosts[id].reciver.id}, make([]int, 0)}

	g.vertices[vertex_id].forwarder_chan <- message
	//printer <- fmt.Sprint("SEND: (", vertex_id, ", ", id, ") => (", g.vertices[vertex_id].hosts[id].reciver.node, ", ", g.vertices[vertex_id].hosts[id].reciver.id, ")")

	costs := make(map[Address]int)

	for {
		message = <-g.vertices[vertex_id].hosts[id].in_channel

		last_cost, ok := costs[message.from]

		path := fmt.Sprint(message.via[0])
		for i := 1; i < len(message.via); i++ {
			path = path + fmt.Sprint(" => ", message.via[i])
		}

		if !ok || last_cost <= len(message.via) {
			printer <- fmt.Sprint("Wiadmomość od (", message.from.node, ", ", message.from.id, ") do (", message.to.node, ", ", message.to.id, "); Koszt = ", len(message.via), "; Droga: ", path)
		} else {
			printer <- fmt.Sprint("Wiadmomość od (", message.from.node, ", ", message.from.id, ") do (", message.to.node, ", ", message.to.id, "); Koszt = ", len(message.via), " poprawa z ", last_cost, "; Droga: ", path)
		}
		if !ok || last_cost > len(message.via) {
			costs[message.from] = len(message.via)
		}

		message = DataMessage{Address{vertex_id, id}, message.from, make([]int, 0)}

		to_sleep := rand.Intn(sleep_max-sleep_min) + sleep_min
		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		g.vertices[vertex_id].forwarder_chan <- message
	}
}

func reciver(g *Graph, vertex_id int) {
	for {
		message := <-g.vertices[vertex_id].in_channel

		g.vertices[vertex_id].routing_table_in <- RoutingTableMessage{0, 0, message.to, 0, false}
		info := <-g.vertices[vertex_id].routing_table_out

		//printer <- fmt.Sprint("Wieszchołek: ", vertex_id, " otrzymal od: ", message.from, " oferte do ", message.to, " za ", message.cost)

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

				//printer <- fmt.Sprint("Wieszchołek: ", vertex_id, " wysyła do: ", id, " oferte do ", message.to, " za ", message.cost)
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

	for i := 0; i < n; i++ {
		//num_of_host := rand.Intn(n-1) + 1
		num_of_host := rand.Intn(n/2+1) + 1

		g.vertices[i].hosts = make([]HostData, num_of_host)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < len(g.vertices[i].hosts); j++ {
			host_data := HostData{Address{i, i}, make(chan DataMessage, 0)}

			for host_data.reciver.node == i {
				host_data.reciver.node = rand.Intn(n)
			}

			host_data.reciver.id = rand.Intn(len(g.vertices[host_data.reciver.node].hosts))

			g.vertices[i].hosts[j] = host_data
		}
	}

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
