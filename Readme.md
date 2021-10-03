[原文地址](https://draveness.me/golang/docs/part2-foundation/ch04-basic/golang-interface) 

版本: GO1.17

# 接口
Go语言中的接口，是一组方法的签名，它是Go语言的重要组成部分。使用接口能让我们写出易于测试的代码。
然而很多工程师对Go接口的了解都非常有限，也不清楚其底层原理的实现。这成为了开发高性能服务的阻碍。

本文会介绍使用接口时遇到的一些常见问题，以及它的设计与实现，包括接口的类型转换、类型断言以及动态派发机制，
帮助读者更好地理解接口类型。

## 概述
在计算机科学中，接口是多个组件共享的边界，不同的组件能在边界上交换信息。
如下图所示，接口的本质是引入一个新的中间层。调用方可以通过接口与具体的实现分离。
接触上下游的耦合，上层的模块不在需要依赖下层的具体模块。只需要依赖一个约定好的的接口。

```
                   GOLANG INTERFACE
                                             ┌────────┐
                            ┌───────────────►│ module │
                            │                └────────┘
                            │
┌────────┐        ┌─────────┴─┐              ┌────────┐
│ module ├───────►│ interface ├─────────────►│ module │
└────────┘        └─────────┬─┘              └────────┘
                            │
                            │                ┌────────┐
                            └───────────────►│ module │
                                             └────────┘
```
**图1 上下游通过接口解耦**

这种面向接口的编程方式有着强大的生命力，无论是在框架中，还是在操作系统中，都能看到接口的身影。
可移植操作系统接口（Portable Operating System Interface, POSIX）就是一个典型的例子，
它定义了应用程序接口和命令行等标准，为计算机软件带来了可移植性，只要操作系统实现了POSIX,
计算机软件就可以在不同操作系统上运行。

除了解耦有依赖关系的上下游，接口还能帮助我们隐藏底层实现，减少关注点。
人能够同时处理的信息非常有限，定义良好的接口能够隔离底层实现，让我们将重点放在当前的代码片段中。
SQL就是接口的一个例子。当我们使用SQL查询数据时，其实不需要关心底层数据库的具体实现，
我们只在乎SQL返回的结果是否符合预期。

```
                   SQL AND DATABASE
                                             ┌────────┐
                            ┌───────────────►│ MYSQL  │
                            │                └────────┘
                            │
                  ┌─────────┴─┐              ┌────────┐
                  │     SQL   ├─────────────►│ SQLITE │
                  └─────────┬─┘              └────────┘
                            │
                            │                ┌────────────┐
                            └───────────────►│ POSTGRESQL │
                                             └────────────┘
```
**图2  SQL和不同数据库**

### 类型
接口也是GO语言中的一种类型，它能够出现在变量的定义，函数的入参和放回值上。
GO语言中有两种略微不同的接口，一种是带一组方法的接口，另一种是不带任何方法的接口。
```
                   GOLANG DIFFERENT INTERFACE
                           
                  ┌─────────┐              ┌────────┐
                  │   iface │              │ eface  │
                  └─────────┘              └────────┘
                            
```
**图3 Go 语言中的两种接口**

Go语言使用`runtime.iface`表示带有一组方法的接口，使用`runtime.eface`表示不带任何方法的接口。
需要注意的是`interface{}`不是任意类型，如果我们将类型转换成了`interface{}`类型，
变量在运行期间的类型也会发生变化。

我们可以通过一个例子理解`Go 语言的接口类型不是任意类型`这一句话，下面的代码在 main 函数中初始化了一个 *Test 类型的变量，由于指针的零值是 nil，所以变量 s 在初始化之后也是 nil
```go
package main

type Test struct{}

func main() {
	var v *Test
	println(v == nil) // true
	var i interface{} = v
	println(i == nil) // false
}
```
由此可见，变量的赋值会触发隐式类型转换，在类型转换时，`*Test`会被转换成`interface{}`
转换后的变量，不仅包含转换前的变量，还包含变量的类型信息。所以转换后的变量不等于`nil`


## 数据结构
我们从源代码和汇编的角度分析一下接口的底层数据结构。
Go语言根据接口是否包含一组方法，将接口分为两类：
1. 使用`runtime.iface`表示包含方法的接口
2. 使用`runtime.eface`表示不包含方法的接口

`runtime.eface`在Go语言中的定义如下:
```go
type eface struct { // 16 字节
	_type *_type
	data  unsafe.Pointer
}
```
这个结构相对简单，只包含类型和数据，从上述结构我们能推断出
Go语言的任意类型能转都能换成 `runtime.eface`

另一个用于表示接口的结构体是`runtime.iface`,这个结构体也有指向原始数据的指针 `data`
不过更重要的是`runtime.itab`类型的`tab`字段
```go
type iface struct { // 16 字节
	tab  *itab
	data unsafe.Pointer
}
```
接下来我们将分析Go语言中的这两个接口类型

### 类型结构体

`runtime._type`是Go语言类型的运行时表示，下面是runtime包中的结构体，
其中包含了很多类型的元信息，例如类型的大小、哈希、对齐以及种类等
```go
type _type struct {
	size       uintptr // 存储了类型的占用空间，为内存空间的分配提供信息
	ptrdata    uintptr
	hash       uint32  // 字段能够帮助我们快速确定类型是否相等
	tflag      tflag
	align      uint8
	fieldAlign uint8
	kind       uint8
	equal      func(unsafe.Pointer, unsafe.Pointer) bool // 字段用于判断当前类型的多个对象是否相等
	gcdata     *byte
	str        nameOff
	ptrToThis  typeOff
}
```
我们只需要对`runtime._type`结构体中的字段有个大体的概念，不需要详细理解每个字段的作用和意义。

### itab 结构体
`runtime.itab`结构体是接口类型的核心组成部分，共占32字节，我们可以把其看成是接口类型和具体类型的组合
`inter`字段表示接口类型,`_type`字段表示具体类型
```go
type itab struct { // 32 字节
	inter *interfacetype
	_type *_type
	hash  uint32
	_     [4]byte
	fun   [1]uintptr
}
```
除了`inter`和`_type`两个字段外，上述结构体的另外两个字段也有自己的作用：
- `hash`是对`_type.hash`的拷贝，当我们想将`interface`类型转成具体类型时,可以使用该字段快速判断目标类型和具体类型的`runtime._type`是否一致
- `fun`是一组方法的首地址,配合`inter`中的方法集使用，可以方便的定位到`_type`实现的方法地址

我们会在类型断言中介绍`hash`的使用，在动态派发中介绍`fun`的使用

## 类型断言
本节会根据接口是否包含方法分两种情况介绍类型断言的执行过程。

### 非空接口

首先分析非空接口，`Person`是一个包含方法的非空接口，我们来分析从
`Person`转换回`Chinese`结构体的过程
```go
func main() {
	var p Person = &Chinese{Name: "chinese"}
	switch p.(type) {
	case *Chinese:
		chinese := p.(*Chinese)
		chinese.GetName()
	}
}
```
我们将编译得到的汇编指令分成两部分，第一部分是变量的初始化，第二部门是
类型断言。
第一部分代码如下：
```
	 00000 	TEXT	"".main(SB), ABIInternal, $136-0
	 ......
	 00038	MOVUPS	X15, ""..autotmp_6+112(SP)               ;; 清空(112-128)(SP)
	 ......
	 00056	MOVQ	$7, ""..autotmp_6+120(SP)                ;; +120(SP) = 7
	 00065	LEAQ	go.string."chinese"(SB), SI              ;; SI = &("chinese")
	 00072	MOVQ	SI, ""..autotmp_6+112(SP)                ;; +112(SP) = SI = &("chinese")
	 ......
	 00082 	LEAQ	go.itab.*"".Chinese,"".Person(SB), SI    ;; SI = &(go.itab.*"".Chinese,"".Person(SB))
	 00089 	MOVQ	SI, "".p+80(SP)                          ;; +80(SP) = SI = &(go.itab.*"".Chinese,"".Person(SB))
	 00044 	LEAQ	""..autotmp_6+112(SP), DX                ;; DX = &(+112(SP))
	 00094 	MOVQ	DX, "".p+88(SP)                          ;; +88(SP) = DX = &(+112(SP))
```
上面的代码初始化了`Person`变量，`Chinese`结构体初始化在 **(112-128)(SP)** 上。
(112-120)(SP)上存的是 go.string."chinese"(SB) .也就是字符串"chinese"的地址
(120-128)(SP)上存的是 长度7
`Person`变量初始化在 **(80-96)(SP)** 上。

下面进入类型转换的部分：

```
	00099 	MOVQ	"".p+80(SP), DX        ;; DX = &(go.itab.*"".Chinese,"".Person(SB))
	00104 	MOVQ	"".p+88(SP), SI        ;; SI = &(+112(SP)) = "chinese"
	00126 	MOVL	16(DX), DX             ;; DX = &(go.itab.*"".Chinese,"".Person(SB)).hash
	00133 	CMPL	DX, $-1430607797       ;; if (p.hash == *"".Chinese.hash) 
```
switch语句生成的汇编指令会将目标类型的 hash 与接口变量中的 itab.hash 进行比较：
如果二者相等，说明断言成功,可以走入分支，如果不相等，说明p变量不是*Chinese类型。

### 空接口
当我们使用空接口类型 interface{} 进行类型断言时,编译器从 eface._type 中获取，汇编指令仍然会使用目标类型的 hash 与变量的类型比较
```go
func main() {
	var p interface{} = &Chinese{Name: "chinese"}
	switch p.(type) {
	case *Chinese:
		chinese := p.(*Chinese)
		chinese.GetName()
	}
}
```

## 动态派发
动态派发是在运行期间选择具体方法执行的过程。调用接口类型的方法时，如果编译期不能确认接口的类型，Go语言会在运行期决定调用该方法的哪个实现。
```go
func main() {
	var p Person = &Chinese{Name: "chinese"}
	PrintName(p)
}

func PrintName(p Person) {
	name1 := p.GetName()
	fmt.Println(name1)
}
```
主要来看动态派发的过程
```
	00000 	TEXT	"".PrintName(SB), ABIInternal, $208-16
	00038 	MOVQ	AX, "".p+216(SP)        ;; "".p+216(SP) = iface.tab
	00046 	MOVQ	BX, "".p+224(SP)        ;; "".p+224(SP) = iface.data
	00056 	MOVQ	24(AX), CX              ;; CX = iface(p).tab.fun[0] = *Chinese.GetName
	00060 	MOVQ	BX, AX                  ;; AX = iface(p).data = (&Chinese{Name: "chinese"})
	00064 	CALL	CX                      ;; (&Chinese{Name: "chinese"}).GetName()
```

PrintName函数接受参数为Person接口 `p` ,也就是一个iface结构体实例，根据1.17的调用规约，
寄存器AX,BX分别存的是iface.tab以及iface.data, 【00056】的24(AX) 是iface.tab.fun[0]
【00064】实际就是接口方法真实调用的地方。
至于【00038】【00046】为什么要把参数存起来，是因为调用接口方法后，返回值会覆盖AX,BX的值。
