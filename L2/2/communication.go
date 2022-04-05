package main

type Node struct {
	ID          int
	log         []int
	in_chanel   chan int
	trap_chanel chan bool

	delay_max    int
	delay_min    int
	print_chanel chan string
}

type Message struct {
	ID   int
	log  []int
	life int
}

func createNode(ID int) Node {
	return Node{ID, make([]int, 0), make(chan int, 0), make(chan bool, 1), 100, 0, nil}
}

func (node *Node) SetDelay(min, max int) {
	node.delay_max = max
	node.delay_min = min
}

func (node *Node) SetPrinter(print_chanel chan string) {
	node.print_chanel = print_chanel
}

func createMessage(ID int, max_life int) *Message {
	return &Message{ID, make([]int, 0), max_life}
}
