package circuit

import "sync"

// Panel manages a batch of circuitbreakers
type Panel struct {
	sync.Mutex
	breakers       map[string]*Breaker
	defaultOptions *Options
}

// NewPanel .
func NewPanel(defaultOptions *Options) (*Panel, error) {
	_, err := NewBreaker(defaultOptions)
	if err != nil {
		return nil, err
	}

	return &Panel{
		breakers:       make(map[string]*Breaker),
		defaultOptions: defaultOptions,
	}, nil
}

// GetBreaker .
func (p *Panel) GetBreaker(key string) *Breaker {
	p.Lock()
	cb, ok := p.breakers[key]
	p.Unlock()

	if ok {
		return cb
	}

	cb, _ = NewBreaker(p.defaultOptions)
	p.Lock()
	_, ok = p.breakers[key]
	if ok == false {
		p.breakers[key] = cb
	} else {
		cb = p.breakers[key]
	}
	p.Unlock()

	return cb
}

// DumpBreakers .
func (p *Panel) DumpBreakers() map[string]*Breaker {
	breakers := make(map[string]*Breaker)
	p.Lock()
	for k, b := range p.breakers {
		breakers[k] = b
	}
	p.Unlock()
	return breakers
}

// NewBreaker .
func (p *Panel) NewBreaker(key string, options *Options) error {
	cb, err := NewBreaker(options)
	if err != nil {
		return err
	}
	p.Lock()
	p.breakers[key] = cb
	p.Unlock()
	return nil
}

// Succeed .
func (p *Panel) Succeed(key string) {
	p.GetBreaker(key).Succeed()
}

// Fail .
func (p *Panel) Fail(key string) {
	p.GetBreaker(key).Fail()
}

// Timeout .
func (p *Panel) Timeout(key string) {
	p.GetBreaker(key).Timeout()
}

// Done .
func (p *Panel) Done(key string) {
	p.GetBreaker(key).Done()
}

// IsAllowed .
func (p *Panel) IsAllowed(key string) bool {
	return p.GetBreaker(key).IsAllowed()
}
