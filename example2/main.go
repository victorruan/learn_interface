package main

/*
类型断言-非空接口
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
	switch p.(type) {
	case *Chinese:
		chinese := p.(*Chinese)
		chinese.GetName()
	}
}
