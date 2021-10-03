package main

/*
类型断言-空接口
*/

type Chinese struct {
	Name string
}

func (chinese *Chinese) GetName() string {
	return chinese.Name
}

func main() {
	var p interface{} = &Chinese{Name: "chinese"}
	switch p.(type) {
	case *Chinese:
		chinese := p.(*Chinese)
		chinese.GetName()
	}
}
