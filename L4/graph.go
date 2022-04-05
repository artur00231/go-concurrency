package main

import (
	"fmt"
	"sync"
)

type Message struct {
	from int
	to   int
	cost int
}

type Address struct {
	node int
	id   int
}

type DataMessage struct {
	from Address
	to   Address

	via []int
}

type HostData struct {
	reciver    Address
	in_channel chan DataMessage
}

type RoutingTableMessage struct {
	op      int
	from    int
	to      int
	cost    int
	changed bool
}

type GraphVertex struct {
	ID int

	edges map[int]bool
	costs map[int]int

	nexthop         []int
	nexthop_changed []bool
	nexthop_cost    []int

	in_channel        chan Message
	routing_table_in  chan RoutingTableMessage
	routing_table_out chan RoutingTableMessage

	hosts []HostData

	forwarder_chan  chan DataMessage
	forwarder_queue []DataMessage
	forwarder_lock  sync.Mutex
}

type Graph struct {
	vertices map[int]*GraphVertex

	avaiable_name int
}

func MakeGraph() Graph {
	g := Graph{make(map[int]*GraphVertex), 0}

	return g
}

func (g *Graph) addVertex() int {
	name := g.avaiable_name
	g.avaiable_name++
	vertex := GraphVertex{name, make(map[int]bool), make(map[int]int), make([]int, 0),
		make([]bool, 0), make([]int, 0), make(chan Message, 0), make(chan RoutingTableMessage, 0), make(chan RoutingTableMessage, 0), make([]HostData, 0),
		make(chan DataMessage, 0), make([]DataMessage, 0), sync.Mutex{}}

	g.vertices[name] = &vertex

	return name
}

func (g *Graph) addEdge(from, to int, cost int) bool {
	if from == to {
		return false
	}

	if from >= g.avaiable_name || to >= g.avaiable_name {
		return false
	}

	_, exists := g.vertices[from].edges[to]
	if exists {
		return false
	}

	g.vertices[from].edges[to] = true
	g.vertices[to].edges[from] = true
	g.vertices[from].costs[to] = cost
	g.vertices[to].costs[from] = cost

	return true
}

func (g *Graph) size() int {
	return g.avaiable_name
}

func (g *Graph) resetHopTables() {
	for i := 0; i < g.size(); i++ {
		g.vertices[i].nexthop = make([]int, g.size())
		g.vertices[i].nexthop_cost = make([]int, g.size())
		g.vertices[i].nexthop_changed = make([]bool, g.size())
	}

	for i := 0; i < g.size(); i++ {
		for j := 0; j < g.size(); j++ {
			if i == j {
				continue
			}

			_, exists := g.vertices[i].edges[j]
			if exists {
				g.vertices[i].nexthop[j] = j
				g.vertices[i].nexthop_cost[j] = g.vertices[i].costs[j]
			} else {
				if j < i {
					g.vertices[i].nexthop[j] = i - 1
				} else {
					g.vertices[i].nexthop[j] = i + 1
				}

				//g.vertices[i].nexthop_cost[j] = math.MaxInt64
				g.vertices[i].nexthop_cost[j] = i - j
				if g.vertices[i].nexthop_cost[j] < 0 {
					g.vertices[i].nexthop_cost[j] = -g.vertices[i].nexthop_cost[j]
				}
			}

			g.vertices[i].nexthop_changed[j] = true
		}
	}
}

func (g *Graph) print() {

	edges_count := 0
	for _, vertex := range g.vertices {
		edges_count += len(vertex.edges)
	}

	fmt.Printf("#Vertices = %d; #Edges = %d", len(g.vertices), edges_count)

	for i := 0; i < g.size(); i++ {
		fmt.Printf("\n(%d)", i)

		for ID := range g.vertices[i].edges {
			fmt.Printf("\n\t=> (%d) || %d", ID, g.vertices[i].costs[ID])
		}

		fmt.Printf("\n\t\t #Hosts:(%d)", len(g.vertices[i].hosts))

		for id, host := range g.vertices[i].hosts {
			fmt.Printf("\n\t\t Host:(%d) wysyła do (%d, %d)", id, host.reciver.node, host.reciver.id)
		}

		fmt.Println("")
	}

	fmt.Println("")
}

