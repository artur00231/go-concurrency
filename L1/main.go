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

	rand.Seed(time.Now().Unix())

	if len(os.Args) != 4 {
		fmt.Print("Invalid argumants!")
		return
	}

	n, _ = strconv.Atoi(os.Args[1])
	d, _ = strconv.Atoi(os.Args[2])
	k, _ = strconv.Atoi(os.Args[3])

	g := generateGraph(n, d)

	g.print()
	g.printf()

	for i := 1; i < n-1; i++ {
		vertex := g.vertices[i]

		go node_loop(vertex, &g)
	}

	end_canel := make(chan bool, 0)

	sender_vertex := g.vertices[0]
	reciver_vertex := g.vertices[n-1]
	go sender(k, sender_vertex, &g)
	go reciver(k, reciver_vertex, &g, end_canel)

	<-end_canel

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
			fmt.Printf("\t%d\n", id)
		}
	}

}

func print_loop(printer_in chan string) {
	for {
		message := <-printer_in

		fmt.Println(message)
	}
}

func sender(mun_of_messages int, me *GraphVertex, graph *Graph) {
	for i := 0; i < mun_of_messages; i++ {
		to_sleep := rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		graph.messages[i] = createMessage(i)

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		to_send := rand.Intn(len(me.edges))
		for key := range me.edges {
			if to_send == 0 {
				to_send = key
				break
			}

			to_send--
		}

		if me.node.print_chanel != nil {
			me.node.print_chanel <- "pakiet " + strconv.Itoa(i) + " został wysłany"
		}

		me.node.log = append(me.node.log, i)

		graph.messages[i].log = append(graph.messages[i].log, me.ID)

		graph.vertices[to_send].node.in_chanel <- i
	}

}

func reciver(mun_of_messages int, me *GraphVertex, graph *Graph, end_chanel chan bool) {
	for i := 0; i < mun_of_messages; i++ {
		to_sleep := rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		message_id := <-me.node.in_chanel

		me.node.log = append(me.node.log, message_id)

		graph.messages[message_id].log = append(graph.messages[message_id].log, me.ID)

		if me.node.print_chanel != nil {
			me.node.print_chanel <- "pakiet " + strconv.Itoa(message_id) + " został odebrany"
		}
	}

	end_chanel <- true
}

func node_loop(me *GraphVertex, graph *Graph) {
	for {
		to_sleep := rand.Intn(me.node.delay_max-me.node.delay_min) + me.node.delay_min

		time.Sleep(time.Duration(to_sleep) * time.Millisecond)

		message_id := <-me.node.in_chanel

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

		graph.vertices[to_send].node.in_chanel <- message_id
	}
}

func generateGraph(n, d int) Graph {
	g := MakeGraph()

	printer := make(chan string)

	for i := 0; i < n; i++ {
		g.addVertex()

		g.vertices[i].node.SetPrinter(printer)
		g.vertices[i].node.SetDelay(400, 800)
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

	go print_loop(printer)

	return g
}
