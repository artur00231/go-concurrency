package main

import "fmt"

type GraphVertex struct {
	ID int

	edges map[int]bool
	node  Node
}

type WatchdogMessage struct {
	vertex_id int
	sending   bool
}

type Graph struct {
	vertices map[int]*GraphVertex

	avaiable_name int
	messages      map[int]*Message
	graph_map     []int

	watchdog_chan      chan WatchdogMessage
	destroyed_messages int
}

func MakeGraph() Graph {
	g := Graph{make(map[int]*GraphVertex), 0, make(map[int]*Message), make([]int, 0), nil, 0}

	return g
}

func (g *Graph) addVertex() int {
	name := g.avaiable_name
	g.avaiable_name++
	vertex := GraphVertex{name, make(map[int]bool), createNode(name)}

	g.vertices[name] = &vertex
	g.graph_map = append(g.graph_map, 0)

	return name
}

func (g *Graph) addEdge(from, to int) bool {
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

	return true
}

func (g *Graph) size() int {
	return g.avaiable_name
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
			fmt.Printf("\n\t=> (%d)", ID)
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
		from int
		used bool
		wait bool
	}

	draw_edges := func(levels *[]Level, on_vertex bool, vertex_id int, g *Graph) {
		max := func(a, b int) int {
			if b > a {
				return b
			}
			return a
		}

		line := make([]rune, 3*len(*levels))
		clear_line := true
		connected := false
		fill_to_level := 0
		for i := 0; i < 3*len(*levels); i++ {
			line[i] = ' '
		}

		if on_vertex {
			for index, level := range *levels {
				if level.wait {
					(*levels)[index].wait = false
				}

				if level.used {
					line[3*index] = '|'
					clear_line = false
				}
			}

			for index, level := range *levels {
				if level.used && level.to == vertex_id {
					(*levels)[index].used = false
					(*levels)[index].wait = true
					line[3*index] = 'v'
					clear_line = false
					fill_to_level = max(index, fill_to_level)
					connected = true
				} else if level.used && level.from == vertex_id && level.to < vertex_id {
					(*levels)[index].used = false
					(*levels)[index].wait = true
					line[3*index] = '*'
					clear_line = false
					fill_to_level = max(index, fill_to_level)
					connected = true
				}
			}

			for id := range g.vertices[vertex_id].edges {
				if id == vertex_id+1 {
					continue
				}
				if id < vertex_id {
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
					*levels = append(*levels, Level{id, vertex_id, true, false})
					line = append(line, '*', ' ', ' ')
					clear_line = false
					fill_to_level = max(i, fill_to_level)
					connected = true

				} else {
					(*levels)[i].to = id
					(*levels)[i].from = vertex_id
					(*levels)[i].used = true
					line[3*i] = '*'
					fill_to_level = max(i, fill_to_level)
					connected = true
				}
			}

			for from_vertex := range g.vertices {
				if from_vertex > vertex_id {
					if _, ok := g.vertices[from_vertex].edges[vertex_id]; ok {
						i := 0
						for _, level := range *levels {
							if !level.used && !level.wait {
								break
							}
							i++
						}

						if i == len(*levels) {
							*levels = append(*levels, Level{vertex_id, from_vertex, true, false})
							line = append(line, '^', ' ', ' ')
							clear_line = false
							fill_to_level = max(i, fill_to_level)
							connected = true

						} else {
							(*levels)[i].to = vertex_id
							(*levels)[i].from = from_vertex
							(*levels)[i].used = true
							line[3*i] = '^'
							fill_to_level = max(i, fill_to_level)
							connected = true
						}
					}
				}
			}

			for i := 0; i < 3*fill_to_level; i++ {
				if line[i] == ' ' {
					line[i] = '-'
				}
			}
		} else {
			for index, level := range *levels {
				if level.used {
					line[3*index] = '|'
					clear_line = false
				}
			}
		}

		if !clear_line {
			if connected {
				fmt.Print("--", string(line))
			} else {
				fmt.Print("  ", string(line))
			}
		}
		fmt.Println("")
	}

	fmt.Printf("#Vertices = %d; #Edges = %d\n", len(g.vertices), edges_count)

	leves := make([]Level, 0)

	for i := 0; i < g.size()-1; i++ {
		fmt.Printf("(%3d) *", i)
		draw_edges(&leves, true, i, g)
		fmt.Print("      |")
		draw_edges(&leves, false, i, g)
		fmt.Print("      |")
		draw_edges(&leves, false, i, g)
		fmt.Print("      |")
		draw_edges(&leves, false, i, g)
	}

	fmt.Printf("(%3d) *", g.size()-1)
	draw_edges(&leves, true, g.size()-1, g)
}