func (g *Graph) printf() {

	edges_count := 0
	for _, vertex := range g.vertices {
		edges_count += len(vertex.edges)
	}

	type Level struct {
		to   int
		cost int
		used bool
		wait bool
	}

	draw_edges := func(levels *[]Level, on_vertex bool, print_cost bool, vertex_id int, g *Graph) {
		max := func(a, b int) int {
			if b > a {
				return b
			}
			return a
		}

		line := make([]rune, 4*len(*levels))
		clear_line := true
		connected := false
		fill_to_level := 0
		for i := 0; i < 4*len(*levels); i++ {
			line[i] = ' '
		}

		if on_vertex {
			for index, level := range *levels {
				if level.wait {
					(*levels)[index].wait = false
				}

				if level.used {
					line[4*index] = '|'
					clear_line = false
				}
			}

			for index, level := range *levels {
				if level.used && level.to == vertex_id {
					(*levels)[index].used = false
					(*levels)[index].wait = true
					line[4*index] = '*'
					clear_line = false
					fill_to_level = max(index, fill_to_level)
					connected = true
				}
			}

			for id := range g.vertices[vertex_id].edges {
				if id == vertex_id+1 {
					continue
				}
				if id <= vertex_id {
					continue
				}

				i := 0
				for _, level := range *levels {
					if !level.used && !level.wait {
						break
					}
					i++
				}

				if i == len(*levels) {
					*levels = append(*levels, Level{id, g.vertices[vertex_id].costs[id], true, false})
					line = append(line, '*', ' ', ' ', ' ')
					clear_line = false
					fill_to_level = max(i, fill_to_level)
					connected = true

				} else {
					(*levels)[i].to = id
					(*levels)[i].used = true
					(*levels)[i].cost = g.vertices[vertex_id].costs[id]
					line[4*i] = '*'
					fill_to_level = max(i, fill_to_level)
					connected = true
				}
			}

			for i := 0; i < 4*fill_to_level; i++ {
				if line[i] == ' ' {
					line[i] = '-'
				}
			}
		} else {
			for index, level := range *levels {
				if level.used {
					if print_cost {
						cost := fmt.Sprint(level.cost)
						for i := 0; i < 4; i++ {
							if len(cost) > i {
								line[4*index+i] = rune(cost[i])
							}
						}
					} else {
						line[4*index] = '|'
					}
					clear_line = false
				}
			}
		}

		if clear_line {
			line = make([]rune, 0)
		}

		if connected {
			fmt.Print("*---", string(line))
		} else {
			if print_cost {
				cost := fmt.Sprint(g.vertices[vertex_id].costs[vertex_id+1])
				for i := 0; i < 4; i++ {
					if len(cost) > i {
						fmt.Print(string(cost[i]))
					} else {
						fmt.Print(" ")
					}
				}
			} else {
				if on_vertex {
					fmt.Print("*   ")
				} else {
					fmt.Print("|   ")
				}

			}

			fmt.Print(string(line))
		}
		fmt.Println("")
	}

	fmt.Printf("#Vertices = %d; #Edges = %d\n", len(g.vertices), edges_count)

	leves := make([]Level, 0)

	for i := 0; i < g.size()-1; i++ {
		fmt.Printf("(%3d) ", i)
		draw_edges(&leves, true, false, i, g)
		fmt.Print("      ")
		draw_edges(&leves, false, false, i, g)
		fmt.Print("      ")
		draw_edges(&leves, false, true, i, g)
		fmt.Print("      ")
		draw_edges(&leves, false, false, i, g)
	}

	fmt.Printf("(%3d) ", g.size()-1)
	draw_edges(&leves, true, false, g.size()-1, g)
}
