package license

type Hooks struct {
	onBlocked          func()
	onHeartbeatError   func(error)
	onGracePeriodStart func()
}

func (s *SDK) OnBlocked(callback func()) {
	s.hooks.onBlocked = callback
}

func (s *SDK) OnHeartbeatError(callback func(error)) {
	s.hooks.onHeartbeatError = callback
}

func (s *SDK) OnGracePeriodStart(callback func()) {
	s.hooks.onGracePeriodStart = callback
}
