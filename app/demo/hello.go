package demo

type Hello struct{}

func (g *Hello) Greet(name string) string {
	return "Hello " + name + "!"
}
