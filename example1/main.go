package main

type Test struct{}

func main() {
	var v *Test
	println(v == nil) // true
	var i interface{} = v
	println(i == nil) // false
}