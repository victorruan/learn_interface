package main

import "fmt"

/*
动态派发
*/

type Person interface {
	GetName() string
}

type Chinese struct {
	Name string
}

func (chinese *Chinese) GetName() string {
	return chinese.Name
}

func main() {
	var p Person = &Chinese{Name: "chinese"}
	PrintName(p)
}

func PrintName(p Person) {
	name1 := p.GetName()
	fmt.Println(name1)
}
