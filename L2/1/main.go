package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var mutex sync.Mutex

func main() {
	n := 10
	d := 4
	k := 10
	b := 5
	h := 15

	rand.Seed(time.Now().Unix())

	if len(os.Args) != 6 {
		fmt.Print("Invalid argumants!")
		return
	}

	n, _ = strconv.Atoi(os.Args[1])
	d, _ = strconv.Atoi(os.Args[2])
	k, _ = strconv.Atoi(os.Args[3])
	b, _ = strconv.Atoi(os.Args[4])
	h, _ = strconv.Atoi(os.Args[5])

	g := generateGraph(n, d, b)

	g.print()
	g.printf()

	watchdog_chan := make(chan WatchdogMessage, 10)
	g.watchdog_chan = watchdog_chan

	for i := 1; i < n-1; i++ {
		vertex := g.vertices[i]

		go node_loop(vertex, &g)
	}

	end_channel := make(chan bool, 10)

	sender_vertex := g.vertices[0]
	reciver_vertex := g.vertices[n-1]
	go sender(k, sender_vertex, &g, h)
	go reciver(k, reciver_vertex, &g)

	go watchdog(&g, watchdog_chan, g.vertices[0].node.print_chanel, end_channel, k)

	//go trapLoop(&g, 10000)

	<-end_channel

	time.Sleep(time.Duration(1) * time.Second)

	fmt.Println("Koniec symulacji")
	fmt.Println("")
	fmt.Println("Historia wieszchołków")

	for i := 0; i < n; i++ {
		vertex := g.vertices[i]
		fmt.Println("Wieszchołek :", vertex.ID)

		for _, id := range vertex.node.log {
			if vertex.ID == 0 {
				fmt.Printf("\t    %d =>\n", id)
			} else if vertex.ID == n-1 {
				fmt.Printf("\t => %d\n", id)
			} else {
				fmt.Printf("\t => %d =>\n", id)
			}
		}
	}

	fmt.Println("")
	fmt.Println("Historia pakietów")

	for i := 0; i < k; i++ {
		message := g.messages[i]
		fmt.Println("Pakiet :", message.ID)

		for _, id := range message.log {
			fmt.Printf("\t%d", id)

			switch id {
			case -1:
				fmt.Printf("\tUsunięto przez blokadę sieci\n")
			case -2:
				fmt.Printf("\tPakietowi skończyło się życie\n")
			case -3:
				fmt.Printf("\tPakiet wpadł w pułapkę\n")
			default:
				fmt.Printf("\n")
			}
		}
	}

}

func print_loop(printer_in chan string) {
	for {
		message := <-printer_in

		fmt.Println(message)
	}
}

func sender(num_of_messages_to_send int, me *GraphVertex, graph *Graph, max_life int) {
	i := 0
	for {
		to_sleep := rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		message_id := 0

		select {
		case message_id = <-me.node.in_chanel:
		case <-time.After(300 * time.Millisecond):
			if i != num_of_messages_to_send {
				graph.messages[i] = createMessage(i, max_life)
				message_id = i
				i += 1

				me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został wysłany"
			} else {
				continue
			}
		}

		select {
		case <-me.node.trap_chanel:
			if me.node.print_chanel != nil {
				me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został złapany w pułapkę"
			}

			graph.messages[message_id].log = append(graph.messages[message_id].log, -3)
			graph.destroyed_messages++
			continue
		case <-time.After(2 * time.Millisecond):
		}

		if me.node.print_chanel != nil {
			me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " jest w wierzchołku " + strconv.Itoa(me.ID)
		}

		to_sleep = rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		to_send := rand.Intn(len(me.edges))
		for key := range me.edges {
			if to_send == 0 {
				to_send = key
				break
			}

			to_send--
		}

		me.node.log = append(me.node.log, message_id)

		graph.messages[message_id].log = append(graph.messages[message_id].log, me.ID)
		graph.messages[message_id].life -= 1
		if graph.messages[message_id].life != 0 {
			graph.graph_map[me.ID] = to_send
			if graph.watchdog_chan != nil {
				graph.watchdog_chan <- WatchdogMessage{me.ID, true}
			}
			graph.vertices[to_send].node.in_chanel <- message_id
			if graph.watchdog_chan != nil {
				graph.watchdog_chan <- WatchdogMessage{me.ID, false}
			}
			graph.graph_map[me.ID] = -1
		} else {
			if me.node.print_chanel != nil {
				me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został zniszczony"
			}
			graph.messages[message_id].log = append(graph.messages[message_id].log, -2)

			graph.destroyed_messages++
		}
	}

}

func reciver(num_of_messages int, me *GraphVertex, graph *Graph) {
	for {
		to_sleep := rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		message_id := 0

		message_id = <-me.node.in_chanel

		select {
		case <-me.node.trap_chanel:
			if me.node.print_chanel != nil {
				me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został złapany w pułapkę"
			}

			graph.messages[message_id].log = append(graph.messages[message_id].log, -3)
			graph.destroyed_messages++
			continue
		case <-time.After(2 * time.Millisecond):
		}

		if me.node.print_chanel != nil {
			me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " jest w wierzchołku " + strconv.Itoa(me.ID)
		}

		to_sleep = rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		to_send := rand.Intn(len(me.edges) + 1)

		if to_send == len(me.edges) {
			graph.messages[message_id].log = append(graph.messages[message_id].log, me.ID)
			me.node.log = append(me.node.log, message_id)
			graph.destroyed_messages++

			me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został odebrany"
		} else {
			for key := range me.edges {
				if to_send == 0 {
					to_send = key
					break
				}

				to_send--
			}

			me.node.log = append(me.node.log, message_id)

			graph.messages[message_id].log = append(graph.messages[message_id].log, me.ID)
			graph.messages[message_id].life -= 1
			if graph.messages[message_id].life != 0 {
				graph.graph_map[me.ID] = to_send
				if graph.watchdog_chan != nil {
					graph.watchdog_chan <- WatchdogMessage{me.ID, true}
				}
				graph.vertices[to_send].node.in_chanel <- message_id
				if graph.watchdog_chan != nil {
					graph.watchdog_chan <- WatchdogMessage{me.ID, false}
				}
				graph.graph_map[me.ID] = -1
			} else {
				if me.node.print_chanel != nil {
					me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został zniszczony"
				}
				graph.messages[message_id].log = append(graph.messages[message_id].log, -2)

				graph.destroyed_messages = graph.destroyed_messages - 1
			}
		}
	}
}

