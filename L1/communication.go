package main

type Node struct {
	ID        int
	log       []int
	in_chanel chan int

	delay_max    int
	delay_min    int
	print_chanel chan string
}

type Message struct {
	ID  int
	log []int
}

func createNode(ID int) Node {
	return Node{ID, make([]int, 0), make(chan int, 1), 100, 0, nil}
}

func (node *Node) SetDelay(min, max int) {
	node.delay_max = max
	node.delay_min = min
}

func (node *Node) SetPrinter(print_chanel chan string) {
	node.print_chanel = print_chanel
}

func createMessage(ID int) *Message {
	return &Message{ID, make([]int, 0)}
}
