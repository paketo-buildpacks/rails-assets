package fakes

import "sync"

type EnvironmentSetup struct {
	RunCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			LayerPath  string
			WorkingDir string
		}
		Returns struct {
			Error error
		}
		Stub func(string, string) error
	}
}

func (f *EnvironmentSetup) Run(param1 string, param2 string) error {
	f.RunCall.Lock()
	defer f.RunCall.Unlock()
	f.RunCall.CallCount++
	f.RunCall.Receives.LayerPath = param1
	f.RunCall.Receives.WorkingDir = param2
	if f.RunCall.Stub != nil {
		return f.RunCall.Stub(param1, param2)
	}
	return f.RunCall.Returns.Error
}