func node_loop(me *GraphVertex, graph *Graph) {
	for {
		to_sleep := rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		message_id := <-me.node.in_chanel
		select {
		case <-me.node.trap_chanel:
			if me.node.print_chanel != nil {
				me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został złapany w pułapkę"
			}

			graph.messages[message_id].log = append(graph.messages[message_id].log, -3)
			graph.destroyed_messages++
			continue
		case <-time.After(2 * time.Millisecond):
		}

		if me.node.print_chanel != nil {
			me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " jest w wierzchołku " + strconv.Itoa(me.ID)
		}

		to_sleep = rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		to_send := rand.Intn(len(me.edges))
		for key := range me.edges {
			if to_send == 0 {
				to_send = key
				break
			}

			to_send--
		}

		me.node.log = append(me.node.log, message_id)

		graph.messages[message_id].log = append(graph.messages[message_id].log, me.ID)
		graph.messages[message_id].life -= 1
		if graph.messages[message_id].life != 0 {
			graph.graph_map[me.ID] = to_send
			if graph.watchdog_chan != nil {
				graph.watchdog_chan <- WatchdogMessage{me.ID, true}
			}
			graph.vertices[to_send].node.in_chanel <- message_id
			if graph.watchdog_chan != nil {
				graph.watchdog_chan <- WatchdogMessage{me.ID, false}
			}
			graph.graph_map[me.ID] = -1
		} else {
			if me.node.print_chanel != nil {
				if me.node.print_chanel != nil {
					me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został zniszczony"
				}
				graph.messages[message_id].log = append(graph.messages[message_id].log, -2)

				graph.destroyed_messages++
			}
		}
	}
}

func watchdog(graph *Graph, watchdog_channel chan WatchdogMessage, printer chan string, end_channel chan bool, num_of_messages int) {
	vetex_states := make([]bool, graph.size())
	vetex_times := make([]time.Time, graph.size())
	last_update := time.Now()
	for index := range vetex_times {
		vetex_times[index] = time.Now()
		vetex_states[index] = false
	}

	isVertexRecivingMesssage := func(vertex_id int) bool {
		for _, v := range graph.graph_map {
			if v == vertex_id {
				return true
			}
		}

		return false
	}

	for {
		select {
		case message := <-watchdog_channel:
			vetex_times[message.vertex_id] = time.Now()
			vetex_states[message.vertex_id] = message.sending
			last_update = time.Now()
		case <-time.After(10 * time.Millisecond):
			if time.Now().Sub(last_update) > 5*time.Second {
				for id := range graph.vertices {
					if time.Now().Sub(vetex_times[id]) > 4*time.Second && vetex_states[id] == true &&
						isVertexRecivingMesssage(id) {
						printer <- "Deadlock detected"
						printer <- "Cleaning locked node: " + strconv.Itoa(id)

						reciver := graph.graph_map[id]
						printer <- "Deadlock detected to: " + strconv.Itoa(graph.graph_map[id])
						select {
						case message := <-graph.vertices[reciver].node.in_chanel:
							graph.messages[message].log = append(graph.messages[message].log, -1)
							printer <- "Message deleted: " + strconv.Itoa(message)

							graph.destroyed_messages++

							for index := range vetex_times {
								vetex_times[index] = time.Now()
							}
							last_update = time.Now()

						case <-time.After(10 * time.Millisecond):
							printer <- "Wathdog error"
						}

						break
					}
				}

				if graph.destroyed_messages == num_of_messages {
					end_channel <- true
				}
			}
		}
	}
}

func trapLoop(graph *Graph, delay int) {
	for {
		time.Sleep(time.Duration(delay) * time.Millisecond)

		vertex_id := rand.Intn(graph.size())

		select {
		case graph.vertices[vertex_id].node.trap_chanel <- true:
		case <-time.After(10 * time.Millisecond):
		}

	}
}

func generateGraph(n, d int, b int) Graph {
	g := MakeGraph()

	printer := make(chan string)

	for i := 0; i < n; i++ {
		g.addVertex()

		g.vertices[i].node.SetDelay(300, 800)
		g.vertices[i].node.SetPrinter(printer)
	}

	for i := 0; i < n-1; i++ {
		g.addEdge(i, i+1)
	}

	for ; d > 0; d-- {
		from, to := 0, 0

		for done := false; !done; done = g.addEdge(from, to) {
			from = rand.Intn(g.size() - 1)
			to = rand.Intn(g.size()-from-1) + from + 1
		}
	}

	for ; b > 0; b-- {
		from, to := 0, 0

		for done := false; !done; done = g.addEdge(from, to) {
			from = rand.Intn(g.size())

			if from == 0 {
				from, to = 0, 0
				continue
			}

			to = rand.Intn(from)
		}
	}

	go print_loop(printer)

	return g
}
