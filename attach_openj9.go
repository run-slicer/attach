package attach

type openJ9VM struct {
	c conn
}

func (vm *openJ9VM) Close() error {
	//TODO implement me
	panic("implement me")
}

func (vm *openJ9VM) Load(agent string, options string) error {
	//TODO implement me
	panic("implement me")
}

func (vm *openJ9VM) LoadLibrary(path string, absolute bool, options string) error {
	//TODO implement me
	panic("implement me")
}

func (vm *openJ9VM) Properties() (map[string]string, error) {
	//TODO implement me
	panic("implement me")
}

func (vm *openJ9VM) ThreadDump() (string, error) {
	//TODO implement me
	panic("implement me")
}
