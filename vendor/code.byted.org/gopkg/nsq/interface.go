package nsq

type Invoker interface {
	Succ()
	Err(msg *MsgEntry, err error)
}

type FooInvoker struct{}

func (fi *FooInvoker) Succ()                        {}
func (fi *FooInvoker) Err(msg *MsgEntry, err error) {}
