package fakes

import "sync"

type InstallProcess struct {
	ExecuteCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			WorkingDir string
		}
		Returns struct {
			Error error
		}
		Stub func(string) error
	}
}

func (f *InstallProcess) Execute(param1 string) error {
	f.ExecuteCall.Lock()
	defer f.ExecuteCall.Unlock()
	f.ExecuteCall.CallCount++
	f.ExecuteCall.Receives.WorkingDir = param1
	if f.ExecuteCall.Stub != nil {
		return f.ExecuteCall.Stub(param1)
	}
	return f.ExecuteCall.Returns.Error
}
